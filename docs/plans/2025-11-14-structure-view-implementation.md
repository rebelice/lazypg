# Structure View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add comprehensive table structure inspection with columns, constraints, and indexes in a tabbed interface.

**Architecture:** Create a tab-based UI container (`StructureView`) that switches between four views (Data, Columns, Constraints, Indexes). Each specialized view is a separate component that queries PostgreSQL system catalogs and displays results in table format. Integrate with existing app state management and keyboard shortcuts.

**Tech Stack:**
- Bubble Tea (TUI framework)
- Lipgloss (styling)
- PostgreSQL system catalogs (pg_catalog, information_schema)
- pgx v5 (database driver)

---

## Task 1: Add Structure View Models

**Files:**
- Modify: `internal/models/models.go`

**Step 1: Add column structure models**

In `internal/models/models.go`, add after existing `ColumnInfo`:

```go
// ColumnDetail contains comprehensive column information for structure view
type ColumnDetail struct {
	Name          string
	DataType      string
	IsNullable    bool
	DefaultValue  string
	IsPrimaryKey  bool
	IsForeignKey  bool
	IsUnique      bool
	HasCheck      bool
	Comment       string
}

// Constraint represents a table constraint
type Constraint struct {
	Name         string
	Type         string // 'p'=PK, 'f'=FK, 'u'=Unique, 'c'=Check
	Columns      []string
	Definition   string
	ForeignTable string // For FK: "schema.table"
	ForeignCols  []string
}

// IndexInfo represents an index
type IndexInfo struct {
	Name        string
	Type        string // btree, hash, gin, gist, brin, spgist
	Columns     []string
	Definition  string
	IsUnique    bool
	IsPrimary   bool
	IsPartial   bool
	Size        int64
	Predicate   string // WHERE clause for partial indexes
}
```

**Step 2: Commit models**

```bash
git add internal/models/models.go
git commit -m "feat: add structure view models for columns, constraints, and indexes"
```

---

## Task 2: Create Columns Metadata Query

**Files:**
- Create: `internal/db/metadata/column_details.go`

**Step 1: Create column details query file**

```go
package metadata

import (
	"context"
	"fmt"

	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/models"
)

// GetColumnDetails retrieves detailed column information including constraints
func GetColumnDetails(ctx context.Context, pool *connection.Pool, schema, table string) ([]models.ColumnDetail, error) {
	query := `
		WITH column_constraints AS (
			SELECT
				a.attname AS column_name,
				bool_or(con.contype = 'p') AS is_pk,
				bool_or(con.contype = 'f') AS is_fk,
				bool_or(con.contype = 'u') AS is_unique,
				bool_or(con.contype = 'c') AS has_check
			FROM pg_catalog.pg_attribute a
			LEFT JOIN pg_catalog.pg_constraint con ON con.conrelid = a.attrelid
				AND a.attnum = ANY(con.conkey)
			WHERE a.attrelid = ($1 || '.' || $2)::regclass
				AND a.attnum > 0
				AND NOT a.attisdropped
			GROUP BY a.attname
		)
		SELECT
			c.column_name,
			c.data_type,
			CASE
				WHEN c.character_maximum_length IS NOT NULL
				THEN c.data_type || '(' || c.character_maximum_length || ')'
				WHEN c.numeric_precision IS NOT NULL
				THEN c.data_type || '(' || c.numeric_precision || ',' || c.numeric_scale || ')'
				ELSE c.data_type
			END AS formatted_type,
			c.is_nullable = 'YES' AS is_nullable,
			COALESCE(c.column_default, '-') AS default_value,
			COALESCE(cc.is_pk, false) AS is_primary_key,
			COALESCE(cc.is_fk, false) AS is_foreign_key,
			COALESCE(cc.is_unique, false) AS is_unique,
			COALESCE(cc.has_check, false) AS has_check,
			COALESCE(d.description, '-') AS comment
		FROM information_schema.columns c
		LEFT JOIN column_constraints cc ON cc.column_name = c.column_name
		LEFT JOIN pg_catalog.pg_attribute a ON a.attname = c.column_name
			AND a.attrelid = ($1 || '.' || $2)::regclass
		LEFT JOIN pg_catalog.pg_description d ON d.objoid = a.attrelid
			AND d.objsubid = a.attnum
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get column details: %w", err)
	}

	var columns []models.ColumnDetail
	for _, row := range rows {
		col := models.ColumnDetail{
			Name:          toString(row["column_name"]),
			DataType:      toString(row["formatted_type"]),
			IsNullable:    toBool(row["is_nullable"]),
			DefaultValue:  toString(row["default_value"]),
			IsPrimaryKey:  toBool(row["is_primary_key"]),
			IsForeignKey:  toBool(row["is_foreign_key"]),
			IsUnique:      toBool(row["is_unique"]),
			HasCheck:      toBool(row["has_check"]),
			Comment:       toString(row["comment"]),
		}
		columns = append(columns, col)
	}

	return columns, nil
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
```

**Step 2: Run build to verify syntax**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit column details query**

```bash
git add internal/db/metadata/column_details.go
git commit -m "feat: add column details metadata query"
```

---

## Task 3: Create Constraints Metadata Query

**Files:**
- Create: `internal/db/metadata/constraints.go`

**Step 1: Create constraints query file**

```go
package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/models"
)

