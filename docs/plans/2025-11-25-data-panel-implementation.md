# Data Panel Enhancements - Implementation Plan

## Overview

This plan covers three features:
1. Column Sorting
2. Horizontal Scrolling
3. Data Search

Implementation order: Sorting → Scrolling → Search

---

## Task 1: Column Sorting

### 1.1 Add Sort State to TableView

**File:** `internal/ui/components/table_view.go`

Add new fields to `TableView` struct (after line 31):

```go
type TableView struct {
    // ... existing fields ...

    // Sort state
    SortColumn    int    // -1 means no sort, otherwise index of sorted column
    SortDirection string // "ASC" or "DESC"
    NullsFirst    bool   // true = NULLS FIRST, false = NULLS LAST (default)
}
```

Update `NewTableView` (after line 40):

```go
func NewTableView(th theme.Theme) *TableView {
    return &TableView{
        // ... existing fields ...
        SortColumn:    -1,
        SortDirection: "ASC",
        NullsFirst:    false,
    }
}
```

### 1.2 Add Sort Methods to TableView

**File:** `internal/ui/components/table_view.go`

Add after `MoveSelectionHorizontal` method (after line 398):

```go
// ToggleSort toggles sorting on the currently selected column
func (tv *TableView) ToggleSort() {
    if tv.SortColumn == tv.SelectedCol {
        // Same column - toggle direction
        if tv.SortDirection == "ASC" {
            tv.SortDirection = "DESC"
        } else {
            tv.SortDirection = "ASC"
        }
    } else {
        // New column - start with ASC
        tv.SortColumn = tv.SelectedCol
        tv.SortDirection = "ASC"
    }
}

// ToggleNullsFirst toggles NULLS FIRST/LAST for current sort
func (tv *TableView) ToggleNullsFirst() {
    tv.NullsFirst = !tv.NullsFirst
}

// GetSortColumn returns the current sort column name, or empty string if no sort
func (tv *TableView) GetSortColumn() string {
    if tv.SortColumn < 0 || tv.SortColumn >= len(tv.Columns) {
        return ""
    }
    return tv.Columns[tv.SortColumn]
}

// GetSortDirection returns the current sort direction
func (tv *TableView) GetSortDirection() string {
    return tv.SortDirection
}

// GetNullsFirst returns whether NULLS FIRST is enabled
func (tv *TableView) GetNullsFirst() bool {
    return tv.NullsFirst
}

// ClearSort clears the current sort
func (tv *TableView) ClearSort() {
    tv.SortColumn = -1
    tv.SortDirection = "ASC"
    tv.NullsFirst = false
}
```

### 1.3 Update Header Rendering with Sort Indicators

**File:** `internal/ui/components/table_view.go`

Modify `renderHeader` method (line 183-229) to add sort indicators:

```go
func (tv *TableView) renderHeader() string {
    s := make([]string, 0, len(tv.Columns)*2-1)

    separatorStyle := lipgloss.NewStyle().
        Foreground(tv.Theme.Border).
        Background(tv.Theme.Selection)
    separator := separatorStyle.Render(" │ ")

    for i, col := range tv.Columns {
        width := tv.ColumnWidths[i]
        if width <= 0 {
            continue
        }

        // Add sort indicator if this column is sorted
        displayCol := col
        if i == tv.SortColumn {
            if tv.SortDirection == "ASC" {
                if tv.NullsFirst {
                    displayCol = col + " ↑ⁿ"
                } else {
                    displayCol = col + " ↑"
                }
            } else {
                if tv.NullsFirst {
                    displayCol = col + " ↓ⁿ"
                } else {
                    displayCol = col + " ↓"
                }
            }
        }

        truncated := runewidth.Truncate(displayCol, width, "…")

        widthStyle := lipgloss.NewStyle().
            Width(width).
            MaxWidth(width).
            Inline(true)

        headerCellStyle := lipgloss.NewStyle().
            Background(tv.Theme.Selection)

        renderedCell := headerCellStyle.Render(widthStyle.Render(truncated))
        s = append(s, renderedCell)

        if i < len(tv.Columns)-1 {
            s = append(s, separator)
        }
    }

    headerRow := lipgloss.JoinHorizontal(lipgloss.Top, s...)

    headerStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(tv.Theme.TableHeader)

    return headerStyle.Render(headerRow)
}
```

