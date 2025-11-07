package app

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/components"
)

// App is the main application model
type App struct {
	state      models.AppState
	leftPanel  components.Panel
	rightPanel components.Panel
}

// New creates a new App instance
func New() *App {
	return &App{
		state: models.NewAppState(),
		leftPanel: components.Panel{
			Title: "Navigation",
			Content: "Databases\n└─ (empty)",
			Style: lipgloss.NewStyle().BorderForeground(lipgloss.Color("62")),
		},
		rightPanel: components.Panel{
			Title: "Content",
			Content: "Select a database object to view",
			Style: lipgloss.NewStyle().BorderForeground(lipgloss.Color("240")),
		},
	}
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
	// Top bar
	topBar := lipgloss.NewStyle().
		Width(a.state.Width).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 2).
		Render("lazypg                                                      ⌘K")

	// Bottom bar
	bottomBar := lipgloss.NewStyle().
		Width(a.state.Width).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("250")).
		Padding(0, 2).
		Render("[tab] Switch panel | [q] Quit                      ⌘K Command")

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

	// Reserve space for top and bottom bars (3 lines each with padding)
	contentHeight := a.state.Height - 3
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Calculate panel widths
	leftWidth := (a.state.Width * a.state.LeftPanelWidth) / 100
	if leftWidth < 20 {
		leftWidth = 20
	}
	rightWidth := a.state.Width - leftWidth - 2 // Account for borders
	if rightWidth < 20 {
		rightWidth = 20
	}

	a.leftPanel.Width = leftWidth
	a.leftPanel.Height = contentHeight
	a.rightPanel.Width = rightWidth
	a.rightPanel.Height = contentHeight
}

// updatePanelStyles updates panel styling based on focus
func (a *App) updatePanelStyles() {
	focusedColor := lipgloss.Color("62")
	unfocusedColor := lipgloss.Color("240")

	if a.state.FocusedPanel == models.LeftPanel {
		a.leftPanel.Style = lipgloss.NewStyle().BorderForeground(focusedColor)
		a.rightPanel.Style = lipgloss.NewStyle().BorderForeground(unfocusedColor)
	} else {
		a.leftPanel.Style = lipgloss.NewStyle().BorderForeground(unfocusedColor)
		a.rightPanel.Style = lipgloss.NewStyle().BorderForeground(focusedColor)
	}
}