// GetConstraints retrieves all constraints for a table
func GetConstraints(ctx context.Context, pool *connection.Pool, schema, table string) ([]models.Constraint, error) {
	query := `
		SELECT
			con.conname AS constraint_name,
			con.contype AS constraint_type,
			pg_get_constraintdef(con.oid) AS definition,
			ARRAY(
				SELECT att.attname
				FROM unnest(con.conkey) WITH ORDINALITY AS u(attnum, attposition)
				JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid
					AND att.attnum = u.attnum
				ORDER BY u.attposition
			) AS columns,
			COALESCE(nf.nspname || '.' || clf.relname, '') AS foreign_table,
			ARRAY(
				SELECT att.attname
				FROM unnest(con.confkey) WITH ORDINALITY AS u(attnum, attposition)
				JOIN pg_catalog.pg_attribute att ON att.attrelid = con.confrelid
					AND att.attnum = u.attnum
				ORDER BY u.attposition
			) AS foreign_columns
		FROM pg_catalog.pg_constraint con
		JOIN pg_catalog.pg_class cl ON con.conrelid = cl.oid
		JOIN pg_catalog.pg_namespace ns ON cl.relnamespace = ns.oid
		LEFT JOIN pg_catalog.pg_class clf ON con.confrelid = clf.oid
		LEFT JOIN pg_catalog.pg_namespace nf ON clf.relnamespace = nf.oid
		WHERE ns.nspname = $1 AND cl.relname = $2
		ORDER BY
			CASE con.contype
				WHEN 'p' THEN 1
				WHEN 'u' THEN 2
				WHEN 'f' THEN 3
				WHEN 'c' THEN 4
			END,
			con.conname
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get constraints: %w", err)
	}

	var constraints []models.Constraint
	for _, row := range rows {
		constraint := models.Constraint{
			Name:         toString(row["constraint_name"]),
			Type:         toString(row["constraint_type"]),
			Definition:   toString(row["definition"]),
			ForeignTable: toString(row["foreign_table"]),
		}

		// Parse columns array
		if colsArray, ok := row["columns"].(pgtype.Array[string]); ok {
			constraint.Columns = colsArray.Elements
		}

		// Parse foreign columns array
		if fkArray, ok := row["foreign_columns"].(pgtype.Array[string]); ok {
			constraint.ForeignCols = fkArray.Elements
		}

		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

// FormatConstraintType returns a short type label
func FormatConstraintType(conType string) string {
	switch conType {
	case "p":
		return "PK"
	case "f":
		return "FK"
	case "u":
		return "UQ"
	case "c":
		return "CK"
	default:
		return strings.ToUpper(conType)
	}
}
```

**Step 2: Run build to verify syntax**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit constraints query**

```bash
git add internal/db/metadata/constraints.go
git commit -m "feat: add constraints metadata query"
```

---

## Task 4: Create Indexes Metadata Query

**Files:**
- Create: `internal/db/metadata/indexes.go`

**Step 1: Create indexes query file**

```go
package metadata

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/models"
)

// GetIndexes retrieves all indexes for a table
func GetIndexes(ctx context.Context, pool *connection.Pool, schema, table string) ([]models.IndexInfo, error) {
	query := `
		SELECT
			i.indexrelid::regclass::text AS index_name,
			am.amname AS index_type,
			pg_get_indexdef(i.indexrelid) AS definition,
			i.indisunique AS is_unique,
			i.indisprimary AS is_primary,
			pg_relation_size(i.indexrelid) AS size,
			ARRAY(
				SELECT a.attname
				FROM unnest(i.indkey) WITH ORDINALITY AS u(attnum, attposition)
				LEFT JOIN pg_catalog.pg_attribute a ON a.attrelid = i.indrelid
					AND a.attnum = u.attnum
				WHERE u.attnum > 0
				ORDER BY u.attposition
			) AS columns,
			pg_get_expr(i.indpred, i.indrelid) AS predicate
		FROM pg_catalog.pg_index i
		JOIN pg_catalog.pg_class c ON c.oid = i.indrelid
		JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_catalog.pg_class ic ON ic.oid = i.indexrelid
		JOIN pg_catalog.pg_am am ON am.oid = ic.relam
		WHERE n.nspname = $1 AND c.relname = $2
		ORDER BY i.indisprimary DESC, i.indisunique DESC, index_name
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}

	var indexes []models.IndexInfo
	for _, row := range rows {
		index := models.IndexInfo{
			Name:       toString(row["index_name"]),
			Type:       toString(row["index_type"]),
			Definition: toString(row["definition"]),
			IsUnique:   toBool(row["is_unique"]),
			IsPrimary:  toBool(row["is_primary"]),
			Predicate:  toString(row["predicate"]),
		}

		if size, ok := row["size"].(int64); ok {
			index.Size = size
		}

		index.IsPartial = index.Predicate != ""

		// Parse columns array
		if colsArray, ok := row["columns"].(pgtype.Array[string]); ok {
			index.Columns = colsArray.Elements
		}

		indexes = append(indexes, index)
	}

	return indexes, nil
}

// FormatSize converts bytes to human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

**Step 2: Run build to verify syntax**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit indexes query**

```bash
git add internal/db/metadata/indexes.go
git commit -m "feat: add indexes metadata query"
```

---

## Task 5: Create ColumnsView Component (Basic Structure)

**Files:**
- Create: `internal/ui/components/columns_view.go`

**Step 1: Create columns view component**

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// ColumnsView displays column information
type ColumnsView struct {
	Width   int
	Height  int
	Theme   theme.Theme
	Columns []models.ColumnDetail

	selectedRow int
	topRow      int
	visibleRows int
}

// NewColumnsView creates a new columns view
func NewColumnsView(th theme.Theme) *ColumnsView {
	return &ColumnsView{
		Theme:       th,
		Columns:     []models.ColumnDetail{},
		selectedRow: 0,
		topRow:      0,
	}
}

// SetColumns updates the columns data
func (cv *ColumnsView) SetColumns(columns []models.ColumnDetail) {
	cv.Columns = columns
	cv.selectedRow = 0
	cv.topRow = 0
}

