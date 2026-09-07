// Command harx-capture opens a real, visible Chrome window and records
// every network request made during the session — exactly like a browser's
// DevTools "Network" tab, but saved as a HAR file when you're done.
//
// Usage:
//
//	harx-capture -url https://example.com -out ./captures
//	harx-capture -out ./captures   # opens a blank tab; navigate wherever you like
//
// Close the browser window (or press Ctrl+C in this terminal) to stop the
// capture and write the HAR.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/persys-dev/harx/capture/harformat"
	"github.com/persys-dev/harx/capture/recorder"
)

func main() {
	var (
		startURL  = flag.String("url", "", "optional URL to open immediately")
		outDir    = flag.String("out", "./captures", "directory to write the .har file into")
		profile   = flag.String("profile", "", "Chrome user-data-dir to reuse (keeps logins between sessions); default: temporary profile")
		sessTitle = flag.String("title", "", "optional label stored in the HAR filename")
	)
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("harx: cannot create output dir: %v", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // this is the whole point: a real, visible window
		chromedp.Flag("start-maximized", true),
	)
	if *profile != "" {
		opts = append(opts, chromedp.UserDataDir(*profile))
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	rec := recorder.New()
	rec.Attach(ctx)

	// Detect the user closing the popup window themselves.
	browserClosed := make(chan struct{})
	chromedp.ListenBrowser(ctx, func(ev interface{}) {
		if _, ok := ev.(*target.EventDetachedFromTarget); ok {
			select {
			case browserClosed <- struct{}{}:
			default:
			}
		}
	})

	// Start the tab, enable the domains we need, and navigate if a URL was given.
	setup := []chromedp.Action{
		network.Enable(),
		page.Enable(),
	}
	if *startURL != "" {
		setup = append(setup, chromedp.Navigate(*startURL))
	} else {
		setup = append(setup, chromedp.Navigate("about:blank"))
	}
	if err := chromedp.Run(ctx, setup...); err != nil {
		log.Fatalf("harx: failed to start browser: %v", err)
	}

	fmt.Println("harx-capture: recording. Browse normally in the opened window.")
	fmt.Println("harx-capture: close the window (or Ctrl+C here) to stop and save the HAR.")

	// Wait for: the browser window closing, or Ctrl+C.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
	case <-browserClosed:
	case <-sigCh:
	}

	entries, pages := rec.Entries()
	if len(entries) == 0 {
		fmt.Println("harx-capture: no requests were captured; nothing written.")
		return
	}

	h := harformat.HAR{Log: harformat.Log{
		Version: "1.2",
		Creator: harformat.Creator{Name: "harx-capture", Version: "0.1.0"},
		Browser: &harformat.Browser{Name: "Chrome", Version: "headful/CDP"},
		Pages:   pages,
		Entries: entries,
	}}

	outPath := buildOutputPath(*outDir, *sessTitle)
	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("harx: cannot write HAR: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(h); err != nil {
		log.Fatalf("harx: cannot encode HAR: %v", err)
	}

	fmt.Printf("harx-capture: saved %d requests across %d page(s) -> %s\n", len(entries), len(pages), outPath)
}

func buildOutputPath(dir, title string) string {
	ts := time.Now().Format("20060102-150405")
	name := ts
	if title != "" {
		name = ts + "-" + sanitize(title)
	}
	return filepath.Join(dir, name+".har")
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	return b.String()
}
