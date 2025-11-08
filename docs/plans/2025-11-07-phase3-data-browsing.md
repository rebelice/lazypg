# Phase 3: Data Browsing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement navigation tree, table data viewing with virtual scrolling, and structure/indexes/constraints tabs.

**Architecture:** Build navigation tree UI component using tree structure, implement table data viewer with pagination-based virtual scrolling, add tabbed views for table metadata, and ensure smooth user experience with async data loading.

**Tech Stack:** Go, Bubble Tea, Lipgloss, Bubbles (table component), pgx v5 for queries

---

## Prerequisites

Phase 3 requires working database connections. Phase 2 left connection logic incomplete (TODO items). We must complete those first.

---

## Task 1: Complete Connection Logic from Phase 2

**Files:**
- Modify: `internal/app/app.go:111` (trigger discovery)
- Modify: `internal/app/app.go:286-298` (implement connection)

**Step 1: Add discovery triggering when opening connection dialog**

In `internal/app/app.go`, update the 'c' key handler (around line 108-112):

```go
case "c":
	// Open connection dialog
	a.showConnectionDialog = true
	// Trigger discovery in background
	go func() {
		ctx := context.Background()
		instances := a.discoverer.DiscoverAll(ctx)
		// Send discovered instances back via Bubble Tea message
		// TODO: This requires adding a custom message type
	}()
	return a, nil
```

**Step 2: Add custom message type for discovery results**

Add to `internal/models/models.go`:

```go
// DiscoveryCompleteMsg is sent when auto-discovery completes
type DiscoveryCompleteMsg struct {
	Instances []DiscoveredInstance
}
```

**Step 3: Update discovery trigger with proper message sending**

```go
case "c":
	// Open connection dialog
	a.showConnectionDialog = true
	// Trigger discovery in background
	return a, func() tea.Msg {
		ctx := context.Background()
		instances := a.discoverer.DiscoverAll(ctx)
		return models.DiscoveryCompleteMsg{Instances: instances}
	}
```

**Step 4: Handle discovery complete message in Update**

Add case in `Update` method:

```go
case models.DiscoveryCompleteMsg:
	a.connectionDialog.DiscoveredInstances = msg.Instances
	return a, nil
```

**Step 5: Implement actual connection logic**

Update `handleConnectionDialog` method around line 281-300:

```go
case "enter":
	ctx := context.Background()

	if a.connectionDialog.ManualMode {
		config, err := a.connectionDialog.GetManualConfig()
		if err != nil {
			// TODO: Show error message to user (will implement in next step)
			return a, nil
		}

		// Attempt connection
		connID, err := a.connectionManager.Connect(ctx, config)
		if err != nil {
			// TODO: Show error message
			return a, nil
		}

		// Store connection info in app state
		a.state.ActiveConnection = &models.Connection{
			ID:          connID,
			Config:      config,
			Connected:   true,
			ConnectedAt: time.Now(),
		}
	} else {
		instance := a.connectionDialog.GetSelectedInstance()
		if instance == nil {
			return a, nil
		}

		// Build config from discovered instance
		config := models.ConnectionConfig{
			Host:    instance.Host,
			Port:    instance.Port,
			User:    os.Getenv("USER"), // Default to current user
			SSLMode: "prefer",
		}

		// Try to find password from .pgpass
		config.Password = discovery.FindPassword(instance.Host, instance.Port, "", config.User)

		// Need database name - prompt user or use default
		config.Database = "postgres" // Default database

		connID, err := a.connectionManager.Connect(ctx, config)
		if err != nil {
			// TODO: Show error message
			return a, nil
		}

		a.state.ActiveConnection = &models.Connection{
			ID:          connID,
			Config:      config,
			Connected:   true,
			ConnectedAt: time.Now(),
		}
	}

	a.showConnectionDialog = false
	return a, nil
```

**Step 6: Add necessary imports**

Add to imports in `internal/app/app.go`:

```go
import (
	// ... existing imports
	"os"
	"time"
)
```

**Step 7: Commit**

