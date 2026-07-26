package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/xgg-2/netcli/internal/types"
)

var (
	colorBlue   = lipgloss.Color("#5C9CF5")
	colorYellow = lipgloss.Color("#F5C842")
	colorOrange = lipgloss.Color("#F5A623")
	colorRed    = lipgloss.Color("#F55C5C")
	colorGreen  = lipgloss.Color("#5CF590")
	colorGray   = lipgloss.Color("#6B7280")
	colorWhite  = lipgloss.Color("#F9FAFB")
	colorBg     = lipgloss.Color("#111827")
	colorBgAlt  = lipgloss.Color("#1F2937")
	colorBorder = lipgloss.Color("#374151")
	colorAccent = lipgloss.Color("#818CF8")
	colorTeal   = lipgloss.Color("#2DD4BF")
	colorPink   = lipgloss.Color("#F472B6")

	styleBase = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorWhite)

	styleSelectedRow = lipgloss.NewStyle().
				Background(colorBgAlt).
				Foreground(colorWhite).
				Bold(true)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleFooter = lipgloss.NewStyle().
			Background(colorBgAlt).
			Foreground(colorGray).
			Padding(0, 1)

	styleFooterKey = lipgloss.NewStyle().
			Background(colorBgAlt).
			Foreground(colorAccent).
			Bold(true)

	styleDivider = lipgloss.NewStyle().
			Foreground(colorBorder)

	styleSectionHeader = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Underline(true)

	styleFilter = lipgloss.NewStyle().
			Background(colorBgAlt).
			Foreground(colorYellow).
			Padding(0, 1)

	styleSavePrompt = lipgloss.NewStyle().
			Background(colorBgAlt).
			Foreground(colorGreen).
			Padding(0, 1)

	styleExportPrompt = lipgloss.NewStyle().
				Background(colorBgAlt).
				Foreground(colorTeal).
				Padding(0, 1)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen)
)

func methodColor(method string) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true)
	switch method {
	case "GET":
		return base.Foreground(colorBlue)
	case "POST":
		return base.Foreground(colorYellow)
	case "PUT", "PATCH":
		return base.Foreground(colorOrange)
	case "DELETE":
		return base.Foreground(colorRed)
	default:
		return base.Foreground(colorGray)
	}
}

func statusColor(code int) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true)
	switch {
	case code >= 200 && code < 300:
		return base.Foreground(colorGreen)
	case code >= 300 && code < 400:
		return base.Foreground(colorYellow)
	case code >= 400:
		return base.Foreground(colorRed)
	default:
		return base.Foreground(colorGray)
	}
}

func typeColor(rt types.ResourceType) lipgloss.Style {
	base := lipgloss.NewStyle()
	switch rt {
	case types.TypeXHR:
		return base.Foreground(colorYellow)
	case types.TypeDoc:
		return base.Foreground(colorBlue)
	case types.TypeJS:
		return base.Foreground(colorOrange)
	case types.TypeCSS:
		return base.Foreground(colorTeal)
	case types.TypeImg:
		return base.Foreground(colorPink)
	case types.TypeFont:
		return base.Foreground(colorAccent)
	case types.TypeMedia:
		return base.Foreground(colorRed)
	default:
		return base.Foreground(colorGray)
	}
}
