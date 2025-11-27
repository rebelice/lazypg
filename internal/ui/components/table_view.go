package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/jsonb"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// TableView displays table data with virtual scrolling
type TableView struct {
	Columns      []string
	Rows         [][]string
	Width        int
	Height       int
	Style        lipgloss.Style
	Theme        theme.Theme // Color theme

	// Virtual scrolling state
	TopRow       int
	VisibleRows  int
	SelectedRow  int
	SelectedCol  int // Currently selected column
	TotalRows    int

	// Column widths (calculated)
	ColumnWidths []int

	// Sort state
	SortColumn    int    // -1 means no sort, otherwise index of sorted column
	SortDirection string // "ASC" or "DESC"
	NullsFirst    bool   // true = NULLS FIRST, false = NULLS LAST (default)

	// Horizontal scrolling state
	LeftColOffset int // First visible column index
	VisibleCols   int // Number of columns that fit in current width

	// Search state
	SearchActive bool
	SearchMode   string     // "local" or "table"
	SearchQuery  string
	Matches      []MatchPos // List of match positions
	CurrentMatch int        // Index in Matches

	// Preview pane for truncated content
	PreviewPane *PreviewPane
}

// MatchPos represents a search match position
type MatchPos struct {
	Row int
	Col int
}

// NewTableView creates a new table view with theme
func NewTableView(th theme.Theme) *TableView {
	return &TableView{
		Columns:       []string{},
		Rows:          [][]string{},
		ColumnWidths:  []int{},
		Theme:         th,
		SortColumn:    -1,
		SortDirection: "ASC",
		NullsFirst:    false,
		PreviewPane:   NewPreviewPane(th),
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

	numColumns := len(tv.Columns)
	tv.ColumnWidths = make([]int, numColumns)

	// Step 1: Calculate desired widths based on content
	desiredWidths := make([]int, numColumns)

	// Start with column header lengths (add 4 chars for sort indicator space)
	for i, col := range tv.Columns {
		desiredWidths[i] = runewidth.StringWidth(col) + 4
	}

	// Check row data
	for _, row := range tv.Rows {
		for i, cell := range row {
			if i < numColumns {
				cellLen := runewidth.StringWidth(cell)
				if cellLen > desiredWidths[i] {
					desiredWidths[i] = cellLen
				}
			}
		}
	}

	// Step 2: Apply constraints (min/max width per column)
	maxWidth := 50
	minWidth := 10

	for i, w := range desiredWidths {
		if w > maxWidth {
			w = maxWidth
		}
		if w < minWidth {
			w = minWidth
		}
		tv.ColumnWidths[i] = w
	}
}

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

		if totalWidth+colWidth+separatorWidth > availableWidth {
			break
		}
		totalWidth += colWidth + separatorWidth
		count++
	}

	if count < 1 && len(tv.ColumnWidths) > 0 {
		count = 1 // Always show at least one column
	}
	tv.VisibleCols = count
}

// View renders the table
func (tv *TableView) View() string {
	if len(tv.Columns) == 0 {
		return tv.Style.Render("No data")
	}

	// Calculate visible columns for horizontal scrolling
	tv.calculateVisibleCols()

	var b strings.Builder

	// Determine edge indicators
	leftIndicator := "  " // 2 spaces placeholder
	if tv.LeftColOffset > 0 {
		leftIndicator = "◀ "
	}
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

	// Calculate how many rows we can show
	// Height is already the content area height
	// Subtract 3 for header + separator + status line
	tv.VisibleRows = tv.Height - 3
	if tv.VisibleRows < 1 {
		tv.VisibleRows = 1
	}

	// Render visible rows
	endRow := tv.TopRow + tv.VisibleRows
	if endRow > len(tv.Rows) {
		endRow = len(tv.Rows)
	}

	for i := tv.TopRow; i < endRow; i++ {
		isSelected := i == tv.SelectedRow
		b.WriteString("  ") // Align with left indicator
		b.WriteString(tv.renderRow(tv.Rows[i], isSelected, i))
		b.WriteString("  ")
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
	s := make([]string, 0, tv.VisibleCols*2-1) // Account for separators

	// Create separator style
	separatorStyle := lipgloss.NewStyle().
		Foreground(tv.Theme.Border).
		Background(tv.Theme.Selection)
	separator := separatorStyle.Render(" │ ")

	// Only render visible columns
	endCol := tv.LeftColOffset + tv.VisibleCols
	if endCol > len(tv.Columns) {
		endCol = len(tv.Columns)
	}

	for idx, i := 0, tv.LeftColOffset; i < endCol; i, idx = i+1, idx+1 {
		col := tv.Columns[i]
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

		// Use runewidth.Truncate for proper truncation
		truncated := runewidth.Truncate(displayCol, width, "…")

		// Create cell width style
		widthStyle := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Inline(true)

		// Create header cell style
		headerCellStyle := lipgloss.NewStyle().
			Background(tv.Theme.Selection)

		// Render cell with header background
		renderedCell := headerCellStyle.Render(widthStyle.Render(truncated))
		s = append(s, renderedCell)

		// Add separator between columns (but not after the last visible column)
		if i < endCol-1 {
			s = append(s, separator)
		}
	}

	// Join header cells horizontally with separators
	headerRow := lipgloss.JoinHorizontal(lipgloss.Top, s...)

	// Apply bold and color to the entire row
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tv.Theme.TableHeader)

	return headerStyle.Render(headerRow)
}

