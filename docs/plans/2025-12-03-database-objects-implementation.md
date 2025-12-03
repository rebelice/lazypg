# Database Objects Tree Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend the left sidebar tree to display all major PostgreSQL database objects (Views, Materialized Views, Functions, Procedures, Trigger Functions, Sequences, Indexes, Triggers, Types, Extensions).

**Architecture:** Add new TreeNodeTypes to models, create metadata query functions for each object type, update tree building logic in app.go, and add icons/handlers in tree_view.go. Objects are grouped by type under schemas, with Indexes/Triggers as children of their parent Tables.

**Tech Stack:** Go, Bubble Tea, Lipgloss, pgx (PostgreSQL driver)

---

## Phase 1: Add TreeNodeTypes

### Task 1.1: Add New TreeNodeType Constants

**Files:**
- Modify: `internal/models/tree.go:11-20`

**Step 1: Add new node type constants**

Add these constants after line 19 (after `TreeNodeTypeColumn`):

```go
const (
	TreeNodeTypeRoot       TreeNodeType = "root"
	TreeNodeTypeDatabase   TreeNodeType = "database"
	TreeNodeTypeSchema     TreeNodeType = "schema"
	TreeNodeTypeTableGroup TreeNodeType = "table_group"
	TreeNodeTypeViewGroup  TreeNodeType = "view_group"
	TreeNodeTypeTable      TreeNodeType = "table"
	TreeNodeTypeView       TreeNodeType = "view"
	TreeNodeTypeColumn     TreeNodeType = "column"

	// New group types
	TreeNodeTypeMaterializedViewGroup TreeNodeType = "materialized_view_group"
	TreeNodeTypeFunctionGroup         TreeNodeType = "function_group"
	TreeNodeTypeProcedureGroup        TreeNodeType = "procedure_group"
	TreeNodeTypeTriggerFunctionGroup  TreeNodeType = "trigger_function_group"
	TreeNodeTypeSequenceGroup         TreeNodeType = "sequence_group"
	TreeNodeTypeTypeGroup             TreeNodeType = "type_group"
	TreeNodeTypeExtensionGroup        TreeNodeType = "extension_group"
	TreeNodeTypeIndexGroup            TreeNodeType = "index_group"
	TreeNodeTypeTriggerGroup          TreeNodeType = "trigger_group"

	// Type subcategory groups
	TreeNodeTypeCompositeTypeGroup TreeNodeType = "composite_type_group"
	TreeNodeTypeEnumTypeGroup      TreeNodeType = "enum_type_group"
	TreeNodeTypeDomainTypeGroup    TreeNodeType = "domain_type_group"
	TreeNodeTypeRangeTypeGroup     TreeNodeType = "range_type_group"

	// New leaf node types
	TreeNodeTypeMaterializedView TreeNodeType = "materialized_view"
	TreeNodeTypeFunction         TreeNodeType = "function"
	TreeNodeTypeProcedure        TreeNodeType = "procedure"
	TreeNodeTypeTriggerFunction  TreeNodeType = "trigger_function"
	TreeNodeTypeSequence         TreeNodeType = "sequence"
	TreeNodeTypeIndex            TreeNodeType = "index"
	TreeNodeTypeTrigger          TreeNodeType = "trigger"
	TreeNodeTypeExtension        TreeNodeType = "extension"
	TreeNodeTypeCompositeType    TreeNodeType = "composite_type"
	TreeNodeTypeEnumType         TreeNodeType = "enum_type"
	TreeNodeTypeDomainType       TreeNodeType = "domain_type"
	TreeNodeTypeRangeType        TreeNodeType = "range_type"
)
```

**Step 2: Update Toggle() to handle new leaf types**

Modify `internal/models/tree.go` Toggle() function (around line 61):

```go
func (n *TreeNode) Toggle() {
	// Leaf nodes that can't be expanded
	switch n.Type {
	case TreeNodeTypeColumn,
		TreeNodeTypeFunction,
		TreeNodeTypeProcedure,
		TreeNodeTypeTriggerFunction,
		TreeNodeTypeSequence,
		TreeNodeTypeIndex,
		TreeNodeTypeTrigger,
		TreeNodeTypeExtension,
		TreeNodeTypeCompositeType,
		TreeNodeTypeEnumType,
		TreeNodeTypeDomainType,
		TreeNodeTypeRangeType:
		return
	}

	if len(n.Children) > 0 || !n.Loaded {
		n.Expanded = !n.Expanded
	}
}
```

**Step 3: Verify changes compile**

Run: `go build ./internal/models/...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/models/tree.go
git commit -m "feat(models): add TreeNodeTypes for all PostgreSQL objects"
```

---

## Phase 2: Add Metadata Structs and Query Functions

### Task 2.1: Create Objects Metadata File

**Files:**
- Create: `internal/db/metadata/objects.go`

**Step 1: Create the file with struct definitions and queries**