```bash
git add internal/app/app.go internal/models/models.go
git commit -m "feat: complete Phase 2 connection logic with discovery and connect"
```

---

## Task 2: Add Error Message Display

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/models/models.go`

**Step 1: Add error message to app state**

In `internal/models/models.go`:

```go
type AppState struct {
	// ... existing fields

	// Error display
	ErrorMessage string
	ShowError    bool
}
```

**Step 2: Add error message type**

```go
// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Message string
}
```

**Step 3: Update connection error handling to show errors**

In `internal/app/app.go`, update the connection logic:

```go
// When connection fails:
if err != nil {
	a.state.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
	a.state.ShowError = true
	return a, nil
}
```

**Step 4: Add error dismissal**

In Update method:

```go
// If showing error, allow ESC to dismiss
if a.state.ShowError {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			a.state.ShowError = false
			a.state.ErrorMessage = ""
			return a, nil
		}
	}
}
```

**Step 5: Render error overlay in View**

```go
func (a *App) View() string {
	// ... existing code

	if a.state.ShowError {
		return a.renderError()
	}

	// ... rest of view logic
}

func (a *App) renderError() string {
	errorStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Error).
		Padding(1, 2).
		Width(60)

	content := lipgloss.NewStyle().Foreground(a.theme.Error).Render(a.state.ErrorMessage)
	help := lipgloss.NewStyle().Foreground(a.theme.TextDim).Render("\nPress ESC to dismiss")

	return lipgloss.Place(
		a.state.Width,
		a.state.Height,
		lipgloss.Center,
		lipgloss.Center,
		errorStyle.Render(content+help),
	)
}
```

**Step 6: Commit**

```bash
git add internal/app/app.go internal/models/models.go
git commit -m "feat: add error message display system"
```

---

## Task 3: Create Navigation Tree Model

**Files:**
- Create: `internal/models/tree.go`

**Step 1: Create tree node structures**

```go
package models

// TreeNode represents a node in the navigation tree
type TreeNode struct {
	ID       string
	Label    string
	Type     NodeType
	Children []*TreeNode
	Parent   *TreeNode
	Expanded bool
	Level    int
}

// NodeType represents the type of tree node
type NodeType int

const (
	NodeTypeDatabase NodeType = iota
	NodeTypeSchema
	NodeTypeTables
	NodeTypeTable
	NodeTypeViews
	NodeTypeView
	NodeTypeFunctions
	NodeTypeFunction
	NodeTypeSequences
	NodeTypeSequence
)

func (t NodeType) String() string {
	switch t {
	case NodeTypeDatabase:
		return "Database"
	case NodeTypeSchema:
		return "Schema"
	case NodeTypeTables:
		return "Tables"
	case NodeTypeTable:
		return "Table"
	case NodeTypeViews:
		return "Views"
	case NodeTypeView:
		return "View"
	case NodeTypeFunctions:
		return "Functions"
	case NodeTypeFunction:
		return "Function"
	case NodeTypeSequences:
		return "Sequences"
	case NodeTypeSequence:
		return "Sequence"
	default:
		return "Unknown"
	}
}

// TreeState tracks navigation tree state
type TreeState struct {
	Root         *TreeNode
	Selected     *TreeNode
	VisibleNodes []*TreeNode
}