func (tv *TableView) renderSeparator() string {
	// Calculate total width of visible columns only
	totalWidth := 0
	endCol := tv.LeftColOffset + tv.VisibleCols
	if endCol > len(tv.ColumnWidths) {
		endCol = len(tv.ColumnWidths)
	}

	for i := tv.LeftColOffset; i < endCol; i++ {
		totalWidth += tv.ColumnWidths[i]
	}

	// Add width for separators: 3 chars (" │ ") * (number of separators)
	visibleCount := endCol - tv.LeftColOffset
	if visibleCount > 1 {
		totalWidth += 3 * (visibleCount - 1)
	}

	// Create a simple horizontal line
	separatorStyle := lipgloss.NewStyle().
		Foreground(tv.Theme.Border)

	return separatorStyle.Render(strings.Repeat("─", totalWidth))
}

func (tv *TableView) renderRow(row []string, selected bool, rowIndex int) string {
	s := make([]string, 0, tv.VisibleCols*2-1) // Account for separators

	// Create separator style (always uses border color, no background)
	separatorStyle := lipgloss.NewStyle().
		Foreground(tv.Theme.Border)
	separator := separatorStyle.Render(" │ ")

	// Only render visible columns
	endCol := tv.LeftColOffset + tv.VisibleCols
	if endCol > len(tv.ColumnWidths) {
		endCol = len(tv.ColumnWidths)
	}

	for i := tv.LeftColOffset; i < endCol; i++ {
		if i >= len(row) || i >= len(tv.ColumnWidths) {
			break
		}
		width := tv.ColumnWidths[i]
		if width <= 0 {
			continue
		}

		value := row[i]

		// Check if this looks like JSONB and format for display
		cellValue := value
		if jsonb.IsJSONB(cellValue) {
			cellValue = jsonb.Truncate(cellValue, 50)
		}

		// Use runewidth.Truncate for proper truncation (handles multibyte chars)
		truncated := runewidth.Truncate(cellValue, width, "…")

		// Create cell width style
		widthStyle := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Inline(true)

		// Determine cell background based on selection and search
		// Priority: selected cell > current match > other matches > selected row > normal
		var cellStyle lipgloss.Style
		if selected && i == tv.SelectedCol {
			// Selected cell - highest priority, bright highlight
			cellStyle = lipgloss.NewStyle().
				Background(tv.Theme.BorderFocused).
				Foreground(tv.Theme.Background).
				Bold(true)
		} else if tv.IsCurrentMatch(rowIndex, i) {
			// Current search match - bright yellow highlight
			cellStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#f9e2af")). // Yellow
				Foreground(lipgloss.Color("#1e1e2e")). // Dark
				Bold(true)
		} else if tv.IsMatch(rowIndex, i) {
			// Other search match - subtle highlight
			cellStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#585b70")). // Surface2
				Foreground(tv.Theme.Foreground)
		} else if selected {
			// Selected row but not selected column - dim highlight
			cellStyle = lipgloss.NewStyle().
				Background(tv.Theme.Selection).
				Foreground(tv.Theme.Foreground)
		} else {
			// Normal cell
			cellStyle = lipgloss.NewStyle()
		}

		// Render cell: first apply width, then apply cell style
		renderedCell := cellStyle.Render(widthStyle.Render(truncated))
		s = append(s, renderedCell)

		// Add separator between columns (but not after the last visible column)
		if i < endCol-1 {
			s = append(s, separator)
		}
	}

	// Join cells horizontally with separators
	rowStr := lipgloss.JoinHorizontal(lipgloss.Top, s...)

	return rowStr
}