// View renders the columns view
func (cv *ColumnsView) View() string {
	if len(cv.Columns) == 0 {
		return lipgloss.NewStyle().
			Foreground(cv.Theme.Metadata).
			Render("No columns to display")
	}

	var b strings.Builder

	// Render header
	b.WriteString(cv.renderHeader())
	b.WriteString("\n")
	b.WriteString(cv.renderSeparator())
	b.WriteString("\n")

	// Calculate visible rows
	cv.visibleRows = cv.Height - 3 // Header + separator + status
	if cv.visibleRows < 1 {
		cv.visibleRows = 1
	}

	// Render visible rows
	endRow := cv.topRow + cv.visibleRows
	if endRow > len(cv.Columns) {
		endRow = len(cv.Columns)
	}

	for i := cv.topRow; i < endRow; i++ {
		isSelected := i == cv.selectedRow
		b.WriteString(cv.renderRow(cv.Columns[i], isSelected))
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}

	// Status line
	b.WriteString("\n")
	b.WriteString(cv.renderStatus())

	return lipgloss.NewStyle().
		Width(cv.Width).
		Height(cv.Height).
		Render(b.String())
}

func (cv *ColumnsView) renderHeader() string {
	headers := []string{"Name", "Type", "Nullable", "Default", "Constraints", "Comment"}
	widths := []int{20, 20, 10, 20, 15, 30}

	parts := make([]string, len(headers))
	for i, header := range headers {
		truncated := runewidth.Truncate(header, widths[i], "…")
		parts[i] = lipgloss.NewStyle().
			Width(widths[i]).
			Bold(true).
			Foreground(cv.Theme.TableHeader).
			Background(cv.Theme.Selection).
			Render(truncated)
	}

	separatorStyle := lipgloss.NewStyle().
		Foreground(cv.Theme.Border).
		Background(cv.Theme.Selection)

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		parts[0], separatorStyle.Render(" │ "),
		parts[1], separatorStyle.Render(" │ "),
		parts[2], separatorStyle.Render(" │ "),
		parts[3], separatorStyle.Render(" │ "),
		parts[4], separatorStyle.Render(" │ "),
		parts[5],
	)

	return row
}

func (cv *ColumnsView) renderSeparator() string {
	// Total width calculation: 20+20+10+20+15+30 + 5*3 (separators) = 130
	totalWidth := 130
	return lipgloss.NewStyle().
		Foreground(cv.Theme.Border).
		Render(strings.Repeat("─", totalWidth))
}

func (cv *ColumnsView) renderRow(col models.ColumnDetail, selected bool) string {
	widths := []int{20, 20, 10, 20, 15, 30}

	// Format constraint markers
	constraints := cv.formatConstraints(col)

	// Prepare cell values
	cells := []string{
		col.Name,
		col.DataType,
		cv.formatNullable(col.IsNullable),
		col.DefaultValue,
		constraints,
		col.Comment,
	}

	parts := make([]string, len(cells))
	for i, cell := range cells {
		truncated := runewidth.Truncate(cell, widths[i], "…")

		var cellStyle lipgloss.Style
		if selected {
			cellStyle = lipgloss.NewStyle().
				Background(cv.Theme.Selection).
				Foreground(cv.Theme.Foreground)
		} else {
			cellStyle = lipgloss.NewStyle()
		}

		parts[i] = cellStyle.Render(
			lipgloss.NewStyle().Width(widths[i]).Render(truncated),
		)
	}

	separatorStyle := lipgloss.NewStyle().Foreground(cv.Theme.Border)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		parts[0], separatorStyle.Render(" │ "),
		parts[1], separatorStyle.Render(" │ "),
		parts[2], separatorStyle.Render(" │ "),
		parts[3], separatorStyle.Render(" │ "),
		parts[4], separatorStyle.Render(" │ "),
		parts[5],
	)
}

func (cv *ColumnsView) formatConstraints(col models.ColumnDetail) string {
	markers := []string{}
	if col.IsPrimaryKey {
		markers = append(markers, "🔑 PK")
	}
	if col.IsForeignKey {
		markers = append(markers, "🔗 FK")
	}
	if col.IsUnique {
		markers = append(markers, "✓ UQ")
	}
	if col.HasCheck {
		markers = append(markers, "⚠️ CK")
	}
	if len(markers) == 0 {
		return "-"
	}
	return strings.Join(markers, ", ")
}

func (cv *ColumnsView) formatNullable(nullable bool) string {
	if nullable {
		return "YES"
	}
	return "NO"
}

func (cv *ColumnsView) renderStatus() string {
	showing := fmt.Sprintf(" 󰠵 %d columns", len(cv.Columns))
	return lipgloss.NewStyle().
		Foreground(cv.Theme.Metadata).
		Italic(true).
		Render(showing)
}

// MoveSelection moves the selected row up/down
func (cv *ColumnsView) MoveSelection(delta int) {
	cv.selectedRow += delta

	if cv.selectedRow < 0 {
		cv.selectedRow = 0
	}
	if cv.selectedRow >= len(cv.Columns) {
		cv.selectedRow = len(cv.Columns) - 1
	}

	// Adjust scroll
	if cv.selectedRow < cv.topRow {
		cv.topRow = cv.selectedRow
	}
	if cv.selectedRow >= cv.topRow+cv.visibleRows {
		cv.topRow = cv.selectedRow - cv.visibleRows + 1
	}
}

