package app

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rebeliceyang/lazypg/internal/config"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/discovery"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/components"
	"github.com/rebeliceyang/lazypg/internal/ui/help"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// App is the main application model
type App struct {
	state      models.AppState
	config     *config.Config
	theme      theme.Theme
	leftPanel  components.Panel
	rightPanel components.Panel

	// Phase 2: Connection management
	connectionManager *connection.Manager
	discoverer        *discovery.Discoverer

	// Connection dialog
	showConnectionDialog bool
	connectionDialog     *components.ConnectionDialog
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
		state:             state,
		config:            cfg,
		theme:             th,
		connectionManager: connection.NewManager(),
		discoverer:        discovery.NewDiscoverer(),
		connectionDialog:  components.NewConnectionDialog(),
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
		// Handle connection dialog first if visible
		if a.showConnectionDialog {
			return a.handleConnectionDialog(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			// Don't quit if in help mode, exit help instead
			if a.state.ViewMode == models.HelpMode {
				a.state.ViewMode = models.NormalMode
				return a, nil
			}
			return a, tea.Quit
		case "?":
			// Toggle help
			if a.state.ViewMode == models.HelpMode {
				a.state.ViewMode = models.NormalMode
			} else {
				a.state.ViewMode = models.HelpMode
			}
		case "esc":
			// Exit help mode
			if a.state.ViewMode == models.HelpMode {
				a.state.ViewMode = models.NormalMode
			}
		case "c":
			// Open connection dialog
			a.showConnectionDialog = true
			// TODO: Trigger discovery
			return a, nil
		case "tab":
			// Only handle tab in normal mode
			if a.state.ViewMode == models.NormalMode {
				if a.state.FocusedPanel == models.LeftPanel {
					a.state.FocusedPanel = models.RightPanel
				} else {
					a.state.FocusedPanel = models.LeftPanel
				}
				a.updatePanelStyles()
			}
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
	// If connection dialog is showing, render it
	if a.showConnectionDialog {
		return a.renderConnectionDialog()
	}

	// If in help mode, show help overlay
	if a.state.ViewMode == models.HelpMode {
		return help.Render(a.state.Width, a.state.Height, lipgloss.NewStyle())
	}

	// Normal view rendering
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

// handleConnectionDialog handles key events when connection dialog is visible
func (a *App) handleConnectionDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showConnectionDialog = false
		return a, nil

	case "up", "k":
		a.connectionDialog.MoveSelection(-1)
		return a, nil

	case "down", "j":
		a.connectionDialog.MoveSelection(1)
		return a, nil

	case "m":
		a.connectionDialog.ManualMode = !a.connectionDialog.ManualMode
		return a, nil

	case "enter":
		// TODO: Implement connection logic
		a.showConnectionDialog = false
		return a, nil

	case "backspace":
		if a.connectionDialog.ManualMode {
			a.connectionDialog.HandleBackspace()
		}
		return a, nil

	default:
		// Handle text input in manual mode
		if a.connectionDialog.ManualMode {
			// Only handle printable characters
			key := msg.String()
			if len(key) == 1 {
				a.connectionDialog.HandleInput(rune(key[0]))
			}
		}
		return a, nil
	}
}

// renderConnectionDialog renders the connection dialog centered on screen
func (a *App) renderConnectionDialog() string {
	// Center the dialog
	dialogWidth := 60
	dialogHeight := 20

	a.connectionDialog.Width = dialogWidth
	a.connectionDialog.Height = dialogHeight

	dialog := a.connectionDialog.View()

	// Center it
	verticalPadding := (a.state.Height - dialogHeight) / 2
	horizontalPadding := (a.state.Width - dialogWidth) / 2

	if verticalPadding < 0 {
		verticalPadding = 0
	}
	if horizontalPadding < 0 {
		horizontalPadding = 0
	}

	style := lipgloss.NewStyle().
		Padding(verticalPadding, 0, 0, horizontalPadding)

	return style.Render(dialog)
}
