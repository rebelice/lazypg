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

// View renders the panel with modern Catppuccin styling
func (p *Panel) View() string {
	if p.Width <= 0 || p.Height <= 0 {
		return ""
	}

	// Modern rounded border style
	borderStyle := lipgloss.RoundedBorder()

	// Calculate content area (subtract border + padding)
	contentHeight := p.Height - 2 // -2 for top and bottom borders
	if p.Title != "" {
		contentHeight -= 1 // -1 for title line
	}
	if contentHeight < 1 {
		contentHeight = 1 //nolint:ineffassign // kept for clarity
	}
	_ = contentHeight // silence unused warning, value used indirectly via layout

	// Create content with modern title
	var finalContent string
	if p.Title != "" {
		// Get border color from style to determine if focused
		borderColor := p.Style.GetBorderTopForeground()
		var titleStyle lipgloss.Style

		// Use blue for focused, lavender for unfocused
		if borderColor == lipgloss.Color("#89b4fa") { // Blue = focused
			titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#89b4fa")). // Blue
				Padding(0, 1)
		} else {
			titleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#b4befe")). // Lavender
				Padding(0, 1)
		}
		finalContent = titleStyle.Render("  "+p.Title) + "\n" + p.Content
	} else {
		finalContent = p.Content
	}

	// Apply border and sizing
	// Note: lipgloss Height() sets content height, then adds borders on top
	// So if we want total height of p.Height, we need to subtract border height (2)
	innerHeight := p.Height - 2 // Subtract top and bottom borders
	if innerHeight < 1 {
		innerHeight = 1
	}

	style := p.Style.
		Width(p.Width).
		Height(innerHeight). // This is the inner content height
		Border(borderStyle).
		Padding(0, 1) // Horizontal padding inside border

	return style.Render(finalContent)
}
