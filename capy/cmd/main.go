package main

import (
    "context"
    "crypto/tls"
    "time"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/http/httptrace"
    "os"
   

    "github.com/google/gopacket"
    "github.com/google/gopacket/pcap"
    "github.com/miekg/dns"
)

func main() {
    fqdn := flag.String("fqdn", "example.com", "for DNS capture")
    url := flag.String("url", "https://example.com", "url")
    dnsServer := flag.String("dns", "8.8.8.8", "DNS server address")
    outputDir := flag.String("dir", "./output", "output directory")

    flag.Parse()

    // Create the output directory if it doesn't exist
    if err := os.MkdirAll(*outputDir, 0755); err!= nil {
        log.Fatal(err)
    }

    // Perform a DNS query for the URL and write the results to a file
    if err := writeDNSToFile(*fqdn, *dnsServer, *outputDir); err!= nil {
        log.Fatal(err)
    } 

    // Use a headless browser to open the URL in the background, capture the network activity, and save the HAR file
    if err := captureNetworkActivity(*url, *outputDir); err!= nil {
        log.Fatal(err)
    }

    log.Printf("your DNS, Pcap, and HAR files were created in: %v", *outputDir)
}

func writeDNSToFile(fqdn, dnsServer, outputDir string) error {
    client := &dns.Client{}
    m := &dns.Msg{}
    m.SetQuestion(dns.Fqdn(fqdn), dns.TypeA)
    r, _, err := client.Exchange(m, dnsServer+":53")
    if err!= nil {
        return err
    }
    if len(r.Answer) == 0 {
        return fmt.Errorf("no results")
    }
    f, err := os.Create(outputDir + "/dns.txt")
    if err!= nil {
        return err
    }
    defer f.Close()
    for _, ans := range r.Answer {
        if a, ok := ans.(*dns.A); ok {
            io.WriteString(f, fmt.Sprintf("%s IN A %s\n", fqdn, a.A.String()))
        }
    }
    return nil
}

func captureNetworkActivity(url, outputDir string) error {
    // Open a handle to the network interface
    handle, err := pcap.OpenLive("eth0", 1600, true, pcap.BlockForever)
    if err!= nil {
        return err
    }
    defer handle.Close()

    // Use a headless browser to open the URL in the background, capture the network activity, and save the HAR file
    transport := &http.Transport{
        DisableCompression: true,
        DisableKeepAlives:  true,
        MaxIdleConns:       100,
        MaxIdleConnsPerHost: 100,
        MaxConnsPerHost:    100,
        IdleConnTimeout:    30 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
        ResponseHeaderTimeout: 10 * time.Second,
    }
    client := &http.Client{
        Transport: transport,
    }
    req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
    if err!= nil {
        return err
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")
    trace := &httptrace.ClientTrace{
        GotConn: func(connInfo httptrace.GotConnInfo) {
            fmt.Printf("Connected to %s\n", connInfo.Conn.RemoteAddr())
        },
        DNSStart: func(info httptrace.DNSStartInfo) {
            fmt.Printf("DNS lookup for %s started\n", info.Host)
        },
        DNSDone: func(info httptrace.DNSDoneInfo) {
            fmt.Printf("DNS lookup for %s done: %v", info.Err)
            if info.Err!= nil {
                fmt.Printf("Failed to connect to %s: %v\n", url, info.Err)
            } else {
                fmt.Printf("Connected to %s\n", url)
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
    resp, err := client.Do(req)
    if err!= nil {
        return err
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err!= nil {
        return err
    }

    // Handle redirects manually
    for resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther {
        location := resp.Header.Get("Location")
        if location == "" {
            break
        }
        req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, location, nil)
        if err!= nil {
            return err
        }
        req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")
        resp, err = client.Do(req)
        if err!= nil {
            return err
        }
        defer resp.Body.Close()
        body, err = io.ReadAll(resp.Body)
        if err!= nil {
            return err
        }
    }

    // Generate the HAR file
    err = harGen(url, req, resp, body, outputDir)
    if err!= nil {
        log.Printf("could not generate HAR file: %v", err)
        return err
    }

    // Stop capturing packets after the browser has stopped
    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
    packetCh := make(chan gopacket.Packet)
    go func() {
        for packet := range packetSource.Packets() {
            packetCh <- packet
        }
        close(packetCh)
    }()
    w, err := os.Create(outputDir + "/output.pcap")
    if err!= nil {
        return err
    }
    defer w.Close()
    for packet := range packetCh {
        w.Write(packet.Data())
    }

    return nil
}