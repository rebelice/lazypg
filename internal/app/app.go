package app

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rebeliceyang/lazypg/internal/config"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/components"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// App is the main application model
type App struct {
	state      models.AppState
	config     *config.Config
	theme      theme.Theme
	leftPanel  components.Panel
	rightPanel components.Panel
}

// New creates a new App instance with config
func New(cfg *config.Config) *App {
	state := models.NewAppState()

	// Load theme
	themeName := "default"
	if cfg != nil && cfg.UI.Theme != "" {
		themeName = cfg.UI.Theme
	}
	th := theme.GetTheme(themeName)

	// Apply config to state
	if cfg != nil && cfg.UI.PanelWidthRatio > 0 && cfg.UI.PanelWidthRatio < 100 {
		state.LeftPanelWidth = cfg.UI.PanelWidthRatio
	}

	app := &App{
		state:  state,
		config: cfg,
		theme:  th,
		leftPanel: components.Panel{
			Title:   "Navigation",
			Content: "Databases\n└─ (empty)",
			Style:   lipgloss.NewStyle().BorderForeground(th.BorderFocused),
		},
		rightPanel: components.Panel{
			Title:   "Content",
			Content: "Select a database object to view",
			Style:   lipgloss.NewStyle().BorderForeground(th.Border),
		},
	}

	// Set initial panel dimensions and styles
	app.updatePanelDimensions()
	app.updatePanelStyles()

	return app
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			// Toggle focus between panels
			if a.state.FocusedPanel == models.LeftPanel {
				a.state.FocusedPanel = models.RightPanel
			} else {
				a.state.FocusedPanel = models.LeftPanel
			}
			a.updatePanelStyles()
		}
	case tea.WindowSizeMsg:
		a.state.Width = msg.Width
		a.state.Height = msg.Height
		a.updatePanelDimensions()
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	// Calculate status bar content dynamically
	topBarLeft := "lazypg"
	topBarRight := "⌘K"
	topBarContent := a.formatStatusBar(topBarLeft, topBarRight)

	// Top bar with theme colors
	topBar := lipgloss.NewStyle().
		Width(a.state.Width).
		Background(a.theme.BorderFocused).
		Foreground(lipgloss.Color("230")).
		Padding(0, 2).
		Render(topBarContent)

	// Calculate bottom bar content dynamically
	bottomBarLeft := "[tab] Switch panel | [q] Quit"
	bottomBarRight := "⌘K Command"
	bottomBarContent := a.formatStatusBar(bottomBarLeft, bottomBarRight)

	// Bottom bar with theme colors
	bottomBar := lipgloss.NewStyle().
		Width(a.state.Width).
		Background(a.theme.Selection).
		Foreground(a.theme.Foreground).
		Padding(0, 2).
		Render(bottomBarContent)

	// Panels side by side
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		a.leftPanel.View(),
		a.rightPanel.View(),
	)

	// Combine all
	return lipgloss.JoinVertical(
		lipgloss.Left,
		topBar,
		panels,
		bottomBar,
	)
}

// updatePanelDimensions calculates panel sizes based on window size
func (a *App) updatePanelDimensions() {
	if a.state.Width <= 0 || a.state.Height <= 0 {
		return
	}

	// Reserve space for top bar (1 line) and bottom bar (1 line)
	// Total: 2 lines, leaving Height - 2 for panels
	contentHeight := a.state.Height - 2
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Calculate panel widths
	// Each panel has a border (2 chars wide: left + right borders)
	// Total border width: 4 chars (2 per panel)
	leftWidth := (a.state.Width * a.state.LeftPanelWidth) / 100
	if leftWidth < 20 {
		leftWidth = 20
	}

	// Right panel gets remaining width after accounting for left panel and both borders
	// Subtract 4 to account for borders on both panels (2 chars each)
	rightWidth := a.state.Width - leftWidth - 4
	if rightWidth < 20 {
		rightWidth = 20
		// If right panel is too small, reduce left panel width
		leftWidth = a.state.Width - rightWidth - 4
	}

	a.leftPanel.Width = leftWidth
	a.leftPanel.Height = contentHeight
	a.rightPanel.Width = rightWidth
	a.rightPanel.Height = contentHeight
}

// updatePanelStyles updates panel styling based on focus
func (a *App) updatePanelStyles() {
	if a.state.FocusedPanel == models.LeftPanel {
		a.leftPanel.Style = lipgloss.NewStyle().BorderForeground(a.theme.BorderFocused)
		a.rightPanel.Style = lipgloss.NewStyle().BorderForeground(a.theme.Border)
	} else {
		a.leftPanel.Style = lipgloss.NewStyle().BorderForeground(a.theme.Border)
		a.rightPanel.Style = lipgloss.NewStyle().BorderForeground(a.theme.BorderFocused)
	}
}

// formatStatusBar formats a status bar with left and right aligned content
func (a *App) formatStatusBar(left, right string) string {
	// Account for padding (2 chars on each side = 4 total)
	availableWidth := a.state.Width - 4
	if availableWidth < 0 {
		availableWidth = 0
	}

	leftLen := len(left)
	rightLen := len(right)

	// If content is too wide, truncate
	if leftLen+rightLen > availableWidth {
		if availableWidth > rightLen {
			return left[:availableWidth-rightLen] + right
		}
		return left[:availableWidth]
	}

	// Calculate spacing between left and right content
	spacing := availableWidth - leftLen - rightLen
	if spacing < 0 {
		spacing = 0
	}

	return left + lipgloss.NewStyle().Width(spacing).Render("") + right
}
