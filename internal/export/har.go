package export

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/xgg-2/netcli/internal/types"
)

type harDocument struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Creator harCreator `json:"creator"`
	Entries []harEntry `json:"entries"`
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harEntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            float64     `json:"time"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
	Timings         harTimings  `json:"timings"`
}

type harRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	HTTPVersion string       `json:"httpVersion"`
	Headers     []harNameVal `json:"headers"`
	QueryString []harNameVal `json:"queryString"`
	PostData    *harPostData `json:"postData,omitempty"`
	BodySize    int          `json:"bodySize"`
	HeadersSize int          `json:"headersSize"`
}

type harResponse struct {
	Status      int          `json:"status"`
	StatusText  string       `json:"statusText"`
	HTTPVersion string       `json:"httpVersion"`
	Headers     []harNameVal `json:"headers"`
	Content     harContent   `json:"content"`
	RedirectURL string       `json:"redirectURL"`
	BodySize    int          `json:"bodySize"`
	HeadersSize int          `json:"headersSize"`
}

type harContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
}

type harPostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type harNameVal struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harTimings struct {
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	SSL     float64 `json:"ssl"`
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

func WriteHAR(path string, entries []*types.RequestEntry) (int, error) {
	var harEntries []harEntry
	for _, e := range entries {
		if !e.Complete {
			continue
		}
		harEntries = append(harEntries, entryToHAR(e))
	}
	if harEntries == nil {
		harEntries = []harEntry{}
	}

	doc := harDocument{
		Log: harLog{
			Version: "1.2",
			Creator: harCreator{Name: "netcli", Version: "1.0"},
			Entries: harEntries,
		},
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return 0, err
	}

	return len(harEntries), nil
}

func entryToHAR(e *types.RequestEntry) harEntry {
	reqHeaders := headersToHAR(e.RequestHeaders)
	respHeaders := headersToHAR(e.ResponseHeaders)

	var postData *harPostData
	if len(e.RequestBody) > 0 && !e.IsBinaryRequest {
		postData = &harPostData{
			MimeType: e.RequestHeaders.Get("Content-Type"),
			Text:     string(e.RequestBody),
		}
	}

	mimeType := e.ResponseHeaders.Get("Content-Type")
	var contentText string
	if !e.IsBinaryResponse {
		contentText = string(e.ResponseBody)
	}

	return harEntry{
		StartedDateTime: e.Timestamp.UTC().Format(time.RFC3339Nano),
		Time:            e.DurationMs,
		Request: harRequest{
			Method:      e.Method,
			URL:         e.URL,
			HTTPVersion: "HTTP/1.1",
			Headers:     reqHeaders,
			QueryString: []harNameVal{},
			PostData:    postData,
			BodySize:    len(e.RequestBody),
			HeadersSize: -1,
		},
		Response: harResponse{
			Status:      e.StatusCode,
			StatusText:  http.StatusText(e.StatusCode),
			HTTPVersion: "HTTP/1.1",
			Headers:     respHeaders,
			Content: harContent{
				Size:     len(e.ResponseBody),
				MimeType: mimeType,
				Text:     contentText,
			},
			RedirectURL: "",
			BodySize:    len(e.ResponseBody),
			HeadersSize: -1,
		},
		Timings: harTimings{
			DNS:     -1,
			Connect: -1,
			SSL:     -1,
			Send:    0,
			Wait:    e.DurationMs,
			Receive: 0,
		},
	}
}

func headersToHAR(h http.Header) []harNameVal {
	result := make([]harNameVal, 0, len(h))
	for k, vals := range h {
		for _, v := range vals {
			result = append(result, harNameVal{Name: k, Value: v})
		}
	}
	return result
}
