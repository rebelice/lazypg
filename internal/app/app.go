package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/commands"
	"github.com/rebeliceyang/lazypg/internal/config"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/discovery"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
	"github.com/rebeliceyang/lazypg/internal/db/query"
	filterBuilder "github.com/rebeliceyang/lazypg/internal/filter"
	"github.com/rebeliceyang/lazypg/internal/history"
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

	// Phase 4: Command palette
	showCommandPalette bool
	commandPalette     *components.CommandPalette
	commandRegistry    *commands.Registry

	// Quick query
	showQuickQuery bool
	quickQuery     *components.QuickQuery

	// History
	historyStore *history.Store

	// Filter builder
	showFilterBuilder bool
	filterBuilder     *components.FilterBuilder
	activeFilter      *models.Filter
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
	Offset    int   // Offset used in the query (0 for initial load)
	Err       error
}

// QueryResultMsg is sent when a query has been executed
type QueryResultMsg struct {
	SQL    string
	Result models.QueryResult
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

	// Initialize command registry
	registry := commands.NewRegistry()
	for _, cmd := range commands.GetBuiltinCommands() {
		registry.Register(cmd)
	}

	// Initialize history store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get home directory: %v", err)
		homeDir = "."
	}
	configDir := filepath.Join(homeDir, ".config", "lazypg")
	os.MkdirAll(configDir, 0755)

	historyPath := filepath.Join(configDir, "history.db")
	historyStore, err := history.NewStore(historyPath)
	if err != nil {
		log.Printf("Warning: Could not open history: %v", err)
	}

	// Initialize filter builder
	filterBuilder := components.NewFilterBuilder(th)

	app := &App{
		state:             state,
		config:            cfg,
		theme:             th,
		connectionManager: connection.NewManager(),
		discoverer:        discovery.NewDiscoverer(),
		connectionDialog:  components.NewConnectionDialog(),
		errorOverlay:      components.NewErrorOverlay(th),
		treeView:          components.NewTreeView(emptyRoot, th),
		commandRegistry:   registry,
		commandPalette:    components.NewCommandPalette(th),
		quickQuery:        components.NewQuickQuery(th),
		historyStore:      historyStore,
		tableView:         components.NewTableView(th),
		showFilterBuilder: false,
		filterBuilder:     filterBuilder,
		activeFilter:      nil,
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
	case commands.ConnectCommandMsg:
		// Handle connect command from palette
		a.showConnectionDialog = true
		return a, a.triggerDiscovery()

	case commands.RefreshCommandMsg:
		// Handle refresh command
		if a.state.ActiveConnection != nil {
			return a, func() tea.Msg {
				return LoadTreeMsg{}
			}
		}
		return a, nil

	case commands.QuickQueryCommandMsg:
		// Open quick query mode
		a.showQuickQuery = true
		return a, nil

	case commands.QueryEditorCommandMsg:
		// TODO: Future task
		a.ShowError("Not Implemented", "Query editor is a future enhancement")
		return a, nil

	case commands.HistoryCommandMsg:
		// TODO: Implement in Task 7
		a.ShowError("Not Implemented", "Query history will be implemented in Task 7")
		return a, nil

	case components.ExecuteQueryMsg:
		// Handle query execution from quick query
		if a.state.ActiveConnection == nil {
			a.ShowError("No Connection", "Please connect to a database first")
			return a, nil
		}

		// Execute query asynchronously
		return a, func() tea.Msg {
			conn, err := a.connectionManager.GetActive()
			if err != nil {
				return QueryResultMsg{
					SQL: msg.SQL,
					Result: models.QueryResult{
						Error: fmt.Errorf("failed to get connection: %w", err),
					},
				}
			}

			result := query.Execute(context.Background(), conn.Pool.GetPool(), msg.SQL)
			return QueryResultMsg{
				SQL:    msg.SQL,
				Result: result,
			}
		}

	case QueryResultMsg:
		// Record query to history
		if a.historyStore != nil {
			connName := ""
			dbName := ""
			if a.state.ActiveConnection != nil {
				connName = a.state.ActiveConnection.Config.Name
				dbName = a.state.ActiveConnection.Config.Database
			}

			entry := history.HistoryEntry{
				ConnectionName: connName,
				DatabaseName:   dbName,
				Query:          msg.SQL,
				Duration:       msg.Result.Duration,
				RowsAffected:   msg.Result.RowsAffected,
				Success:        msg.Result.Error == nil,
			}

			if msg.Result.Error != nil {
				entry.ErrorMessage = msg.Result.Error.Error()
			}

			// Record to history (ignore errors to not interrupt user flow)
			_ = a.historyStore.Add(entry)
		}

		// Handle query result
		if msg.Result.Error != nil {
			a.ShowError("Query Error", msg.Result.Error.Error())
			return a, nil
		}

		// Display results in table view
		a.tableView.SetData(msg.Result.Columns, msg.Result.Rows, len(msg.Result.Rows))
		a.state.FocusedPanel = models.RightPanel
		a.updatePanelStyles()

		// Show success message briefly (could add a toast notification system)
		// For now, just show in the status that would be visible

		return a, nil

	case components.ApplyFilterMsg:
		// Apply the filter and reload table data
		a.showFilterBuilder = false
		a.activeFilter = &msg.Filter

		// Reload table with filter
		if a.state.TreeSelected != nil && a.state.TreeSelected.Type == models.TreeNodeTypeTable {
			return a, a.loadTableDataWithFilter(*a.activeFilter)
		}
		return a, nil

	case components.CloseFilterBuilderMsg:
		a.showFilterBuilder = false
		return a, nil

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

		// Handle command palette if visible
		if a.showCommandPalette {
			return a.handleCommandPalette(msg)
		}

		// Handle quick query if visible
		if a.showQuickQuery {
			return a.handleQuickQuery(msg)
		}

		// Handle filter builder input
		if a.showFilterBuilder {
			return a.handleFilterBuilder(msg)
		}

		switch msg.String() {
		case "ctrl+p":
			// Open quick query
			a.showQuickQuery = true
			return a, nil
		case "ctrl+k":
			// Open command palette and populate with commands including history
			a.commandPalette.SetCommands(a.getCommandsWithHistory())
			a.showCommandPalette = true
			return a, nil
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
		case "f":
			// Open filter builder if on table view
			if a.state.FocusedPanel == models.RightPanel && a.state.TreeSelected != nil {
				if a.state.TreeSelected.Type == models.TreeNodeTypeTable {
					// Get column info from current table
					columns := a.getTableColumns()
					// Extract schema and table names
					schemaNode := a.state.TreeSelected.Parent
					if schemaNode != nil {
						a.filterBuilder.SetColumns(columns)
						a.filterBuilder.SetTable(schemaNode.Label, a.state.TreeSelected.Label)
						a.showFilterBuilder = true
					}
				}
			}
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
				case "enter", " ":
					// Consume enter/space in table view (no action needed for now)
					// This prevents the key from propagating to tree view
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
		// Initial load if:
		// 1. No existing rows (first load ever)
		// 2. Offset is 0 (fresh load request, even for same table)
		// 3. Columns changed (different table selected)
		isInitialLoad := len(a.tableView.Rows) == 0 ||
			msg.Offset == 0 ||
			(len(msg.Columns) > 0 && len(a.tableView.Columns) > 0 && msg.Columns[0] != a.tableView.Columns[0])

		if isInitialLoad {
			// Initial load - replace all data
			a.tableView.SetData(msg.Columns, msg.Rows, msg.TotalRows)
			a.tableView.SelectedRow = 0
			a.tableView.TopRow = 0
			a.state.FocusedPanel = models.RightPanel
			a.updatePanelStyles()
		} else {
			// Append paginated data (same table, loading more rows)
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

	// If command palette is showing, render it on top of everything
	if a.showCommandPalette {
		normalView := a.renderNormalView()
		// Set command palette dimensions
		a.commandPalette.Width = 80
		if a.commandPalette.Width > a.state.Width-4 {
			a.commandPalette.Width = a.state.Width - 4
		}
		a.commandPalette.Height = 20

		paletteView := lipgloss.Place(
			a.state.Width, a.state.Height,
			lipgloss.Center, lipgloss.Center,
			a.commandPalette.View(),
		)

		// Overlay palette on normal view
		return lipgloss.Place(
			a.state.Width, a.state.Height,
			lipgloss.Left, lipgloss.Top,
			normalView,
		) + "\n" + paletteView
	}

	// If in help mode, show help overlay
	if a.state.ViewMode == models.HelpMode {
		return help.Render(a.state.Width, a.state.Height, lipgloss.NewStyle())
	}

	return a.renderNormalView()
}

// renderNormalView renders the normal application view
func (a *App) renderNormalView() string {

	// Top bar with app name and connection status
	appNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(a.theme.BorderFocused).
		Background(a.theme.Selection)

	connStatus := ""
	if a.state.ActiveConnection != nil {
		// Build connection string
		conn := a.state.ActiveConnection
		connStr := fmt.Sprintf("%s@%s:%d/%s",
			conn.Config.User,
			conn.Config.Host,
			conn.Config.Port,
			conn.Config.Database)

		connStatus = " " + lipgloss.NewStyle().
			Foreground(a.theme.Success).
			Render("●") + " " +
			lipgloss.NewStyle().
			Foreground(a.theme.Foreground).
			Render(connStr)
	} else {
		connStatus = " " + lipgloss.NewStyle().
			Foreground(a.theme.Metadata).
			Render("○ Not connected")
	}

	topBarLeft := appNameStyle.Render(" 󱘖 LazyPG ") + connStatus
	topBarRight := lipgloss.NewStyle().
		Foreground(a.theme.Metadata).
		Render("? help")
	topBarContent := a.formatStatusBar(topBarLeft, topBarRight)

	// Create top bar as a bordered box
	topBarStyle := lipgloss.NewStyle().
		Width(a.state.Width). // Full width - lipgloss will handle borders
		Background(a.theme.Selection).
		Foreground(a.theme.Foreground).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		Padding(0, 1)

	topBar := topBarStyle.Render(topBarContent)

	// Context-sensitive bottom bar
	var bottomBarLeft string
	if a.state.FocusedPanel == models.LeftPanel {
		// Tree navigation keys
		keyStyle := lipgloss.NewStyle().Foreground(a.theme.BorderFocused)
		dimStyle := lipgloss.NewStyle().Foreground(a.theme.Metadata)

		bottomBarLeft = keyStyle.Render("↑↓") + dimStyle.Render(" navigate") +
			dimStyle.Render(" │ ") +
			keyStyle.Render("→←") + dimStyle.Render(" expand/collapse") +
			dimStyle.Render(" │ ") +
			keyStyle.Render("Enter") + dimStyle.Render(" select")
	} else {
		// Table navigation keys
		keyStyle := lipgloss.NewStyle().Foreground(a.theme.BorderFocused)
		dimStyle := lipgloss.NewStyle().Foreground(a.theme.Metadata)

		bottomBarLeft = keyStyle.Render("↑↓") + dimStyle.Render(" navigate") +
			dimStyle.Render(" │ ") +
			keyStyle.Render("Ctrl+D/U") + dimStyle.Render(" page")
	}

	// Common keys on the right
	keyStyle := lipgloss.NewStyle().Foreground(a.theme.BorderFocused)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Metadata)
	bottomBarRight := keyStyle.Render("Tab") + dimStyle.Render(" switch") +
		dimStyle.Render(" │ ") +
		keyStyle.Render("q") + dimStyle.Render(" quit")

	bottomBarContent := a.formatStatusBar(bottomBarLeft, bottomBarRight)

	// Create bottom bar as a bordered box
	bottomBarStyle := lipgloss.NewStyle().
		Width(a.state.Width). // Full width - lipgloss will handle borders
		Background(a.theme.Selection).
		Foreground(a.theme.Foreground).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Border).
		Padding(0, 1)

	bottomBar := bottomBarStyle.Render(bottomBarContent)

	// Update tree view dimensions and render
	// Calculate available content height: panel height - borders (2) - title line (1) - padding (0)
	treeContentHeight := a.leftPanel.Height - 3 // -2 for top/bottom borders, -1 for title
	if treeContentHeight < 1 {
		treeContentHeight = 1
	}
	a.treeView.Width = a.leftPanel.Width - 2 // -2 for horizontal padding inside panel
	a.treeView.Height = treeContentHeight
	a.leftPanel.Content = a.treeView.View()

	// Update table view dimensions and render
	// Calculate available content height: panel height - borders (2) - title line (1) - padding (0)
	tableContentHeight := a.rightPanel.Height - 3 // -2 for top/bottom borders, -1 for title
	if tableContentHeight < 1 {
		tableContentHeight = 1
	}
	a.tableView.Width = a.rightPanel.Width - 2 // -2 for horizontal padding inside panel
	a.tableView.Height = tableContentHeight
	a.rightPanel.Content = a.tableView.View()

	// Panels side by side
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		a.leftPanel.View(),
		a.rightPanel.View(),
	)

	// Combine all
	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		topBar,
		panels,
		bottomBar,
	)

	// If quick query is showing, replace bottom bar with it
	if a.showQuickQuery {
		a.quickQuery.Width = a.state.Width
		quickQueryView := a.quickQuery.View()

		// Replace bottom bar with quick query
		mainView = lipgloss.JoinVertical(
			lipgloss.Left,
			topBar,
			panels,
			quickQueryView,
		)
	}

	// Render filter builder if visible
	if a.showFilterBuilder {
		mainView = lipgloss.Place(
			a.state.Width,
			a.state.Height,
			lipgloss.Center,
			lipgloss.Center,
			a.filterBuilder.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#555555")),
		)
	}

	return mainView
}

