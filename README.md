# HAR-X

A rewrite of the old `capy`/`har.analyze.py` pair into two small, coherent
tools joined by one interface: the **HAR file**.

```text
harx/
├── capture/     Go tool: pops up a real Chrome window, records every
│                network request during your session, writes a spec-
│                compliant .har file.
├── analyzer/    Python + Streamlit app: loads .har files (from capture/
│                or any browser export) and gives you an interactive
│                waterfall, filters, and summary stats.
└── captures/    Shared output folder. capture/ writes here by default;
                 analyzer/ scans it by default. (Two real-world sample
                 .har files are included so you can try the analyzer
                 immediately.)
```

## Why the old version didn't work

The old `capy` tool did a single raw `net/http.Get()` and faked a one-entry
HAR from it, plus an unrelated raw `pcap` dump. It never ran a browser, so
it could never see JS-driven requests, CSS/images/XHR/fetch, or real
per-phase timings (DNS/connect/TLS/send/wait/receive) — everything that
makes a HAR a HAR. `capture/` replaces that outright by driving a real
Chrome instance over the DevTools Protocol (CDP) and recording its actual
`Network.*` event stream, the same way DevTools' own "Network" tab and
Lighthouse do it.

## 1. Capture

```bash
cd capture
go build -o harx-capture ./cmd/harx-capture
./harx-capture -out ../captures                 # opens a blank window
./harx-capture -url https://example.com -out ../captures -title example
```

A real, visible Chrome window opens. Browse normally — log in, click
around, trigger whatever XHR/fetch/SPA traffic you care about. Close the
window (or Ctrl+C the terminal) to stop recording; a `.har` file lands in
`captures/`.

Flags:

- `-url` — optional starting page (default: opens a blank tab)
- `-out` — output directory (default `./captures`)
- `-profile` — reuse a Chrome user-data-dir, so you don't have to log in
  every session
- `-title` — label included in the output filename

## 2. Analyze

```bash
cd analyzer
pip install -r requirements.txt
streamlit run app.py
```

Opens in your browser. Pick a capture from the sidebar (auto-discovered
from `../captures/`) or upload any `.har` file — this works on browser
DevTools exports too, not just `harx-capture` output. You get:

- an interactive waterfall (stacked by timing phase: blocked/DNS/connect/
  SSL/send/wait/receive)
- filters: status range, method, resource type, domain, content-type
- status code / content-type / size distributions
- a sortable request table
- per-request detail: headers, POST body, response body preview, full
  timing breakdown

## Status

- [x] Capture rewritten on chromedp/CDP — produces real, multi-entry,
      spec-compliant HAR with per-phase timings
- [x] Analyzer rewritten in Streamlit — interactive replacement for
      `har.analyze.py`
