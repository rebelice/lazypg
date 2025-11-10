package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/jsonb"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// JSONBViewMode represents the display mode
type JSONBViewMode int

const (
	JSONBViewFormatted JSONBViewMode = iota
	JSONBViewTree
	JSONBViewQuery
)

// CloseJSONBViewerMsg is sent when viewer should close
type CloseJSONBViewerMsg struct{}

// JSONBViewer displays JSONB data in multiple modes
type JSONBViewer struct {
	Width  int
	Height int
	Theme  theme.Theme

	// Data
	value       interface{}
	formatted   string
	paths       []jsonb.Path
	currentMode JSONBViewMode

	// Tree view state
	selected     int
	offset       int
	expandedKeys map[string]bool
}

// NewJSONBViewer creates a new JSONB viewer
func NewJSONBViewer(th theme.Theme) *JSONBViewer {
	return &JSONBViewer{
		Width:        80,
		Height:       30,
		Theme:        th,
		currentMode:  JSONBViewFormatted,
		expandedKeys: make(map[string]bool),
	}
}

// SetValue sets the JSONB value to display
func (jv *JSONBViewer) SetValue(value interface{}) error {
	jv.value = value

	// Format the value
	formatted, err := jsonb.Format(value)
	if err != nil {
		return err
	}
	jv.formatted = formatted

	// Extract paths
	jv.paths = jsonb.ExtractPaths(value)
	jv.selected = 0
	jv.offset = 0

	return nil
}

// Update handles keyboard input
func (jv *JSONBViewer) Update(msg tea.KeyMsg) (*JSONBViewer, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return jv, func() tea.Msg {
			return CloseJSONBViewerMsg{}
		}
	case "1":
		jv.currentMode = JSONBViewFormatted
	case "2":
		jv.currentMode = JSONBViewTree
	case "3":
		jv.currentMode = JSONBViewQuery
	case "up", "k":
		if jv.currentMode == JSONBViewTree && jv.selected > 0 {
			jv.selected--
			if jv.selected < jv.offset {
				jv.offset = jv.selected
			}
		}
	case "down", "j":
		if jv.currentMode == JSONBViewTree && jv.selected < len(jv.paths)-1 {
			jv.selected++
			visibleHeight := jv.Height - 8
			if jv.selected >= jv.offset+visibleHeight {
				jv.offset = jv.selected - visibleHeight + 1
			}
		}
	}

	return jv, nil
}

// View renders the JSONB viewer
func (jv *JSONBViewer) View() string {
	var sections []string

	// Title bar with mode indicators
	titleStyle := lipgloss.NewStyle().
		Foreground(jv.Theme.Foreground).
		Background(jv.Theme.Info).
		Padding(0, 1).
		Bold(true)

	modes := []string{"1:Formatted", "2:Tree", "3:Query"}
	for i, mode := range modes {
		if JSONBViewMode(i) == jv.currentMode {
			modes[i] = "[" + mode + "]"
		}
	}
	title := "JSONB Viewer    " + strings.Join(modes, "  ")
	sections = append(sections, titleStyle.Render(title))

	// Instructions
	instrStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Padding(0, 1)
	sections = append(sections, instrStyle.Render("1-3: Switch mode  ↑↓: Navigate  Esc: Close"))

	// Content based on mode
	contentHeight := jv.Height - 6
	var content string

	switch jv.currentMode {
	case JSONBViewFormatted:
		content = jv.renderFormatted(contentHeight)
	case JSONBViewTree:
		content = jv.renderTree(contentHeight)
	case JSONBViewQuery:
		content = jv.renderQuery(contentHeight)
	}

	sections = append(sections, content)

	// Container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(jv.Theme.Border).
		Width(jv.Width).
		Height(jv.Height).
		Padding(1)

	return containerStyle.Render(strings.Join(sections, "\n"))
}

func (jv *JSONBViewer) renderFormatted(height int) string {
	lines := strings.Split(jv.formatted, "\n")

	// Limit to visible height
	if len(lines) > height {
		lines = lines[:height]
		lines = append(lines, "...")
	}

	style := lipgloss.NewStyle().
		Foreground(jv.Theme.Foreground).
		Padding(0, 1)

	return style.Render(strings.Join(lines, "\n"))
}

func (jv *JSONBViewer) renderTree(height int) string {
	var lines []string

	visibleStart := jv.offset
	visibleEnd := jv.offset + height
	if visibleEnd > len(jv.paths) {
		visibleEnd = len(jv.paths)
	}

	for i := visibleStart; i < visibleEnd; i++ {
		path := jv.paths[i]
		indent := strings.Repeat("  ", len(path.Parts))

		// Get path label
		label := "$"
		if len(path.Parts) > 0 {
			label = path.Parts[len(path.Parts)-1]
		}

		// Get value at path
		value, err := jsonb.GetValueAtPath(jv.value, path)
		valueStr := ""
		if err == nil {
			switch v := value.(type) {
			case string:
				valueStr = fmt.Sprintf(": \"%s\"", truncate(v, 30))
			case float64:
				valueStr = fmt.Sprintf(": %v", v)
			case bool:
				valueStr = fmt.Sprintf(": %v", v)
			case nil:
				valueStr = ": null"
			case map[string]interface{}:
				valueStr = fmt.Sprintf(" {%d keys}", len(v))
			case []interface{}:
				valueStr = fmt.Sprintf(" [%d items]", len(v))
			}
		}

		line := indent + label + valueStr

		// Highlight selected
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == jv.selected {
			style = style.Background(jv.Theme.Selection).Foreground(jv.Theme.Foreground)
		}

		lines = append(lines, style.Render(line))
	}

	return strings.Join(lines, "\n")
}

func (jv *JSONBViewer) renderQuery(height int) string {
	if jv.selected >= len(jv.paths) {
		return ""
	}

	selectedPath := jv.paths[jv.selected]

	var lines []string
	lines = append(lines, "Selected Path:")
	lines = append(lines, "  " + selectedPath.String())
	lines = append(lines, "")
	lines = append(lines, "PostgreSQL Queries:")
	lines = append(lines, "")

	// Assume column name is 'data'
	colName := "data"

	// #> operator (returns JSONB)
	lines = append(lines, fmt.Sprintf("Get JSONB value:"))
	lines = append(lines, fmt.Sprintf("  %s #> '%s'", colName, selectedPath.PostgreSQLPath()))
	lines = append(lines, "")

	// #>> operator (returns text)
	lines = append(lines, fmt.Sprintf("Get text value:"))
	lines = append(lines, fmt.Sprintf("  %s #>> '%s'", colName, selectedPath.PostgreSQLPath()))
	lines = append(lines, "")

	// @> operator (contains)
	lines = append(lines, fmt.Sprintf("Filter rows containing this path:"))
	lines = append(lines, fmt.Sprintf("  %s @> '{...}'", colName))

	style := lipgloss.NewStyle().
		Foreground(jv.Theme.Foreground).
		Padding(0, 1)

	return style.Render(strings.Join(lines, "\n"))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