// updatePanelDimensions calculates panel sizes based on window size
func (a *App) updatePanelDimensions() {
	if a.state.Width <= 0 || a.state.Height <= 0 {
		return
	}

	// Reserve space for top bar (3 lines) and bottom bar (3 lines)
	// Total: 6 lines, leaving Height - 6 for panels
	// Note: Panel.Height includes borders, so the actual content area is Height - 6
	contentHeight := a.state.Height - 6
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
	// Calculate available width accounting for:
	// - Borders: 2 chars (left + right)
	// - Padding: 2 chars (left + right, from Padding(0, 1))
	// Total: 4 chars
	availableWidth := a.state.Width - 4
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Use lipgloss.Width to get actual display width (ignoring ANSI codes)
	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)

	// If content is too wide, truncate
	if leftLen+rightLen > availableWidth {
		if availableWidth > rightLen {
			// Simple truncation - in a real app you'd want smarter truncation
			return left + right
		}
		return left
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

// getCommandsWithHistory returns all commands including recent history
func (a *App) getCommandsWithHistory() []models.Command {
	// Start with built-in commands
	commands := a.commandRegistry.GetAll()

	// Add recent history entries
	if a.historyStore != nil {
		entries, err := a.historyStore.GetRecent(10)
		if err == nil {
			for _, entry := range entries {
				// Truncate long queries for display
				displayQuery := entry.Query
				if len(displayQuery) > 60 {
					displayQuery = displayQuery[:57] + "..."
				}

				// Create command from history entry
				cmd := models.Command{
					Type:        models.CommandTypeHistory,
					Label:       displayQuery,
					Description: fmt.Sprintf("From %s • %s", entry.DatabaseName, entry.ExecutedAt.Format("Jan 2 15:04")),
					Icon:        "📜",
					Tags:        []string{"history", entry.DatabaseName},
					Action: func(sql string) tea.Cmd {
						return func() tea.Msg {
							return components.ExecuteQueryMsg{SQL: sql}
						}
					}(entry.Query), // Capture the query in closure
				}
				commands = append(commands, cmd)
			}
		}
	}

	return commands
}

// handleCommandPalette handles key events when command palette is visible
func (a *App) handleCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle close message
	var cmd tea.Cmd
	a.commandPalette, cmd = a.commandPalette.Update(msg)

	// Check if we got a close message
	if msg.String() == "esc" || msg.String() == "ctrl+c" {
		a.showCommandPalette = false
		return a, nil
	}

	// Execute the command if Enter was pressed
	if msg.String() == "enter" {
		a.showCommandPalette = false
	}

	return a, cmd
}

