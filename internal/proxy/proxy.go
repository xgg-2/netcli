package proxy

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gomitmproxy "github.com/lqqyt2423/go-mitmproxy/proxy"
	"github.com/sirupsen/logrus"
	"github.com/xgg-2/netcli/internal/cert"
	"github.com/xgg-2/netcli/internal/types"
)

type Config struct {
	Addr      string
	CaRootDir string
	Filter    string
	EntryChan chan<- *types.RequestEntry
}

type inspectorAddon struct {
	gomitmproxy.BaseAddon
	mu      sync.Mutex
	pending map[string]*pendingEntry
	config  *Config
}

type pendingEntry struct {
	entry     *types.RequestEntry
	startTime time.Time
}

func (a *inspectorAddon) Request(f *gomitmproxy.Flow) {
	if f.Request == nil {
		return
	}

	host := f.Request.URL.Host
	path := f.Request.URL.Path

	if a.config.Filter != "" {
		filter := a.config.Filter
		if !strings.Contains(host, filter) && !strings.Contains(path, filter) {
			return
		}
	}

	body := f.Request.Body
	isBinary := isBinaryContent(f.Request.Header.Get("Content-Type"), body)

	entry := &types.RequestEntry{
		ID:              f.Id.String(),
		Timestamp:       time.Now(),
		Method:          f.Request.Method,
		URL:             f.Request.URL.String(),
		Host:            host,
		Path:            path,
		RequestHeaders:  f.Request.Header.Clone(),
		RequestBody:     append([]byte(nil), body...),
		IsBinaryRequest: isBinary,
		Complete:        false,
	}

	a.mu.Lock()
	a.pending[entry.ID] = &pendingEntry{entry: entry, startTime: entry.Timestamp}
	a.mu.Unlock()
}

func (a *inspectorAddon) Response(f *gomitmproxy.Flow) {
	if f.Response == nil {
		return
	}

	a.mu.Lock()
	p, ok := a.pending[f.Id.String()]
	if !ok {
		a.mu.Unlock()
		return
	}
	delete(a.pending, f.Id.String())
	a.mu.Unlock()

	respBody := f.Response.Body
	isBinary := isBinaryContent(f.Response.Header.Get("Content-Type"), respBody)

	p.entry.StatusCode = f.Response.StatusCode
	p.entry.ResponseHeaders = f.Response.Header.Clone()
	p.entry.ResponseBody = append([]byte(nil), respBody...)
	p.entry.IsBinaryResponse = isBinary
	p.entry.DurationMs = float64(time.Since(p.startTime).Milliseconds())
	p.entry.Complete = true
	p.entry.ResourceType = types.ClassifyResourceType(
		f.Response.Header.Get("Content-Type"),
		p.entry.RequestHeaders,
	)

	select {
	case a.config.EntryChan <- p.entry:
	default:
	}
}

func Start(cfg *Config) error {
	addon := &inspectorAddon{
		pending: make(map[string]*pendingEntry),
		config:  cfg,
	}

	redirectInternalLogging()

	opts := &gomitmproxy.Options{
		Addr:        cfg.Addr,
		CaRootPath:  cfg.CaRootDir,
		SslInsecure: false,
	}

	p, err := gomitmproxy.NewProxy(opts)
	if err != nil {
		return err
	}

	p.AddAddon(addon)
	return p.Start()
}

func redirectInternalLogging() {
	dir, err := cert.ConfigDir()
	if err != nil {
		logrus.SetOutput(io.Discard)
		return
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		logrus.SetOutput(io.Discard)
		return
	}
	logPath := filepath.Join(dir, "proxy.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		logrus.SetOutput(io.Discard)
		return
	}
	logrus.SetOutput(f)
}

func isBinaryContent(contentType string, body []byte) bool {
	textTypes := []string{
		"text/", "application/json", "application/xml",
		"application/x-www-form-urlencoded", "application/javascript",
		"application/graphql", "+json", "+xml",
	}
	for _, t := range textTypes {
		if strings.Contains(contentType, t) {
			return false
		}
	}
	if len(body) == 0 {
		return false
	}
	sample := body
	if len(sample) > 512 {
		sample = sample[:512]
	}
	return bytes.IndexByte(sample, 0) >= 0
}
