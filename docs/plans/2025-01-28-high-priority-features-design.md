# High Priority Features Design

## Overview

This document describes the design for 4 high-priority features to enhance lazypg usability.

## Features

### 1. Views Support

**Goal:** Display database views in the tree alongside tables.

**Design:**
- Separate "Tables" and "Views" groups under each schema
- Tree structure:
  ```
  ▼ database
    ▼ public (35 tables, 5 views)
      ▼ Tables
        ├── users
        └── orders
      ▼ Views
        ├── active_users
        └── order_summary
  ```

**Implementation:**
1. Add `ListViews()` function in `internal/db/metadata/tables.go`
2. Modify tree building logic in `internal/app/app.go` to add Tables/Views grouping
3. Update `internal/models/tree.go` to support new node types (TableGroup, ViewGroup)
4. Views should be selectable and display data like tables

**Files to modify:**
- `internal/db/metadata/tables.go` - Add view query
- `internal/models/tree.go` - Add node types
- `internal/app/app.go` - Update tree building

---

### 2. Quick Cell Copy

**Goal:** Copy cell content with `y` key, preview content with `Y` key.

**Design:**
- `y` - Copy current cell content to clipboard (works in all tabs)
- `Y` - Copy preview pane content (only when preview is visible)

**Implementation:**
1. Add `y` key handler to copy `tableView.Rows[selectedRow][selectedCol]`
2. Change existing `y` (preview copy) to `Y`
3. Use `clipboard.WriteAll()` from atotto/clipboard

**Files to modify:**
- `internal/app/app.go` - Update key handlers

---

### 3. Table Search

**Goal:** Search tables by name with two methods.

**Design:**

#### Method A: Tree Filter (`/` key)
- When left panel focused, `/` opens search input
- Filters tree to show only matching tables/views
- `Esc` clears filter and shows all

#### Method B: Global Jump (`Ctrl+T`)
- Opens modal search dialog (like command palette)
- Shows all tables across all schemas
- Fuzzy search by table name
- `Enter` selects and navigates to table
- Format: `schema.table_name`

**Implementation:**
1. Add search input to tree view component
2. Add filter logic to tree rendering
3. Create table jump dialog (reuse command palette style)
4. Add `Ctrl+T` global handler

**Files to modify:**
- `internal/ui/components/tree_view.go` - Add search/filter
- `internal/app/app.go` - Add Ctrl+T handler and dialog

---

### 4. Refresh Table Data

**Goal:** Refresh current table data without refreshing entire tree.

**Design:**
- `Ctrl+R` - Refresh current table data (preserves sort and filter)
- `Ctrl+X` - Clear filter (moved from Ctrl+R)

**Behavior:**
- Re-executes current query with same sort/filter parameters
- Resets to first page (offset 0)
- Shows loading indicator

**Implementation:**
1. Change `Ctrl+R` from "clear filter" to "refresh table"
2. Add `Ctrl+X` for "clear filter"
3. Refresh reloads data with current SortColumn, SortDirection, NullsFirst

**Files to modify:**
- `internal/app/app.go` - Update key handlers

---

## Implementation Order

1. **Quick Cell Copy** (simplest, low risk)
2. **Refresh Table Data** (simple key rebinding + reload logic)
3. **Views Support** (moderate complexity, schema change)
4. **Table Search** (most complex, new UI components)

## Keyboard Shortcuts Summary

| Key | Action | Context |
|-----|--------|---------|
| `y` | Copy current cell | Right panel, all tabs |
| `Y` | Copy preview pane content | Right panel, preview visible |
| `/` | Open tree search/filter | Left panel |
| `Ctrl+T` | Open table jump dialog | Global |
| `Ctrl+R` | Refresh current table | Right panel |
| `Ctrl+X` | Clear filter | Right panel |
