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
		Columns:      []string{},
		Rows:         [][]string{},
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
	// Modern header style with color
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("105")). // Purple
		Background(lipgloss.Color("236"))  // Dark gray background
	return headerStyle.Render(" " + strings.Join(parts, " │ ") + " ")
}

func (tv *TableView) renderSeparator() string {
	var parts []string
	for _, width := range tv.ColumnWidths {
		parts = append(parts, strings.Repeat("─", width))
	}
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")) // Gray
	return separatorStyle.Render("─" + strings.Join(parts, "─┼─") + "─")
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

	line := " " + strings.Join(parts, " │ ") + " "

	if selected {
		// Modern selection with gradient-like effect
		return lipgloss.NewStyle().
			Background(lipgloss.Color("25")). // Brighter blue
			Foreground(lipgloss.Color("15")). // White text
			Bold(true).
			Render(line)
	}
	return line
}

func (tv *TableView) renderStatus() string {
	endRow := tv.TopRow + len(tv.Rows)
	if endRow > tv.TotalRows {
		endRow = tv.TotalRows
	}

	showing := fmt.Sprintf(" 󰈙 %d-%d of %d rows", tv.TopRow+1, endRow, tv.TotalRows)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")). // Medium gray
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
