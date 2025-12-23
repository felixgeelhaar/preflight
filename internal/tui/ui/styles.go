// Package ui provides shared styles, key bindings, and messages for TUI components.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme colors (Catppuccin Mocha inspired).
var (
	ColorPrimary    = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"} // Blue
	ColorSecondary  = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#cba6f7"} // Mauve
	ColorSuccess    = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"} // Green
	ColorWarning    = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"} // Yellow
	ColorError      = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"} // Red
	ColorMuted      = lipgloss.AdaptiveColor{Light: "#6c6f85", Dark: "#6c7086"} // Overlay0
	ColorText       = lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"} // Text
	ColorSubtle     = lipgloss.AdaptiveColor{Light: "#9ca0b0", Dark: "#a6adc8"} // Subtext0
	ColorBackground = lipgloss.AdaptiveColor{Light: "#eff1f5", Dark: "#1e1e2e"} // Base
	ColorSurface    = lipgloss.AdaptiveColor{Light: "#e6e9ef", Dark: "#313244"} // Surface0
)

// Styles contains reusable lipgloss styles for the TUI.
type Styles struct {
	// Base styles
	App       lipgloss.Style
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Paragraph lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// Interactive elements
	Button         lipgloss.Style
	ButtonActive   lipgloss.Style
	ButtonDisabled lipgloss.Style

	// List items
	ListItem       lipgloss.Style
	ListItemActive lipgloss.Style

	// Panels
	Panel       lipgloss.Style
	PanelTitle  lipgloss.Style
	PanelBorder lipgloss.Style

	// Help text
	Help    lipgloss.Style
	HelpKey lipgloss.Style

	// Progress
	ProgressBar lipgloss.Style
	Spinner     lipgloss.Style

	// Diff view
	DiffAdd    lipgloss.Style
	DiffRemove lipgloss.Style
	DiffHeader lipgloss.Style
}

// DefaultStyles returns the default TUI styles.
func DefaultStyles() Styles {
	return Styles{
		// Base styles
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(ColorSecondary),

		Paragraph: lipgloss.NewStyle().
			Foreground(ColorText),

		// Status styles
		Success: lipgloss.NewStyle().
			Foreground(ColorSuccess),

		Warning: lipgloss.NewStyle().
			Foreground(ColorWarning),

		Error: lipgloss.NewStyle().
			Foreground(ColorError),

		Info: lipgloss.NewStyle().
			Foreground(ColorPrimary),

		// Button styles
		Button: lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorText).
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted),

		ButtonActive: lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorBackground).
			Background(ColorPrimary).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary),

		ButtonDisabled: lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorMuted).
			Background(ColorSurface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted),

		// List items
		ListItem: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(ColorText),

		ListItemActive: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(ColorPrimary).
			Bold(true),

		// Panels
		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2),

		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1),

		PanelBorder: lipgloss.NewStyle().
			BorderForeground(ColorPrimary),

		// Help text
		Help: lipgloss.NewStyle().
			Foreground(ColorMuted),

		HelpKey: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		// Progress
		ProgressBar: lipgloss.NewStyle().
			Foreground(ColorSuccess),

		Spinner: lipgloss.NewStyle().
			Foreground(ColorPrimary),

		// Diff view
		DiffAdd: lipgloss.NewStyle().
			Foreground(ColorSuccess),

		DiffRemove: lipgloss.NewStyle().
			Foreground(ColorError),

		DiffHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary),
	}
}

// WithWidth returns styles adapted for a specific terminal width.
func (s Styles) WithWidth(width int) Styles {
	s.Panel = s.Panel.Width(width - 4)
	s.App = s.App.Width(width)
	return s
}