```go
package metadata

import (
	"context"
	"fmt"

	"github.com/rebelice/lazypg/internal/db/connection"
)

// MaterializedView represents a PostgreSQL materialized view
type MaterializedView struct {
	Schema string
	Name   string
}

// Function represents a PostgreSQL function
type Function struct {
	Schema    string
	Name      string
	Arguments string // e.g., "(id integer, name text)"
}

// Procedure represents a PostgreSQL procedure (PG11+)
type Procedure struct {
	Schema    string
	Name      string
	Arguments string
}

// TriggerFunction represents a PostgreSQL trigger function
type TriggerFunction struct {
	Schema string
	Name   string
}

// Sequence represents a PostgreSQL sequence
type Sequence struct {
	Schema     string
	Name       string
	StartValue int64
	MinValue   int64
	MaxValue   int64
	Increment  int64
	Cycle      bool
}

// Index represents a PostgreSQL index
type Index struct {
	Schema     string
	Table      string
	Name       string
	Definition string
}

// Trigger represents a PostgreSQL trigger
type Trigger struct {
	Schema     string
	Table      string
	Name       string
	Definition string
}

// Extension represents a PostgreSQL extension
type Extension struct {
	Name    string
	Version string
	Schema  string
}

// CompositeType represents a PostgreSQL composite type
type CompositeType struct {
	Schema string
	Name   string
}

// EnumType represents a PostgreSQL enum type
type EnumType struct {
	Schema string
	Name   string
	Labels []string
}

// DomainType represents a PostgreSQL domain type
type DomainType struct {
	Schema   string
	Name     string
	BaseType string
}

// RangeType represents a PostgreSQL range type
type RangeType struct {
	Schema   string
	Name     string
	Subtype  string
}

// ListMaterializedViews returns all materialized views in a schema
func ListMaterializedViews(ctx context.Context, pool *connection.Pool, schema string) ([]MaterializedView, error) {
	query := `
		SELECT schemaname, matviewname
		FROM pg_matviews
		WHERE schemaname = $1
		ORDER BY matviewname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	views := make([]MaterializedView, 0, len(rows))
	for _, row := range rows {
		views = append(views, MaterializedView{
			Schema: toString(row["schemaname"]),
			Name:   toString(row["matviewname"]),
		})
	}

	return views, nil
}

// ListFunctions returns all regular functions in a schema (excluding trigger functions and procedures)
func ListFunctions(ctx context.Context, pool *connection.Pool, schema string) ([]Function, error) {
	query := `
		SELECT p.proname, pg_get_function_identity_arguments(p.oid) as args
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname = $1
		  AND p.prokind = 'f'
		  AND p.prorettype != 'trigger'::regtype
		ORDER BY p.proname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	functions := make([]Function, 0, len(rows))
	for _, row := range rows {
		functions = append(functions, Function{
			Schema:    schema,
			Name:      toString(row["proname"]),
			Arguments: toString(row["args"]),
		})
	}

	return functions, nil
}

// ListProcedures returns all procedures in a schema (PG11+)
func ListProcedures(ctx context.Context, pool *connection.Pool, schema string) ([]Procedure, error) {
	// Check if prokind column exists (PG11+)
	query := `
		SELECT p.proname, pg_get_function_identity_arguments(p.oid) as args
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname = $1
		  AND p.prokind = 'p'
		ORDER BY p.proname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		// If prokind doesn't exist (PG10 or earlier), return empty list
		return []Procedure{}, nil
	}

	procedures := make([]Procedure, 0, len(rows))
	for _, row := range rows {
		procedures = append(procedures, Procedure{
			Schema:    schema,
			Name:      toString(row["proname"]),
			Arguments: toString(row["args"]),
		})
	}

	return procedures, nil
}

// ListTriggerFunctions returns all trigger functions in a schema
func ListTriggerFunctions(ctx context.Context, pool *connection.Pool, schema string) ([]TriggerFunction, error) {
	query := `
		SELECT p.proname
		FROM pg_proc p
		JOIN pg_namespace n ON p.pronamespace = n.oid
		WHERE n.nspname = $1
		  AND p.prorettype = 'trigger'::regtype
		ORDER BY p.proname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	functions := make([]TriggerFunction, 0, len(rows))
	for _, row := range rows {
		functions = append(functions, TriggerFunction{
			Schema: schema,
			Name:   toString(row["proname"]),
		})
	}

	return functions, nil
}

// ListSequences returns all sequences in a schema
func ListSequences(ctx context.Context, pool *connection.Pool, schema string) ([]Sequence, error) {
	query := `
		SELECT sequencename, start_value, min_value, max_value, increment_by, cycle
		FROM pg_sequences
		WHERE schemaname = $1
		ORDER BY sequencename;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	sequences := make([]Sequence, 0, len(rows))
	for _, row := range rows {
		sequences = append(sequences, Sequence{
			Schema:     schema,
			Name:       toString(row["sequencename"]),
			StartValue: toInt64(row["start_value"]),
			MinValue:   toInt64(row["min_value"]),
			MaxValue:   toInt64(row["max_value"]),
			Increment:  toInt64(row["increment_by"]),
			Cycle:      toBool(row["cycle"]),
		})
	}

	return sequences, nil
}