// handleQuickQuery handles key events when quick query is visible
func (a *App) handleQuickQuery(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.quickQuery, cmd = a.quickQuery.Update(msg)

	// Check if we should close
	if msg.String() == "esc" || msg.String() == "ctrl+c" {
		a.showQuickQuery = false
		return a, nil
	}

	// Execute query if Enter was pressed
	if msg.String() == "enter" {
		a.showQuickQuery = false
	}

	return a, cmd
}

// handleFilterBuilder handles key events when filter builder is visible
func (a *App) handleFilterBuilder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.filterBuilder, cmd = a.filterBuilder.Update(msg)
	return a, cmd
}

// getTableColumns returns column info for the current table
func (a *App) getTableColumns() []models.ColumnInfo {
	if a.state.TreeSelected == nil || a.state.TreeSelected.Type != models.TreeNodeTypeTable {
		return nil
	}

	// Extract columns from table view
	columns := []models.ColumnInfo{}

	// Get column names from the current table view
	// This is a placeholder - actual implementation depends on how you store column metadata
	// For now, we'll return columns from the tableView
	if len(a.tableView.Columns) > 0 {
		for _, header := range a.tableView.Columns {
			columns = append(columns, models.ColumnInfo{
				Name:     header,
				DataType: "text", // Default type, should be enhanced with actual type info
				IsArray:  false,
				IsJsonb:  false,
			})
		}
	}

	return columns
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
			Offset:    msg.Offset,
		}
	}
}

