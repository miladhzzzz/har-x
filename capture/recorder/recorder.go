// Package recorder drives a real Chrome instance over the DevTools Protocol
// and assembles a browser-accurate HAR from the live network traffic of an
// interactive session (the user drives the popup window; we just listen).
package recorder

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/persys-dev/harx/capture/harformat"
)

const maxCapturedBodyBytes = 2 * 1024 * 1024 // 2MB cap per body, avoids bloating the HAR

// inflight tracks one HTTP exchange from requestWillBeSent through completion.
type inflight struct {
	requestID network.RequestID
	pageRef   string

	req     *network.Request
	resp    *network.Response
	resType network.ResourceType

	wallTime time.Time // real clock time at request start (for startedDateTime)
	startAt  time.Time // monotonic clock at request start
	respAt   time.Time // monotonic clock when responseReceived fired
	endAt    time.Time // monotonic clock when the exchange finished

	bodyText string
	bodySize int64

	errText string
}

// Recorder accumulates network events for the lifetime of a browser tab.
// Safe for concurrent use; events arrive on chromedp's dispatch goroutine
// while Entries() may be called from the CLI's shutdown path.
type Recorder struct {
	mu      sync.Mutex
	active  map[network.RequestID]*inflight
	done    []*inflight
	pages   []harformat.Page
	curPage string
}

func New() *Recorder {
	return &Recorder{active: make(map[network.RequestID]*inflight)}
}

// Attach registers event listeners on the given browser-tab context. Call
// this once per target (tab) before navigation starts. It returns
// immediately; events are handled on chromedp's own goroutine as they arrive.
func (r *Recorder) Attach(ctx context.Context) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {

		case *page.EventFrameNavigated:
			if e.Frame == nil || e.Frame.ParentID != "" {
				return // only top-level (main frame) navigations count as a "page"
			}
			r.mu.Lock()
			id := fmt.Sprintf("page_%d", len(r.pages)+1)
			r.pages = append(r.pages, harformat.Page{
				ID:              id,
				Title:           e.Frame.URL,
				StartedDateTime: time.Now().Format(time.RFC3339Nano),
				PageTimings:     harformat.PageTimings{OnContentLoad: -1, OnLoad: -1},
			})
			r.curPage = id
			r.mu.Unlock()

		case *network.EventRequestWillBeSent:
			r.mu.Lock()
			// A redirect re-fires requestWillBeSent with the SAME request ID
			// and a populated RedirectResponse describing the *previous* hop.
			// Close that hop out using the redirect response, then start
			// tracking the new one under the same ID.
			if e.RedirectResponse != nil {
				if prev, ok := r.active[e.RequestID]; ok {
					prev.resp = e.RedirectResponse
					prev.endAt = monoTime(e.Timestamp)
					r.finalizeLocked(prev)
				}
			}
			r.active[e.RequestID] = &inflight{
				requestID: e.RequestID,
				pageRef:   r.curPage,
				req:       e.Request,
				resType:   e.Type,
				wallTime:  epochTime(e.WallTime),
				startAt:   monoTime(e.Timestamp),
			}
			r.mu.Unlock()

		case *network.EventResponseReceived:
			r.mu.Lock()
			if f, ok := r.active[e.RequestID]; ok {
				f.resp = e.Response
				f.respAt = monoTime(e.Timestamp)
				if f.resType == "" {
					f.resType = e.Type
				}
			}
			r.mu.Unlock()

		case *network.EventLoadingFinished:
			r.mu.Lock()
			f, ok := r.active[e.RequestID]
			if ok {
				f.endAt = monoTime(e.Timestamp)
			}
			r.mu.Unlock()
			if ok {
				r.fetchBodyAndFinalize(ctx, f)
			}

		case *network.EventLoadingFailed:
			r.mu.Lock()
			if f, ok := r.active[e.RequestID]; ok {
				f.endAt = monoTime(e.Timestamp)
				f.errText = e.ErrorText
				if f.resType == "" {
					f.resType = e.Type
				}
				r.finalizeLocked(f)
			}
			r.mu.Unlock()
		}
	})
}

