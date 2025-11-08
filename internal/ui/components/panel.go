package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Panel represents a UI panel
type Panel struct {
	Title   string
	Content string
	Width   int
	Height  int
	Style   lipgloss.Style
}

// View renders the panel with modern styling
func (p *Panel) View() string {
	if p.Width <= 0 || p.Height <= 0 {
		return ""
	}

	// Modern double border style
	borderStyle := lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	// Calculate content area (subtract border + padding)
	contentHeight := p.Height - 2 // -2 for top and bottom borders
	if p.Title != "" {
		contentHeight -= 1 // -1 for title line
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Create content with title
	var finalContent string
	if p.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")). // Bright blue
			Padding(0, 1)
		finalContent = titleStyle.Render(p.Title) + "\n" + p.Content
	} else {
		finalContent = p.Content
	}

	// Apply border and sizing
	style := p.Style.
		Width(p.Width).
		Height(p.Height).
		Border(borderStyle).
		Padding(0, 1) // Horizontal padding inside border

	return style.Render(finalContent)
}