// BuildDatabaseTree builds a tree from database metadata
func BuildDatabaseTree(databases []string, schemasMap map[string][]string, tablesMap map[string][]string) *TreeNode {
	root := &TreeNode{
		ID:       "root",
		Label:    "Databases",
		Type:     NodeTypeDatabase,
		Expanded: true,
		Level:    0,
	}

	for _, dbName := range databases {
		dbNode := &TreeNode{
			ID:       "db:" + dbName,
			Label:    dbName,
			Type:     NodeTypeDatabase,
			Parent:   root,
			Expanded: false,
			Level:    1,
		}

		// Add schemas
		if schemas, ok := schemasMap[dbName]; ok {
			for _, schemaName := range schemas {
				schemaNode := &TreeNode{
					ID:       "schema:" + dbName + ":" + schemaName,
					Label:    schemaName,
					Type:     NodeTypeSchema,
					Parent:   dbNode,
					Expanded: false,
					Level:    2,
				}

				// Add tables folder
				tablesFolder := &TreeNode{
					ID:       "tables:" + dbName + ":" + schemaName,
					Label:    "Tables",
					Type:     NodeTypeTables,
					Parent:   schemaNode,
					Expanded: false,
					Level:    3,
				}

				// Add individual tables
				tableKey := dbName + ":" + schemaName
				if tables, ok := tablesMap[tableKey]; ok {
					for _, tableName := range tables {
						tableNode := &TreeNode{
							ID:     "table:" + dbName + ":" + schemaName + ":" + tableName,
							Label:  tableName,
							Type:   NodeTypeTable,
							Parent: tablesFolder,
							Level:  4,
						}
						tablesFolder.Children = append(tablesFolder.Children, tableNode)
					}
				}

				schemaNode.Children = append(schemaNode.Children, tablesFolder)
				dbNode.Children = append(dbNode.Children, schemaNode)
			}
		}

		root.Children = append(root.Children, dbNode)
	}

	return root
}

// GetVisibleNodes returns nodes that should be displayed (respecting expand/collapse)
func GetVisibleNodes(root *TreeNode) []*TreeNode {
	var visible []*TreeNode
	var traverse func(*TreeNode)

	traverse = func(node *TreeNode) {
		visible = append(visible, node)
		if node.Expanded {
			for _, child := range node.Children {
				traverse(child)
			}
		}
	}

	if root != nil {
		traverse(root)
	}

	return visible
}
```

**Step 2: Commit**

```bash
git add internal/models/tree.go
git commit -m "feat: add navigation tree data models"
```

---

## Task 4: Create Navigation Tree UI Component

**Files:**
- Create: `internal/ui/components/tree.go`

**Step 1: Create tree UI component**

```go
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/models"
)

// Tree represents a navigation tree UI component
type Tree struct {
	State         *models.TreeState
	Width         int
	Height        int
	Style         lipgloss.Style
	SelectedIndex int
}

// NewTree creates a new tree component
func NewTree() *Tree {
	return &Tree{
		State: &models.TreeState{},
	}
}

// View renders the tree
func (t *Tree) View() string {
	if t.State.Root == nil {
		return t.Style.Render("No data")
	}

	t.State.VisibleNodes = models.GetVisibleNodes(t.State.Root)

	var b strings.Builder

	for i, node := range t.State.VisibleNodes {
		if i >= t.Height {
			break
		}

		line := t.renderNode(node, i == t.SelectedIndex)
		b.WriteString(line)
		if i < len(t.State.VisibleNodes)-1 {
			b.WriteString("\n")
		}
	}

	return t.Style.Width(t.Width).Height(t.Height).Render(b.String())
}

func (t *Tree) renderNode(node *models.TreeNode, selected bool) string {
	indent := strings.Repeat("  ", node.Level)

	icon := t.getIcon(node)

	label := node.Label
	if selected {
		label = lipgloss.NewStyle().Bold(true).Render(label)
	}

	prefix := "  "
	if selected {
		prefix = "> "
	}

	return prefix + indent + icon + " " + label
}

func (t *Tree) getIcon(node *models.TreeNode) string {
	if len(node.Children) > 0 {
		if node.Expanded {
			return "▼"
		}
		return "▶"
	}

	switch node.Type {
	case models.NodeTypeDatabase:
		return "🗄"
	case models.NodeTypeSchema:
		return "📁"
	case models.NodeTypeTables, models.NodeTypeViews, models.NodeTypeFunctions, models.NodeTypeSequences:
		return "📂"
	case models.NodeTypeTable:
		return "📊"
	case models.NodeTypeView:
		return "👁"
	case models.NodeTypeFunction:
		return "⚙"
	case models.NodeTypeSequence:
		return "🔢"
	default:
		return "•"
	}
}

