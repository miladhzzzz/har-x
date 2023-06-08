package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/miekg/dns"
)

func main() {

	fqdn := flag.String("fqdn", "example.com" ,"for DNS capture")
    url := flag.String("url", "https://example.com", "url")
    dnsServer := flag.String("dns" ,"8.8.8.8" ,"DNS server address")
    outputDir := flag.String("dir", "./output", "output directory")

	flag.Parse()

    // Create the output directory if it doesn't exist
    if _, err := os.Stat(*outputDir); os.IsNotExist(err) {
        os.Mkdir(*outputDir, 0755)
    }

    // Perform a DNS query for the URL and write the results to a file
    client := &dns.Client{}
    m := &dns.Msg{}
    m.SetQuestion(dns.Fqdn(*fqdn), dns.TypeA)
    r, _, err := client.Exchange(m, *dnsServer+":53")
    if err!= nil {
        log.Fatal(err)
    }
    if len(r.Answer) == 0 {
        log.Fatal("No results")
    }
    f, err := os.Create(*outputDir + "/output.txt")
    if err!= nil {
        log.Fatal(err)
    }
    defer f.Close()
    for _, ans := range r.Answer {
        if a, ok := ans.(*dns.A); ok {
            io.WriteString(f, fmt.Sprintf("%s IN A %s\n", *url, a.A.String()))
        }
    }

    // Capture all TCP, TLS, HTTP handshakes, and everything to a pcap file for the URL that was provided
    handle, err := pcap.OpenLive("eth0", 1600, true, pcap.BlockForever)
    if err!= nil {
        log.Fatal(err)
    }
    defer handle.Close()
    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
    w, err := os.Create(*outputDir + "/output.pcap")
    if err!= nil {
        log.Fatal(err)
    }
    defer w.Close()
    packetCount := 0
    for packet := range packetSource.Packets() {
        w.Write(packet.Data())
        packetCount++
        if packetCount > 100 {
            break
        }
    }

    // Use a headless browser to open the URL in the background, capture the network activity, wait till the page renders completely, and save the.har file
    webClient := &http.Client{}
    req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, *url, nil)
    if err!= nil {
        log.Fatal(err)
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")
    resp, err := webClient.Do(req)
    if err!= nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    trace := &httptrace.ClientTrace{
        GotConn: func(connInfo httptrace.GotConnInfo) {
            fmt.Printf("Connected to %s\n", connInfo.Conn.RemoteAddr())
        },
        DNSStart: func(info httptrace.DNSStartInfo) {
            fmt.Printf("DNS lookup for %s started\n", info.Host)
        },
        DNSDone: func(info httptrace.DNSDoneInfo) {
            fmt.Printf("DNS lookup for %s done: %v\n", info, info.Err)
        },
        ConnectStart: func(network, addr string) {
            fmt.Printf("Connecting to %s\n", addr)
        },
        ConnectDone: func(network, addr string, err error) {
            if err!= nil {
                fmt.Printf("Failed to connect to %s: %v\n", addr, err)
            } else {
                fmt.Printf("Connected to %s\n", addr)
            }
        },
        TLSHandshakeStart: func() {
            fmt.Printf("TLS handshake started\n")
        },
        TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
            if err!= nil {
                fmt.Printf("TLS handshake failed: %v\n", err)
            } else {
                fmt.Printf("TLS handshake successful\n")
            }
        },
    }
    req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
    resp, err = webClient.Do(req)
    if err!= nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err!= nil {
        log.Fatal(err)
    }
    har := &HAR{
        Log: Log{
            Version: "1.2",
            Creator: Creator{
                Name:    "Golang",
                Version: "1.16",
            },
            Pages: []Page{
                {
                    ID:     *url,
                    Title:  *url,
                    URL:    *url,
                    StartedDateTime: time.Now().Format(time.RFC3339),
                    PageTimings: PageTimings{
                        OnContentLoad: -1,
                        OnLoad:        -1,
                    },
                },
            },
            Entries: []Entry{
                {
                    StartedDateTime: time.Now().Format(time.RFC3339),
                    Time:            -1,
                    Request: Request{
                        Method:  req.Method,
                        URL:     req.URL.String(),
                        Headers: req.Header,
                        PostData: PostData{
                            MimeType: "application/x-www-form-urlencoded",
                            Params:   req.PostForm,
                        },
                    },
                    Response: Response{
                        Status:         resp.StatusCode,
                        StatusText:     resp.Status,
                        HTTPVersion:    resp.Proto,
                        Headers:        resp.Header,
                        Content:        Content{Size: len(body), Text: string(body)},
                        RedirectURL:    resp.Header.Get("Location"),
                        Cookies:        resp.Cookies(),
                        HeadersSize:    len(resp.Header.Get("Content-Length")),
                        BodySize:       len(resp.Header.Get("Content-Length")),
                    },
                    Cache: Cache{},
                    Timings: Timings{
                        Send:  -1,
                        Wait:  -1,
                        Receive: -1,
                    },
                    ServerIPAddress: "",
                    Connection:      "",
                    Comment:         "",
                },
            },
        },
    }
    harContent, err := json.MarshalIndent(har, "", "  ")
    if err!= nil {
        log.Fatal(err)
    }
    if err := os.WriteFile(*outputDir+"/output.har", harContent, 0644); err!= nil {
        log.Fatal(err)
    }
	log.Printf("your DNS , Pcap , HAR files were created in : %v", *outputDir)
}