// GetSelectedColumn returns the currently selected column
func (cv *ColumnsView) GetSelectedColumn() *models.ColumnDetail {
	if cv.selectedRow < 0 || cv.selectedRow >= len(cv.Columns) {
		return nil
	}
	return &cv.Columns[cv.selectedRow]
}
```

**Step 2: Run build to verify syntax**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit columns view**

```bash
git add internal/ui/components/columns_view.go
git commit -m "feat: add columns view component"
```

---

## Task 6: Create ConstraintsView Component

**Files:**
- Create: `internal/ui/components/constraints_view.go`

**Step 1: Create constraints view component**

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// ConstraintsView displays constraint information
type ConstraintsView struct {
	Width       int
	Height      int
	Theme       theme.Theme
	Constraints []models.Constraint

	selectedRow int
	topRow      int
	visibleRows int
}

// NewConstraintsView creates a new constraints view
func NewConstraintsView(th theme.Theme) *ConstraintsView {
	return &ConstraintsView{
		Theme:       th,
		Constraints: []models.Constraint{},
		selectedRow: 0,
		topRow:      0,
	}
}

// SetConstraints updates the constraints data
func (cv *ConstraintsView) SetConstraints(constraints []models.Constraint) {
	cv.Constraints = constraints
	cv.selectedRow = 0
	cv.topRow = 0
}

// View renders the constraints view
func (cv *ConstraintsView) View() string {
	if len(cv.Constraints) == 0 {
		return lipgloss.NewStyle().
			Foreground(cv.Theme.Metadata).
			Render("No constraints to display")
	}

	var b strings.Builder

	b.WriteString(cv.renderHeader())
	b.WriteString("\n")
	b.WriteString(cv.renderSeparator())
	b.WriteString("\n")

	cv.visibleRows = cv.Height - 3
	if cv.visibleRows < 1 {
		cv.visibleRows = 1
	}

	endRow := cv.topRow + cv.visibleRows
	if endRow > len(cv.Constraints) {
		endRow = len(cv.Constraints)
	}

	for i := cv.topRow; i < endRow; i++ {
		isSelected := i == cv.selectedRow
		b.WriteString(cv.renderRow(cv.Constraints[i], isSelected))
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(cv.renderStatus())

	return lipgloss.NewStyle().
		Width(cv.Width).
		Height(cv.Height).
		Render(b.String())
}

func (cv *ConstraintsView) renderHeader() string {
	headers := []string{"Type", "Name", "Columns", "Definition", "Description"}
	widths := []int{6, 25, 20, 45, 30}

	parts := make([]string, len(headers))
	for i, header := range headers {
		truncated := runewidth.Truncate(header, widths[i], "…")
		parts[i] = lipgloss.NewStyle().
			Width(widths[i]).
			Bold(true).
			Foreground(cv.Theme.TableHeader).
			Background(cv.Theme.Selection).
			Render(truncated)
	}

	separatorStyle := lipgloss.NewStyle().
		Foreground(cv.Theme.Border).
		Background(cv.Theme.Selection)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		parts[0], separatorStyle.Render(" │ "),
		parts[1], separatorStyle.Render(" │ "),
		parts[2], separatorStyle.Render(" │ "),
		parts[3], separatorStyle.Render(" │ "),
		parts[4],
	)
}

func (cv *ConstraintsView) renderSeparator() string {
	totalWidth := 6 + 25 + 20 + 45 + 30 + 4*3 // widths + separators
	return lipgloss.NewStyle().
		Foreground(cv.Theme.Border).
		Render(strings.Repeat("─", totalWidth))
}

func (cv *ConstraintsView) renderRow(con models.Constraint, selected bool) string {
	widths := []int{6, 25, 20, 45, 30}

	// Format type with color
	typeLabel := metadata.FormatConstraintType(con.Type)
	typeColor := cv.getTypeColor(con.Type)
	typeCell := lipgloss.NewStyle().
		Foreground(typeColor).
		Bold(true).
		Render(typeLabel)

	// Format columns
	columnsStr := strings.Join(con.Columns, ", ")

	// Format definition
	definition := cv.formatDefinition(con)

	// Format description
	description := cv.formatDescription(con)

	cells := []string{
		typeCell,
		con.Name,
		columnsStr,
		definition,
		description,
	}

	parts := make([]string, len(cells))
	for i, cell := range cells {
		truncated := runewidth.Truncate(cell, widths[i], "…")

		var cellStyle lipgloss.Style
		if selected {
			cellStyle = lipgloss.NewStyle().
				Background(cv.Theme.Selection).
				Foreground(cv.Theme.Foreground)
		} else {
			cellStyle = lipgloss.NewStyle()
		}

		parts[i] = cellStyle.Render(
			lipgloss.NewStyle().Width(widths[i]).Render(truncated),
		)
	}

	separatorStyle := lipgloss.NewStyle().Foreground(cv.Theme.Border)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		parts[0], separatorStyle.Render(" │ "),
		parts[1], separatorStyle.Render(" │ "),
		parts[2], separatorStyle.Render(" │ "),
		parts[3], separatorStyle.Render(" │ "),
		parts[4],
	)
}

func (cv *ConstraintsView) getTypeColor(conType string) lipgloss.Color {
	switch conType {
	case "p":
		return cv.Theme.Info // Blue
	case "f":
		return cv.Theme.Warning // Orange
	case "u":
		return cv.Theme.Success // Green
	case "c":
		return cv.Theme.Metadata // Gray
	default:
		return cv.Theme.Foreground
	}
}

func (cv *ConstraintsView) formatDefinition(con models.Constraint) string {
	if con.Type == "f" && con.ForeignTable != "" {
		// Format as: → table(columns)
		fkCols := strings.Join(con.ForeignCols, ", ")
		return fmt.Sprintf("→ %s(%s)", con.ForeignTable, fkCols)
	}
	return con.Definition
}

func (cv *ConstraintsView) formatDescription(con models.Constraint) string {
	switch con.Type {
	case "p":
		return "Primary key constraint"
	case "f":
		return fmt.Sprintf("References %s", con.ForeignTable)
	case "u":
		return "Unique constraint"
	case "c":
		return "Check constraint"
	default:
		return "-"
	}
}

func (cv *ConstraintsView) renderStatus() string {
	showing := fmt.Sprintf(" 󰌆 %d constraints", len(cv.Constraints))
	return lipgloss.NewStyle().
		Foreground(cv.Theme.Metadata).
		Italic(true).
		Render(showing)
}

// MoveSelection moves the selected row up/down
func (cv *ConstraintsView) MoveSelection(delta int) {
	cv.selectedRow += delta

	if cv.selectedRow < 0 {
		cv.selectedRow = 0
	}
	if cv.selectedRow >= len(cv.Constraints) {
		cv.selectedRow = len(cv.Constraints) - 1
	}

	if cv.selectedRow < cv.topRow {
		cv.topRow = cv.selectedRow
	}
	if cv.selectedRow >= cv.topRow+cv.visibleRows {
		cv.topRow = cv.selectedRow - cv.visibleRows + 1
	}
}

// GetSelectedConstraint returns the currently selected constraint
func (cv *ConstraintsView) GetSelectedConstraint() *models.Constraint {
	if cv.selectedRow < 0 || cv.selectedRow >= len(cv.Constraints) {
		return nil
	}
	return &cv.Constraints[cv.selectedRow]
}
```