// MoveSelection moves selection up or down
func (t *Tree) MoveSelection(delta int) {
	t.SelectedIndex += delta
	if t.SelectedIndex < 0 {
		t.SelectedIndex = 0
	}
	if t.SelectedIndex >= len(t.State.VisibleNodes) {
		t.SelectedIndex = len(t.State.VisibleNodes) - 1
	}

	if t.SelectedIndex >= 0 && t.SelectedIndex < len(t.State.VisibleNodes) {
		t.State.Selected = t.State.VisibleNodes[t.SelectedIndex]
	}
}

// ToggleExpand toggles the expansion of the selected node
func (t *Tree) ToggleExpand() {
	if t.State.Selected != nil && len(t.State.Selected.Children) > 0 {
		t.State.Selected.Expanded = !t.State.Selected.Expanded
		t.State.VisibleNodes = models.GetVisibleNodes(t.State.Root)
	}
}

// GetSelected returns the currently selected node
func (t *Tree) GetSelected() *models.TreeNode {
	return t.State.Selected
}
```

**Step 2: Commit**

```bash
git add internal/ui/components/tree.go
git commit -m "feat: implement navigation tree UI component"
```

---

## Task 5: Integrate Navigation Tree into App

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add tree to app struct**

```go
type App struct {
	// ... existing fields

	// Navigation tree
	navTree *components.Tree
}
```

**Step 2: Initialize tree in New()**

```go
func New(cfg *config.Config) *App {
	// ... existing code

	app := &App{
		// ... existing fields
		navTree: components.NewTree(),
	}

	// ... rest of initialization
}
```

**Step 3: Load tree data after successful connection**

After connection succeeds in `handleConnectionDialog`:

```go
// After successful connection:
a.showConnectionDialog = false

// Load database structure
return a, func() tea.Msg {
	// This will be implemented as a command that loads tree data
	return models.LoadTreeMsg{}
}
```

**Step 4: Add LoadTreeMsg and handler**

In `internal/models/models.go`:

```go
// LoadTreeMsg requests loading the navigation tree
type LoadTreeMsg struct{}

// TreeLoadedMsg is sent when tree data is loaded
type TreeLoadedMsg struct {
	Root *TreeNode
	Err  error
}
```

In `internal/app/app.go` Update method:

```go
case models.LoadTreeMsg:
	return a, a.loadTree

case models.TreeLoadedMsg:
	if msg.Err != nil {
		a.state.ErrorMessage = fmt.Sprintf("Failed to load database structure: %v", msg.Err)
		a.state.ShowError = true
		return a, nil
	}
	a.navTree.State.Root = msg.Root
	a.navTree.State.VisibleNodes = models.GetVisibleNodes(msg.Root)
	if len(a.navTree.State.VisibleNodes) > 0 {
		a.navTree.State.Selected = a.navTree.State.VisibleNodes[0]
	}
	return a, nil
