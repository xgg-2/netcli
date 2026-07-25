package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xgg-2/netcli/internal/types"
)

func (m *model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	leftW := m.leftWidth()
	rightW := m.rightWidth()
	lh := m.listHeight()

	leftLines := strings.Split(m.buildList(leftW, lh), "\n")
	rightLines := strings.Split(m.buildDetail(rightW, lh), "\n")

	var rows []string
	for i := range lh {
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		rows = append(rows, l+styleDivider.Render(" ")+r)
	}

	body := strings.Join(rows, "\n")

	var parts []string
	parts = append(parts, body)

	if m.filterMode {
		parts = append(parts, styleFilter.Render("Filter: ")+m.filterInput.View())
	} else if m.saveMode {
		parts = append(parts, styleSavePrompt.Render("Save to: ")+m.saveInput.View())
	}

	if m.statusMsg != "" {
		if m.statusIsErr {
			parts = append(parts, styleError.Render(m.statusMsg))
		} else {
			parts = append(parts, styleSuccess.Render(m.statusMsg))
		}
	}

	parts = append(parts, m.buildFooter())
	return strings.Join(parts, "\n")
}

func (m *model) buildList(width, height int) string {
	lines := make([]string, 0, height)

	header := styleHeader.Width(width).Render(
		padRight("Method", 8) + padRight("Status", 7) + padRight("ms", 7) + "Host + Path",
	)
	lines = append(lines, header)

	visibleCount := height - 1
	if visibleCount < 0 {
		visibleCount = 0
	}

	for i := range visibleCount {
		idx := m.listOffset + i
		if idx >= len(m.filtered) {
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(""))
			continue
		}
		e := m.filtered[idx]
		lines = append(lines, m.renderRow(e, width, idx == m.selected))
	}

	return strings.Join(lines, "\n")
}

func (m *model) renderRow(e *types.RequestEntry, width int, selected bool) string {
	methodW := 8
	statusW := 7
	timeW := 7
	hostPathW := width - methodW - statusW - timeW
	if hostPathW < 8 {
		hostPathW = 8
	}

	method := padRight(e.Method, methodW)
	var statusStr, timeStr string

	if e.Complete {
		statusStr = padRight(fmt.Sprintf("%d", e.StatusCode), statusW)
		timeStr = padRight(fmt.Sprintf("%.0f", e.DurationMs), timeW)
	} else {
		statusStr = padRight("...", statusW)
		timeStr = padRight("-", timeW)
	}

	hostPath := truncate(e.Host+e.DisplayPath(), hostPathW)

	var row string
	if selected {
		row = styleSelectedRow.Width(width).Render(method + statusStr + timeStr + hostPath)
	} else {
		methodStyled := methodColor(e.Method).Render(padRight(e.Method, methodW))
		var statusStyled string
		if e.Complete {
			statusStyled = statusColor(e.StatusCode).Render(statusStr)
		} else {
			statusStyled = lipgloss.NewStyle().Foreground(colorGray).Render(statusStr)
		}
		timeStyled := lipgloss.NewStyle().Foreground(colorGray).Render(timeStr)
		row = methodStyled + statusStyled + timeStyled + hostPath
	}

	return row
}

func (m *model) buildDetail(width, height int) string {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return lipgloss.NewStyle().
			Foreground(colorGray).
			Width(width).
			Height(height).
			Render("No request selected.")
	}

	m.viewport.Width = width
	m.viewport.Height = height
	return m.viewport.View()
}

func (m *model) buildDetailContent() string {
	if len(m.filtered) == 0 || m.selected >= len(m.filtered) {
		return ""
	}

	e := m.filtered[m.selected]
	var b strings.Builder

	label := func(s string) string {
		return lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(s)
	}

	b.WriteString(styleSectionHeader.Render("Request") + "\n")
	b.WriteString(label("URL:    ") + e.URL + "\n")
	b.WriteString(label("Method: ") + methodColor(e.Method).Render(e.Method) + "\n")
	b.WriteString(label("Time:   ") + e.Timestamp.Format("15:04:05.000") + "\n\n")

	b.WriteString(styleSectionHeader.Render("Request Headers") + "\n")
	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(label(k+": ") + v + "\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(styleSectionHeader.Render("Request Body") + "\n")
	b.WriteString(formatBody(e.RequestBody, e.IsBinaryRequest, e.RequestHeaders.Get("Content-Type")))
	b.WriteString("\n\n")

	if !e.Complete {
		b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("Waiting for response..."))
		return b.String()
	}

	b.WriteString(styleSectionHeader.Render("Response") + "\n")
	b.WriteString(label("Status:   ") + statusColor(e.StatusCode).Render(fmt.Sprintf("%d", e.StatusCode)) + "\n")
	b.WriteString(label("Duration: ") + fmt.Sprintf("%.0f ms", e.DurationMs) + "\n\n")

	b.WriteString(styleSectionHeader.Render("Response Headers") + "\n")
	for k, vals := range e.ResponseHeaders {
		for _, v := range vals {
			b.WriteString(label(k+": ") + v + "\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(styleSectionHeader.Render("Response Body") + "\n")
	b.WriteString(formatBody(e.ResponseBody, e.IsBinaryResponse, e.ResponseHeaders.Get("Content-Type")))

	return b.String()
}

func (m *model) buildFooter() string {
	bind := func(key, action string) string {
		return styleFooterKey.Render(key) + styleFooter.Render(" "+action)
	}
	parts := []string{
		bind("arrows", "navigate"),
		bind("/", "filter"),
		bind("s", "save"),
		bind("e", "curl"),
		bind("q", "quit"),
	}
	if len(m.filtered) > 0 {
		parts = append(parts, styleFooter.Render(
			fmt.Sprintf("[%d/%d requests]", m.selected+1, len(m.filtered)),
		))
	} else {
		parts = append(parts, styleFooter.Render("[0 requests]"))
	}
	if m.savePath != "" {
		parts = append(parts, styleFooter.Render("saving to: "+m.savePath))
	}
	return strings.Join(parts, styleFooter.Render("  "))
}

func formatBody(body []byte, isBinary bool, contentType string) string {
	if len(body) == 0 {
		return lipgloss.NewStyle().Foreground(colorGray).Render("(empty)")
	}
	if isBinary {
		return lipgloss.NewStyle().Foreground(colorGray).Render(
			fmt.Sprintf("[binary data — %d bytes, content-type: %s]", len(body), contentType),
		)
	}
	text := string(body)
	if strings.Contains(contentType, "application/json") || isJSONLike(text) {
		var v interface{}
		if err := json.Unmarshal(body, &v); err == nil {
			if pretty, err := json.MarshalIndent(v, "", "  "); err == nil {
				return string(pretty)
			}
		}
	}
	return text
}

func isJSONLike(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[")
}

func buildCurlCommand(e *types.RequestEntry) string {
	var b strings.Builder
	b.WriteString("curl -X " + e.Method + " \\\n  '" + e.URL + "'")
	for k, vals := range e.RequestHeaders {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf(" \\\n  -H '%s: %s'", k, v))
		}
	}
	if len(e.RequestBody) > 0 && !e.IsBinaryRequest {
		escaped := strings.ReplaceAll(string(e.RequestBody), "'", "'\\''")
		b.WriteString(" \\\n  --data '" + escaped + "'")
	}
	return b.String()
}

func padRight(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 3 {
		return string(r[:n])
	}
	return string(r[:n-3]) + "..."
}