// ListTableIndexes returns all indexes for a specific table
func ListTableIndexes(ctx context.Context, pool *connection.Pool, schema, table string) ([]Index, error) {
	query := `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2
		ORDER BY indexname;
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}

	indexes := make([]Index, 0, len(rows))
	for _, row := range rows {
		indexes = append(indexes, Index{
			Schema:     schema,
			Table:      table,
			Name:       toString(row["indexname"]),
			Definition: toString(row["indexdef"]),
		})
	}

	return indexes, nil
}

// ListTableTriggers returns all triggers for a specific table
func ListTableTriggers(ctx context.Context, pool *connection.Pool, schema, table string) ([]Trigger, error) {
	query := `
		SELECT t.tgname, pg_get_triggerdef(t.oid) as definition
		FROM pg_trigger t
		JOIN pg_class c ON t.tgrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = $1 AND c.relname = $2
		  AND NOT t.tgisinternal
		ORDER BY t.tgname;
	`

	rows, err := pool.Query(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}

	triggers := make([]Trigger, 0, len(rows))
	for _, row := range rows {
		triggers = append(triggers, Trigger{
			Schema:     schema,
			Table:      table,
			Name:       toString(row["tgname"]),
			Definition: toString(row["definition"]),
		})
	}

	return triggers, nil
}

// ListExtensions returns all extensions in the database
func ListExtensions(ctx context.Context, pool *connection.Pool) ([]Extension, error) {
	query := `
		SELECT e.extname, e.extversion, n.nspname as schema
		FROM pg_extension e
		JOIN pg_namespace n ON e.extnamespace = n.oid
		ORDER BY e.extname;
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	extensions := make([]Extension, 0, len(rows))
	for _, row := range rows {
		extensions = append(extensions, Extension{
			Name:    toString(row["extname"]),
			Version: toString(row["extversion"]),
			Schema:  toString(row["schema"]),
		})
	}

	return extensions, nil
}

// ListCompositeTypes returns all composite types in a schema
func ListCompositeTypes(ctx context.Context, pool *connection.Pool, schema string) ([]CompositeType, error) {
	query := `
		SELECT t.typname
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		LEFT JOIN pg_class c ON t.typrelid = c.oid
		WHERE n.nspname = $1
		  AND t.typtype = 'c'
		  AND (c.relkind IS NULL OR c.relkind = 'c')
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]CompositeType, 0, len(rows))
	for _, row := range rows {
		types = append(types, CompositeType{
			Schema: schema,
			Name:   toString(row["typname"]),
		})
	}

	return types, nil
}

// ListEnumTypes returns all enum types in a schema
func ListEnumTypes(ctx context.Context, pool *connection.Pool, schema string) ([]EnumType, error) {
	query := `
		SELECT t.typname,
		       array_agg(e.enumlabel ORDER BY e.enumsortorder) as labels
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		JOIN pg_enum e ON t.oid = e.enumtypid
		WHERE n.nspname = $1
		  AND t.typtype = 'e'
		GROUP BY t.typname
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]EnumType, 0, len(rows))
	for _, row := range rows {
		types = append(types, EnumType{
			Schema: schema,
			Name:   toString(row["typname"]),
			Labels: toStringSlice(row["labels"]),
		})
	}

	return types, nil
}

// ListDomainTypes returns all domain types in a schema
func ListDomainTypes(ctx context.Context, pool *connection.Pool, schema string) ([]DomainType, error) {
	query := `
		SELECT t.typname, pg_catalog.format_type(t.typbasetype, t.typtypmod) as basetype
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		WHERE n.nspname = $1
		  AND t.typtype = 'd'
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]DomainType, 0, len(rows))
	for _, row := range rows {
		types = append(types, DomainType{
			Schema:   schema,
			Name:     toString(row["typname"]),
			BaseType: toString(row["basetype"]),
		})
	}

	return types, nil
}

// ListRangeTypes returns all range types in a schema
func ListRangeTypes(ctx context.Context, pool *connection.Pool, schema string) ([]RangeType, error) {
	query := `
		SELECT t.typname, st.typname as subtype
		FROM pg_type t
		JOIN pg_namespace n ON t.typnamespace = n.oid
		JOIN pg_range r ON t.oid = r.rngtypid
		JOIN pg_type st ON r.rngsubtype = st.oid
		WHERE n.nspname = $1
		  AND t.typtype = 'r'
		ORDER BY t.typname;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	types := make([]RangeType, 0, len(rows))
	for _, row := range rows {
		types = append(types, RangeType{
			Schema:  schema,
			Name:    toString(row["typname"]),
			Subtype: toString(row["subtype"]),
		})
	}

	return types, nil
}

// Helper functions for type conversion
func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
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

func toStringSlice(v interface{}) []string {
	if v == nil {
		return []string{}
	}
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return []string{}
	}
}
```

**Step 2: Verify the file compiles**

Run: `go build ./internal/db/metadata/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/db/metadata/objects.go
git commit -m "feat(metadata): add query functions for all PostgreSQL object types"
```

---

## Phase 3: Add Theme Colors for New Icons

### Task 3.1: Add New Icon Colors to Theme

**Files:**
- Modify: `internal/ui/theme/theme.go`
- Modify: `internal/ui/theme/catppuccin.go`
- Modify: `internal/ui/theme/default.go`

**Step 1: Add new color fields to Theme struct**

In `internal/ui/theme/theme.go`, add after line 57 (after `ForeignKey`):

```go
	// Additional tree icon colors
	MaterializedViewIcon lipgloss.Color
	ProcedureIcon        lipgloss.Color
	TriggerFunctionIcon  lipgloss.Color
	SequenceIcon         lipgloss.Color
	IndexIcon            lipgloss.Color
	TriggerIcon          lipgloss.Color
	ExtensionIcon        lipgloss.Color
	TypeIcon             lipgloss.Color
