package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors — exported for use by modules and layout.
	Subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	Accent    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	Warn      = lipgloss.AdaptiveColor{Light: "#F25D94", Dark: "#F55385"}
	Highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	DimText   = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}

	statLabelStyle = lipgloss.NewStyle().
			Foreground(Subtle).
			Transform(strings.ToUpper).
			Width(16)

	statValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Bold(true)
)

// RenderBar draws a horizontal bar chart.
func RenderBar(percent float64, width int, color lipgloss.TerminalColor) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(float64(width) * (percent / 100))
	empty := width - filled
	if empty < 0 {
		empty = 0
	}
	if filled > width {
		filled = width
	}
	return lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(Subtle).Render(strings.Repeat("░", empty))
}

// RenderStat renders a label-value pair.
func RenderStat(label, val string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		statLabelStyle.Render(label),
		statValueStyle.Render(val),
	)
}

// Truncate shortens a string to max runes.
func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}
