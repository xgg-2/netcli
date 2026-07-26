package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xgg-2/netcli/internal/export"
	"github.com/xgg-2/netcli/internal/types"
)

type newEntryMsg struct {
	entry *types.RequestEntry
}

type tickMsg time.Time

type statusMsg struct {
	text  string
	isErr bool
}

type clearStatusMsg struct{}

type model struct {
	entries        []*types.RequestEntry
	pendingEntries []*types.RequestEntry
	filtered       []*types.RequestEntry
	selected       int
	listOffset     int
	filterMode     bool
	filterInput    textinput.Model
	saveMode       bool
	saveInput      textinput.Model
	savePath       string
	exporter       *export.Writer
	exportMode     bool
	exportSelected int
	harMode        bool
	harInput       textinput.Model
	viewport       viewport.Model
	width          int
	height         int
	ready          bool
	entryChan      <-chan *types.RequestEntry
	statusMsg      string
	statusIsErr    bool
	statusExpires  time.Time
}

func NewModel(entryChan <-chan *types.RequestEntry, savePath string) (*model, error) {
	fi := textinput.New()
	fi.Placeholder = "filter by host/path or type:xhr type:img ..."
	fi.CharLimit = 200

	si := textinput.New()
	si.Placeholder = "filename (e.g. session.jsonl)"
	si.CharLimit = 256

	hi := textinput.New()
	hi.Placeholder = "filename (e.g. session.har)"
	hi.CharLimit = 256

	var exp *export.Writer
	if savePath != "" {
		var err error
		exp, err = export.NewWriter(savePath)
		if err != nil {
			return nil, fmt.Errorf("open save file: %w", err)
		}
	}

	return &model{
		entries:        make([]*types.RequestEntry, 0, 64),
		pendingEntries: make([]*types.RequestEntry, 0, 64),
		filtered:       make([]*types.RequestEntry, 0, 64),
		filterMode:     false,
		filterInput:    fi,
		saveMode:       false,
		saveInput:      si,
		savePath:       savePath,
		exporter:       exp,
		harInput:       hi,
		entryChan:      entryChan,
	}, nil
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		waitForEntry(m.entryChan),
		scheduleTick(),
	)
}

func waitForEntry(ch <-chan *types.RequestEntry) tea.Cmd {
	return func() tea.Msg {
		entry := <-ch
		return newEntryMsg{entry: entry}
	}
}

func scheduleTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func scheduleStatusClear(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (m *model) applyFilter() {
	raw := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	if raw == "" {
		m.filtered = m.entries
		return
	}

	var typeFilter string
	var textParts []string
	for _, part := range strings.Fields(raw) {
		if strings.HasPrefix(part, "type:") {
			typeFilter = strings.TrimPrefix(part, "type:")
		} else {
			textParts = append(textParts, part)
		}
	}
	textFilter := strings.Join(textParts, " ")

	result := make([]*types.RequestEntry, 0, len(m.entries))
	for _, e := range m.entries {
		if typeFilter != "" && string(e.ResourceType) != typeFilter {
			continue
		}
		if textFilter != "" {
			host := strings.ToLower(e.Host)
			path := strings.ToLower(e.Path)
			if !strings.Contains(host, textFilter) && !strings.Contains(path, textFilter) {
				continue
			}
		}
		result = append(result, e)
	}
	m.filtered = result
}

func (m *model) clampSelected() {
	if len(m.filtered) == 0 {
		m.selected = 0
		m.listOffset = 0
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m *model) listHeight() int {
	reserved := 2
	if m.filterMode || m.saveMode || m.exportMode || m.harMode {
		reserved = 3
	}
	if m.statusMsg != "" {
		reserved++
	}
	h := m.height - reserved
	if h < 1 {
		h = 1
	}
	return h
}

func (m *model) updateListScroll() {
	lh := m.listHeight()
	if m.selected < m.listOffset {
		m.listOffset = m.selected
	}
	if m.selected >= m.listOffset+lh {
		m.listOffset = m.selected - lh + 1
	}
}

func (m *model) exporterEntry(e *types.RequestEntry) *export.Entry {
	return &export.Entry{
		Timestamp:        e.Timestamp,
		Method:           e.Method,
		URL:              e.URL,
		RequestHeaders:   e.RequestHeaders,
		RequestBody:      e.RequestBody,
		IsBinaryRequest:  e.IsBinaryRequest,
		StatusCode:       e.StatusCode,
		ResponseHeaders:  e.ResponseHeaders,
		ResponseBody:     e.ResponseBody,
		IsBinaryResponse: e.IsBinaryResponse,
		DurationMs:       e.DurationMs,
	}
}
