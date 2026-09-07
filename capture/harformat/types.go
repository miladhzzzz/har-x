// Package harformat defines Go types for the HAR (HTTP Archive) 1.2 spec.
// http://www.softwareishard.com/blog/har-12-spec/
package harformat

type HAR struct {
	Log Log `json:"log"`
}

type Log struct {
	Version string   `json:"version"`
	Creator Creator  `json:"creator"`
	Browser *Browser `json:"browser,omitempty"`
	Pages   []Page   `json:"pages,omitempty"`
	Entries []Entry  `json:"entries"`
	Comment string   `json:"comment,omitempty"`
}

type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Browser struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Page struct {
	StartedDateTime string      `json:"startedDateTime"`
	ID              string      `json:"id"`
	Title           string      `json:"title"`
	PageTimings     PageTimings `json:"pageTimings"`
}

type PageTimings struct {
	OnContentLoad float64 `json:"onContentLoad"` // ms, -1 if not available
	OnLoad        float64 `json:"onLoad"`
}

type Entry struct {
	PageRef         string  `json:"pageref,omitempty"`
	StartedDateTime string  `json:"startedDateTime"`
	Time            float64 `json:"time"` // total ms
	Request         Request `json:"request"`
	Response        Response `json:"response"`
	Cache           Cache   `json:"cache"`
	Timings         Timings `json:"timings"`
	ServerIPAddress string  `json:"serverIPAddress,omitempty"`
	Connection      string  `json:"connection,omitempty"`
	Comment         string  `json:"comment,omitempty"`

	// Resource type ("Document", "XHR", "Fetch", "Script", "Image", "WebSocket", ...)
	// Not part of the HAR spec but very useful for analysis; namespaced as a custom field.
	ResourceType string `json:"_resourceType,omitempty"`
	Initiator    string `json:"_initiator,omitempty"`
	Failed       string `json:"_error,omitempty"`
}

type Request struct {
	Method      string        `json:"method"`
	URL         string        `json:"url"`
	HTTPVersion string        `json:"httpVersion"`
	Cookies     []Cookie      `json:"cookies"`
	Headers     []NameValue   `json:"headers"`
	QueryString []NameValue   `json:"queryString"`
	PostData    *PostData     `json:"postData,omitempty"`
	HeadersSize int           `json:"headersSize"`
	BodySize    int           `json:"bodySize"`
}

type Response struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Cookies     []Cookie    `json:"cookies"`
	Headers     []NameValue `json:"headers"`
	Content     Content     `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int64       `json:"bodySize"`
}

type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Expires  string `json:"expires,omitempty"`
	HTTPOnly bool   `json:"httpOnly,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PostData struct {
	MimeType string      `json:"mimeType"`
	Params   []NameValue `json:"params,omitempty"`
	Text     string      `json:"text,omitempty"`
}

type Content struct {
	Size        int64  `json:"size"`
	Compression int64  `json:"compression,omitempty"`
	MimeType    string `json:"mimeType"`
	Text        string `json:"text,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
}

type Cache struct{}

// Timings, all in ms. -1 means "not applicable / not measured".
type Timings struct {
	Blocked float64 `json:"blocked"`
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	SSL     float64 `json:"ssl"`
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}