**Step 2: Run build to verify**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit constraints view**

```bash
git add internal/ui/components/constraints_view.go
git commit -m "feat: add constraints view component"
```

---

## Task 7: Create IndexesView Component

**Files:**
- Create: `internal/ui/components/indexes_view.go`

**Step 1: Create indexes view component**

```go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// IndexesView displays index information
type IndexesView struct {
	Width   int
	Height  int
	Theme   theme.Theme
	Indexes []models.IndexInfo

	selectedRow int
	topRow      int
	visibleRows int
}

// NewIndexesView creates a new indexes view
func NewIndexesView(th theme.Theme) *IndexesView {
	return &IndexesView{
		Theme:       th,
		Indexes:     []models.IndexInfo{},
		selectedRow: 0,
		topRow:      0,
	}
}

// SetIndexes updates the indexes data
func (iv *IndexesView) SetIndexes(indexes []models.IndexInfo) {
	iv.Indexes = indexes
	iv.selectedRow = 0
	iv.topRow = 0
}

// View renders the indexes view
func (iv *IndexesView) View() string {
	if len(iv.Indexes) == 0 {
		return lipgloss.NewStyle().
			Foreground(iv.Theme.Metadata).
			Render("No indexes to display")
	}

	var b strings.Builder

	b.WriteString(iv.renderHeader())
	b.WriteString("\n")
	b.WriteString(iv.renderSeparator())
	b.WriteString("\n")

	iv.visibleRows = iv.Height - 3
	if iv.visibleRows < 1 {
		iv.visibleRows = 1
	}

	endRow := iv.topRow + iv.visibleRows
	if endRow > len(iv.Indexes) {
		endRow = len(iv.Indexes)
	}

	for i := iv.topRow; i < endRow; i++ {
		isSelected := i == iv.selectedRow
		b.WriteString(iv.renderRow(iv.Indexes[i], isSelected))
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(iv.renderStatus())

	return lipgloss.NewStyle().
		Width(iv.Width).
		Height(iv.Height).
		Render(b.String())
}

func (iv *IndexesView) renderHeader() string {
	headers := []string{"Name", "Type", "Columns", "Properties", "Size", "Definition"}
	widths := []int{25, 10, 20, 20, 10, 40}

	parts := make([]string, len(headers))
	for i, header := range headers {
		truncated := runewidth.Truncate(header, widths[i], "…")
		parts[i] = lipgloss.NewStyle().
			Width(widths[i]).
			Bold(true).
			Foreground(iv.Theme.TableHeader).
			Background(iv.Theme.Selection).
			Render(truncated)
	}

	separatorStyle := lipgloss.NewStyle().
		Foreground(iv.Theme.Border).
		Background(iv.Theme.Selection)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		parts[0], separatorStyle.Render(" │ "),
		parts[1], separatorStyle.Render(" │ "),
		parts[2], separatorStyle.Render(" │ "),
		parts[3], separatorStyle.Render(" │ "),
		parts[4], separatorStyle.Render(" │ "),
		parts[5],
	)
}

func (iv *IndexesView) renderSeparator() string {
	totalWidth := 25 + 10 + 20 + 20 + 10 + 40 + 5*3 // widths + separators
	return lipgloss.NewStyle().
		Foreground(iv.Theme.Border).
		Render(strings.Repeat("─", totalWidth))
}

func (iv *IndexesView) renderRow(idx models.IndexInfo, selected bool) string {
	widths := []int{25, 10, 20, 20, 10, 40}

	// Format columns
	columnsStr := strings.Join(idx.Columns, ", ")

	// Format properties
	properties := iv.formatProperties(idx)

	// Format size
	sizeStr := metadata.FormatSize(idx.Size)

	// Format definition
	definition := idx.Definition

	cells := []string{
		idx.Name,
		idx.Type,
		columnsStr,
		properties,
		sizeStr,
		definition,
	}

	parts := make([]string, len(cells))
	for i, cell := range cells {
		truncated := runewidth.Truncate(cell, widths[i], "…")

		var cellStyle lipgloss.Style
		if selected {
			cellStyle = lipgloss.NewStyle().
				Background(iv.Theme.Selection).
				Foreground(iv.Theme.Foreground)
		} else {
			cellStyle = lipgloss.NewStyle()
		}

		parts[i] = cellStyle.Render(
			lipgloss.NewStyle().Width(widths[i]).Render(truncated),
		)
	}

	separatorStyle := lipgloss.NewStyle().Foreground(iv.Theme.Border)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		parts[0], separatorStyle.Render(" │ "),
		parts[1], separatorStyle.Render(" │ "),
		parts[2], separatorStyle.Render(" │ "),
		parts[3], separatorStyle.Render(" │ "),
		parts[4], separatorStyle.Render(" │ "),
		parts[5],
	)
}

func (iv *IndexesView) formatProperties(idx models.IndexInfo) string {
	props := []string{}
	if idx.IsPrimary {
		props = append(props, "🔑 PK")
	}
	if idx.IsUnique {
		props = append(props, "✓ UQ")
	}
	if idx.IsPartial {
		props = append(props, "📋 Partial")
	}
	if len(props) == 0 {
		return "-"
	}
	return strings.Join(props, ", ")
}

func (iv *IndexesView) renderStatus() string {
	showing := fmt.Sprintf(" 󰘚 %d indexes", len(iv.Indexes))
	return lipgloss.NewStyle().
		Foreground(iv.Theme.Metadata).
		Italic(true).
		Render(showing)
}

// MoveSelection moves the selected row up/down
func (iv *IndexesView) MoveSelection(delta int) {
	iv.selectedRow += delta

	if iv.selectedRow < 0 {
		iv.selectedRow = 0
	}
	if iv.selectedRow >= len(iv.Indexes) {
		iv.selectedRow = len(iv.Indexes) - 1
	}

	if iv.selectedRow < iv.topRow {
		iv.topRow = iv.selectedRow
	}
	if iv.selectedRow >= iv.topRow+iv.visibleRows {
		iv.topRow = iv.selectedRow - iv.visibleRows + 1
	}
}

// GetSelectedIndex returns the currently selected index
func (iv *IndexesView) GetSelectedIndex() *models.IndexInfo {
	if iv.selectedRow < 0 || iv.selectedRow >= len(iv.Indexes) {
		return nil
	}
	return &iv.Indexes[iv.selectedRow]
}
```

