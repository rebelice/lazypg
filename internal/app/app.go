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
	"github.com/charmbracelet/x/ansi"
	"github.com/rebeliceyang/lazypg/internal/commands"
	"github.com/rebeliceyang/lazypg/internal/config"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/discovery"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
	"github.com/rebeliceyang/lazypg/internal/db/query"
	"github.com/rebeliceyang/lazypg/internal/connection_history"
	"github.com/rebeliceyang/lazypg/internal/favorites"
	filterBuilder "github.com/rebeliceyang/lazypg/internal/filter"
	"github.com/rebeliceyang/lazypg/internal/history"
	"github.com/rebeliceyang/lazypg/internal/jsonb"
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

	// JSONB viewer
	showJSONBViewer bool
	jsonbViewer     *components.JSONBViewer

	// Structure view
	showStructureView bool
	structureView     *components.StructureView
	currentTab        int // 0=Data, 1=Columns, 2=Constraints, 3=Indexes

	// Favorites
	showFavorites    bool
	favoritesManager *favorites.Manager
	favoritesDialog  *components.FavoritesDialog

	// Connection history
	connectionHistory *connection_history.Manager

	// Search input
	showSearch  bool
	searchInput *components.SearchInput
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
	Schema     string
	Table      string
	Offset     int
	Limit      int
	SortColumn string
	SortDir    string
	NullsFirst bool
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

	// Initialize favorites manager
	favoritesManager, err := favorites.NewManager(configDir)
	if err != nil {
		log.Printf("Warning: Could not initialize favorites: %v", err)
	}

	// Initialize connection history manager
	connectionHistory, err := connection_history.NewManager(configDir)
	if err != nil {
		log.Printf("Warning: Could not initialize connection history: %v", err)
	}

	// Initialize filter builder
	filterBuilder := components.NewFilterBuilder(th)

	// Initialize JSONB viewer
	jsonbViewer := components.NewJSONBViewer(th)

	// Initialize table view (needed by structure view)
	tableView := components.NewTableView(th)

	// Initialize structure view with shared table view
	structureView := components.NewStructureView(th, tableView)

	// Initialize favorites dialog
	favoritesDialog := components.NewFavoritesDialog(th)

	// Initialize search input
	searchInput := components.NewSearchInput(th)

	app := &App{
		state:             state,
		config:            cfg,
		theme:             th,
		connectionManager: connection.NewManager(),
		discoverer:        discovery.NewDiscoverer(),
		connectionDialog:  components.NewConnectionDialog(th),
		errorOverlay:      components.NewErrorOverlay(th),
		treeView:          components.NewTreeView(emptyRoot, th),
		commandRegistry:   registry,
		commandPalette:    components.NewCommandPalette(th),
		quickQuery:        components.NewQuickQuery(th),
		historyStore:      historyStore,
		tableView:         tableView,
		showFilterBuilder: false,
		filterBuilder:     filterBuilder,
		activeFilter:      nil,
		showJSONBViewer:   false,
		jsonbViewer:       jsonbViewer,
		showStructureView: false,
		structureView:     structureView,
		currentTab:        0,
		showFavorites:     false,
		favoritesManager:  favoritesManager,
		favoritesDialog:   favoritesDialog,
		connectionHistory: connectionHistory,
		showSearch:        false,
		searchInput:       searchInput,
		leftPanel: components.Panel{
			Title:   "Navigation",
			Content: "Databases\n└─ (empty)",
			Style:   lipgloss.NewStyle().BorderForeground(th.BorderFocused),
		},
		rightPanel: components.Panel{
			Title:   "", // No title for right panel
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
	// Load connection history if available
	if a.connectionHistory != nil {
		history := a.connectionHistory.GetRecent(10) // Show up to 10 recent connections
		a.connectionDialog.SetHistoryEntries(history)
	}

	// If no active connection, automatically show connection dialog on startup
	if a.state.ActiveConnection == nil {
		a.showConnectionDialog = true
		return tea.Batch(
			a.triggerDiscovery(),
			a.connectionDialog.Init(), // Start cursor blinking
		)
	}
	return a.connectionDialog.Init() // Always init textinput cursors
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

	case commands.FavoritesCommandMsg:
		// Open favorites dialog
		if a.favoritesManager != nil {
			a.favoritesDialog.SetFavorites(a.favoritesManager.GetAll())
		}
		a.showFavorites = true
		return a, nil

	case commands.ExportFavoritesCSVMsg:
		// Export favorites to CSV
		if a.favoritesManager == nil {
			a.ShowError("Export Not Available", "Favorites manager is not initialized.\n\nPlease restart the application.")
			return a, nil
		}

		path, err := a.favoritesManager.ExportToCSV()
		if err != nil {
			a.ShowError("Export Failed", fmt.Sprintf("Failed to export favorites to CSV:\n\n%v\n\nPlease check that you have write permissions and try again.", err))
			return a, nil
		}

		// Show success notification
		a.ShowError("Export Complete", fmt.Sprintf("Successfully exported favorites to:\n\n%s\n\nYou can now import this file or share it with others.", path))
		return a, nil

	case commands.ExportFavoritesJSONMsg:
		// Export favorites to JSON
		if a.favoritesManager == nil {
			a.ShowError("Export Not Available", "Favorites manager is not initialized.\n\nPlease restart the application.")
			return a, nil
		}

		path, err := a.favoritesManager.ExportToJSON()
		if err != nil {
			a.ShowError("Export Failed", fmt.Sprintf("Failed to export favorites to JSON:\n\n%v\n\nPlease check that you have write permissions and try again.", err))
			return a, nil
		}

		// Show success notification
		a.ShowError("Export Complete", fmt.Sprintf("Successfully exported favorites to:\n\n%s\n\nYou can now import this file or share it with others.", path))
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

	case components.CloseJSONBViewerMsg:
		a.showJSONBViewer = false
		return a, nil

	case components.ExecuteFavoriteMsg:
		// Execute favorite query
		if a.state.ActiveConnection == nil {
			a.ShowError("No Database Connection", "Please connect to a database before executing queries.\n\nPress 'c' to open the connection dialog.")
			return a, nil
		}

		// Record usage
		if a.favoritesManager != nil {
			if err := a.favoritesManager.RecordUsage(msg.Favorite.ID); err != nil {
				// Log error but don't block execution
				log.Printf("Warning: Failed to record favorite usage: %v", err)
			}
		}

		// Execute query asynchronously
		a.showFavorites = false
		return a, func() tea.Msg {
			conn, err := a.connectionManager.GetActive()
			if err != nil {
				return QueryResultMsg{
					SQL: msg.Favorite.Query,
					Result: models.QueryResult{
						Error: fmt.Errorf("connection error: %w", err),
					},
				}
			}

			result := query.Execute(context.Background(), conn.Pool.GetPool(), msg.Favorite.Query)
			return QueryResultMsg{
				SQL:    msg.Favorite.Query,
				Result: result,
			}
		}

	case components.CloseFavoritesDialogMsg:
		a.showFavorites = false
		return a, nil

	case components.SearchInputMsg:
		// Handle search request from search input
		a.showSearch = false
		if msg.Query == "" {
			return a, nil
		}

		if msg.Mode == "local" {
			// Local search - search only loaded data
			a.tableView.SearchLocal(msg.Query)
		} else {
			// Table search - query the database
			if a.state.ActiveConnection == nil {
				a.ShowError("No Connection", "Please connect to a database first")
				return a, nil
			}

			if a.currentTable == "" {
				a.ShowError("No Table", "Please select a table first")
				return a, nil
			}

			// Execute table search
			return a, a.searchTable(msg.Query)
		}
		return a, nil

	case components.CloseSearchMsg:
		a.showSearch = false
		a.searchInput.Reset()
		return a, nil

	case SearchTableResultMsg:
		if msg.Err != nil {
			a.ShowError("Search Error", msg.Err.Error())
			return a, nil
		}

		if msg.Data == nil || len(msg.Data.Rows) == 0 {
			a.ShowError("No Results", fmt.Sprintf("No matches found for '%s'", msg.Query))
			return a, nil
		}

		// Replace table data with search results
		a.tableView.SetData(msg.Data.Columns, msg.Data.Rows, int(msg.Data.TotalRows))

		// Build matches from all cells that contain the query
		queryLower := strings.ToLower(msg.Query)
		var matches []components.MatchPos
		for rowIdx, row := range msg.Data.Rows {
			for colIdx, cell := range row {
				if strings.Contains(strings.ToLower(cell), queryLower) {
					matches = append(matches, components.MatchPos{Row: rowIdx, Col: colIdx})
				}
			}
		}

		a.tableView.SetSearchResults(msg.Query, matches)
		return a, nil

	case components.AddFavoriteMsg:
		if a.favoritesManager != nil {
			conn := ""
			if a.state.ActiveConnection != nil {
				conn = a.state.ActiveConnection.Config.Name
			}
			db := a.state.CurrentDatabase
			_, err := a.favoritesManager.Add(msg.Name, msg.Description, msg.Query, conn, db, msg.Tags)
			if err != nil {
				a.ShowError("Cannot Add Favorite", fmt.Sprintf("Failed to add favorite:\n\n%v\n\nPlease check your input and try again.", err))
			} else {
				// Refresh the dialog
				a.favoritesDialog.SetFavorites(a.favoritesManager.GetAll())
			}
		} else {
			a.ShowError("Favorites Not Available", "Favorites manager is not initialized.\n\nPlease restart the application.")
		}
		return a, nil

	case components.EditFavoriteMsg:
		if a.favoritesManager != nil {
			err := a.favoritesManager.Update(msg.FavoriteID, msg.Name, msg.Description, msg.Query, msg.Tags)
			if err != nil {
				a.ShowError("Cannot Update Favorite", fmt.Sprintf("Failed to update favorite:\n\n%v\n\nPlease check your input and try again.", err))
			} else {
				// Refresh the dialog
				a.favoritesDialog.SetFavorites(a.favoritesManager.GetAll())
			}
		} else {
			a.ShowError("Favorites Not Available", "Favorites manager is not initialized.\n\nPlease restart the application.")
		}
		return a, nil

	case components.DeleteFavoriteMsg:
		if a.favoritesManager != nil {
			err := a.favoritesManager.Delete(msg.FavoriteID)
			if err != nil {
				a.ShowError("Cannot Delete Favorite", fmt.Sprintf("Failed to delete favorite:\n\n%v\n\nThe favorite may have already been deleted.", err))
			} else {
				// Refresh the dialog
				a.favoritesDialog.SetFavorites(a.favoritesManager.GetAll())
				// Adjust selection if needed
				if a.favoritesDialog != nil {
					favorites := a.favoritesManager.GetAll()
					if len(favorites) > 0 {
						// Keep selection valid
						a.favoritesDialog.SetFavorites(favorites)
					}
				}
			}
		} else {
			a.ShowError("Favorites Not Available", "Favorites manager is not initialized.\n\nPlease restart the application.")
		}
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

		// Handle JSONB viewer input
		if a.showJSONBViewer {
			return a.handleJSONBViewer(msg)
		}

		// Handle favorites dialog if visible
		if a.showFavorites {
			return a.handleFavoritesDialog(msg)
		}

		// Handle search input if visible
		if a.showSearch {
			return a.handleSearchInput(msg)
		}

		switch msg.String() {
		// Tab switching (1/2/3/4 when right panel focused, or Ctrl+1/2/3/4 globally)
		case "1", "ctrl+1":
			if a.state.FocusedPanel == models.RightPanel || msg.String() == "ctrl+1" {
				a.currentTab = 0
				a.structureView.SwitchTab(0)
				return a, nil
			}

		case "2", "ctrl+2":
			if a.state.FocusedPanel == models.RightPanel || msg.String() == "ctrl+2" {
				a.currentTab = 1
				a.structureView.SwitchTab(1)
				return a, nil
			}

		case "3", "ctrl+3":
			if a.state.FocusedPanel == models.RightPanel || msg.String() == "ctrl+3" {
				a.currentTab = 2
				a.structureView.SwitchTab(2)
				return a, nil
			}

		case "4", "ctrl+4":
			if a.state.FocusedPanel == models.RightPanel || msg.String() == "ctrl+4" {
				a.currentTab = 3
				a.structureView.SwitchTab(3)
				return a, nil
			}

		case "ctrl+p":
			// Open quick query
			a.showQuickQuery = true
			return a, nil
		case "ctrl+k":
			// Open command palette and populate with commands including history
			a.commandPalette.SetCommands(a.getCommandsWithHistory())
			a.showCommandPalette = true
			return a, nil
		case "ctrl+b":
			// Open favorites dialog
			if a.favoritesManager != nil {
				a.favoritesDialog.SetFavorites(a.favoritesManager.GetAll())
			}
			a.showFavorites = true
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
		case "ctrl+f":
			// Quick filter from current cell
			if a.state.FocusedPanel == models.RightPanel && a.tableView != nil {
				selectedRow, selectedCol := a.tableView.GetSelectedCell()
				if selectedRow >= 0 && selectedCol >= 0 && selectedCol < len(a.tableView.Columns) {
					// Get column name and value
					columnName := a.tableView.Columns[selectedCol]
					cellValue := a.tableView.Rows[selectedRow][selectedCol]

					// Create quick filter
					columns := a.getTableColumns()
					var columnInfo models.ColumnInfo
					for _, col := range columns {
						if col.Name == columnName {
							columnInfo = col
							break
						}
					}

					// Create filter with single condition
					quickFilter := models.Filter{
						Schema:    a.state.TreeSelected.Parent.Label,
						TableName: a.state.TreeSelected.Label,
						RootGroup: models.FilterGroup{
							Conditions: []models.FilterCondition{
								{
									Column:   columnName,
									Operator: models.OpEqual,
									Value:    cellValue,
									Type:     columnInfo.DataType,
								},
							},
							Logic: "AND",
						},
					}

					a.activeFilter = &quickFilter
					return a, a.loadTableDataWithFilter(quickFilter)
				}
			}
			return a, nil
		case "ctrl+r":
			// Clear filter and reload
			if a.activeFilter != nil && a.state.TreeSelected != nil {
				a.activeFilter = nil
				schemaNode := a.state.TreeSelected.Parent
				if schemaNode != nil {
					return a, a.loadTableData(LoadTableDataMsg{
						Schema: schemaNode.Label,
						Table:  a.state.TreeSelected.Label,
						Limit:  100,
						Offset: 0,
					})
				}
			}
			return a, nil
		case "y":
			// Copy functionality in structure view (copy name)
			if a.currentTab > 0 {
				statusMsg := a.structureView.CopyCurrentName()
				if statusMsg != "" {
					// Show status message (log.Println is acceptable per plan)
					log.Println(statusMsg)
				}
				return a, nil
			}
		case "Y":
			// Copy functionality in structure view (copy definition)
			if a.currentTab > 0 {
				statusMsg := a.structureView.CopyCurrentDefinition()
				if statusMsg != "" {
					log.Println(statusMsg)
				}
				return a, nil
			}
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
				// If structure view is active and not on Data tab, route to structure view
				if a.currentTab > 0 {
					a.structureView.Update(msg)
					return a, nil
				}

				// Handle preview pane scrolling (when visible)
				if a.tableView.PreviewPane != nil && a.tableView.PreviewPane.Visible {
					switch msg.String() {
					case "ctrl+up":
						a.tableView.PreviewPane.ScrollUp()
						return a, nil
					case "ctrl+down":
						a.tableView.PreviewPane.ScrollDown()
						return a, nil
					}
				}

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
				case "left", "h":
					a.tableView.MoveSelectionHorizontal(-1)
					return a, nil
				case "right", "l":
					a.tableView.MoveSelectionHorizontal(1)
					return a, nil
				case "H":
					// Jump scroll left (half screen)
					a.tableView.JumpScrollHorizontal(-1)
					return a, nil
				case "L":
					// Jump scroll right (half screen)
					a.tableView.JumpScrollHorizontal(1)
					return a, nil
				case "0":
					// Jump to first column
					a.tableView.JumpToFirstColumn()
					return a, nil
				case "$":
					// Jump to last column
					a.tableView.JumpToLastColumn()
					return a, nil
				case "ctrl+u":
					a.tableView.PageUp()
					return a, nil
				case "ctrl+d":
					a.tableView.PageDown()
					return a, nil
				case "s":
					// Sort by current column
					a.tableView.ToggleSort()
					// Reload data with new sort
					if a.currentTable != "" {
						parts := strings.Split(a.currentTable, ".")
						if len(parts) == 2 {
							return a, func() tea.Msg {
								return LoadTableDataMsg{
									Schema:     parts[0],
									Table:      parts[1],
									Offset:     0,
									Limit:      100,
									SortColumn: a.tableView.GetSortColumn(),
									SortDir:    a.tableView.GetSortDirection(),
									NullsFirst: a.tableView.GetNullsFirst(),
								}
							}
						}
					}
					return a, nil
				case "S":
					// Toggle NULLS FIRST/LAST
					if a.tableView.SortColumn >= 0 {
						a.tableView.ToggleNullsFirst()
						// Reload data
						if a.currentTable != "" {
							parts := strings.Split(a.currentTable, ".")
							if len(parts) == 2 {
								return a, func() tea.Msg {
									return LoadTableDataMsg{
										Schema:     parts[0],
										Table:      parts[1],
										Offset:     0,
										Limit:      100,
										SortColumn: a.tableView.GetSortColumn(),
										SortDir:    a.tableView.GetSortDirection(),
										NullsFirst: a.tableView.GetNullsFirst(),
									}
								}
							}
						}
					}
					return a, nil
				case "J":
					// Open JSONB viewer if cell contains JSONB (uppercase J to avoid conflict with vim down)
					selectedRow, selectedCol := a.tableView.GetSelectedCell()
					if selectedRow >= 0 && selectedCol >= 0 && selectedRow < len(a.tableView.Rows) && selectedCol < len(a.tableView.Columns) {
						cellValue := a.tableView.Rows[selectedRow][selectedCol]
						if jsonb.IsJSONB(cellValue) {
							if err := a.jsonbViewer.SetValue(cellValue); err == nil {
								a.showJSONBViewer = true
							}
						}
					}
					return a, nil
				case "/":
					// Open search input
					a.searchInput.Reset()
					a.searchInput.Width = a.rightPanel.Width - 4
					a.showSearch = true
					return a, nil
				case "p":
					// Toggle preview pane
					a.tableView.TogglePreviewPane()
					return a, nil
				case "y":
					// Copy preview pane content (yank)
					if a.tableView.PreviewPane != nil && a.tableView.PreviewPane.Visible {
						if err := a.tableView.PreviewPane.CopyContent(); err == nil {
							log.Println("Copied preview content to clipboard")
						}
					}
					return a, nil
				case "n":
					// Next search match
					if a.tableView.SearchActive {
						a.tableView.NextMatch()
					}
					return a, nil
				case "N":
					// Previous search match
					if a.tableView.SearchActive {
						a.tableView.PrevMatch()
					}
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
		a.connectionDialog.SetDiscoveredInstances(msg.Instances)
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

	// Top bar with modern Catppuccin styling
	appNameStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#cba6f7")). // Mauve
		Background(lipgloss.Color("#313244"))   // Surface0

	connStatus := ""
	if a.state.ActiveConnection != nil {
		// Build connection string with elegant formatting
		conn := a.state.ActiveConnection
		connStr := fmt.Sprintf("%s@%s:%d/%s",
			conn.Config.User,
			conn.Config.Host,
			conn.Config.Port,
			conn.Config.Database)

		connStatus = "  " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6e3a1")). // Green
			Render("") + " " +
			lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cdd6f4")). // Text
			Render(connStr)
	} else {
		connStatus = "  " + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")). // Overlay0
			Render("") + " " +
			lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Render("Not connected")
	}

	topBarLeft := appNameStyle.Render("  LazyPG ") + connStatus
	topBarRight := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89b4fa")). // Blue
		Render("? ") +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Render("help")
	topBarContent := a.formatStatusBar(topBarLeft, topBarRight)

	// Create modern top bar with subtle border
	topBarStyle := lipgloss.NewStyle().
		Width(a.state.Width).
		Background(lipgloss.Color("#313244")). // Surface0
		Foreground(lipgloss.Color("#cdd6f4")). // Text
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475a")). // Surface1
		Padding(0, 1)

	topBar := topBarStyle.Render(topBarContent)

	// Context-sensitive bottom bar with modern styling
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89b4fa")). // Blue for keys
		Bold(true)
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")) // Overlay0 for descriptions
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#45475a")) // Surface1 for separators

	var bottomBarLeft string
	if a.state.FocusedPanel == models.LeftPanel {
		// Tree navigation keys with icons
		bottomBarLeft = keyStyle.Render("↑↓") + dimStyle.Render(" navigate") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("→←") + dimStyle.Render(" expand") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("Enter") + dimStyle.Render(" select")
	} else {
		// Table navigation keys
		bottomBarLeft = keyStyle.Render("↑↓") + dimStyle.Render(" navigate") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("Ctrl+D/U") + dimStyle.Render(" page") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("p") + dimStyle.Render(" preview") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("j") + dimStyle.Render(" jsonb")
	}

	// Add filter indicator if active
	if a.activeFilter != nil && len(a.activeFilter.RootGroup.Conditions) > 0 {
		filterCount := len(a.activeFilter.RootGroup.Conditions)
		filterSuffix := ""
		if filterCount > 1 {
			filterSuffix = "s"
		}
		filterStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f9e2af")) // Yellow for filter
		filterIndicator := separatorStyle.Render(" │ ") +
			filterStyle.Render("") + dimStyle.Render(fmt.Sprintf(" %d filter%s", filterCount, filterSuffix))
		bottomBarLeft = bottomBarLeft + filterIndicator
	}

	// Common keys on the right with icons
	bottomBarRight := keyStyle.Render("Tab") + dimStyle.Render(" switch") +
		separatorStyle.Render(" │ ") +
		keyStyle.Render("c") + dimStyle.Render(" connect") +
		separatorStyle.Render(" │ ") +
		keyStyle.Render("q") + dimStyle.Render(" quit")

	bottomBarContent := a.formatStatusBar(bottomBarLeft, bottomBarRight)

	// Create modern bottom bar
	bottomBarStyle := lipgloss.NewStyle().
		Width(a.state.Width).
		Background(lipgloss.Color("#313244")). // Surface0
		Foreground(lipgloss.Color("#cdd6f4")). // Text
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475a")). // Surface1
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

	// Update right panel content
	// Calculate available content height: panel height - borders (2) - padding (0)
	rightContentHeight := a.rightPanel.Height - 2 // -2 for top/bottom borders (no title)
	if rightContentHeight < 1 {
		rightContentHeight = 1
	}
	rightContentWidth := a.rightPanel.Width - 2 // -2 for horizontal padding inside panel

	a.rightPanel.Content = a.renderRightPanel(rightContentWidth, rightContentHeight)

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

	// Render JSONB viewer if visible
	if a.showJSONBViewer {
		mainView = lipgloss.Place(
			a.state.Width,
			a.state.Height,
			lipgloss.Center,
			lipgloss.Center,
			a.jsonbViewer.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#555555")),
		)
	}

	// Render favorites dialog if visible
	if a.showFavorites {
		mainView = lipgloss.Place(
			a.state.Width,
			a.state.Height,
			lipgloss.Center,
			lipgloss.Center,
			a.favoritesDialog.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("#555555")),
		)
	}

	// Render search input if visible (as overlay on top of mainView)
	if a.showSearch {
		a.searchInput.Width = 60
		if a.searchInput.Width > a.state.Width-4 {
			a.searchInput.Width = a.state.Width - 4
		}
		mainView = a.overlaySearchInput(mainView)
	}

	return mainView
}

// renderRightPanel renders the right panel content based on current state
func (a *App) renderRightPanel(width, height int) string {
	// If table is selected, show structure view with tabs
	if a.currentTable != "" {
		// Calculate preview pane height
		previewHeight := 0
		if a.currentTab == 0 && a.tableView.PreviewPane != nil {
			// Set preview pane dimensions (max 1/3 of available height)
			maxPreviewHeight := height / 3
			if maxPreviewHeight < 5 {
				maxPreviewHeight = 5
			}
			a.tableView.SetPreviewPaneDimensions(width, maxPreviewHeight)
			a.tableView.UpdatePreviewPane()
			previewHeight = a.tableView.GetPreviewPaneHeight()
		}

		// Calculate main content height (subtract preview pane height)
		mainHeight := height - previewHeight
		if mainHeight < 5 {
			mainHeight = 5
		}

		// Update structure view dimensions
		a.structureView.Width = width
		a.structureView.Height = mainHeight

		// Load table structure if needed (when table changes)
		conn, err := a.connectionManager.GetActive()
		if err == nil && conn != nil && conn.Pool != nil {
			parts := strings.Split(a.currentTable, ".")
			if len(parts) == 2 {
				// Only load if we haven't loaded this table yet
				if !a.structureView.HasTableLoaded(parts[0], parts[1]) {
					ctx := context.Background()
					err := a.structureView.SetTable(ctx, conn.Pool, parts[0], parts[1])
					if err != nil {
						log.Printf("Failed to load structure: %v", err)
					}
				}
			}
		}

		// Render main content
		mainContent := a.structureView.View()

		// If on Data tab and preview pane is visible, append it
		if a.currentTab == 0 && previewHeight > 0 {
			previewContent := a.tableView.PreviewPane.View()
			return lipgloss.JoinVertical(lipgloss.Left, mainContent, previewContent)
		}

		return mainContent
	}

	// No table selected - show table view (will display empty state)
	a.tableView.Width = width
	a.tableView.Height = height
	return a.tableView.View()
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

// updatePanelStyles updates panel styling based on focus with Catppuccin colors
func (a *App) updatePanelStyles() {
	if a.state.FocusedPanel == models.LeftPanel {
		// Focused left panel - Blue border, transparent background like connection dialog
		a.leftPanel.Style = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#89b4fa")). // Blue
			Foreground(lipgloss.Color("#cdd6f4"))         // Text
		// Unfocused right panel - Surface border
		a.rightPanel.Style = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#45475a")). // Surface1
			Foreground(lipgloss.Color("#cdd6f4"))        // Text
	} else {
		// Unfocused left panel
		a.leftPanel.Style = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#45475a")). // Surface1
			Foreground(lipgloss.Color("#cdd6f4"))        // Text
		// Focused right panel - Blue border
		a.rightPanel.Style = lipgloss.NewStyle().
			BorderForeground(lipgloss.Color("#89b4fa")). // Blue
			Foreground(lipgloss.Color("#cdd6f4"))        // Text
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
	// Handle search mode
	if a.connectionDialog.SearchMode {
		switch msg.String() {
		case "esc":
			// Exit search mode and clear search
			a.connectionDialog.ExitSearchMode(true)
			return a, nil
		case "enter":
			// Exit search mode but keep search results
			a.connectionDialog.ExitSearchMode(false)
			return a, nil
		default:
			// Pass keys to search input
			var cmd tea.Cmd
			a.connectionDialog, cmd = a.connectionDialog.Update(msg)
			return a, cmd
		}
	}

	switch msg.String() {
	case "esc":
		a.showConnectionDialog = false
		return a, nil

	case "/", "ctrl+f":
		// Enter search mode (only in discovery mode)
		if !a.connectionDialog.ManualMode {
			a.connectionDialog.EnterSearchMode()
		}
		return a, nil

	case "up", "k":
		if !a.connectionDialog.ManualMode {
			a.connectionDialog.MoveSelection(-1)
		}
		return a, nil

	case "down", "j":
		if !a.connectionDialog.ManualMode {
			a.connectionDialog.MoveSelection(1)
		}
		return a, nil

	case "tab":
		if a.connectionDialog.ManualMode {
			a.connectionDialog.NextInput()
		} else {
			// In discovery mode, switch between history and discovered sections
			a.connectionDialog.SwitchSection()
		}
		return a, nil

	case "shift+tab":
		if a.connectionDialog.ManualMode {
			a.connectionDialog.PrevInput()
		}
		return a, nil

	case "m":
		// Only handle 'm' key in discovery mode, not in manual mode (to allow typing 'm')
		if !a.connectionDialog.ManualMode {
			a.connectionDialog.ToggleMode()
			return a, nil
		}
		// In manual mode, pass 'm' to textinput
		var cmd tea.Cmd
		a.connectionDialog, cmd = a.connectionDialog.Update(msg)
		return a, cmd

	case "ctrl+d":
		// Use Ctrl+D to switch back to discovery mode to avoid conflict with typing 'd'
		if a.connectionDialog.ManualMode {
			a.connectionDialog.ToggleMode()
		}
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

			// Save to connection history (ignore errors)
			if a.connectionHistory != nil {
				if err := a.connectionHistory.Add(config); err != nil {
					log.Printf("Warning: Failed to save connection to history: %v", err)
				} else {
					// Reload history in dialog
					history := a.connectionHistory.GetRecent(10)
					a.connectionDialog.SetHistoryEntries(history)
				}
			}

			// Trigger tree loading
			a.showConnectionDialog = false
			return a, func() tea.Msg {
				return LoadTreeMsg{}
			}
		} else {
			var config models.ConnectionConfig

			// Check if browsing history or discovered instances
			if a.connectionDialog.InHistorySection {
				// Get selected history entry
				historyEntry := a.connectionDialog.GetSelectedHistory()
				if historyEntry == nil {
					// No history entry selected
					return a, nil
				}

				// Convert history entry to connection config WITH password from keyring
				if a.connectionHistory != nil {
					config = a.connectionHistory.GetConnectionConfigWithPassword(historyEntry)
				} else {
					config = historyEntry.ToConnectionConfig()
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
				config = models.ConnectionConfig{
					Host:     instance.Host,
					Port:     instance.Port,
					Database: "postgres",          // Default database
					User:     os.Getenv("USER"),   // Current user
					Password: "",                  // No password for now
					SSLMode:  "prefer",
				}
			}

			// Connect using the configuration
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

			// Save to connection history (ignore errors)
			if a.connectionHistory != nil {
				if err := a.connectionHistory.Add(config); err != nil {
					log.Printf("Warning: Failed to save connection to history: %v", err)
				} else {
					// Reload history in dialog
					history := a.connectionHistory.GetRecent(10)
					a.connectionDialog.SetHistoryEntries(history)
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

	default:
		// In manual mode, delegate to textinput for cursor and text handling
		if a.connectionDialog.ManualMode {
			var cmd tea.Cmd
			a.connectionDialog, cmd = a.connectionDialog.Update(msg)
			return a, cmd
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

// handleJSONBViewer handles key events when JSONB viewer is visible
func (a *App) handleJSONBViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.jsonbViewer, cmd = a.jsonbViewer.Update(msg)
	return a, cmd
}

// handleFavoritesDialog handles key events when favorites dialog is visible
func (a *App) handleFavoritesDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.favoritesDialog, cmd = a.favoritesDialog.Update(msg)
	return a, cmd
}

// handleSearchInput handles key events when search input is visible
func (a *App) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.searchInput, cmd = a.searchInput.Update(msg)
	return a, cmd
}

// getTableColumns returns column info for the current table
func (a *App) getTableColumns() []models.ColumnInfo {
	if a.state.TreeSelected == nil || a.state.TreeSelected.Type != models.TreeNodeTypeTable {
		return nil
	}

	conn, err := a.connectionManager.GetActive()
	if err != nil {
		return nil
	}

	// Get schema from parent node
	schemaNode := a.state.TreeSelected.Parent
	if schemaNode == nil {
		return nil
	}

	columns, err := metadata.GetTableColumns(
		context.Background(),
		conn.Pool,
		schemaNode.Label,
		a.state.TreeSelected.Label,
	)
	if err != nil {
		return nil
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

		var sort *metadata.SortOptions
		if msg.SortColumn != "" {
			sort = &metadata.SortOptions{
				Column:     msg.SortColumn,
				Direction:  msg.SortDir,
				NullsFirst: msg.NullsFirst,
			}
		}

		data, err := metadata.QueryTableData(ctx, conn.Pool, msg.Schema, msg.Table, msg.Offset, msg.Limit, sort)
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
			`SELECT * FROM "%s"."%s" %s LIMIT 100`,
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

// overlaySearchInput renders the search input as an overlay on top of background
func (a *App) overlaySearchInput(background string) string {
	searchView := a.searchInput.View()
	searchLines := strings.Split(searchView, "\n")
	bgLines := strings.Split(background, "\n")

	// Calculate center position
	searchHeight := len(searchLines)
	searchWidth := lipgloss.Width(searchLines[0]) // Use first line width

	startY := (a.state.Height - searchHeight) / 2
	startX := (a.state.Width - searchWidth) / 2

	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	// Overlay search box on background
	result := make([]string, len(bgLines))
	for i, bgLine := range bgLines {
		if i >= startY && i < startY+searchHeight {
			searchLineIdx := i - startY
			if searchLineIdx < len(searchLines) {
				// Overlay this search line onto background
				result[i] = a.overlayLine(bgLine, searchLines[searchLineIdx], startX)
			} else {
				result[i] = bgLine
			}
		} else {
			result[i] = bgLine
		}
	}

	return strings.Join(result, "\n")
}

// overlayLine overlays foreground onto background at given x position
// Handles ANSI escape sequences correctly
func (a *App) overlayLine(background, foreground string, startX int) string {
	fgWidth := ansi.StringWidth(foreground)

	// Truncate background to startX (visual width)
	leftPart := ansi.Truncate(background, startX, "")

	// Get visible width of left part
	leftWidth := ansi.StringWidth(leftPart)

	// Pad if needed
	if leftWidth < startX {
		leftPart += strings.Repeat(" ", startX-leftWidth)
	}

	// Cut the right part of background after the overlay
	rightStart := startX + fgWidth
	bgWidth := ansi.StringWidth(background)

	var rightPart string
	if rightStart < bgWidth {
		// Cut from background starting at rightStart position
		rightPart = ansi.Cut(background, rightStart, bgWidth)
	}

	return leftPart + foreground + rightPart
}

// SearchTableResultMsg is sent when table search completes
type SearchTableResultMsg struct {
	Query   string
	Data    *metadata.TableData
	Err     error
}

// searchTable executes a table-wide search
func (a *App) searchTable(query string) tea.Cmd {
	return func() tea.Msg {
		conn, err := a.connectionManager.GetActive()
		if err != nil {
			return SearchTableResultMsg{Query: query, Err: fmt.Errorf("no active connection: %w", err)}
		}

		parts := strings.Split(a.currentTable, ".")
		if len(parts) != 2 {
			return SearchTableResultMsg{Query: query, Err: fmt.Errorf("invalid table: %s", a.currentTable)}
		}

		schema, table := parts[0], parts[1]

		data, err := metadata.SearchTableData(
			context.Background(),
			conn.Pool,
			schema,
			table,
			a.tableView.Columns,
			query,
			500, // Max results
		)
		if err != nil {
			return SearchTableResultMsg{Query: query, Err: err}
		}

		return SearchTableResultMsg{Query: query, Data: data}
	}
}
