package theme

import "github.com/charmbracelet/lipgloss"

// DefaultTheme returns the default dark theme
func DefaultTheme() Theme {
	return Theme{
		Name: "default",

		// Background colors
		Background: lipgloss.Color("235"),
		Foreground: lipgloss.Color("252"),

		// UI elements
		Border:        lipgloss.Color("240"),
		BorderFocused: lipgloss.Color("62"),
		Selection:     lipgloss.Color("237"),
		Cursor:        lipgloss.Color("248"),

		// Status colors
		Success: lipgloss.Color("42"),
		Warning: lipgloss.Color("220"),
		Error:   lipgloss.Color("196"),
		Info:    lipgloss.Color("75"),

		// Syntax highlighting
		Keyword:  lipgloss.Color("75"),
		String:   lipgloss.Color("180"),
		Number:   lipgloss.Color("150"),
		Comment:  lipgloss.Color("65"),
		Function: lipgloss.Color("220"),
		Operator: lipgloss.Color("252"),

		// Table colors
		TableHeader:      lipgloss.Color("62"),
		TableRowEven:     lipgloss.Color("235"),
		TableRowOdd:      lipgloss.Color("236"),
		TableRowSelected: lipgloss.Color("237"),

		// JSONB colors
		JSONKey:     lipgloss.Color("117"),
		JSONString:  lipgloss.Color("180"),
		JSONNumber:  lipgloss.Color("150"),
		JSONBoolean: lipgloss.Color("75"),
		JSONNull:    lipgloss.Color("244"),
	}
}