### 1.4 Add ORDER BY Support to QueryTableData

**File:** `internal/db/metadata/data.go`

Update function signature and implementation:

```go
// SortOptions holds sorting configuration
type SortOptions struct {
    Column     string
    Direction  string // "ASC" or "DESC"
    NullsFirst bool
}

// QueryTableData fetches paginated table data with optional sorting
func QueryTableData(ctx context.Context, pool *connection.Pool, schema, table string, offset, limit int, sort *SortOptions) (*TableData, error) {
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

    // Build query with optional ORDER BY
    query := fmt.Sprintf("SELECT * FROM %s.%s", schema, table)

    if sort != nil && sort.Column != "" {
        nullsClause := "NULLS LAST"
        if sort.NullsFirst {
            nullsClause = "NULLS FIRST"
        }
        query += fmt.Sprintf(" ORDER BY %s %s %s", sort.Column, sort.Direction, nullsClause)
    }

    query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

    result, err := pool.QueryWithColumns(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to query table data: %w", err)
    }

    // ... rest of function unchanged ...
}
```

### 1.5 Update LoadTableDataMsg and App

**File:** `internal/app/app.go`

Update `LoadTableDataMsg` struct (line 114-119):

```go
type LoadTableDataMsg struct {
    Schema     string
    Table      string
    Offset     int
    Limit      int
    SortColumn string
    SortDir    string
    NullsFirst bool
}
```

Update `loadTableData` method (line 1718-1738):

```go
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
```

### 1.6 Add Key Handlers for Sort

**File:** `internal/app/app.go`

Add in the right panel key handling section (around line 752-806):

```go
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
```

### 1.7 Update Help Text

**File:** `internal/ui/help/help.go`

Add sorting keybindings to the Data View section.

---

## Task 2: Horizontal Scrolling

### 2.1 Add Scroll State to TableView

**File:** `internal/ui/components/table_view.go`

Add new fields to `TableView` struct:

```go
type TableView struct {
    // ... existing fields ...

    // Horizontal scrolling
    LeftColOffset int // First visible column index
    VisibleCols   int // Number of columns that fit in current width
}
```

### 2.2 Add calculateVisibleCols Method

**File:** `internal/ui/components/table_view.go`

```go
// calculateVisibleCols calculates how many columns fit in the current width
func (tv *TableView) calculateVisibleCols() {
    if len(tv.ColumnWidths) == 0 {
        tv.VisibleCols = 0
        return
    }

    // Reserve space for edge indicators (2 chars each side)
    availableWidth := tv.Width - 4

    // Count columns that fit starting from LeftColOffset
    totalWidth := 0
    count := 0
    for i := tv.LeftColOffset; i < len(tv.ColumnWidths); i++ {
        colWidth := tv.ColumnWidths[i]
        separatorWidth := 0
        if count > 0 {
            separatorWidth = 3 // " │ "
        }

        if totalWidth + colWidth + separatorWidth > availableWidth {
            break
        }
        totalWidth += colWidth + separatorWidth
        count++
    }

    if count < 1 {
        count = 1 // Always show at least one column
    }
    tv.VisibleCols = count
}
```

### 2.3 Update MoveSelectionHorizontal for Auto-Scroll

**File:** `internal/ui/components/table_view.go`

Replace existing `MoveSelectionHorizontal` method:

```go
// MoveSelectionHorizontal moves the selected column left or right with auto-scroll
func (tv *TableView) MoveSelectionHorizontal(delta int) {
    tv.SelectedCol += delta

    // Bounds checking
    if tv.SelectedCol < 0 {
        tv.SelectedCol = 0
    }
    if tv.SelectedCol >= len(tv.Columns) {
        tv.SelectedCol = len(tv.Columns) - 1
    }

    // Auto-scroll to keep selected column visible
    if tv.SelectedCol < tv.LeftColOffset {
        tv.LeftColOffset = tv.SelectedCol
    }
    if tv.SelectedCol >= tv.LeftColOffset+tv.VisibleCols {
        tv.LeftColOffset = tv.SelectedCol - tv.VisibleCols + 1
    }

    // Bounds check LeftColOffset
    if tv.LeftColOffset < 0 {
        tv.LeftColOffset = 0
    }
    maxOffset := len(tv.Columns) - tv.VisibleCols
    if maxOffset < 0 {
        maxOffset = 0
    }
    if tv.LeftColOffset > maxOffset {
        tv.LeftColOffset = maxOffset
    }
}
```