func (tv *TableView) renderStatus() string {
	endRow := tv.TopRow + len(tv.Rows)
	if endRow > tv.TotalRows {
		endRow = tv.TotalRows
	}

	// Search match info
	matchInfo := ""
	if tv.SearchActive && len(tv.Matches) > 0 {
		matchInfo = fmt.Sprintf("Match %d of %d │ ", tv.CurrentMatch+1, len(tv.Matches))
	}

	// Column info for horizontal scrolling
	colInfo := ""
	if len(tv.Columns) > tv.VisibleCols {
		endCol := tv.LeftColOffset + tv.VisibleCols
		if endCol > len(tv.Columns) {
			endCol = len(tv.Columns)
		}
		colInfo = fmt.Sprintf("Cols %d-%d of %d │ ", tv.LeftColOffset+1, endCol, len(tv.Columns))
	}

	showing := fmt.Sprintf(" 󰈙 %s%s%d-%d of %d rows", matchInfo, colInfo, tv.TopRow+1, endRow, tv.TotalRows)
	return lipgloss.NewStyle().
		Foreground(tv.Theme.Metadata).
		Italic(true).
		Render(showing)
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

	// Update preview pane
	tv.UpdatePreviewPane()
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

// GetSelectedCell returns the currently selected row and column indices
func (tv *TableView) GetSelectedCell() (row int, col int) {
	return tv.SelectedRow, tv.SelectedCol
}

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

	// Update preview pane
	tv.UpdatePreviewPane()
}

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

	// Update scroll position via MoveSelectionHorizontal's auto-scroll
	tv.MoveSelectionHorizontal(0)
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

// SetSearchResults sets search results from table search
func (tv *TableView) SetSearchResults(query string, matches []MatchPos) {
	tv.SearchQuery = query
	tv.SearchMode = "table"
	tv.Matches = matches
	tv.CurrentMatch = 0
	tv.SearchActive = len(matches) > 0

	if len(matches) > 0 {
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

	// Scroll to show match (vertical)
	if tv.SelectedRow < tv.TopRow {
		tv.TopRow = tv.SelectedRow
	}
	if tv.SelectedRow >= tv.TopRow+tv.VisibleRows {
		tv.TopRow = tv.SelectedRow - tv.VisibleRows + 1
	}

	// Horizontal scroll via MoveSelectionHorizontal's auto-scroll
	tv.MoveSelectionHorizontal(0)
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

// GetMatchInfo returns current match info for status bar
func (tv *TableView) GetMatchInfo() (current int, total int) {
	if !tv.SearchActive || len(tv.Matches) == 0 {
		return 0, 0
	}
	return tv.CurrentMatch + 1, len(tv.Matches)
}

// IsCellTruncated checks if the currently selected cell content is truncated
func (tv *TableView) IsCellTruncated() bool {
	if tv.SelectedRow < 0 || tv.SelectedRow >= len(tv.Rows) {
		return false
	}
	if tv.SelectedCol < 0 || tv.SelectedCol >= len(tv.ColumnWidths) {
		return false
	}
	if tv.SelectedCol >= len(tv.Rows[tv.SelectedRow]) {
		return false
	}

	cellValue := tv.Rows[tv.SelectedRow][tv.SelectedCol]
	colWidth := tv.ColumnWidths[tv.SelectedCol]

	// Check if cell content width exceeds column width
	return runewidth.StringWidth(cellValue) > colWidth
}

// GetSelectedCellContent returns the full content of the selected cell
func (tv *TableView) GetSelectedCellContent() string {
	if tv.SelectedRow < 0 || tv.SelectedRow >= len(tv.Rows) {
		return ""
	}
	if tv.SelectedCol < 0 || tv.SelectedCol >= len(tv.Rows[tv.SelectedRow]) {
		return ""
	}
	return tv.Rows[tv.SelectedRow][tv.SelectedCol]
}

// GetSelectedColumnName returns the name of the currently selected column
func (tv *TableView) GetSelectedColumnName() string {
	if tv.SelectedCol < 0 || tv.SelectedCol >= len(tv.Columns) {
		return ""
	}
	return tv.Columns[tv.SelectedCol]
}

// UpdatePreviewPane updates the preview pane with current selection
func (tv *TableView) UpdatePreviewPane() {
	if tv.PreviewPane == nil {
		return
	}

	content := tv.GetSelectedCellContent()
	title := tv.GetSelectedColumnName()
	isTruncated := tv.IsCellTruncated()

	tv.PreviewPane.SetContent(content, title, isTruncated)
}

// SetPreviewPaneDimensions sets the dimensions for the preview pane
func (tv *TableView) SetPreviewPaneDimensions(width, maxHeight int) {
	if tv.PreviewPane != nil {
		tv.PreviewPane.Width = width
		tv.PreviewPane.MaxHeight = maxHeight
	}
}

// TogglePreviewPane toggles the preview pane visibility
func (tv *TableView) TogglePreviewPane() {
	if tv.PreviewPane != nil {
		tv.PreviewPane.Toggle()
	}
}

// GetPreviewPaneHeight returns the current preview pane height
func (tv *TableView) GetPreviewPaneHeight() int {
	if tv.PreviewPane != nil {
		return tv.PreviewPane.Height()
	}
	return 0
}
