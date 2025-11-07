package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme and styling
type Theme struct {
	Name string

	// Background colors
	Background lipgloss.Color
	Foreground lipgloss.Color

	// UI elements
	Border        lipgloss.Color
	BorderFocused lipgloss.Color
	Selection     lipgloss.Color
	Cursor        lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Syntax highlighting (SQL)
	Keyword  lipgloss.Color
	String   lipgloss.Color
	Number   lipgloss.Color
	Comment  lipgloss.Color
	Function lipgloss.Color
	Operator lipgloss.Color

	// Table colors
	TableHeader      lipgloss.Color
	TableRowEven     lipgloss.Color
	TableRowOdd      lipgloss.Color
	TableRowSelected lipgloss.Color

	// JSONB colors
	JSONKey     lipgloss.Color
	JSONString  lipgloss.Color
	JSONNumber  lipgloss.Color
	JSONBoolean lipgloss.Color
	JSONNull    lipgloss.Color
}

// GetTheme returns a theme by name
func GetTheme(name string) Theme {
	switch name {
	case "default":
		return DefaultTheme()
	default:
		return DefaultTheme()
	}
}