// fetchBodyAndFinalize best-effort grabs the response body (skipped for
// non-text content, or when it's unavailable) then finalizes the entry.
// Runs off the event-dispatch goroutine so it never blocks capture.
func (r *Recorder) fetchBodyAndFinalize(ctx context.Context, f *inflight) {
	go func() {
		if f.resp != nil && isTextualMime(f.resp.MimeType) {
			var body []byte
			_ = chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
				b, err := network.GetResponseBody(f.requestID).Do(c)
				if err == nil {
					body = b
				}
				return nil // never fail the run over one missing body
			}))
			if len(body) > 0 {
				if len(body) > maxCapturedBodyBytes {
					body = body[:maxCapturedBodyBytes]
				}
				f.bodyText = string(body)
				f.bodySize = int64(len(body))
			}
		}
		r.mu.Lock()
		r.finalizeLocked(f)
		r.mu.Unlock()
	}()
}

// finalizeLocked moves an in-flight exchange to the completed list. Caller
// must hold r.mu. Guards against double-finalizing the same pointer (e.g. a
// redirect hop closed out both explicitly and via a stray later event).
func (r *Recorder) finalizeLocked(f *inflight) {
	if _, stillActive := r.active[f.requestID]; !stillActive {
		for _, d := range r.done {
			if d == f {
				return
			}
		}
	}
	delete(r.active, f.requestID)
	r.done = append(r.done, f)
}

// Entries converts every completed exchange into HAR entries, sorted by
// start time, along with the pages observed during the session.
func (r *Recorder) Entries() ([]harformat.Entry, []harformat.Page) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sort.SliceStable(r.done, func(i, j int) bool {
		return r.done[i].startAt.Before(r.done[j].startAt)
	})

	entries := make([]harformat.Entry, 0, len(r.done))
	for _, f := range r.done {
		e, err := toEntry(f)
		if err != nil {
			log.Printf("harx: skipping %s: %v", f.requestID, err)
			continue
		}
		entries = append(entries, e)
	}
	return entries, r.pages
}

// --- conversion helpers ---

func monoTime(t *cdp.MonotonicTime) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.Time()
}

func epochTime(t *cdp.TimeSinceEpoch) time.Time {
	if t == nil {
		return time.Now()
	}
	return t.Time()
}

func toEntry(f *inflight) (harformat.Entry, error) {
	if f.req == nil {
		return harformat.Entry{}, fmt.Errorf("no request data")
	}

	totalMS := durMS(f.startAt, f.endAt)
	if totalMS < 0 {
		totalMS = 0
	}

	entry := harformat.Entry{
		PageRef:         f.pageRef,
		StartedDateTime: f.wallTime.Format(time.RFC3339Nano),
		Time:            totalMS,
		Request:         buildRequest(f.req),
		Cache:           harformat.Cache{},
		ResourceType:    string(f.resType),
	}

	if f.errText != "" {
		entry.Failed = f.errText
	}

	if f.resp != nil {
		entry.Response = buildResponse(f.resp, f.bodyText, f.bodySize)
		entry.ServerIPAddress = f.resp.RemoteIPAddress
		entry.Timings = buildTimings(f.resp.Timing, f.startAt, f.respAt, totalMS)
	} else {
		// No response at all (blocked/aborted/DNS failure, etc).
		entry.Response = harformat.Response{Status: 0, StatusText: entry.Failed}
		entry.Timings = harformat.Timings{Blocked: -1, DNS: -1, Connect: -1, SSL: -1, Send: 0, Wait: totalMS, Receive: 0}
	}

	return entry, nil
}

func buildRequest(req *network.Request) harformat.Request {
	headers := headersToNV(req.Headers)
	r := harformat.Request{
		Method:      req.Method,
		URL:         req.URL,
		HTTPVersion: "HTTP/1.1",
		Cookies:     []harformat.Cookie{},
		Headers:     headers,
		QueryString: parseQueryString(req.URL),
		HeadersSize: -1,
		BodySize:    len(req.PostData),
	}
	if req.HasPostData && req.PostData != "" {
		ct := headerValue(headers, "content-type")
		r.PostData = &harformat.PostData{MimeType: ct, Text: req.PostData}
	}
	return r
}