```

**Step 2: Add colors to CatppuccinMochaTheme**

In `internal/ui/theme/catppuccin.go`, add the new colors in the return statement:

```go
	// Additional tree icon colors
	MaterializedViewIcon: lipgloss.Color("#89dceb"), // Sky - cached view
	ProcedureIcon:        lipgloss.Color("#cba6f7"), // Mauve - procedure
	TriggerFunctionIcon:  lipgloss.Color("#f9e2af"), // Yellow - trigger func
	SequenceIcon:         lipgloss.Color("#94e2d5"), // Teal - sequential
	IndexIcon:            lipgloss.Color("#fab387"), // Peach - performance
	TriggerIcon:          lipgloss.Color("#f38ba8"), // Red - event trigger
	ExtensionIcon:        lipgloss.Color("#a6e3a1"), // Green - extension
	TypeIcon:             lipgloss.Color("#74c7ec"), // Sapphire - type
```

**Step 3: Add colors to DefaultTheme**

In `internal/ui/theme/default.go`, add matching colors (can use same or similar values).

**Step 4: Verify changes compile**

Run: `go build ./internal/ui/theme/...`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/ui/theme/
git commit -m "feat(theme): add icon colors for new database object types"
```

---

## Phase 4: Update Tree View Icons

### Task 4.1: Add Icons for New Node Types

**Files:**
- Modify: `internal/ui/components/tree_view.go:286-360`

**Step 1: Update getNodeIcon() function**

Replace the switch statement in `getNodeIcon()` to include all new types:

```go
func (tv *TreeView) getNodeIcon(node *models.TreeNode) string {
	var icon string
	var iconColor lipgloss.Color

	switch node.Type {
	case models.TreeNodeTypeDatabase:
		isActive := false
		if meta, ok := node.Metadata.(map[string]interface{}); ok {
			if active, ok := meta["active"].(bool); ok && active {
				isActive = true
			}
		}
		if isActive {
			icon = "●"
			iconColor = tv.Theme.DatabaseActive
		} else {
			icon = "○"
			iconColor = tv.Theme.DatabaseInactive
		}

	case models.TreeNodeTypeSchema:
		if node.Expanded {
			icon = "▾"
			iconColor = tv.Theme.SchemaExpanded
		} else {
			icon = "▸"
			iconColor = tv.Theme.SchemaCollapsed
		}

	case models.TreeNodeTypeTableGroup,
		models.TreeNodeTypeViewGroup,
		models.TreeNodeTypeMaterializedViewGroup,
		models.TreeNodeTypeFunctionGroup,
		models.TreeNodeTypeProcedureGroup,
		models.TreeNodeTypeTriggerFunctionGroup,
		models.TreeNodeTypeSequenceGroup,
		models.TreeNodeTypeTypeGroup,
		models.TreeNodeTypeExtensionGroup,
		models.TreeNodeTypeIndexGroup,
		models.TreeNodeTypeTriggerGroup,
		models.TreeNodeTypeCompositeTypeGroup,
		models.TreeNodeTypeEnumTypeGroup,
		models.TreeNodeTypeDomainTypeGroup,
		models.TreeNodeTypeRangeTypeGroup:
		if node.Expanded {
			icon = "▾"
		} else {
			icon = "▸"
		}
		// Color based on group type
		switch node.Type {
		case models.TreeNodeTypeTableGroup:
			iconColor = tv.Theme.TableIcon
		case models.TreeNodeTypeViewGroup:
			iconColor = tv.Theme.ViewIcon
		case models.TreeNodeTypeMaterializedViewGroup:
			iconColor = tv.Theme.MaterializedViewIcon
		case models.TreeNodeTypeFunctionGroup:
			iconColor = tv.Theme.FunctionIcon
		case models.TreeNodeTypeProcedureGroup:
			iconColor = tv.Theme.ProcedureIcon
		case models.TreeNodeTypeTriggerFunctionGroup:
			iconColor = tv.Theme.TriggerFunctionIcon
		case models.TreeNodeTypeSequenceGroup:
			iconColor = tv.Theme.SequenceIcon
		case models.TreeNodeTypeTypeGroup,
			models.TreeNodeTypeCompositeTypeGroup,
			models.TreeNodeTypeEnumTypeGroup,
			models.TreeNodeTypeDomainTypeGroup,
			models.TreeNodeTypeRangeTypeGroup:
			iconColor = tv.Theme.TypeIcon
		case models.TreeNodeTypeExtensionGroup:
			iconColor = tv.Theme.ExtensionIcon
		case models.TreeNodeTypeIndexGroup:
			iconColor = tv.Theme.IndexIcon
		case models.TreeNodeTypeTriggerGroup:
			iconColor = tv.Theme.TriggerIcon
		default:
			iconColor = tv.Theme.Foreground
		}

	case models.TreeNodeTypeTable:
		icon = "▦"
		iconColor = tv.Theme.TableIcon

	case models.TreeNodeTypeView:
		icon = "◎"
		iconColor = tv.Theme.ViewIcon

	case models.TreeNodeTypeMaterializedView:
		icon = "◉"
		iconColor = tv.Theme.MaterializedViewIcon

	case models.TreeNodeTypeFunction:
		icon = "ƒ"
		iconColor = tv.Theme.FunctionIcon

	case models.TreeNodeTypeProcedure:
		icon = "⚙"
		iconColor = tv.Theme.ProcedureIcon

	case models.TreeNodeTypeTriggerFunction:
		icon = "⚡"
		iconColor = tv.Theme.TriggerFunctionIcon

	case models.TreeNodeTypeSequence:
		icon = "#"
		iconColor = tv.Theme.SequenceIcon

	case models.TreeNodeTypeIndex:
		icon = "⊕"
		iconColor = tv.Theme.IndexIcon

	case models.TreeNodeTypeTrigger:
		icon = "↯"
		iconColor = tv.Theme.TriggerIcon

	case models.TreeNodeTypeExtension:
		icon = "◈"
		iconColor = tv.Theme.ExtensionIcon

	case models.TreeNodeTypeCompositeType:
		icon = "◫"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeEnumType:
		icon = "◧"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeDomainType:
		icon = "◨"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeRangeType:
		icon = "◩"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeColumn:
		icon = "•"
		iconColor = tv.Theme.ColumnIcon

	default:
		if node.Expanded {
			icon = "▾"
			iconColor = tv.Theme.Foreground
		} else {
			icon = "▸"
			iconColor = tv.Theme.Foreground
		}
	}

	iconStyle := lipgloss.NewStyle().Foreground(iconColor)
	return iconStyle.Render(icon)
}
```

