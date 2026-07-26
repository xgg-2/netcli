package types

import (
	"net/http"
	"strings"
	"time"
)

type ResourceType string

const (
	TypeDoc   ResourceType = "doc"
	TypeXHR   ResourceType = "xhr"
	TypeJS    ResourceType = "js"
	TypeCSS   ResourceType = "css"
	TypeImg   ResourceType = "img"
	TypeFont  ResourceType = "font"
	TypeMedia ResourceType = "media"
	TypeOther ResourceType = "other"
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
	ResourceType     ResourceType
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

func ClassifyResourceType(responseContentType string, requestHeaders http.Header) ResourceType {
	ct := strings.ToLower(responseContentType)
	if i := strings.Index(ct, ";"); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}

	switch {
	case strings.HasPrefix(ct, "text/html"):
		return TypeDoc
	case strings.HasPrefix(ct, "text/css"):
		return TypeCSS
	case ct == "application/javascript" || ct == "text/javascript" || strings.HasSuffix(ct, "/javascript"):
		return TypeJS
	case strings.HasPrefix(ct, "image/"):
		return TypeImg
	case strings.HasPrefix(ct, "font/") ||
		strings.HasPrefix(ct, "application/font-") ||
		ct == "application/vnd.ms-fontobject" ||
		ct == "application/x-font-ttf" ||
		ct == "application/x-font-woff":
		return TypeFont
	case strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/"):
		return TypeMedia
	case ct == "application/json" || strings.HasSuffix(ct, "+json"):
		return TypeXHR
	}

	if requestHeaders != nil {
		if requestHeaders.Get("X-Requested-With") == "XMLHttpRequest" {
			return TypeXHR
		}
		if requestHeaders.Get("Sec-Fetch-Mode") == "cors" {
			return TypeXHR
		}
	}

	return TypeOther
}
