package models

// AppState holds the application state
type AppState struct {
	Width          int
	Height         int
	LeftPanelWidth int
	FocusedPanel   PanelType
	ViewMode       ViewMode
}

// PanelType identifies which panel is focused
type PanelType int

const (
	LeftPanel PanelType = iota
	RightPanel
)

// ViewMode identifies the current view
type ViewMode int

const (
	NormalMode ViewMode = iota
	HelpMode
)

// NewAppState creates a new AppState with defaults
func NewAppState() AppState {
	return AppState{
		Width:          80,
		Height:         24,
		LeftPanelWidth: 25,
		FocusedPanel:   LeftPanel,
		ViewMode:       NormalMode,
	}
}