```

**Step 5: Implement loadTree command**

```go
func (a *App) loadTree() tea.Msg {
	ctx := context.Background()

	conn, err := a.connectionManager.GetActive()
	if err != nil {
		return models.TreeLoadedMsg{Err: err}
	}

	// Get databases (for now just use current database)
	databases := []string{conn.Config.Database}

	// Get schemas
	schemas, err := metadata.ListSchemas(ctx, conn.Pool)
	if err != nil {
		return models.TreeLoadedMsg{Err: err}
	}

	schemasMap := make(map[string][]string)
	schemaNames := make([]string, len(schemas))
	for i, s := range schemas {
		schemaNames[i] = s.Name
	}
	schemasMap[conn.Config.Database] = schemaNames

	// Get tables for each schema
	tablesMap := make(map[string][]string)
	for _, s := range schemas {
		tables, err := metadata.ListTables(ctx, conn.Pool, s.Name)
		if err != nil {
			continue
		}

		tableNames := make([]string, len(tables))
		for i, t := range tables {
			tableNames[i] = t.Name
		}
		tablesMap[conn.Config.Database+":"+s.Name] = tableNames
	}

	root := models.BuildDatabaseTree(databases, schemasMap, tablesMap)

	return models.TreeLoadedMsg{Root: root}
}
```

**Step 6: Update left panel rendering to show tree**

In View method, update left panel content:

```go
func (a *App) View() string {
	// ... existing code

	// Render tree in left panel
	a.navTree.Width = a.leftPanel.Width - 4 // Account for padding
	a.navTree.Height = a.leftPanel.Height - 4
	a.leftPanel.Content = a.navTree.View()

	// ... rest of view code
}
```

**Step 7: Add tree navigation keys**

In Update method, when left panel is focused:

```go
if a.state.FocusedPanel == models.LeftPanel {
	switch msg.String() {
	case "up", "k":
		a.navTree.MoveSelection(-1)
		return a, nil
	case "down", "j":
		a.navTree.MoveSelection(1)
		return a, nil
	case "enter", "space":
		a.navTree.ToggleExpand()
		return a, nil
	}
}
```

**Step 8: Commit**

```bash
git add internal/app/app.go internal/models/models.go
git commit -m "feat: integrate navigation tree into app"
```

---

## Task 6: Create Table Data View Component

**Files:**
- Create: `internal/ui/components/table_view.go`

**Step 1: Create table view component**

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TableView displays table data with virtual scrolling
type TableView struct {
	Columns      []string
	Rows         [][]string
	Width        int
	Height       int
	Style        lipgloss.Style

	// Virtual scrolling state
	TopRow       int
	VisibleRows  int
	SelectedRow  int
	TotalRows    int

	// Column widths (calculated)
	ColumnWidths []int
}

// NewTableView creates a new table view
func NewTableView() *TableView {
	return &TableView{
		Columns:     []string{},
		Rows:        [][]string{},
		ColumnWidths: []int{},
	}
}

// SetData sets the table data
func (tv *TableView) SetData(columns []string, rows [][]string, totalRows int) {
	tv.Columns = columns
	tv.Rows = rows
	tv.TotalRows = totalRows
	tv.calculateColumnWidths()
}

// calculateColumnWidths calculates optimal column widths
func (tv *TableView) calculateColumnWidths() {
	if len(tv.Columns) == 0 {
		return
	}

	tv.ColumnWidths = make([]int, len(tv.Columns))

	// Start with column header lengths
	for i, col := range tv.Columns {
		tv.ColumnWidths[i] = len(col)
	}

	// Check row data
	for _, row := range tv.Rows {
		for i, cell := range row {
			if i < len(tv.ColumnWidths) {
				cellLen := len(cell)
				if cellLen > tv.ColumnWidths[i] {
					tv.ColumnWidths[i] = cellLen
				}
			}
		}
	}

	// Apply max width constraint
	maxWidth := 50
	for i := range tv.ColumnWidths {
		if tv.ColumnWidths[i] > maxWidth {
			tv.ColumnWidths[i] = maxWidth
		}
		// Min width
		if tv.ColumnWidths[i] < 10 {
			tv.ColumnWidths[i] = 10
		}
	}
}

// View renders the table
func (tv *TableView) View() string {
	if len(tv.Columns) == 0 {
		return tv.Style.Render("No data")
	}

	var b strings.Builder

	// Render header
	b.WriteString(tv.renderHeader())
	b.WriteString("\n")
	b.WriteString(tv.renderSeparator())
	b.WriteString("\n")

	// Calculate how many rows we can show
	tv.VisibleRows = tv.Height - 3 // Header + separator + status

	// Render visible rows
	endRow := tv.TopRow + tv.VisibleRows
	if endRow > len(tv.Rows) {
		endRow = len(tv.Rows)
	}

	for i := tv.TopRow; i < endRow; i++ {
		isSelected := i == tv.SelectedRow
		b.WriteString(tv.renderRow(tv.Rows[i], isSelected))
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}

	// Render status
	b.WriteString("\n")
	b.WriteString(tv.renderStatus())

	return tv.Style.Width(tv.Width).Height(tv.Height).Render(b.String())
}

func (tv *TableView) renderHeader() string {
	var parts []string
	for i, col := range tv.Columns {
		width := tv.ColumnWidths[i]
		parts = append(parts, tv.pad(col, width))
	}
	headerStyle := lipgloss.NewStyle().Bold(true)
	return headerStyle.Render(strings.Join(parts, " │ "))
}

func (tv *TableView) renderSeparator() string {
	var parts []string
	for _, width := range tv.ColumnWidths {
		parts = append(parts, strings.Repeat("─", width))
	}
	return strings.Join(parts, "─┼─")
}

func (tv *TableView) renderRow(row []string, selected bool) string {
	var parts []string
	for i, cell := range row {
		if i >= len(tv.ColumnWidths) {
			break
		}
		width := tv.ColumnWidths[i]
		parts = append(parts, tv.pad(cell, width))
	}

	line := strings.Join(parts, " │ ")

	if selected {
		return lipgloss.NewStyle().Background(lipgloss.Color("62")).Render(line)
	}
	return line
}

func (tv *TableView) renderStatus() string {
	showing := fmt.Sprintf("Rows %d-%d of %d", tv.TopRow+1, tv.TopRow+len(tv.Rows), tv.TotalRows)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(showing)
}

func (tv *TableView) pad(s string, width int) string {
	if len(s) > width {
		return s[:width-3] + "..."
	}
	return s + strings.Repeat(" ", width-len(s))
}

// MoveSelection moves the selection up or down
func (tv *TableView) MoveSelection(delta int) {
	tv.SelectedRow += delta

	// Bounds checking
	if tv.SelectedRow < 0 {
		tv.SelectedRow = 0
	}
	if tv.SelectedRow >= len(tv.Rows) {
		tv.SelectedRow = len(tv.Rows) - 1
	}

	// Adjust visible window if needed
	if tv.SelectedRow < tv.TopRow {
		tv.TopRow = tv.SelectedRow
	}
	if tv.SelectedRow >= tv.TopRow+tv.VisibleRows {
		tv.TopRow = tv.SelectedRow - tv.VisibleRows + 1
	}
}

// PageUp/PageDown
func (tv *TableView) PageUp() {
	tv.SelectedRow -= tv.VisibleRows
	if tv.SelectedRow < 0 {
		tv.SelectedRow = 0
	}
	tv.TopRow = tv.SelectedRow
}

func (tv *TableView) PageDown() {
	tv.SelectedRow += tv.VisibleRows
	if tv.SelectedRow >= len(tv.Rows) {
		tv.SelectedRow = len(tv.Rows) - 1
	}
	tv.TopRow = tv.SelectedRow
	if tv.TopRow+tv.VisibleRows > len(tv.Rows) {
		tv.TopRow = len(tv.Rows) - tv.VisibleRows
		if tv.TopRow < 0 {
			tv.TopRow = 0
		}
	}
}
```