**Step 2: Run build to verify**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit indexes view**

```bash
git add internal/ui/components/indexes_view.go
git commit -m "feat: add indexes view component"
```

---

## Task 8: Create StructureView Container

**Files:**
- Create: `internal/ui/components/structure_view.go`

**Step 1: Create structure view container**

```go
package components

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// StructureView is a tabbed container for viewing table structure
type StructureView struct {
	Width  int
	Height int
	Theme  theme.Theme

	// Current active tab (0=Data, 1=Columns, 2=Constraints, 3=Indexes)
	activeTab int

	// Tab views
	columnsView     *ColumnsView
	constraintsView *ConstraintsView
	indexesView     *IndexesView

	// Table info
	schema string
	table  string
	pool   *connection.Pool

	// Status
	loading      bool
	errorMessage string
}

// NewStructureView creates a new structure view
func NewStructureView(th theme.Theme) *StructureView {
	return &StructureView{
		Theme:           th,
		activeTab:       1, // Start with Columns tab
		columnsView:     NewColumnsView(th),
		constraintsView: NewConstraintsView(th),
		indexesView:     NewIndexesView(th),
	}
}

// SetTable sets the current table and loads structure data
func (sv *StructureView) SetTable(ctx context.Context, pool *connection.Pool, schema, table string) error {
	sv.schema = schema
	sv.table = table
	sv.pool = pool
	sv.loading = true
	sv.errorMessage = ""

	// Load columns
	columns, err := metadata.GetColumnDetails(ctx, pool, schema, table)
	if err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to load columns: %v", err)
		sv.loading = false
		return err
	}
	sv.columnsView.SetColumns(columns)

	// Load constraints
	constraints, err := metadata.GetConstraints(ctx, pool, schema, table)
	if err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to load constraints: %v", err)
		sv.loading = false
		return err
	}
	sv.constraintsView.SetConstraints(constraints)

	// Load indexes
	indexes, err := metadata.GetIndexes(ctx, pool, schema, table)
	if err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to load indexes: %v", err)
		sv.loading = false
		return err
	}
	sv.indexesView.SetIndexes(indexes)

	sv.loading = false
	return nil
}

// SwitchTab switches to a specific tab
func (sv *StructureView) SwitchTab(tabIndex int) {
	if tabIndex >= 0 && tabIndex <= 3 {
		sv.activeTab = tabIndex
	}
}

// Update handles keyboard input
func (sv *StructureView) Update(msg tea.KeyMsg) {
	if sv.activeTab == 0 {
		// Data tab - handled by app.go with existing table view
		return
	}

	// Handle navigation keys for structure tabs
	switch msg.String() {
	case "up", "k":
		sv.getCurrentView().MoveSelection(-1)
	case "down", "j":
		sv.getCurrentView().MoveSelection(1)
	case "left", "h":
		sv.SwitchTab(sv.activeTab - 1)
	case "right", "l":
		sv.SwitchTab(sv.activeTab + 1)
	}
}

type structureViewNavigator interface {
	MoveSelection(delta int)
}

func (sv *StructureView) getCurrentView() structureViewNavigator {
	switch sv.activeTab {
	case 1:
		return sv.columnsView
	case 2:
		return sv.constraintsView
	case 3:
		return sv.indexesView
	default:
		return sv.columnsView
	}
}

// View renders the structure view
func (sv *StructureView) View() string {
	if sv.loading {
		return lipgloss.NewStyle().
			Foreground(sv.Theme.Metadata).
			Render("Loading structure...")
	}

	if sv.errorMessage != "" {
		return lipgloss.NewStyle().
			Foreground(sv.Theme.Error).
			Render(sv.errorMessage)
	}

	var b strings.Builder

	// Render tab bar
	b.WriteString(sv.renderTabBar())
	b.WriteString("\n")

	// Calculate content height (subtract tab bar)
	contentHeight := sv.Height - 1

	// Update view dimensions
	sv.columnsView.Width = sv.Width
	sv.columnsView.Height = contentHeight
	sv.constraintsView.Width = sv.Width
	sv.constraintsView.Height = contentHeight
	sv.indexesView.Width = sv.Width
	sv.indexesView.Height = contentHeight

	// Render active tab content
	switch sv.activeTab {
	case 1:
		b.WriteString(sv.columnsView.View())
	case 2:
		b.WriteString(sv.constraintsView.View())
	case 3:
		b.WriteString(sv.indexesView.View())
	default:
		b.WriteString("Data view handled separately")
	}

	return b.String()
}

func (sv *StructureView) renderTabBar() string {
	tabs := []struct {
		index int
		label string
	}{
		{0, "Data"},
		{1, "Columns"},
		{2, "Constraints"},
		{3, "Indexes"},
	}

	tabParts := make([]string, len(tabs))
	for i, tab := range tabs {
		var style lipgloss.Style
		if tab.index == sv.activeTab {
			// Active tab
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(sv.Theme.BorderFocused).
				Background(sv.Theme.Selection).
				Padding(0, 2)
		} else {
			// Inactive tab
			style = lipgloss.NewStyle().
				Foreground(sv.Theme.Metadata).
				Padding(0, 2)
		}
		tabParts[i] = style.Render(tab.label)
	}

	separator := lipgloss.NewStyle().
		Foreground(sv.Theme.Border).
		Render(" │ ")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		tabParts[0], separator,
		tabParts[1], separator,
		tabParts[2], separator,
		tabParts[3],
	)
}
```

