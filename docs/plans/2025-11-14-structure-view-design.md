# Structure View Design

**Date**: 2025-11-14
**Status**: Design Complete - Ready for Implementation

## Overview

Add a comprehensive structure view for PostgreSQL tables, allowing users to inspect columns, constraints, and indexes through a tabbed interface in the right panel.

## Requirements

### Information to Display
- **Columns**: Name, type, nullable, default value, constraint markers, comments
- **Constraints**: Primary keys, foreign keys, unique constraints, check constraints
- **Indexes**: Name, type, columns, properties, size, definition

### UI Organization
- **Tabbed interface** in right panel
- Four tabs: Data | Columns | Constraints | Indexes
- Switch between tabs with Ctrl+1/2/3/4, arrow keys, or mouse

## Architecture

### Overall Layout

When a table is selected in the left navigation tree:

```
┌─────────────────────────────────────────┐
│ Data | Columns | Constraints | Indexes  │ ← Tab bar (Ctrl+1/2/3/4)
├─────────────────────────────────────────┤
│                                         │
│         Current tab content             │
│                                         │
└─────────────────────────────────────────┘
```

### Tab Switching
- **Ctrl+1** - Data tab (existing table data view)
- **Ctrl+2** - Columns tab
- **Ctrl+3** - Constraints tab
- **Ctrl+4** - Indexes tab
- **Left/Right arrows** - Navigate between tabs when tab bar is focused
- **Mouse click** - Click tab to switch (if mouse enabled)

### State Management
- Active tab is highlighted
- Each tab preserves scroll position and selection
- Tab bar only shown when table node is selected

## Columns Tab Design

### Display Format

Table with the following columns:

| Column Name | Type | Nullable | Default | Constraints | Comment |
|-------------|------|----------|---------|-------------|---------|
| id | integer | NO | nextval(...) | 🔑 PK | Primary key |
| name | varchar(100) | NO | - | ✓ UQ | User name |
| email | varchar(255) | YES | NULL | - | Email address |
| user_id | integer | NO | - | 🔗 FK | References users table |

### Constraint Markers
- 🔑 **PK** - Primary Key
- 🔗 **FK** - Foreign Key
- ✓ **UQ** - Unique
- ⚠️ **CK** - Check Constraint

### Data Type Display
- Show complete type: `varchar(100)`, `numeric(10,2)`, `timestamp with time zone`
- Array types: `integer[]`, `text[]`
- Custom types: Display type name

### Comments
- Display PostgreSQL `COMMENT ON COLUMN` information
- Show `-` if no comment
- Truncate long comments, show full on selection

### Interactions
- **↑↓** - Navigate rows
- **y** - Copy column name
- **Y** - Copy full column definition
- **/** - Search column names

## Constraints Tab Design

### Display Format

Single table showing all constraints:

| Type | Name | Columns | Definition/Reference | Description |
|------|------|---------|----------------------|-------------|
| PK | users_pkey | id | PRIMARY KEY (id) | Primary key constraint |
| FK | orders_user_id_fkey | user_id | → users(id) | References users table |
| UQ | users_email_key | email | UNIQUE (email) | Email must be unique |
| CK | users_age_check | age | age > 0 AND age < 150 | Age range validation |

### Constraint Type Styling
- **PK** - Primary Key - `theme.Info` color (blue)
- **FK** - Foreign Key - `theme.Warning` color (orange)
- **UQ** - Unique - `theme.Success` color (green)
- **CK** - Check - `theme.Metadata` color (gray)

### Foreign Key Format
- Format: `→ referenced_table(referenced_columns)`
- Example: `→ users(id)`
- Multi-column: `→ users(tenant_id, user_id)`

### Check Constraint Definition
- Show complete CHECK expression
- Truncate if too long, show full on selection

### Interactions
- **↑↓** - Navigate rows
- **y** - Copy constraint name
- **Y** - Copy constraint definition SQL
- **/** - Search constraint names

## Indexes Tab Design

### Display Format

Table showing index information:

| Name | Type | Columns | Properties | Size | Definition |
|------|------|---------|------------|------|------------|
| users_pkey | btree | id | 🔑 PK, ✓ UQ | 16 KB | CREATE UNIQUE INDEX... |
| idx_users_email | btree | email | ✓ UQ | 32 KB | CREATE UNIQUE INDEX... |
| idx_users_name | btree | name | - | 24 KB | CREATE INDEX... |
| idx_users_active | btree | status | 📋 Partial | 8 KB | WHERE status = 'active' |

### Index Types
- **btree** - B-tree index (most common)
- **hash** - Hash index
- **gin** - GIN index (full-text, JSONB)
- **gist** - GiST index (geometric, full-text)
- **brin** - BRIN index (block range)
- **spgist** - SP-GiST index

### Property Markers
- 🔑 **PK** - Primary Key index
- ✓ **UQ** - Unique index
- 📋 **Partial** - Partial index (has WHERE clause)
- 🔤 **Expression** - Expression index

### Index Definition
- Show complete `CREATE INDEX` statement
- Truncate if too long, show full on selection
- Include WHERE clause for partial indexes
- Include expression for expression indexes

### Multi-Column Indexes
- Comma-separated columns: `name, created_at`
- Preserve column order (important for index performance)

