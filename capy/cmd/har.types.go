package main

import (
	"net/http"
	"net/url"
)

type HAR struct {
    Log Log `json:"log"`
}

type Log struct {
    Version  string    `json:"version"`
    Creator  Creator   `json:"creator"`
    Pages    []Page    `json:"pages"`
    Entries  []Entry   `json:"entries"`
    Browser  Browser   `json:"browser"`
    Metadata Metadata  `json:"metadata"`
    Comment  string    `json:"comment"`
}

type Creator struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type Page struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    URL    string `json:"url"`
    StartedDateTime string `json:"startedDateTime"`
    PageTimings PageTimings `json:"pageTimings"`
}

type PageTimings struct {
    OnContentLoad float64 `json:"onContentLoad"`
    OnLoad        float64 `json:"onLoad"`
}

type Entry struct {
    StartedDateTime string `json:"startedDateTime"`
    Time            float64 `json:"time"`
    Request         Request `json:"request"`
    Response        Response `json:"response"`
    Cache           Cache   `json:"cache"`
    Timings         Timings `json:"timings"`
    ServerIPAddress string `json:"serverIPAddress"`
    Connection      string `json:"connection"`
    Comment         string `json:"comment"`
}

type Request struct {
    Method  string `json:"method"`
    URL     string `json:"url"`
    HTTPVersion string `json:"httpVersion"`
    Headers http.Header `json:"headers"`
    QueryString []QueryString `json:"queryString"`
    PostData PostData `json:"postData"`
    HeadersSize int `json:"headersSize"`
    BodySize int `json:"bodySize"`
}

type Header struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type QueryString struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type PostData struct {
    MimeType string `json:"mimeType"`
    Params url.Values `json:"params"`
}

type Param struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type Response struct {
    Status         int `json:"status"`
    StatusText     string `json:"statusText"`
    HTTPVersion    string `json:"httpVersion"`
    Cookies        []*http.Cookie `json:"cookies"`
    Headers        http.Header `json:"headers"`
    Content        Content `json:"content"`
    RedirectURL    string `json:"redirectURL"`
    HeadersSize    int `json:"headersSize"`
    BodySize       int `json:"bodySize"`
    MimeType       string `json:"mimeType"`
    Connection     string `json:"connection"`
    EncodedDataLength int `json:"_encodedDataLength"`
    FromDiskCache  bool `json:"fromDiskCache"`
    FromServiceWorker bool `json:"fromServiceWorker"`
    FromPrefetchCache bool `json:"fromPrefetchCache"`
    Timing         Timing `json:"timing"`
}

type Cookie struct {
    Name     string `json:"name"`
    Value    string `json:"value"`
    Path     string `json:"path"`
    Domain   string `json:"domain"`
    Expires  string `json:"expires"`
    HTTPOnly bool `json:"httpOnly"`
    Secure   bool `json:"secure"`
    SameSite string `json:"sameSite"`
}

type Content struct {
    Size int `json:"size"`
    Text string `json:"text"`
}

type Timing struct {
    Blocked float64 `json:"blocked"`
    DNS     float64 `json:"dns"`
    Connect float64 `json:"connect"`
    Send    float64 `json:"send"`
    Wait    float64 `json:"wait"`
    Receive float64 `json:"receive"`
    SSL     float64 `json:"ssl"`
    Comment string `json:"comment"`
}

type Browser struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type Metadata struct {
    Device string `json:"device"`
    Platform string `json:"platform"`
    UserAgent string `json:"userAgent"`
}

type Cache struct {
    BeforeRequest CacheDetails `json:"beforeRequest"`
    AfterRequest  CacheDetails `json:"afterRequest"`
}

type CacheDetails struct {
    Expires    string `json:"expires"`
    LastAccess string `json:"lastAccess"`
    ETag       string `json:"eTag"`
    HitCount   int    `json:"hitCount"`
    Comment    string `json:"comment"`
}

type Timings struct {
    Blocked float64 `json:"blocked"`
    DNS     float64 `json:"dns"`
    Connect float64 `json:"connect"`
    Send    float64 `json:"send"`
    Wait    float64 `json:"wait"`
    Receive float64 `json:"receive"`
    SSL     float64 `json:"ssl"`
    Comment string `json:"comment"`
}