package export

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type record struct {
	Timestamp       time.Time   `json:"timestamp"`
	Method          string      `json:"method"`
	URL             string      `json:"url"`
	RequestHeaders  http.Header `json:"request_headers"`
	RequestBody     string      `json:"request_body"`
	StatusCode      int         `json:"status_code"`
	ResponseHeaders http.Header `json:"response_headers"`
	ResponseBody    string      `json:"response_body"`
	DurationMs      float64     `json:"duration_ms"`
}

type Entry struct {
	Timestamp       time.Time
	Method          string
	URL             string
	RequestHeaders  http.Header
	RequestBody     []byte
	IsBinaryRequest bool
	StatusCode      int
	ResponseHeaders  http.Header
	ResponseBody     []byte
	IsBinaryResponse bool
	DurationMs      float64
}

type Writer struct {
	path string
	file *os.File
}

func NewWriter(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open export file: %w", err)
	}
	return &Writer{path: path, file: f}, nil
}

func (w *Writer) Write(e *Entry) error {
	reqBody := encodeBody(e.RequestBody, e.IsBinaryRequest)
	respBody := encodeBody(e.ResponseBody, e.IsBinaryResponse)

	r := record{
		Timestamp:       e.Timestamp,
		Method:          e.Method,
		URL:             e.URL,
		RequestHeaders:  e.RequestHeaders,
		RequestBody:     reqBody,
		StatusCode:      e.StatusCode,
		ResponseHeaders: e.ResponseHeaders,
		ResponseBody:    respBody,
		DurationMs:      e.DurationMs,
	}

	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	_, err = fmt.Fprintf(w.file, "%s\n", data)
	return err
}

func (w *Writer) Close() error {
	return w.file.Close()
}

func (w *Writer) Path() string {
	return w.path
}

func encodeBody(body []byte, isBinary bool) string {
	if isBinary {
		return "base64:" + base64.StdEncoding.EncodeToString(body)
	}
	return string(body)
}

func WriteAll(path string, entries []*Entry) error {
	w, err := NewWriter(path)
	if err != nil {
		return err
	}
	defer w.Close()

	for _, e := range entries {
		if err := w.Write(e); err != nil {
			return err
		}
	}
	return nil
}
