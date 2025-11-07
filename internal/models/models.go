package models

// AppState holds the application state
type AppState struct {
	Width         int
	Height        int
	LeftPanelWidth int  // Width percentage (0-100)
	FocusedPanel  PanelType
}

// PanelType identifies which panel is focused
type PanelType int

const (
	LeftPanel PanelType = iota
	RightPanel
)

// NewAppState creates a new AppState with defaults
func NewAppState() AppState {
	return AppState{
		Width:         80,
		Height:        24,
		LeftPanelWidth: 25, // 25% for left panel
		FocusedPanel:  LeftPanel,
	}
}
