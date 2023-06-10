package main

import (
	"net/http"
	"encoding/json"
	"time"
	"os"

)

// Generate the HAR file

 func harGen(url string, req *http.Request, resp *http.Response , body []byte, outputDir string) error {

	har := &HAR{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "Golang",
				Version: "1.16",
			},
			Pages: []Page{
				{
					ID:     url,
					Title:  url,
					URL:    url,
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
        return err
    }
    if err := os.WriteFile(outputDir+"/output.har", harContent, 0644); err!= nil {
        return err
    }

	return nil

 }
 