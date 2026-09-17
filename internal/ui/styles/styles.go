// Package styles centralizes the Lipgloss theme used across all UI
// components. Colors are AdaptiveColor so the TUI looks correct on both
// light and dark terminal backgrounds.
package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#5B9BF7"} // true blue accent (text/titles/selection)
	// ColorStatusBarBg is a deep, muted navy used only for the status bar's
	// full-width background. It's deliberately its own (darker, desaturated)
	// color rather than reusing ColorPrimary: a bright accent blue as a solid
	// background bar reads as glaring, whereas this stays legible without
	// being harsh in either light or dark terminals.
	ColorStatusBarBg = lipgloss.AdaptiveColor{Light: "#1E3A6E", Dark: "#152A4D"}
	ColorSecondary   = lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#9B9B9B"} // muted gray
	ColorSuccess     = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"} // green
	ColorWarning     = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"} // orange
	ColorError       = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"} // red
	ColorBorder      = lipgloss.AdaptiveColor{Light: "#D0D7DE", Dark: "#3D3D3D"}
	ColorOnAccent    = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"} // text on StatusBar/Error backgrounds
)

var (
	StatusBar = lipgloss.NewStyle().
			Background(ColorStatusBarBg).
			Foreground(ColorOnAccent).
			Padding(0, 1)

	HelpBar = lipgloss.NewStyle().
		Foreground(ColorSecondary)

	ErrorBanner = lipgloss.NewStyle().
			Background(ColorError).
			Foreground(ColorOnAccent).
			Padding(0, 1)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

	Selected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	Faint = lipgloss.NewStyle().
		Foreground(ColorSecondary)

	Border = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder)
)