**Step 2: Verify changes compile**

Run: `go build ./internal/ui/components/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/components/tree_view.go
git commit -m "feat(tree_view): add icons for all database object types"
```

---

## Phase 5: Update Tree Building Logic

### Task 5.1: Update loadTree Function in app.go

**Files:**
- Modify: `internal/app/app.go` (loadTree function, around line 2900-2986)

**Step 1: Update loadTree to build complete object tree**

This is a large change. Replace the content of the loadTree function's schema building section:

```go
// In loadTree function, replace the schema node building section

for _, schema := range schemas {
	// Load all objects for this schema
	tables, _ := metadata.ListTables(ctx, conn.Pool, schema.Name)
	views, _ := metadata.ListViews(ctx, conn.Pool, schema.Name)
	matViews, _ := metadata.ListMaterializedViews(ctx, conn.Pool, schema.Name)
	functions, _ := metadata.ListFunctions(ctx, conn.Pool, schema.Name)
	procedures, _ := metadata.ListProcedures(ctx, conn.Pool, schema.Name)
	triggerFuncs, _ := metadata.ListTriggerFunctions(ctx, conn.Pool, schema.Name)
	sequences, _ := metadata.ListSequences(ctx, conn.Pool, schema.Name)
	compositeTypes, _ := metadata.ListCompositeTypes(ctx, conn.Pool, schema.Name)
	enumTypes, _ := metadata.ListEnumTypes(ctx, conn.Pool, schema.Name)
	domainTypes, _ := metadata.ListDomainTypes(ctx, conn.Pool, schema.Name)
	rangeTypes, _ := metadata.ListRangeTypes(ctx, conn.Pool, schema.Name)

	// Count total objects
	totalObjects := len(tables) + len(views) + len(matViews) + len(functions) +
		len(procedures) + len(triggerFuncs) + len(sequences) +
		len(compositeTypes) + len(enumTypes) + len(domainTypes) + len(rangeTypes)

	// Skip empty schemas
	if totalObjects == 0 {
		continue
	}

	// Create schema label
	schemaLabel := schema.Name

	schemaNode := models.NewTreeNode(
		fmt.Sprintf("schema:%s.%s", currentDB, schema.Name),
		models.TreeNodeTypeSchema,
		schemaLabel,
	)
	schemaNode.Selectable = true

	// Add Tables group
	if len(tables) > 0 {
		tablesGroup := models.NewTreeNode(
			fmt.Sprintf("tables:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeTableGroup,
			fmt.Sprintf("Tables (%d)", len(tables)),
		)
		tablesGroup.Selectable = false

		for _, table := range tables {
			tableNode := models.NewTreeNode(
				fmt.Sprintf("table:%s.%s.%s", currentDB, schema.Name, table.Name),
				models.TreeNodeTypeTable,
				table.Name,
			)
			tableNode.Selectable = true

			// Load indexes and triggers for this table
			indexes, _ := metadata.ListTableIndexes(ctx, conn.Pool, schema.Name, table.Name)
			triggers, _ := metadata.ListTableTriggers(ctx, conn.Pool, schema.Name, table.Name)

			// Add Indexes group under table
			if len(indexes) > 0 {
				indexGroup := models.NewTreeNode(
					fmt.Sprintf("indexes:%s.%s.%s", currentDB, schema.Name, table.Name),
					models.TreeNodeTypeIndexGroup,
					fmt.Sprintf("Indexes (%d)", len(indexes)),
				)
				indexGroup.Selectable = false
				for _, idx := range indexes {
					idxNode := models.NewTreeNode(
						fmt.Sprintf("index:%s.%s.%s.%s", currentDB, schema.Name, table.Name, idx.Name),
						models.TreeNodeTypeIndex,
						idx.Name,
					)
					idxNode.Selectable = true
					idxNode.Metadata = idx
					idxNode.Loaded = true
					indexGroup.AddChild(idxNode)
				}
				indexGroup.Loaded = true
				tableNode.AddChild(indexGroup)
			}

			// Add Triggers group under table
			if len(triggers) > 0 {
				triggerGroup := models.NewTreeNode(
					fmt.Sprintf("triggers:%s.%s.%s", currentDB, schema.Name, table.Name),
					models.TreeNodeTypeTriggerGroup,
					fmt.Sprintf("Triggers (%d)", len(triggers)),
				)
				triggerGroup.Selectable = false
				for _, trg := range triggers {
					trgNode := models.NewTreeNode(
						fmt.Sprintf("trigger:%s.%s.%s.%s", currentDB, schema.Name, table.Name, trg.Name),
						models.TreeNodeTypeTrigger,
						trg.Name,
					)
					trgNode.Selectable = true
					trgNode.Metadata = trg
					trgNode.Loaded = true
					triggerGroup.AddChild(trgNode)
				}
				triggerGroup.Loaded = true
				tableNode.AddChild(triggerGroup)
			}

			tableNode.Loaded = len(indexes) == 0 && len(triggers) == 0
			tablesGroup.AddChild(tableNode)
		}
		tablesGroup.Loaded = true
		schemaNode.AddChild(tablesGroup)
	}

	// Add Views group
	if len(views) > 0 {
		viewsGroup := models.NewTreeNode(
			fmt.Sprintf("views:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeViewGroup,
			fmt.Sprintf("Views (%d)", len(views)),
		)
		viewsGroup.Selectable = false

		for _, view := range views {
			viewNode := models.NewTreeNode(
				fmt.Sprintf("view:%s.%s.%s", currentDB, schema.Name, view.Name),
				models.TreeNodeTypeView,
				view.Name,
			)
			viewNode.Selectable = true
			viewNode.Loaded = true
			viewsGroup.AddChild(viewNode)
		}
		viewsGroup.Loaded = true
		schemaNode.AddChild(viewsGroup)
	}

	// Add Materialized Views group
	if len(matViews) > 0 {
		matViewsGroup := models.NewTreeNode(
			fmt.Sprintf("matviews:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeMaterializedViewGroup,
			fmt.Sprintf("Materialized Views (%d)", len(matViews)),
		)
		matViewsGroup.Selectable = false

		for _, mv := range matViews {
			mvNode := models.NewTreeNode(
				fmt.Sprintf("matview:%s.%s.%s", currentDB, schema.Name, mv.Name),
				models.TreeNodeTypeMaterializedView,
				mv.Name,
			)
			mvNode.Selectable = true
			mvNode.Loaded = true
			matViewsGroup.AddChild(mvNode)
		}
		matViewsGroup.Loaded = true
		schemaNode.AddChild(matViewsGroup)
	}

	// Add Functions group
	if len(functions) > 0 {
		funcsGroup := models.NewTreeNode(
			fmt.Sprintf("functions:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeFunctionGroup,
			fmt.Sprintf("Functions (%d)", len(functions)),
		)
		funcsGroup.Selectable = false

		for _, fn := range functions {
			label := fn.Name
			if fn.Arguments != "" {
				label = fmt.Sprintf("%s(%s)", fn.Name, fn.Arguments)
			}
			fnNode := models.NewTreeNode(
				fmt.Sprintf("function:%s.%s.%s", currentDB, schema.Name, fn.Name),
				models.TreeNodeTypeFunction,
				label,
			)
			fnNode.Selectable = true
			fnNode.Metadata = fn
			fnNode.Loaded = true
			funcsGroup.AddChild(fnNode)
		}
		funcsGroup.Loaded = true
		schemaNode.AddChild(funcsGroup)
	}

	// Add Procedures group
	if len(procedures) > 0 {
		procsGroup := models.NewTreeNode(
			fmt.Sprintf("procedures:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeProcedureGroup,
			fmt.Sprintf("Procedures (%d)", len(procedures)),
		)
		procsGroup.Selectable = false

		for _, proc := range procedures {
			label := proc.Name
			if proc.Arguments != "" {
				label = fmt.Sprintf("%s(%s)", proc.Name, proc.Arguments)
			}
			procNode := models.NewTreeNode(
				fmt.Sprintf("procedure:%s.%s.%s", currentDB, schema.Name, proc.Name),
				models.TreeNodeTypeProcedure,
				label,
			)
			procNode.Selectable = true
			procNode.Metadata = proc
			procNode.Loaded = true
			procsGroup.AddChild(procNode)
		}
		procsGroup.Loaded = true
		schemaNode.AddChild(procsGroup)
	}

	// Add Trigger Functions group
	if len(triggerFuncs) > 0 {
		trigFuncsGroup := models.NewTreeNode(
			fmt.Sprintf("triggerfuncs:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeTriggerFunctionGroup,
			fmt.Sprintf("Trigger Functions (%d)", len(triggerFuncs)),
		)
		trigFuncsGroup.Selectable = false

		for _, tf := range triggerFuncs {
			tfNode := models.NewTreeNode(
				fmt.Sprintf("triggerfunc:%s.%s.%s", currentDB, schema.Name, tf.Name),
				models.TreeNodeTypeTriggerFunction,
				tf.Name,
			)
			tfNode.Selectable = true
			tfNode.Metadata = tf
			tfNode.Loaded = true
			trigFuncsGroup.AddChild(tfNode)
		}
		trigFuncsGroup.Loaded = true
		schemaNode.AddChild(trigFuncsGroup)
	}

	// Add Sequences group
	if len(sequences) > 0 {
		seqsGroup := models.NewTreeNode(
			fmt.Sprintf("sequences:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeSequenceGroup,
			fmt.Sprintf("Sequences (%d)", len(sequences)),
		)
		seqsGroup.Selectable = false

		for _, seq := range sequences {
			seqNode := models.NewTreeNode(
				fmt.Sprintf("sequence:%s.%s.%s", currentDB, schema.Name, seq.Name),
				models.TreeNodeTypeSequence,
				seq.Name,
			)
			seqNode.Selectable = true
			seqNode.Metadata = seq
			seqNode.Loaded = true
			seqsGroup.AddChild(seqNode)
		}
		seqsGroup.Loaded = true
		schemaNode.AddChild(seqsGroup)
	}

	// Add Types group (with subgroups)
	hasTypes := len(compositeTypes) > 0 || len(enumTypes) > 0 || len(domainTypes) > 0 || len(rangeTypes) > 0
	if hasTypes {
		typesGroup := models.NewTreeNode(
			fmt.Sprintf("types:%s.%s", currentDB, schema.Name),
			models.TreeNodeTypeTypeGroup,
			fmt.Sprintf("Types (%d)", len(compositeTypes)+len(enumTypes)+len(domainTypes)+len(rangeTypes)),
		)
		typesGroup.Selectable = false

		// Composite Types
		if len(compositeTypes) > 0 {
			compGroup := models.NewTreeNode(
				fmt.Sprintf("compositetypes:%s.%s", currentDB, schema.Name),
				models.TreeNodeTypeCompositeTypeGroup,
				fmt.Sprintf("Composite (%d)", len(compositeTypes)),
			)
			compGroup.Selectable = false
			for _, ct := range compositeTypes {
				ctNode := models.NewTreeNode(
					fmt.Sprintf("compositetype:%s.%s.%s", currentDB, schema.Name, ct.Name),
					models.TreeNodeTypeCompositeType,
					ct.Name,
				)
				ctNode.Selectable = true
				ctNode.Loaded = true
				compGroup.AddChild(ctNode)
			}
			compGroup.Loaded = true
			typesGroup.AddChild(compGroup)
		}

		// Enum Types
		if len(enumTypes) > 0 {
			enumGroup := models.NewTreeNode(
				fmt.Sprintf("enumtypes:%s.%s", currentDB, schema.Name),
				models.TreeNodeTypeEnumTypeGroup,
				fmt.Sprintf("Enum (%d)", len(enumTypes)),
			)
			enumGroup.Selectable = false
			for _, et := range enumTypes {
				etNode := models.NewTreeNode(
					fmt.Sprintf("enumtype:%s.%s.%s", currentDB, schema.Name, et.Name),
					models.TreeNodeTypeEnumType,
					et.Name,
				)
				etNode.Selectable = true
				etNode.Metadata = et
				etNode.Loaded = true
				enumGroup.AddChild(etNode)
			}
			enumGroup.Loaded = true
			typesGroup.AddChild(enumGroup)
		}

		// Domain Types
		if len(domainTypes) > 0 {
			domGroup := models.NewTreeNode(
				fmt.Sprintf("domaintypes:%s.%s", currentDB, schema.Name),
				models.TreeNodeTypeDomainTypeGroup,
				fmt.Sprintf("Domain (%d)", len(domainTypes)),
			)
			domGroup.Selectable = false
			for _, dt := range domainTypes {
				dtNode := models.NewTreeNode(
					fmt.Sprintf("domaintype:%s.%s.%s", currentDB, schema.Name, dt.Name),
					models.TreeNodeTypeDomainType,
					fmt.Sprintf("%s → %s", dt.Name, dt.BaseType),
				)
				dtNode.Selectable = true
				dtNode.Metadata = dt
				dtNode.Loaded = true
				domGroup.AddChild(dtNode)
			}
			domGroup.Loaded = true
			typesGroup.AddChild(domGroup)
		}

		// Range Types
		if len(rangeTypes) > 0 {
			rangeGroup := models.NewTreeNode(
				fmt.Sprintf("rangetypes:%s.%s", currentDB, schema.Name),
				models.TreeNodeTypeRangeTypeGroup,
				fmt.Sprintf("Range (%d)", len(rangeTypes)),
			)
			rangeGroup.Selectable = false
			for _, rt := range rangeTypes {
				rtNode := models.NewTreeNode(
					fmt.Sprintf("rangetype:%s.%s.%s", currentDB, schema.Name, rt.Name),
					models.TreeNodeTypeRangeType,
					fmt.Sprintf("%s [%s]", rt.Name, rt.Subtype),
				)
				rtNode.Selectable = true
				rtNode.Metadata = rt
				rtNode.Loaded = true
				rangeGroup.AddChild(rtNode)
			}
			rangeGroup.Loaded = true
			typesGroup.AddChild(rangeGroup)
		}

		typesGroup.Loaded = true
		schemaNode.AddChild(typesGroup)
	}

	schemaNode.Loaded = true
	dbNode.AddChild(schemaNode)
}
```