**Step 2: Run build to verify**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit structure view container**

```bash
git add internal/ui/components/structure_view.go
git commit -m "feat: add structure view tabbed container"
```

---

## Task 9: Integrate StructureView into App

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add structure view field to App struct**

In `app.go`, find the App struct and add after the `jsonbViewer` field:

```go
// Structure view
showStructureView bool
structureView     *components.StructureView
currentTab        int // 0=Data, 1=Columns, 2=Constraints, 3=Indexes
```

**Step 2: Initialize structure view in New function**

In the `New()` function, after creating other components, add:

```go
structureView := components.NewStructureView(th)
```

And add to the returned App:

```go
structureView: structureView,
currentTab:    0,
```

**Step 3: Add keyboard shortcuts in Update method**

In the `Update()` method, find the keyboard handling section and add:

```go
// Tab switching (Ctrl+1/2/3/4)
case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+1"))):
	a.currentTab = 0
	a.structureView.SwitchTab(0)
	return a, nil

case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+2"))):
	a.currentTab = 1
	a.structureView.SwitchTab(1)
	return a, nil

case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+3"))):
	a.currentTab = 2
	a.structureView.SwitchTab(2)
	return a, nil

case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+4"))):
	a.currentTab = 3
	a.structureView.SwitchTab(3)
	return a, nil
```

**Step 4: Route keyboard events to structure view**

In the keyboard handling section, when in right panel and table is selected, add:

```go
// If structure view is active and not on Data tab, route to structure view
if a.currentTab > 0 {
	a.structureView.Update(msg)
	return a, nil
}
```

**Step 5: Run build**

```bash
go build ./...
```

Expected: No errors

**Step 6: Commit app integration**

```bash
git add internal/app/app.go
git commit -m "feat: integrate structure view into app with keyboard shortcuts"
```

---

## Task 10: Update Right Panel Rendering

**Files:**
- Modify: `internal/app/app.go` (View method)

**Step 1: Modify renderRightPanel to show structure view**

Find the `renderRightPanel()` method and update it to render structure view when a table is selected:

```go
func (a *App) renderRightPanel(width, height int) string {
	// ... existing checks for overlays ...

	// If table is selected, show structure view
	if a.currentTable != "" {
		// Update structure view dimensions
		a.structureView.Width = width
		a.structureView.Height = height

		// Load table structure if needed
		if a.structureView.table != a.currentTable && a.connectionManager.CurrentPool() != nil {
			parts := strings.Split(a.currentTable, ".")
			if len(parts) == 2 {
				ctx := context.Background()
				err := a.structureView.SetTable(ctx, a.connectionManager.CurrentPool(), parts[0], parts[1])
				if err != nil {
					log.Printf("Failed to load structure: %v", err)
				}
			}
		}

		// If Data tab is active, show existing table view
		if a.currentTab == 0 {
			// Existing table view rendering
			a.tableView.Width = width
			a.tableView.Height = height
			return a.tableView.View()
		}

		// Otherwise show structure view
		return a.structureView.View()
	}

	// ... existing fallback rendering ...
}
```

**Step 2: Run build**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit rendering updates**

```bash
git add internal/app/app.go
git commit -m "feat: render structure view in right panel based on active tab"
```

---

## Task 11: Manual Testing

**Step 1: Build and run the application**

```bash
make build
./bin/lazypg
```

**Step 2: Test tab switching**

1. Connect to a PostgreSQL database
2. Select a table in the left tree
3. Press `Ctrl+2` - should show Columns tab
4. Press `Ctrl+3` - should show Constraints tab
5. Press `Ctrl+4` - should show Indexes tab
6. Press `Ctrl+1` - should show Data tab (existing table view)

**Step 3: Test navigation within tabs**

1. Switch to Columns tab (`Ctrl+2`)
2. Press `↑` and `↓` to navigate rows
3. Verify row selection highlights correctly
4. Repeat for Constraints and Indexes tabs

**Step 4: Verify data display**

1. Check that column types are formatted correctly
2. Check that constraint markers (🔑 PK, 🔗 FK, etc.) appear
3. Check that index properties are displayed
4. Verify all emoji icons render correctly

**Expected Results:**
- Tab switching works smoothly
- Column/constraint/index data displays correctly
- Navigation within tabs works
- Visual styling is consistent with existing UI

**Step 5: Document any issues**

If there are issues:
- Note specific error messages
- Check logs for query errors
- Verify PostgreSQL permissions

---

## Task 12: Add Copy Functionality (y/Y keys)

