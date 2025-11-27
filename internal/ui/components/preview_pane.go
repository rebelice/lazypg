package components

import (
	"encoding/json"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/jsonb"
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

// SetContent sets the content to display
// isTruncated indicates whether the content was truncated in the parent view
func (p *PreviewPane) SetContent(content, title string, isTruncated bool) {
	p.Content = content
	p.Title = title
	p.IsTruncated = isTruncated
	p.scrollY = 0

	// Format content
	p.formatContent()

	// Update visibility (only auto-show if not force hidden)
	if !p.ForceHidden {
		p.Visible = isTruncated && content != "" && content != "NULL"
	}
}

// formatContent formats the raw content for display
func (p *PreviewPane) formatContent() {
	if p.Content == "" {
		p.contentLines = []string{}
		return
	}

	// Calculate available width for content
	contentWidth := p.Width - p.style.GetHorizontalFrameSize()
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Try to format as JSON if it looks like JSONB
	formatted := p.Content
	if jsonb.IsJSONB(p.Content) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(p.Content), &parsed); err == nil {
			if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				formatted = string(pretty)
			}
		}
	}

	// Wrap lines to fit width
	p.contentLines = p.wrapText(formatted, contentWidth)
}

// wrapText wraps text to fit within maxWidth
func (p *PreviewPane) wrapText(text string, maxWidth int) []string {
	var result []string
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if runewidth.StringWidth(line) <= maxWidth {
			result = append(result, line)
			continue
		}

		// Wrap long lines
		current := ""
		currentWidth := 0
		for _, r := range line {
			rWidth := runewidth.RuneWidth(r)
			if currentWidth+rWidth > maxWidth {
				result = append(result, current)
				current = string(r)
				currentWidth = rWidth
			} else {
				current += string(r)
				currentWidth += rWidth
			}
		}
		if current != "" {
			result = append(result, current)
		}
	}

	return result
}

// Toggle toggles the preview pane visibility
// When toggled off, sets ForceHidden to prevent auto-show
// When toggled on, clears ForceHidden to allow auto-show
func (p *PreviewPane) Toggle() {
	if p.Visible {
		p.Visible = false
		p.ForceHidden = true
	} else {
		p.ForceHidden = false
		// Only show if content is truncated
		p.Visible = p.IsTruncated && p.Content != "" && p.Content != "NULL"
	}
}

// Height returns the actual rendered height including borders
// Returns 0 if not visible
func (p *PreviewPane) Height() int {
	if !p.Visible {
		return 0
	}

	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	if maxContentHeight < 3 {
		maxContentHeight = 3
	}

	// Content lines that will be shown (excluding header and footer)
	contentLinesCount := len(p.contentLines)
	maxContentLines := maxContentHeight - 2 // -2 for header and footer
	if maxContentLines < 1 {
		maxContentLines = 1
	}
	if contentLinesCount > maxContentLines {
		contentLinesCount = maxContentLines
	}

	// Total = header (1) + content + footer (1)
	totalLines := 1 + contentLinesCount + 1

	return totalLines + p.style.GetVerticalFrameSize()
}

// IsScrollable returns true if content exceeds visible area
func (p *PreviewPane) IsScrollable() bool {
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	return len(p.contentLines) > maxContentHeight
}

// ScrollUp scrolls content up
func (p *PreviewPane) ScrollUp() {
	if p.scrollY > 0 {
		p.scrollY--
	}
}

// ScrollDown scrolls content down
func (p *PreviewPane) ScrollDown() {
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	maxContentLines := maxContentHeight - 2 // -2 for header and footer
	if maxContentLines < 1 {
		maxContentLines = 1
	}
	maxScroll := len(p.contentLines) - maxContentLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollY < maxScroll {
		p.scrollY++
	}
}

// GetContent returns the raw content for copying
func (p *PreviewPane) GetContent() string {
	return p.Content
}

// CopyContent copies the preview content to clipboard
func (p *PreviewPane) CopyContent() error {
	return clipboard.WriteAll(p.Content)
}

// View renders the preview pane
func (p *PreviewPane) View() string {
	if !p.Visible {
		return ""
	}

	// Calculate dimensions
	contentWidth := p.Width - p.style.GetHorizontalFrameSize()
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	if maxContentHeight < 1 {
		maxContentHeight = 1
	}

	// Build header
	titleStyle := lipgloss.NewStyle().
		Foreground(p.Theme.Info).
		Bold(true)

	header := titleStyle.Render("Preview")
	if p.Title != "" {
		header = titleStyle.Render("Preview: " + p.Title)
	}

	// Truncate header if too long
	if runewidth.StringWidth(header) > contentWidth-4 {
		header = runewidth.Truncate(header, contentWidth-4, "...")
	}

	// Get visible content lines
	startLine := p.scrollY
	endLine := startLine + maxContentHeight - 2 // -2 for header and footer
	if endLine > len(p.contentLines) {
		endLine = len(p.contentLines)
	}

	var contentParts []string
	contentParts = append(contentParts, header)

	// Add content lines
	contentStyle := lipgloss.NewStyle().Foreground(p.Theme.Foreground)
	for i := startLine; i < endLine; i++ {
		line := p.contentLines[i]
		// Truncate line if too long
		if runewidth.StringWidth(line) > contentWidth {
			line = runewidth.Truncate(line, contentWidth, "...")
		}
		contentParts = append(contentParts, contentStyle.Render(line))
	}

	// Build help text
	helpParts := []string{}
	if p.IsScrollable() {
		helpParts = append(helpParts, "↑↓: Scroll")
	}
	helpParts = append(helpParts, "y: Copy", "p: Toggle")

	// Add JSONB hint if content is JSON
	if jsonb.IsJSONB(p.Content) {
		helpParts = append(helpParts, "J: Tree")
	}

	helpText := strings.Join(helpParts, " │ ")
	helpStyle := lipgloss.NewStyle().
		Foreground(p.Theme.Metadata).
		Italic(true)

	// Build footer with right-aligned help
	footerPadding := contentWidth - runewidth.StringWidth(helpText)
	if footerPadding < 0 {
		footerPadding = 0
	}
	footer := strings.Repeat(" ", footerPadding) + helpStyle.Render(helpText)
	contentParts = append(contentParts, footer)

	// Join content
	content := strings.Join(contentParts, "\n")

	// Apply container style (p.style already has borders and padding)
	// Don't set Width/MaxHeight here as they would be added to frame size
	return p.style.Render(content)
}