### 2.4 Add Jump Scroll and First/Last Column Methods

**File:** `internal/ui/components/table_view.go`

```go
// JumpScrollHorizontal scrolls horizontally by half the visible columns
func (tv *TableView) JumpScrollHorizontal(delta int) {
    jumpAmount := tv.VisibleCols / 2
    if jumpAmount < 1 {
        jumpAmount = 1
    }

    tv.SelectedCol += delta * jumpAmount

    // Bounds checking
    if tv.SelectedCol < 0 {
        tv.SelectedCol = 0
    }
    if tv.SelectedCol >= len(tv.Columns) {
        tv.SelectedCol = len(tv.Columns) - 1
    }

    // Update scroll position
    tv.MoveSelectionHorizontal(0) // Trigger auto-scroll logic
}

// JumpToFirstColumn jumps to the first column
func (tv *TableView) JumpToFirstColumn() {
    tv.SelectedCol = 0
    tv.LeftColOffset = 0
}

// JumpToLastColumn jumps to the last column
func (tv *TableView) JumpToLastColumn() {
    if len(tv.Columns) > 0 {
        tv.SelectedCol = len(tv.Columns) - 1
        // Scroll to show last column
        maxOffset := len(tv.Columns) - tv.VisibleCols
        if maxOffset < 0 {
            maxOffset = 0
        }
        tv.LeftColOffset = maxOffset
    }
}
```

### 2.5 Update View() to Render Only Visible Columns

**File:** `internal/ui/components/table_view.go`

Update `View()` method to calculate visible columns and add edge indicators:

```go
func (tv *TableView) View() string {
    if len(tv.Columns) == 0 {
        return tv.Style.Render("No data")
    }

    // Calculate visible columns
    tv.calculateVisibleCols()

    var b strings.Builder

    // Add left edge indicator if scrolled
    leftIndicator := "  " // 2 spaces placeholder
    if tv.LeftColOffset > 0 {
        leftIndicator = "◀ "
    }

    // Add right edge indicator if more columns to the right
    rightIndicator := "  "
    if tv.LeftColOffset+tv.VisibleCols < len(tv.Columns) {
        rightIndicator = " ▶"
    }

    // Render header with indicators
    b.WriteString(leftIndicator)
    b.WriteString(tv.renderHeader())
    b.WriteString(rightIndicator)
    b.WriteString("\n")

    // Render separator
    b.WriteString("  ") // Align with left indicator
    b.WriteString(tv.renderSeparator())
    b.WriteString("  ")
    b.WriteString("\n")

    // ... rest of rendering with visible columns only ...
}
```

### 2.6 Update renderHeader/renderRow for Visible Columns

Modify `renderHeader()` and `renderRow()` to only render columns from `LeftColOffset` to `LeftColOffset + VisibleCols`.

### 2.7 Update Status Bar

**File:** `internal/ui/components/table_view.go`

Update `renderStatus()` to include column info:

```go
func (tv *TableView) renderStatus() string {
    endRow := tv.TopRow + len(tv.Rows)
    if endRow > tv.TotalRows {
        endRow = tv.TotalRows
    }

    // Column info
    colInfo := ""
    if len(tv.Columns) > tv.VisibleCols {
        endCol := tv.LeftColOffset + tv.VisibleCols
        if endCol > len(tv.Columns) {
            endCol = len(tv.Columns)
        }
        colInfo = fmt.Sprintf("Cols %d-%d of %d │ ", tv.LeftColOffset+1, endCol, len(tv.Columns))
    }

    showing := fmt.Sprintf(" 󰈙 %s%d-%d of %d rows", colInfo, tv.TopRow+1, endRow, tv.TotalRows)
    return lipgloss.NewStyle().
        Foreground(tv.Theme.Metadata).
        Italic(true).
        Render(showing)
}
```

### 2.8 Add Key Handlers for Horizontal Scroll

**File:** `internal/app/app.go`

Add in right panel key handling:

```go
case "H":
    // Jump scroll left
    a.tableView.JumpScrollHorizontal(-1)
    return a, nil

case "L":
    // Jump scroll right
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
```

---

## Task 3: Data Search

### 3.1 Add Search State to TableView

**File:** `internal/ui/components/table_view.go`

