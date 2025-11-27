package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// PreviewPane displays full content for truncated values
type PreviewPane struct {
	Width     int
	MaxHeight int    // Maximum height (screen 1/3)
	Content   string // Raw content to display
	Title     string // Title (column name or JSON path)

	// Visibility state
	Visible       bool // Whether pane should be shown
	ForceHidden   bool // User manually hid the pane (overrides auto-show)
	IsTruncated   bool // Whether content was truncated in parent view

	// Scrolling
	scrollY       int
	contentLines  []string // Formatted content split into lines

	// Styling
	Theme theme.Theme
	style lipgloss.Style
}

// NewPreviewPane creates a new preview pane
func NewPreviewPane(th theme.Theme) *PreviewPane {
	return &PreviewPane{
		Width:     80,
		MaxHeight: 10,
		Theme:     th,
		style: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(th.Border).
			Padding(0, 1),
	}
}
