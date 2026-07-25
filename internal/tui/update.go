package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/xgg-2/netcli/internal/export"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		rightW := m.rightWidth()
		detailH := m.listHeight() - 1
		if detailH < 1 {
			detailH = 1
		}
		if !m.ready {
			m.viewport = viewport.New(rightW, detailH)
			m.ready = true
		} else {
			m.viewport.Width = rightW
			m.viewport.Height = detailH
		}
		m.refreshViewport()

	case newEntryMsg:
		m.entries = append(m.entries, msg.entry)
		m.applyFilter()
		if m.exporter != nil && msg.entry.Complete {
			_ = m.exporter.Write(m.exporterEntry(msg.entry))
		}
		cmds = append(cmds, waitForEntry(m.entryChan))

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
		if m.filterMode {
			return m.handleFilterInput(msg, cmds)
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
		cmds = append(cmds, m.exportCurl())
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

func (m *model) exportCurl() tea.Cmd {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return func() tea.Msg {
			return statusMsg{text: "no request selected", isErr: true}
		}
	}
	entry := m.filtered[m.selected]
	cmd := buildCurlCommand(entry)
	return func() tea.Msg {
		err := clipboard.WriteAll(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stdout, "\nWARNING: curl command below may contain auth headers, cookies, or tokens\n%s\n", cmd)
			return statusMsg{text: "curl printed to terminal (clipboard unavailable) — may contain credentials"}
		}
		return statusMsg{text: "curl copied to clipboard — command may contain auth headers/cookies"}
	}
}

func (m *model) refreshViewport() {
	if !m.ready {
		return
	}
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