```go
// MatchPos represents a search match position
type MatchPos struct {
    Row int
    Col int
}

type TableView struct {
    // ... existing fields ...

    // Search state
    SearchActive  bool
    SearchMode    string     // "local" or "table"
    SearchQuery   string
    Matches       []MatchPos
    CurrentMatch  int
}
```

### 3.2 Create Search Input Component

**File:** `internal/ui/components/search_input.go`

Create a new component for the search input box:

```go
package components

import (
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/lipgloss"
    "github.com/rebeliceyang/lazypg/internal/ui/theme"
)

type SearchInput struct {
    Input     textinput.Model
    Mode      string // "local" or "table"
    Theme     theme.Theme
    Width     int
    Visible   bool
}

func NewSearchInput(th theme.Theme) *SearchInput {
    ti := textinput.New()
    ti.Placeholder = "Search..."
    ti.Focus()

    return &SearchInput{
        Input: ti,
        Mode:  "local",
        Theme: th,
    }
}

func (s *SearchInput) ToggleMode() {
    if s.Mode == "local" {
        s.Mode = "table"
    } else {
        s.Mode = "local"
    }
}

func (s *SearchInput) View() string {
    modeIndicator := "[Local]"
    if s.Mode == "table" {
        modeIndicator = "[Table]"
    }

    modeStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#89b4fa"))

    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(s.Theme.BorderFocused).
        Padding(0, 1).
        Width(s.Width)

    content := modeStyle.Render(modeIndicator) + " 🔍 " + s.Input.View()
    return boxStyle.Render(content)
}
```

### 3.3 Add Search Methods to TableView

**File:** `internal/ui/components/table_view.go`

```go
// SearchLocal searches only loaded data
func (tv *TableView) SearchLocal(query string) {
    tv.SearchQuery = query
    tv.SearchMode = "local"
    tv.Matches = nil
    tv.CurrentMatch = 0

    if query == "" {
        tv.SearchActive = false
        return
    }

    tv.SearchActive = true
    queryLower := strings.ToLower(query)

    for rowIdx, row := range tv.Rows {
        for colIdx, cell := range row {
            if strings.Contains(strings.ToLower(cell), queryLower) {
                tv.Matches = append(tv.Matches, MatchPos{Row: rowIdx, Col: colIdx})
            }
        }
    }

    if len(tv.Matches) > 0 {
        tv.jumpToMatch(0)
    }
}

// jumpToMatch jumps to match at given index
func (tv *TableView) jumpToMatch(idx int) {
    if idx < 0 || idx >= len(tv.Matches) {
        return
    }

    tv.CurrentMatch = idx
    match := tv.Matches[idx]

    // Move selection to match
    tv.SelectedRow = match.Row
    tv.SelectedCol = match.Col

    // Scroll to show match
    if tv.SelectedRow < tv.TopRow {
        tv.TopRow = tv.SelectedRow
    }
    if tv.SelectedRow >= tv.TopRow+tv.VisibleRows {
        tv.TopRow = tv.SelectedRow - tv.VisibleRows + 1
    }

    // Horizontal scroll
    tv.MoveSelectionHorizontal(0) // Trigger auto-scroll
}

// NextMatch jumps to next match
func (tv *TableView) NextMatch() {
    if len(tv.Matches) == 0 {
        return
    }
    nextIdx := (tv.CurrentMatch + 1) % len(tv.Matches)
    tv.jumpToMatch(nextIdx)
}

// PrevMatch jumps to previous match
func (tv *TableView) PrevMatch() {
    if len(tv.Matches) == 0 {
        return
    }
    prevIdx := tv.CurrentMatch - 1
    if prevIdx < 0 {
        prevIdx = len(tv.Matches) - 1
    }
    tv.jumpToMatch(prevIdx)
}

// ClearSearch clears search state
func (tv *TableView) ClearSearch() {
    tv.SearchActive = false
    tv.SearchQuery = ""
    tv.Matches = nil
    tv.CurrentMatch = 0
}

// IsMatch checks if a cell is a match
func (tv *TableView) IsMatch(row, col int) bool {
    for _, m := range tv.Matches {
        if m.Row == row && m.Col == col {
            return true
        }
    }
    return false
}

// IsCurrentMatch checks if a cell is the current match
func (tv *TableView) IsCurrentMatch(row, col int) bool {
    if tv.CurrentMatch < 0 || tv.CurrentMatch >= len(tv.Matches) {
        return false
    }
    m := tv.Matches[tv.CurrentMatch]
    return m.Row == row && m.Col == col
}
```