**Step 2: Commit**

```bash
git add internal/ui/components/table_view.go
git commit -m "feat: implement table data view component with virtual scrolling"
```

---

## Task 7: Add Table Data Query Functions

**Files:**
- Create: `internal/db/metadata/data.go`

**Step 1: Create data query functions**

```go
package metadata

import (
	"context"
	"fmt"

	"github.com/rebeliceyang/lazypg/internal/db/connection"
)

// TableData represents paginated table data
type TableData struct {
	Columns   []string
	Rows      [][]string
	TotalRows int64
}

// QueryTableData fetches paginated table data
func QueryTableData(ctx context.Context, pool *connection.Pool, schema, table string, offset, limit int) (*TableData, error) {
	// First get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) as count FROM %s.%s", schema, table)
	countRow, err := pool.QueryRow(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to count rows: %w", err)
	}

	totalRows := int64(0)
	if count, ok := countRow["count"].(int64); ok {
		totalRows = count
	}

	// Query paginated data
	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d OFFSET %d", schema, table, limit, offset)
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table data: %w", err)
	}

	if len(rows) == 0 {
		return &TableData{
			Columns:   []string{},
			Rows:      [][]string{},
			TotalRows: totalRows,
		}, nil
	}

	// Extract columns from first row
	var columns []string
	firstRow := rows[0]
	for col := range firstRow {
		columns = append(columns, col)
	}

	// Convert rows to string slices
	data := make([][]string, len(rows))
	for i, row := range rows {
		rowData := make([]string, len(columns))
		for j, col := range columns {
			val := row[col]
			if val == nil {
				rowData[j] = "NULL"
			} else {
				rowData[j] = fmt.Sprintf("%v", val)
			}
		}
		data[i] = rowData
	}

	return &TableData{
		Columns:   columns,
		Rows:      data,
		TotalRows: totalRows,
	}, nil
}
```