**Step 2: Add Extensions at Database Level**

Before the schema loop, add extensions loading:

```go
// Load extensions at database level
extensions, _ := metadata.ListExtensions(ctx, conn.Pool)
if len(extensions) > 0 {
	extGroup := models.NewTreeNode(
		fmt.Sprintf("extensions:%s", currentDB),
		models.TreeNodeTypeExtensionGroup,
		fmt.Sprintf("Extensions (%d)", len(extensions)),
	)
	extGroup.Selectable = false

	for _, ext := range extensions {
		extNode := models.NewTreeNode(
			fmt.Sprintf("extension:%s.%s", currentDB, ext.Name),
			models.TreeNodeTypeExtension,
			fmt.Sprintf("%s v%s", ext.Name, ext.Version),
		)
		extNode.Selectable = true
		extNode.Metadata = ext
		extNode.Loaded = true
		extGroup.AddChild(extNode)
	}
	extGroup.Loaded = true
	dbNode.AddChild(extGroup)
}
```

**Step 3: Verify changes compile**

Run: `go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): load all database object types into tree"
```

---

## Phase 6: Handle Object Selection

### Task 6.1: Update TreeNodeSelectedMsg Handler

**Files:**
- Modify: `internal/app/app.go` (around line 1320)

**Step 1: Update the selection handler to support new object types**

