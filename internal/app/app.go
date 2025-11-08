package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/config"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/discovery"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
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

	// Error overlay
	showError    bool
	errorOverlay *components.ErrorOverlay

	// Phase 3: Navigation tree
	treeView *components.TreeView

	// Table view
	tableView    *components.TableView
	currentTable string // "schema.table"
}

// DiscoveryCompleteMsg is sent when discovery completes
type DiscoveryCompleteMsg struct {
	Instances []models.DiscoveredInstance
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Title   string
	Message string
}

// LoadTreeMsg requests loading the navigation tree
type LoadTreeMsg struct{}

// TreeLoadedMsg is sent when tree data is loaded
type TreeLoadedMsg struct {
	Root *models.TreeNode
	Err  error
}

// LoadTableDataMsg requests loading table data
type LoadTableDataMsg struct {
	Schema string
	Table  string
	Offset int
	Limit  int
}

// TableDataLoadedMsg is sent when table data is loaded
type TableDataLoadedMsg struct {
	Columns   []string
	Rows      [][]string
	TotalRows int
	Err       error
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

	// Create empty tree root
	emptyRoot := models.NewTreeNode("root", models.TreeNodeTypeRoot, "Databases")
	emptyRoot.Expanded = true

	app := &App{
		state:             state,
		config:            cfg,
		theme:             th,
		connectionManager: connection.NewManager(),
		discoverer:        discovery.NewDiscoverer(),
		connectionDialog:  components.NewConnectionDialog(),
		errorOverlay:      components.NewErrorOverlay(th),
		treeView:          components.NewTreeView(emptyRoot, th),
		tableView:         components.NewTableView(),
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
	case ErrorMsg:
		// Handle error messages
		a.ShowError(msg.Title, msg.Message)
		return a, nil

	case tea.KeyMsg:
		// Handle error overlay dismissal first if visible
		if a.showError {
			key := msg.String()
			if key == "esc" || key == "enter" {
				a.DismissError()
				return a, nil
			}
			// Allow quit keys to pass through even when error is showing
			if key == "q" || key == "ctrl+c" {
				return a, tea.Quit
			}
			// Consume all other keys when error is showing
			return a, nil
		}

		// Handle connection dialog if visible
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
			// Open connection dialog and trigger discovery
			a.showConnectionDialog = true
			return a, a.triggerDiscovery()
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
		default:
			// Handle tree navigation when left panel is focused
			if a.state.FocusedPanel == models.LeftPanel && a.state.ViewMode == models.NormalMode {
				var cmd tea.Cmd
				a.treeView, cmd = a.treeView.Update(msg)
				return a, cmd
			}

			// Handle table navigation when right panel is focused
			if a.state.FocusedPanel == models.RightPanel && a.state.ViewMode == models.NormalMode {
				switch msg.String() {
				case "up", "k":
					a.tableView.MoveSelection(-1)
					return a, nil
				case "down", "j":
					a.tableView.MoveSelection(1)

					// Check if we need to load more data (lazy loading)
					if a.tableView.SelectedRow >= len(a.tableView.Rows)-10 &&
						len(a.tableView.Rows) < a.tableView.TotalRows &&
						a.currentTable != "" {
						// Parse schema and table from currentTable
						parts := strings.Split(a.currentTable, ".")
						if len(parts) == 2 {
							return a, func() tea.Msg {
								return LoadTableDataMsg{
									Schema: parts[0],
									Table:  parts[1],
									Offset: len(a.tableView.Rows),
									Limit:  100,
								}
							}
						}
					}
					return a, nil
				case "ctrl+u":
					a.tableView.PageUp()
					return a, nil
				case "ctrl+d":
					a.tableView.PageDown()
					return a, nil
				}
			}
		}
	case DiscoveryCompleteMsg:
		// Update connection dialog with discovered instances
		a.connectionDialog.DiscoveredInstances = msg.Instances
		return a, nil

	case LoadTreeMsg:
		return a, a.loadTree

	case TreeLoadedMsg:
		if msg.Err != nil {
			a.ShowError("Database Error", fmt.Sprintf("Failed to load database structure:\n\n%v", msg.Err))
			return a, nil
		}
		// Update tree view with loaded data
		a.treeView.Root = msg.Root
		return a, nil

	case components.TreeNodeSelectedMsg:
		// Handle table selection
		if msg.Node != nil && msg.Node.Type == models.TreeNodeTypeTable {
			// Parse table info from node ID: "table:db.schema.table"
			// For now, we need to get schema and table name
			// Since we're using schema nodes with ID "schema:db.schema", we can get parent
			schemaNode := msg.Node.Parent
			if schemaNode != nil && schemaNode.Type == models.TreeNodeTypeSchema {
				schema := schemaNode.Label
				table := msg.Node.Label
				a.currentTable = schema + "." + table

				return a, func() tea.Msg {
					return LoadTableDataMsg{
						Schema: schema,
						Table:  table,
						Offset: 0,
						Limit:  100,
					}
				}
			}
		}
		return a, nil

	case LoadTableDataMsg:
		return a, a.loadTableData(msg)

	case TableDataLoadedMsg:
		if msg.Err != nil {
			a.ShowError("Database Error", fmt.Sprintf("Failed to load table data:\n\n%v", msg.Err))
			return a, nil
		}

		// Check if this is initial load or pagination
		if len(a.tableView.Rows) == 0 {
			// Initial load
			a.tableView.SetData(msg.Columns, msg.Rows, msg.TotalRows)
			a.state.FocusedPanel = models.RightPanel
			a.updatePanelStyles()
		} else {
			// Append paginated data
			a.tableView.Rows = append(a.tableView.Rows, msg.Rows...)
			a.tableView.TotalRows = msg.TotalRows
		}
		return a, nil

	case tea.WindowSizeMsg:
		a.state.Width = msg.Width
		a.state.Height = msg.Height
		a.updatePanelDimensions()
	}
	return a, nil
}