**Step 2: Commit**

```bash
git add internal/db/metadata/data.go
git commit -m "feat: add table data query functions with pagination"
```

---

## Task 8: Integrate Table View into App

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/models/models.go`

**Step 1: Add table view to app**

```go
type App struct {
	// ... existing fields

	// Table view
	tableView *components.TableView
	currentTable string // "schema.table"
}
```

**Step 2: Initialize table view**

```go
func New(cfg *config.Config) *App {
	// ... existing code

	app := &App{
		// ... existing fields
		tableView: components.NewTableView(),
	}

	// ... rest
}
```

**Step 3: Load table data when table is selected**

Add to Update when Enter is pressed on a table node:

```go
if a.state.FocusedPanel == models.LeftPanel {
	switch msg.String() {
	case "enter", "space":
		// If it's a table node, load data
		selected := a.navTree.GetSelected()
		if selected != nil && selected.Type == models.NodeTypeTable {
			// Extract schema and table name from node ID
			// Format: "table:database:schema:table"
			parts := strings.Split(selected.ID, ":")
			if len(parts) == 4 {
				schema := parts[2]
				table := parts[3]
				a.currentTable = schema + "." + table

				return a, func() tea.Msg {
					return models.LoadTableDataMsg{
						Schema: schema,
						Table:  table,
						Offset: 0,
						Limit:  100,
					}
				}
			}
		} else {
			a.navTree.ToggleExpand()
		}
		return a, nil
	}
}
```

**Step 4: Add message types**

In `internal/models/models.go`:

```go
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
```

**Step 5: Handle messages**

```go
case models.LoadTableDataMsg:
	return a, a.loadTableData(msg)

case models.TableDataLoadedMsg:
	if msg.Err != nil {
		a.state.ErrorMessage = fmt.Sprintf("Failed to load table data: %v", msg.Err)
		a.state.ShowError = true
		return a, nil
	}
	a.tableView.SetData(msg.Columns, msg.Rows, msg.TotalRows)
	a.state.FocusedPanel = models.RightPanel // Switch focus to data
	return a, nil