**Files:**
- Modify: `internal/ui/components/structure_view.go`
- Modify: `internal/app/app.go`

**Step 1: Add clipboard import to structure_view.go**

At top of file:

```go
import (
	// ... existing imports ...
	"github.com/atotto/clipboard"
)
```

**Step 2: Add copy methods to StructureView**

Add these methods to `structure_view.go`:

```go
// CopyCurrentName copies the name of the selected item
func (sv *StructureView) CopyCurrentName() string {
	var name string
	switch sv.activeTab {
	case 1:
		if col := sv.columnsView.GetSelectedColumn(); col != nil {
			name = col.Name
		}
	case 2:
		if con := sv.constraintsView.GetSelectedConstraint(); con != nil {
			name = con.Name
		}
	case 3:
		if idx := sv.indexesView.GetSelectedIndex(); idx != nil {
			name = idx.Name
		}
	}

	if name != "" {
		clipboard.WriteAll(name)
		return fmt.Sprintf("✓ Copied: %s", name)
	}
	return ""
}

// CopyCurrentDefinition copies the full definition of the selected item
func (sv *StructureView) CopyCurrentDefinition() string {
	var definition string
	switch sv.activeTab {
	case 1:
		if col := sv.columnsView.GetSelectedColumn(); col != nil {
			definition = fmt.Sprintf("%s %s %s DEFAULT %s",
				col.Name, col.DataType,
				map[bool]string{true: "NULL", false: "NOT NULL"}[col.IsNullable],
				col.DefaultValue)
		}
	case 2:
		if con := sv.constraintsView.GetSelectedConstraint(); con != nil {
			definition = con.Definition
		}
	case 3:
		if idx := sv.indexesView.GetSelectedIndex(); idx != nil {
			definition = idx.Definition
		}
	}

	if definition != "" {
		clipboard.WriteAll(definition)
		preview := definition
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		return fmt.Sprintf("✓ Copied: %s", preview)
	}
	return ""
}
```

**Step 3: Handle y/Y keys in app.go**

In the `Update()` method keyboard handling section, add:

```go
// Copy functionality in structure view
case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
	if a.currentTab > 0 {
		statusMsg := a.structureView.CopyCurrentName()
		if statusMsg != "" {
			// Show status message (you may want to add a status bar)
			log.Println(statusMsg)
		}
		return a, nil
	}

case key.Matches(msg, key.NewBinding(key.WithKeys("Y"))):
	if a.currentTab > 0 {
		statusMsg := a.structureView.CopyCurrentDefinition()
		if statusMsg != "" {
			log.Println(statusMsg)
		}
		return a, nil
	}
```

**Step 4: Run build**

```bash
go build ./...
```

Expected: No errors

**Step 5: Test copy functionality**

1. Switch to Columns tab
2. Press `y` - should copy column name
3. Press `Y` - should copy column definition
4. Paste in another application to verify
5. Test in Constraints and Indexes tabs

**Step 6: Commit copy functionality**

```bash
git add internal/ui/components/structure_view.go internal/app/app.go
git commit -m "feat: add copy functionality (y/Y) for structure view"
```

---

## Task 13: Update Help Documentation

**Files:**
- Modify: `internal/ui/help/help.go`

**Step 1: Add structure view keybindings**

Find the help text and add a new section:

```go
### Structure View
Ctrl+1/2/3/4   Switch tabs (Data/Columns/Constraints/Indexes)
↑↓ or j/k      Navigate rows
←→ or h/l      Switch tabs
y              Copy name
Y              Copy definition
```

**Step 2: Run build**

```bash
go build ./...
```

Expected: No errors

**Step 3: Commit help updates**

```bash
git add internal/ui/help/help.go
git commit -m "docs: add structure view keybindings to help"
```

---

## Task 14: Final Testing and Polish

**Step 1: Run comprehensive manual tests**

Test matrix:
- [ ] Tab switching with Ctrl+1/2/3/4
- [ ] Tab switching with arrow keys
- [ ] Navigation within each tab
- [ ] Copy with y (names)
- [ ] Copy with Y (definitions)
- [ ] Tables with no constraints
- [ ] Tables with no indexes
- [ ] Tables with complex foreign keys
- [ ] Help overlay shows structure view keys

**Step 2: Test with different table types**

```sql
-- Create test tables
CREATE TABLE test_simple (id serial primary key, name text);
CREATE TABLE test_complex (
    id serial primary key,
    name varchar(100) not null,
    email varchar(255) unique,
    status varchar(20) check (status in ('active', 'inactive')),
    created_at timestamp default now()
);
CREATE INDEX idx_test_status ON test_complex(status) WHERE status = 'active';
```

**Step 3: Check edge cases**

- Empty tables
- Tables with many columns (>20)
- Tables with long constraint names
- Very large indexes
- Unicode in comments

**Step 4: Final build and verification**

```bash
make build
./bin/lazypg
```

Verify all features work correctly.

**Step 5: Create final commit if any fixes needed**

```bash
git add .
git commit -m "fix: address edge cases and polish structure view"
```

---

## Success Criteria

- ✅ Tab switching works with Ctrl+1/2/3/4
- ✅ Columns tab shows: name, type, nullable, default, constraints, comments
- ✅ Constraints tab shows all constraint types with proper formatting
- ✅ Indexes tab shows properties and size information
- ✅ Navigation works within all tabs
- ✅ Copy functionality works (y/Y keys)
- ✅ Help documentation updated
- ✅ Visual styling consistent with existing UI
- ✅ No crashes or query errors

---

## Notes

- Use `@superpowers:test-driven-development` for any new features
- Use `@superpowers:systematic-debugging` if issues arise
- Follow existing patterns in table_view.go for consistency
- Reference CLAUDE.md for lipgloss width calculations
- Keep commits small and focused