// View implements tea.Model
func (a *App) View() string {
	// If error overlay is showing, render it centered on top of everything
	if a.showError {
		return lipgloss.Place(
			a.state.Width, a.state.Height,
			lipgloss.Center, lipgloss.Center,
			a.errorOverlay.View(),
		)
	}

	// If connection dialog is showing, render it
	if a.showConnectionDialog {
		return a.renderConnectionDialog()
	}

	// If in help mode, show help overlay
	if a.state.ViewMode == models.HelpMode {
		return help.Render(a.state.Width, a.state.Height, lipgloss.NewStyle())
	}

	return a.renderNormalView()
}

// renderNormalView renders the normal application view
func (a *App) renderNormalView() string {

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

	// Update tree view dimensions and render
	a.treeView.Width = a.leftPanel.Width
	a.treeView.Height = a.leftPanel.Height
	a.leftPanel.Content = a.treeView.View()

	// Update table view dimensions and render
	a.tableView.Width = a.rightPanel.Width
	a.tableView.Height = a.rightPanel.Height
	a.rightPanel.Content = a.tableView.View()

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
		if a.connectionDialog.ManualMode {
			config, err := a.connectionDialog.GetManualConfig()
			if err != nil {
				// Invalid input - show error and don't close dialog
				a.ShowError("Invalid Configuration", fmt.Sprintf("Could not parse connection configuration\n\nError: %v", err))
				return a, nil
			}

			// Connect using manual configuration
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			connID, err := a.connectionManager.Connect(ctx, config)
			if err != nil {
				// Show error overlay
				a.ShowError("Connection Failed", fmt.Sprintf("Could not connect to %s:%d\n\nError: %v",
					config.Host, config.Port, err))
				return a, nil
			}

			// Update active connection in state
			conn, err := a.connectionManager.GetActive()
			if err == nil && conn != nil {
				a.state.ActiveConnection = &models.Connection{
					ID:          connID,
					Config:      config,
					Connected:   conn.Connected,
					ConnectedAt: conn.ConnectedAt,
					LastPing:    conn.LastPing,
					Error:       conn.Error,
				}
			}

			// Trigger tree loading
			a.showConnectionDialog = false
			return a, func() tea.Msg {
				return LoadTreeMsg{}
			}
		} else {
			// Get selected discovered instance
			instance := a.connectionDialog.GetSelectedInstance()
			if instance == nil {
				// No instance selected
				return a, nil
			}

			// Create connection config from discovered instance
			// Note: We'll need to prompt for database/user/password in future
			// For now, use common defaults
			config := models.ConnectionConfig{
				Host:     instance.Host,
				Port:     instance.Port,
				Database: "postgres", // Default database
				User:     os.Getenv("USER"), // Current user
				Password: "", // No password for now
				SSLMode:  "prefer",
			}

			// Connect using discovered instance
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			connID, err := a.connectionManager.Connect(ctx, config)
			if err != nil {
				// Show error overlay
				a.ShowError("Connection Failed", fmt.Sprintf("Could not connect to %s:%d\n\nError: %v",
					config.Host, config.Port, err))
				return a, nil
			}

			// Update active connection in state
			conn, err := a.connectionManager.GetActive()
			if err == nil && conn != nil {
				a.state.ActiveConnection = &models.Connection{
					ID:          connID,
					Config:      config,
					Connected:   conn.Connected,
					ConnectedAt: conn.ConnectedAt,
					LastPing:    conn.LastPing,
					Error:       conn.Error,
				}
			}

			// Trigger tree loading
			a.showConnectionDialog = false
			return a, func() tea.Msg {
				return LoadTreeMsg{}
			}
		}

		// Should not reach here
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

// triggerDiscovery runs discovery in the background and returns a command
func (a *App) triggerDiscovery() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		instances := a.discoverer.DiscoverAll(ctx)
		return DiscoveryCompleteMsg{Instances: instances}
	}
}