Find the `case components.TreeNodeSelectedMsg:` block and update it:

```go
case components.TreeNodeSelectedMsg:
	// Handle selection based on node type
	if msg.Node == nil {
		return a, nil
	}

	switch msg.Node.Type {
	case models.TreeNodeTypeTable, models.TreeNodeTypeView, models.TreeNodeTypeMaterializedView:
		// Get schema name by traversing up the tree
		var schemaName string
		current := msg.Node.Parent
		for current != nil {
			if current.Type == models.TreeNodeTypeSchema {
				schemaName = strings.Split(current.Label, " ")[0]
				break
			}
			current = current.Parent
		}

		if schemaName == "" {
			return a, nil
		}

		// Clear any active filter when switching tables
		a.activeFilter = nil

		// Store selected node
		a.state.TreeSelected = msg.Node

		// Load table/view data
		return a, a.loadTableData(LoadTableDataMsg{
			Schema: schemaName,
			Table:  msg.Node.Label,
			Offset: 0,
			Limit:  500,
		})

	case models.TreeNodeTypeFunction, models.TreeNodeTypeProcedure, models.TreeNodeTypeTriggerFunction:
		// TODO: Display function/procedure source code
		a.state.TreeSelected = msg.Node
		return a, nil

	case models.TreeNodeTypeSequence:
		// TODO: Display sequence properties
		a.state.TreeSelected = msg.Node
		return a, nil

	case models.TreeNodeTypeIndex, models.TreeNodeTypeTrigger:
		// TODO: Display DDL definition
		a.state.TreeSelected = msg.Node
		return a, nil

	case models.TreeNodeTypeExtension:
		// TODO: Display extension info
		a.state.TreeSelected = msg.Node
		return a, nil

	case models.TreeNodeTypeCompositeType, models.TreeNodeTypeEnumType,
		models.TreeNodeTypeDomainType, models.TreeNodeTypeRangeType:
		// TODO: Display type definition
		a.state.TreeSelected = msg.Node
		return a, nil

	default:
		return a, nil
	}
```

