package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xgg-2/netcli/internal/export"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		lh := m.listHeight()
		rightW := m.rightWidth()
		if !m.ready {
			m.viewport = viewport.New(rightW, lh)
			m.ready = true
		} else {
			m.viewport.Width = rightW
			m.viewport.Height = lh
		}
		m.refreshViewport()

	case newEntryMsg:
		m.pendingEntries = append(m.pendingEntries, msg.entry)
		if m.exporter != nil && msg.entry.Complete {
			_ = m.exporter.Write(m.exporterEntry(msg.entry))
		}
		cmds = append(cmds, waitForEntry(m.entryChan))

	case tickMsg:
		if len(m.pendingEntries) > 0 {
			m.entries = append(m.entries, m.pendingEntries...)
			m.pendingEntries = m.pendingEntries[:0]
			m.applyFilter()
			m.clampSelected()
			m.updateListScroll()
			m.refreshViewport()
		}
		cmds = append(cmds, scheduleTick())

	case statusMsg:
		m.statusMsg = msg.text
		m.statusIsErr = msg.isErr
		m.statusExpires = time.Now().Add(3 * time.Second)
		cmds = append(cmds, scheduleStatusClear(3*time.Second))

	case clearStatusMsg:
		m.statusMsg = ""

	case tea.KeyMsg:
		if m.saveMode {
			return m.handleSaveInput(msg, cmds)
		}
		if m.harMode {
			return m.handleHarInput(msg, cmds)
		}
		if m.filterMode {
			return m.handleFilterInput(msg, cmds)
		}
		if m.exportMode {
			return m.handleExportInput(msg, cmds)
		}
		return m.handleNormalKey(msg, cmds)
	}

	if m.ready {
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleNormalKey(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.exporter != nil {
			_ = m.exporter.Close()
		}
		return m, tea.Quit

	case "up", "k":
		if m.selected > 0 {
			m.selected--
			m.updateListScroll()
			m.refreshViewport()
		}

	case "down", "j":
		if m.selected < len(m.filtered)-1 {
			m.selected++
			m.updateListScroll()
			m.refreshViewport()
		}

	case "pgup":
		lh := m.listHeight()
		m.selected -= lh
		if m.selected < 0 {
			m.selected = 0
		}
		m.updateListScroll()
		m.refreshViewport()

	case "pgdown":
		lh := m.listHeight()
		m.selected += lh
		if m.selected >= len(m.filtered) {
			m.selected = len(m.filtered) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		m.updateListScroll()
		m.refreshViewport()

	case "/":
		m.filterMode = true
		m.filterInput.Focus()
		cmds = append(cmds, m.filterInput.Focus())

	case "s":
		if m.savePath == "" {
			m.saveMode = true
			m.saveInput.SetValue("")
			m.saveInput.Focus()
			cmds = append(cmds, m.saveInput.Focus())
		} else {
			cmds = append(cmds, m.saveAllNow())
		}

	case "e":
		if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
			cmds = append(cmds, func() tea.Msg {
				return statusMsg{text: "no request selected", isErr: true}
			})
		} else {
			m.exportMode = true
			m.exportSelected = 0
		}

	case "y":
		cmds = append(cmds, m.copyResponseBody())

	case "h":
		m.harMode = true
		m.harInput.SetValue("")
		m.harInput.Focus()
		cmds = append(cmds, m.harInput.Focus())
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleFilterInput(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filterMode = false
		m.filterInput.Blur()
		m.applyFilter()
		m.clampSelected()
		m.updateListScroll()
		m.refreshViewport()
		return m, tea.Batch(cmds...)

	case "enter":
		m.filterMode = false
		m.filterInput.Blur()
		m.applyFilter()
		m.clampSelected()
		m.updateListScroll()
		m.refreshViewport()
		return m, tea.Batch(cmds...)

	default:
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		cmds = append(cmds, cmd)
		m.applyFilter()
		m.clampSelected()
		m.updateListScroll()
		m.refreshViewport()
		return m, tea.Batch(cmds...)
	}
}

func (m *model) handleSaveInput(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.saveMode = false
		m.saveInput.Blur()
		return m, tea.Batch(cmds...)

	case "enter":
		m.saveMode = false
		m.saveInput.Blur()
		path := strings.TrimSpace(m.saveInput.Value())
		if path == "" {
			return m, tea.Batch(cmds...)
		}
		if !strings.HasSuffix(path, ".jsonl") {
			path += ".jsonl"
		}
		cmds = append(cmds, m.saveAllTo(path))
		return m, tea.Batch(cmds...)

	default:
		var cmd tea.Cmd
		m.saveInput, cmd = m.saveInput.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
}

func (m *model) saveAllNow() tea.Cmd {
	return func() tea.Msg {
		if m.exporter == nil {
			return statusMsg{text: "no save path configured", isErr: true}
		}
		for _, e := range m.entries {
			if e.Complete {
				_ = m.exporter.Write(m.exporterEntry(e))
			}
		}
		return statusMsg{text: fmt.Sprintf("session saved to %s", m.exporter.Path())}
	}
}

func (m *model) saveAllTo(path string) tea.Cmd {
	entries := make([]*export.Entry, 0, len(m.entries))
	for _, e := range m.entries {
		if e.Complete {
			entries = append(entries, m.exporterEntry(e))
		}
	}
	return func() tea.Msg {
		if err := export.WriteAll(path, entries); err != nil {
			return statusMsg{text: fmt.Sprintf("save failed: %v", err), isErr: true}
		}
		return statusMsg{text: fmt.Sprintf("session saved to %s", path)}
	}
}

func (m *model) handleExportInput(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.exportMode = false
		return m, tea.Batch(cmds...)

	case "up", "k":
		if m.exportSelected > 0 {
			m.exportSelected--
		}
		return m, tea.Batch(cmds...)

	case "down", "j":
		if m.exportSelected < len(export.Formats)-1 {
			m.exportSelected++
		}
		return m, tea.Batch(cmds...)

	case "enter":
		format := export.Formats[m.exportSelected]
		m.exportMode = false
		cmds = append(cmds, m.doExport(format))
		return m, tea.Batch(cmds...)

	case "1", "2", "3", "4", "5":
		idx := int(msg.String()[0] - '1')
		if idx >= 0 && idx < len(export.Formats) {
			format := export.Formats[idx]
			m.exportMode = false
			cmds = append(cmds, m.doExport(format))
		}
		return m, tea.Batch(cmds...)
	}
	return m, tea.Batch(cmds...)
}

