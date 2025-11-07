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

// View renders the panel
func (p *Panel) View() string {
	if p.Width <= 0 || p.Height <= 0 {
		return ""
	}

	// Create border style
	style := p.Style.
		Width(p.Width).
		Height(p.Height).
		Border(lipgloss.RoundedBorder())

	// Add title if present
	content := p.Content
	if p.Title != "" {
		titleStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
		content = titleStyle.Render(p.Title) + "\n" + content
	}

	return style.Render(content)
}