// loadTableDataWithFilter loads table data with an applied filter
func (a *App) loadTableDataWithFilter(filter models.Filter) tea.Cmd {
	return func() tea.Msg {
		conn, err := a.connectionManager.GetActive()
		if err != nil {
			return ErrorMsg{Title: "Connection Error", Message: err.Error()}
		}

		node := a.state.TreeSelected
		if node == nil || node.Type != models.TreeNodeTypeTable {
			return ErrorMsg{Title: "Error", Message: "No table selected"}
		}

		// Get schema from parent node
		schemaNode := node.Parent
		if schemaNode == nil {
			return ErrorMsg{Title: "Error", Message: "Cannot determine schema"}
		}

		// Build filtered query
		builder := filterBuilder.NewBuilder()
		whereClause, args, err := builder.BuildWhere(filter)
		if err != nil {
			return ErrorMsg{Title: "Filter Error", Message: err.Error()}
		}

		// Construct query
		query := fmt.Sprintf(
			"SELECT * FROM %s.%s %s LIMIT 100",
			schemaNode.Label,
			node.Label,
			whereClause,
		)

		// Execute query
		result, err := conn.Pool.QueryWithColumns(context.Background(), query, args...)
		if err != nil {
			return ErrorMsg{Title: "Query Error", Message: err.Error()}
		}

		// Convert to string rows for display
		var rows [][]string
		for _, row := range result.Rows {
			var strRow []string
			for _, col := range result.Columns {
				val := row[col]
				if val == nil {
					strRow = append(strRow, "NULL")
				} else {
					strRow = append(strRow, fmt.Sprintf("%v", val))
				}
			}
			rows = append(rows, strRow)
		}

		return TableDataLoadedMsg{
			Columns:   result.Columns,
			Rows:      rows,
			TotalRows: len(rows),
			Offset:    0,
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