**Step 2: Verify changes compile**

Run: `go build ./...`
Expected: No errors

**Step 3: Run the application to test**

Run: `go run ./cmd/lazypg`
Expected: Tree shows all object types with proper icons

**Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): handle selection of all database object types"
```

---

## Phase 7: Future Enhancement Placeholder

### Task 7.1: Document TODO Items for Right Panel Display

The following features are marked as TODO in the selection handler and should be implemented in future iterations:

1. **Function/Procedure Source Code Display**
   - Query: `SELECT pg_get_functiondef(oid) FROM pg_proc WHERE proname = $1`
   - Display with SQL syntax highlighting

2. **Sequence Properties Display**
   - Show: current_value, start_value, min_value, max_value, increment, cycle
   - Use a properties table format

3. **Index/Trigger DDL Display**
   - Index: Already have `indexdef` in metadata
   - Trigger: Already have `definition` in metadata
   - Display as formatted SQL

4. **Extension Info Display**
   - Show: name, version, schema, description
   - Query for description: `SELECT comment FROM pg_description WHERE objoid = ext.oid`

5. **Type Definition Display**
   - Composite: Show fields and their types
   - Enum: Show all enum labels
   - Domain: Show base type and constraints
   - Range: Show subtype and operators

---

## Summary

This plan adds support for all major PostgreSQL database objects in the tree view:

| Object Type | Icon | Selectable | Parent |
|-------------|------|------------|--------|
| Extension | ◈ | Yes | Database |
| Table | ▦ | Yes | Tables group |
| View | ◎ | Yes | Views group |
| Materialized View | ◉ | Yes | Mat Views group |
| Function | ƒ | Yes | Functions group |
| Procedure | ⚙ | Yes | Procedures group |
| Trigger Function | ⚡ | Yes | Trigger Funcs group |
| Sequence | # | Yes | Sequences group |
| Index | ⊕ | Yes | Table > Indexes |
| Trigger | ↯ | Yes | Table > Triggers |
| Composite Type | ◫ | Yes | Types > Composite |
| Enum Type | ◧ | Yes | Types > Enum |
| Domain Type | ◨ | Yes | Types > Domain |
| Range Type | ◩ | Yes | Types > Range |

**Total commits:** 6
**Estimated implementation time:** Follow tasks sequentially