func buildResponse(resp *network.Response, bodyText string, bodySize int64) harformat.Response {
	headers := headersToNV(resp.Headers)
	c := harformat.Content{
		MimeType: resp.MimeType,
		Size:     bodySize,
	}
	if bodyText != "" {
		c.Text = bodyText
	}
	return harformat.Response{
		Status:      int(resp.Status),
		StatusText:  resp.StatusText,
		HTTPVersion: protocolToHTTPVersion(resp.Protocol),
		Cookies:     []harformat.Cookie{},
		Headers:     headers,
		Content:     c,
		RedirectURL: headerValue(headers, "location"),
		HeadersSize: -1,
		BodySize:    int64(resp.EncodedDataLength),
	}
}

// buildTimings maps Chrome's ResourceTiming (ms relative to timing.RequestTime)
// onto the HAR 1.2 timings breakdown. Falls back to a coarse blocked/wait
// split (using wall-clock deltas between our own recorded timestamps) when
// Chrome didn't supply per-phase timing, e.g. for cached or data: responses.
func buildTimings(t *network.ResourceTiming, start, respAt time.Time, totalMS float64) harformat.Timings {
	if t == nil {
		wait := durMS(start, respAt)
		if wait < 0 {
			wait = totalMS
		}
		receive := totalMS - wait
		if receive < 0 {
			receive = 0
		}
		return harformat.Timings{Blocked: -1, DNS: -1, Connect: -1, SSL: -1, Send: 0, Wait: wait, Receive: receive}
	}

	phase := func(a, b float64) float64 {
		if a < 0 || b < 0 {
			return -1
		}
		d := b - a
		if d < 0 {
			return 0
		}
		return d
	}

	blocked := firstNonNegative(t.DNSStart, t.ConnectStart, t.SendStart)
	dns := phase(t.DNSStart, t.DNSEnd)
	connect := phase(t.ConnectStart, t.ConnectEnd)
	ssl := phase(t.SslStart, t.SslEnd)
	send := phase(t.SendStart, t.SendEnd)
	wait := phase(t.SendEnd, t.ReceiveHeadersEnd)
	if wait < 0 {
		wait = 0
	}

	used := 0.0
	for _, v := range []float64{blocked, dns, connect, send, wait} {
		if v > 0 {
			used += v
		}
	}
	receive := totalMS - used
	if receive < 0 {
		receive = 0
	}

	return harformat.Timings{Blocked: blocked, DNS: dns, Connect: connect, SSL: ssl, Send: send, Wait: wait, Receive: receive}
}

func firstNonNegative(vals ...float64) float64 {
	for _, v := range vals {
		if v >= 0 {
			return v
		}
	}
	return 0
}

func durMS(a, b time.Time) float64 {
	if a.IsZero() || b.IsZero() {
		return -1
	}
	return float64(b.Sub(a)) / float64(time.Millisecond)
}

func headersToNV(h network.Headers) []harformat.NameValue {
	out := make([]harformat.NameValue, 0, len(h))
	for k, v := range h {
		out = append(out, harformat.NameValue{Name: k, Value: fmt.Sprintf("%v", v)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func headerValue(headers []harformat.NameValue, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

func parseQueryString(rawURL string) []harformat.NameValue {
	idx := strings.IndexByte(rawURL, '?')
	if idx < 0 {
		return []harformat.NameValue{}
	}
	out := []harformat.NameValue{}
	for _, pair := range strings.Split(rawURL[idx+1:], "&") {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		nv := harformat.NameValue{Name: kv[0]}
		if len(kv) == 2 {
			nv.Value = kv[1]
		}
		out = append(out, nv)
	}
	return out
}

func protocolToHTTPVersion(proto string) string {
	switch strings.ToLower(proto) {
	case "h2", "h2c":
		return "HTTP/2"
	case "http/1.0":
		return "HTTP/1.0"
	case "":
		return "HTTP/1.1"
	default:
		return proto
	}
}

func isTextualMime(mime string) bool {
	mime = strings.ToLower(mime)
	switch {
	case strings.HasPrefix(mime, "text/"),
		strings.Contains(mime, "json"),
		strings.Contains(mime, "xml"),
		strings.Contains(mime, "javascript"),
		strings.Contains(mime, "html"),
		strings.Contains(mime, "css"):
		return true
	default:
		return false
	}
}