// loadTree loads the database structure and builds the navigation tree
func (a *App) loadTree() tea.Msg {
	ctx := context.Background()

	conn, err := a.connectionManager.GetActive()
	if err != nil {
		return TreeLoadedMsg{Err: fmt.Errorf("no active connection: %w", err)}
	}

	// Get current database name
	currentDB := conn.Config.Database

	// Build simple tree with just current database for now
	// Later we'll expand this to load schemas and tables
	root := models.BuildDatabaseTree([]string{currentDB}, currentDB)

	// Load schemas for the current database
	schemas, err := metadata.ListSchemas(ctx, conn.Pool)
	if err != nil {
		return TreeLoadedMsg{Err: fmt.Errorf("failed to load schemas: %w", err)}
	}

	// Find the database node
	dbNode := root.FindByID(fmt.Sprintf("db:%s", currentDB))
	if dbNode != nil {
		// Add schema nodes as children
		for _, schema := range schemas {
			schemaNode := models.NewTreeNode(
				fmt.Sprintf("schema:%s.%s", currentDB, schema.Name),
				models.TreeNodeTypeSchema,
				schema.Name,
			)
			schemaNode.Selectable = true

			// Load tables for this schema
			tables, err := metadata.ListTables(ctx, conn.Pool, schema.Name)
			if err == nil {
				for _, table := range tables {
					tableNode := models.NewTreeNode(
						fmt.Sprintf("table:%s.%s.%s", currentDB, schema.Name, table.Name),
						models.TreeNodeTypeTable,
						table.Name,
					)
					tableNode.Selectable = true
					schemaNode.AddChild(tableNode)
				}
				schemaNode.Loaded = true
			}

			dbNode.AddChild(schemaNode)
		}
		dbNode.Loaded = true
	}

	return TreeLoadedMsg{Root: root}
}

// loadTableData loads table data with pagination
func (a *App) loadTableData(msg LoadTableDataMsg) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		conn, err := a.connectionManager.GetActive()
		if err != nil {
			return TableDataLoadedMsg{Err: fmt.Errorf("no active connection: %w", err)}
		}

		data, err := metadata.QueryTableData(ctx, conn.Pool, msg.Schema, msg.Table, msg.Offset, msg.Limit)
		if err != nil {
			return TableDataLoadedMsg{Err: err}
		}

		return TableDataLoadedMsg{
			Columns:   data.Columns,
			Rows:      data.Rows,
			TotalRows: int(data.TotalRows),
		}
	}
}

// ShowError displays an error overlay with the given title and message
func (a *App) ShowError(title, message string) {
	a.errorOverlay.SetError(title, message)
	a.showError = true
}

// DismissError hides the error overlay
func (a *App) DismissError() {
	a.showError = false
}