### Interactions
- **↑↓** - Navigate rows
- **y** - Copy index name
- **Y** - Copy index definition SQL
- **/** - Search index names

## Technical Implementation

### Database Queries

**Columns Information**:
```sql
SELECT
    column_name,
    data_type,
    is_nullable,
    column_default,
    character_maximum_length,
    numeric_precision,
    numeric_scale
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position;

-- Get column comments
SELECT
    a.attname AS column_name,
    d.description AS comment
FROM pg_catalog.pg_attribute a
LEFT JOIN pg_catalog.pg_description d ON d.objoid = a.attrelid AND d.objsubid = a.attnum
WHERE a.attrelid = $1::regclass AND a.attnum > 0 AND NOT a.attisdropped;
```

**Constraints Information**:
```sql
SELECT
    con.conname AS constraint_name,
    con.contype AS constraint_type,
    pg_get_constraintdef(con.oid) AS definition,
    array_agg(att.attname ORDER BY u.attposition) AS columns,
    nf.nspname AS foreign_schema,
    clf.relname AS foreign_table
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class cl ON con.conrelid = cl.oid
LEFT JOIN pg_catalog.pg_namespace nf ON con.confrelid = nf.oid
LEFT JOIN pg_catalog.pg_class clf ON con.confrelid = clf.oid
LATERAL unnest(con.conkey) WITH ORDINALITY AS u(attnum, attposition)
JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = u.attnum
WHERE cl.relname = $1 AND cl.relnamespace = $2::regnamespace
GROUP BY con.conname, con.contype, con.oid, nf.nspname, clf.relname;
```

**Indexes Information**:
```sql
SELECT
    i.indexrelid::regclass AS index_name,
    am.amname AS index_type,
    pg_get_indexdef(i.indexrelid) AS definition,
    i.indisunique AS is_unique,
    i.indisprimary AS is_primary,
    pg_relation_size(i.indexrelid) AS size,
    array_agg(a.attname ORDER BY array_position(i.indkey, a.attnum)) AS columns,
    pg_get_expr(i.indpred, i.indrelid) AS predicate
FROM pg_catalog.pg_index i
JOIN pg_catalog.pg_class c ON c.oid = i.indrelid
JOIN pg_catalog.pg_am am ON am.oid = c.relam
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
WHERE c.relname = $1 AND c.relnamespace = $2::regnamespace
GROUP BY i.indexrelid, am.amname, i.indisunique, i.indisprimary, i.indpred, i.indrelid;
```

### Component Structure

```
internal/ui/components/
  ├── structure_view.go       # Main container, manages tab switching
  ├── columns_view.go         # Columns tab table
  ├── constraints_view.go     # Constraints tab table
  └── indexes_view.go         # Indexes tab table

internal/db/metadata/
  ├── columns.go              # Query column information
  ├── constraints.go          # Query constraint information
  └── indexes.go              # Query index information
```

### App State

```go
type App struct {
    // ... existing fields

    // Structure view
    showStructure   bool
    structureView   *components.StructureView
    currentTab      int  // 0=Data, 1=Columns, 2=Constraints, 3=Indexes
}
```

### StructureView Component

```go
type StructureView struct {
    Width  int
    Height int
    Theme  theme.Theme

    // Current active tab (0-3)
    activeTab int

    // Tab views
    columnsView     *ColumnsView
    constraintsView *ConstraintsView
    indexesView     *IndexesView

    // Table info
    schema string
    table  string
}

func (sv *StructureView) Update(msg tea.KeyMsg) (*StructureView, tea.Cmd)
func (sv *StructureView) View() string
func (sv *StructureView) SetTable(schema, table string) error
func (sv *StructureView) SwitchTab(tabIndex int)
```

### Keyboard Handling in App

```go
// In app.go Update method
case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+1"))):
    a.currentTab = 0  // Data

case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+2"))):
    a.currentTab = 1  // Columns

case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+3"))):
    a.currentTab = 2  // Constraints

case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+4"))):
    a.currentTab = 3  // Indexes
```

## Implementation Phases

### Phase 1: Foundation
1. Create `StructureView` container component
2. Implement tab bar rendering and switching
3. Add keyboard shortcuts (Ctrl+1/2/3/4)
4. Integrate into `app.go`

### Phase 2: Columns Tab
1. Create database queries for column information
2. Implement `ColumnsView` component
3. Add constraint marker logic
4. Add copy functionality (y/Y)

### Phase 3: Constraints Tab
1. Create database queries for constraints
2. Implement `ConstraintsView` component
3. Add constraint type styling
4. Format foreign key references

### Phase 4: Indexes Tab
1. Create database queries for indexes
2. Implement `IndexesView` component
3. Add property markers
4. Format index definitions

### Phase 5: Polish
1. Add search functionality to each tab
2. Optimize SQL queries
3. Add error handling
4. Update help documentation

## Success Criteria

- ✅ Can view column definitions with types and constraints
- ✅ Can view all constraint types in one place
- ✅ Can view index information with properties
- ✅ Tab switching works smoothly with keyboard shortcuts
- ✅ Copy functionality works for names and definitions
- ✅ Visual styling is consistent with existing UI
- ✅ Search works within each tab
- ✅ Performance is acceptable for tables with many columns/indexes

## Future Enhancements

- Show index usage statistics (scan counts, cache hit ratio)
- Show table-level statistics (row count, size, last vacuum/analyze)
- Add ability to generate ALTER TABLE statements
- Show column dependencies (views, functions using the column)
- Export structure information to SQL/Markdown