func (m *model) handleHarInput(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.harMode = false
		m.harInput.Blur()
		return m, tea.Batch(cmds...)

	case "enter":
		m.harMode = false
		m.harInput.Blur()
		path := strings.TrimSpace(m.harInput.Value())
		if path == "" {
			path = "session.har"
		}
		if !strings.HasSuffix(path, ".har") {
			path += ".har"
		}
		cmds = append(cmds, m.exportHAR(path))
		return m, tea.Batch(cmds...)

	default:
		var cmd tea.Cmd
		m.harInput, cmd = m.harInput.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
}

func (m *model) doExport(format string) tea.Cmd {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return func() tea.Msg {
			return statusMsg{text: "no request selected", isErr: true}
		}
	}
	entry := m.filtered[m.selected]
	code := export.GenerateCode(format, entry)
	return func() tea.Msg {
		err := clipboard.WriteAll(code)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "\nWARNING: %s output below may contain auth headers, cookies, or tokens\n%s\n", format, code)
			return statusMsg{text: fmt.Sprintf("%s printed to terminal (clipboard unavailable) — may contain credentials", format)}
		}
		return statusMsg{text: fmt.Sprintf("%s copied to clipboard — may contain auth headers/cookies", format)}
	}
}

func (m *model) copyResponseBody() tea.Cmd {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return nil
	}
	entry := m.filtered[m.selected]
	if !entry.Complete {
		return nil
	}
	text := plainResponseBody(entry)
	return func() tea.Msg {
		err := clipboard.WriteAll(text)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "\n%s\n", text)
			return statusMsg{text: "response body printed to terminal (clipboard unavailable)"}
		}
		return statusMsg{text: "response body copied to clipboard"}
	}
}

func (m *model) exportHAR(path string) tea.Cmd {
	entries := m.filtered
	return func() tea.Msg {
		count, err := export.WriteHAR(path, entries)
		if err != nil {
			return statusMsg{text: fmt.Sprintf("HAR export failed: %v", err), isErr: true}
		}
		return statusMsg{text: fmt.Sprintf("%d entries exported to %s", count, path)}
	}
}

func (m *model) refreshViewport() {
	if !m.ready {
		return
	}
	lh := m.listHeight()
	m.viewport.Width = m.rightWidth()
	m.viewport.Height = lh
	m.viewport.SetContent(m.buildDetailContent())
}

func (m *model) rightWidth() int {
	leftW := m.leftWidth()
	w := m.width - leftW - 1
	if w < 20 {
		w = 20
	}
	return w
}

func (m *model) leftWidth() int {
	w := m.width * 38 / 100
	if w < 30 {
		w = 30
	}
	return w
}
