package theme

import "github.com/charmbracelet/lipgloss"

// CatppuccinMochaTheme returns the Catppuccin Mocha theme
// A soothing pastel theme for cozy TUIs
// Based on: https://github.com/catppuccin/catppuccin
func CatppuccinMochaTheme() Theme {
	return Theme{
		Name: "catppuccin-mocha",

		// Background colors
		Background: lipgloss.Color("#1e1e2e"), // Base
		Foreground: lipgloss.Color("#cdd6f4"), // Text

		// UI elements
		Border:        lipgloss.Color("#45475a"), // Surface1
		BorderFocused: lipgloss.Color("#89b4fa"), // Blue
		Selection:     lipgloss.Color("#313244"), // Surface0
		Cursor:        lipgloss.Color("#f5e0dc"), // Rosewater

		// Status colors
		Success: lipgloss.Color("#a6e3a1"), // Green
		Warning: lipgloss.Color("#f9e2af"), // Yellow
		Error:   lipgloss.Color("#f38ba8"), // Red
		Info:    lipgloss.Color("#89dceb"), // Sky

		// Syntax highlighting
		Keyword:  lipgloss.Color("#cba6f7"), // Mauve
		String:   lipgloss.Color("#a6e3a1"), // Green
		Number:   lipgloss.Color("#fab387"), // Peach
		Comment:  lipgloss.Color("#6c7086"), // Overlay0
		Function: lipgloss.Color("#89b4fa"), // Blue
		Operator: lipgloss.Color("#94e2d5"), // Teal

		// Table colors
		TableHeader:      lipgloss.Color("#89b4fa"), // Blue
		TableRowEven:     lipgloss.Color("#1e1e2e"), // Base
		TableRowOdd:      lipgloss.Color("#181825"), // Mantle
		TableRowSelected: lipgloss.Color("#313244"), // Surface0

		// JSONB colors
		JSONKey:     lipgloss.Color("#89b4fa"), // Blue
		JSONString:  lipgloss.Color("#a6e3a1"), // Green
		JSONNumber:  lipgloss.Color("#fab387"), // Peach
		JSONBoolean: lipgloss.Color("#f9e2af"), // Yellow
		JSONNull:    lipgloss.Color("#6c7086"), // Overlay0

		// Tree/Navigator colors
		DatabaseActive:   lipgloss.Color("#a6e3a1"), // Green - active database
		DatabaseInactive: lipgloss.Color("#6c7086"), // Overlay0 - inactive database
		SchemaExpanded:   lipgloss.Color("#89b4fa"), // Blue - expanded schema
		SchemaCollapsed:  lipgloss.Color("#6c7086"), // Overlay0 - collapsed schema
		TableIcon:        lipgloss.Color("#cba6f7"), // Mauve - table icon
		ViewIcon:         lipgloss.Color("#94e2d5"), // Teal - view icon
		FunctionIcon:     lipgloss.Color("#f9e2af"), // Yellow - function icon
		ColumnIcon:       lipgloss.Color("#a6adc8"), // Subtext0 - column icon
		Metadata:         lipgloss.Color("#6c7086"), // Overlay0 - metadata text
		PrimaryKey:       lipgloss.Color("#f9e2af"), // Yellow - PK indicator
		ForeignKey:       lipgloss.Color("#89dceb"), // Sky - FK indicator
	}
}

// Additional Catppuccin colors available for future use:
// Rosewater: #f5e0dc
// Flamingo:  #f2cdcd
// Pink:      #f5c2e7
// Mauve:     #cba6f7
// Red:       #f38ba8
// Maroon:    #eba0ac
// Peach:     #fab387
// Yellow:    #f9e2af
// Green:     #a6e3a1
// Teal:      #94e2d5
// Sky:       #89dceb
// Sapphire:  #74c7ec
// Blue:      #89b4fa
// Lavender:  #b4befe
// Text:      #cdd6f4
// Subtext1:  #bac2de
// Subtext0:  #a6adc8
// Overlay2:  #9399b2
// Overlay1:  #7f849c
// Overlay0:  #6c7086
// Surface2:  #585b70
// Surface1:  #45475a
// Surface0:  #313244
// Base:      #1e1e2e
// Mantle:    #181825
// Crust:     #11111b
