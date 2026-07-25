package types

import (
	"net/http"
	"time"
)

type RequestEntry struct {
	ID               string
	Timestamp        time.Time
	Method           string
	URL              string
	Host             string
	Path             string
	RequestHeaders   http.Header
	RequestBody      []byte
	IsBinaryRequest  bool
	StatusCode       int
	ResponseHeaders  http.Header
	ResponseBody     []byte
	IsBinaryResponse bool
	DurationMs       float64
	Complete         bool
}

func (e *RequestEntry) MethodLabel() string {
	return e.Method
}

func (e *RequestEntry) DisplayPath() string {
	path := e.Path
	if len(path) == 0 {
		path = "/"
	}
	return path
}
