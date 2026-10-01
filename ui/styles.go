package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	koStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	selectStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	boldStyle   = lipgloss.NewStyle().Bold(true)
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	focusStyle  = paneStyle.BorderForeground(lipgloss.Color("69"))
)

// fit truncates or pads s to exactly w cells
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) > w {
		r := []rune(s)
		for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
			r = r[:len(r)-1]
		}
		s = string(r) + "…"
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

// fitLeft is like fit but keeps the end of s, which is the meaningful part of a path
func fitLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) > w {
		r := []rune(s)
		for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
			r = r[1:]
		}
		s = "…" + string(r)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}