```

**Step 6: Implement loadTableData**

```go
func (a *App) loadTableData(msg models.LoadTableDataMsg) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		conn, err := a.connectionManager.GetActive()
		if err != nil {
			return models.TableDataLoadedMsg{Err: err}
		}

		data, err := metadata.QueryTableData(ctx, conn.Pool, msg.Schema, msg.Table, msg.Offset, msg.Limit)
		if err != nil {
			return models.TableDataLoadedMsg{Err: err}
		}

		return models.TableDataLoadedMsg{
			Columns:   data.Columns,
			Rows:      data.Rows,
			TotalRows: int(data.TotalRows),
		}
	}
}
```

**Step 7: Update right panel to show table view**

```go
func (a *App) View() string {
	// ... existing code

	// Update right panel
	a.tableView.Width = a.rightPanel.Width - 4
	a.tableView.Height = a.rightPanel.Height - 4
	a.rightPanel.Content = a.tableView.View()

	// ... rest
}
```

**Step 8: Add table navigation keys**

```go
if a.state.FocusedPanel == models.RightPanel {
	switch msg.String() {
	case "up", "k":
		a.tableView.MoveSelection(-1)
		return a, nil
	case "down", "j":
		a.tableView.MoveSelection(1)
		return a, nil
	case "ctrl+u":
		a.tableView.PageUp()
		return a, nil
	case "ctrl+d":
		a.tableView.PageDown()
		return a, nil
	}
}
```

**Step 9: Add necessary import**

```go
import (
	// ... existing
	"strings"
)
```

**Step 10: Commit**

```bash
git add internal/app/app.go internal/models/models.go
git commit -m "feat: integrate table data view into app with navigation"
```

---

## Task 9: Implement Lazy Loading for Virtual Scrolling

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add auto-load logic when scrolling near bottom**

In the right panel navigation handler:

```go
if a.state.FocusedPanel == models.RightPanel {
	switch msg.String() {
	case "down", "j":
		a.tableView.MoveSelection(1)

		// Check if we need to load more data
		if a.tableView.SelectedRow >= len(a.tableView.Rows)-10 &&
		   len(a.tableView.Rows) < a.tableView.TotalRows {
			// Load next page
			parts := strings.Split(a.currentTable, ".")
			if len(parts) == 2 {
				return a, func() tea.Msg {
					return models.LoadTableDataMsg{
						Schema: parts[0],
						Table:  parts[1],
						Offset: len(a.tableView.Rows),
						Limit:  100,
					}
				}
			}
		}
		return a, nil
	// ... rest
	}
}
```

**Step 2: Update TableDataLoadedMsg handler to append data**

```go
case models.TableDataLoadedMsg:
	if msg.Err != nil {
		a.state.ErrorMessage = fmt.Sprintf("Failed to load table data: %v", msg.Err)
		a.state.ShowError = true
		return a, nil
	}

	// Check if this is initial load or pagination
	if len(a.tableView.Rows) == 0 {
		// Initial load
		a.tableView.SetData(msg.Columns, msg.Rows, msg.TotalRows)
		a.state.FocusedPanel = models.RightPanel
	} else {
		// Append paginated data
		a.tableView.Rows = append(a.tableView.Rows, msg.Rows...)
		a.tableView.TotalRows = msg.TotalRows
	}
	return a, nil
```

**Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: implement lazy loading for virtual scrolling"
```

---

## Task 10: Update README and Documentation

**Files:**
- Modify: `README.md`

**Step 1: Update status and features**

Update README.md:

```markdown
## Status

🚧 **In Development** - Phase 3 (Data Browsing) Complete

### Completed Features

- ✅ Multi-panel layout (left navigation, right content)
- ✅ Configuration system (YAML-based)
- ✅ Theme support
- ✅ Help system with keyboard shortcuts
- ✅ Panel focus management
- ✅ Responsive layout
- ✅ PostgreSQL connection management
- ✅ Connection pooling with pgx v5
- ✅ Auto-discovery (port scan, environment, .pgpass)
- ✅ Connection dialog UI
- ✅ Basic metadata queries
- ✅ Navigation tree (databases, schemas, tables)
- ✅ Table data viewing with virtual scrolling
- ✅ Pagination and lazy loading

### In Progress

- 🔄 Structure/Indexes/Constraints tabs
- 🔄 Advanced filtering
```

Update roadmap:

```markdown
### Phase 3: Data Browsing ✅
- Navigation tree
- Table data view
- Virtual scrolling with pagination
- Interactive data navigation

### Phase 4: Command Palette & Query (Next)
- Command palette UI
- Query execution
- Result display
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README for Phase 3 completion"
```

---

## Summary

**Total Tasks:** 10
**Estimated Time:** 4-6 hours

**Phase 3 delivers:**
1. ✅ Complete Phase 2 connection logic (TODO items)
2. ✅ Error message display system
3. ✅ Navigation tree data model and UI component
4. ✅ Tree integration with database metadata
5. ✅ Table data view with virtual scrolling
6. ✅ Pagination and lazy loading
7. ✅ Interactive navigation (keyboard + mouse support)
8. ✅ Updated documentation

**Ready for Phase 4:** Command Palette & Query execution