### 3.4 Update renderRow for Match Highlighting

**File:** `internal/ui/components/table_view.go`

Update `renderRow` to highlight matches:

```go
// In renderRow, update the cell styling logic:
var cellStyle lipgloss.Style
if tv.IsCurrentMatch(rowIndex, i) {
    // Current match - bright highlight
    cellStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("#f9e2af")). // Yellow
        Foreground(lipgloss.Color("#1e1e2e")). // Dark
        Bold(true)
} else if tv.IsMatch(rowIndex, i) {
    // Other match - subtle highlight
    cellStyle = lipgloss.NewStyle().
        Background(lipgloss.Color("#45475a")). // Surface1
        Foreground(tv.Theme.Foreground)
} else if selected && i == tv.SelectedCol {
    // ... existing selected cell logic
}
```

### 3.5 Add Table Search Query Builder

**File:** `internal/db/metadata/data.go`

```go
// SearchTableData searches entire table
func SearchTableData(ctx context.Context, pool *connection.Pool, schema, table string, columns []string, keyword string, limit int) (*TableData, error) {
    // Build WHERE clause with ILIKE for all columns
    var conditions []string
    for _, col := range columns {
        conditions = append(conditions, fmt.Sprintf("%s::text ILIKE '%%%s%%'", col, keyword))
    }

    whereClause := strings.Join(conditions, " OR ")

    query := fmt.Sprintf("SELECT * FROM %s.%s WHERE %s LIMIT %d",
        schema, table, whereClause, limit)

    result, err := pool.QueryWithColumns(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("search query failed: %w", err)
    }

    // ... convert to TableData ...
}
```

### 3.6 Add Search State to App

**File:** `internal/app/app.go`

```go
type App struct {
    // ... existing fields ...

    // Search
    showSearch  bool
    searchInput *components.SearchInput
}
```

### 3.7 Add Key Handlers for Search

**File:** `internal/app/app.go`

```go
case "/", "ctrl+f":
    // Open search (when right panel focused)
    if a.state.FocusedPanel == models.RightPanel {
        a.showSearch = true
        a.searchInput = components.NewSearchInput(a.theme)
    }
    return a, nil

case "n":
    // Next match (when search active)
    if a.tableView.SearchActive {
        a.tableView.NextMatch()
    }
    return a, nil

case "N":
    // Previous match
    if a.tableView.SearchActive {
        a.tableView.PrevMatch()
    }
    return a, nil
```

### 3.8 Handle Search Input

When search input is visible, handle Tab to toggle mode, Enter to execute, Esc to cancel.

### 3.9 Update Status Bar for Search

Show match count: `Match 3 of 12 │ Cols 1-5 of 12 │ 1-100 of 5000 rows`

---

## Verification Steps

### Task 1 (Sorting)
1. Select a column with h/l
2. Press s - should sort ascending, show ↑ indicator
3. Press s again - should sort descending, show ↓
4. Press S - should toggle NULLS FIRST, show ↑ⁿ or ↓ⁿ
5. Verify data reloads from database with ORDER BY

### Task 2 (Horizontal Scrolling)
1. Load a table with many columns
2. Use h/l to move - selected column should stay visible
3. Use H/L to jump scroll half screen
4. Press 0 to go to first column
5. Press $ to go to last column
6. Verify ◀ ▶ indicators appear/disappear correctly
7. Verify status bar shows column range

### Task 3 (Search)
1. Press / or Ctrl+F to open search
2. Type a keyword - matches should highlight
3. Press Tab to toggle Local/Table mode
4. Press Enter to execute search
5. Press n to go to next match
6. Press N to go to previous match
7. Verify status shows "Match X of Y"
8. Press Esc to close search

---

## Files Modified Summary

| File | Changes |
|------|---------|
| `internal/ui/components/table_view.go` | Sort state, scroll state, search state, methods |
| `internal/ui/components/search_input.go` | New file for search input |
| `internal/db/metadata/data.go` | SortOptions, SearchTableData |
| `internal/app/app.go` | LoadTableDataMsg, key handlers, search state |
| `internal/ui/help/help.go` | New keybindings documentation |
