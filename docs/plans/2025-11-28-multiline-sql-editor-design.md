# Multiline SQL Editor Design

## Overview

Replace the single-line QuickQuery with a full-featured multiline SQL editor. The new layout follows VS Code style: sidebar on the left, right side split into Data Panel (top) and SQL Editor (bottom).

## Layout

```
┌─────────────┬──────────────────────────────────────────┐
│             │ [1] users (128 rows) [2] orders (50 rows)│  ← Result Tabs
│  Sidebar    ├──────────────────────────────────────────┤
│             │                                          │
│  (Schema    │            Data Panel                    │  ← Current Tab result
│   Browser)  │         (Table View)                     │
│             │                                          │
│             ├──────────────────────────────────────────┤
│             │  1 │ SELECT * FROM users                 │  ← SQL Editor
│             │  2 │ WHERE active = true;                │    (collapsible, with line numbers)
└─────────────┴──────────────────────────────────────────┘
```

### Key Changes

- Remove existing QuickQuery component
- Right side splits into: Result Tabs + Data Panel + SQL Editor
- SQL Editor is collapsible with three height presets
- Data Panel supports multiple tabs for different query results

## SQL Editor Behavior

### Collapsed State

- Shows 1-2 lines height
- Preserves last SQL content
- Displays line numbers

### Expanded State

- Three height presets: Small (20%) / Medium (35%) / Large (50%)
- `Ctrl+Shift+↑/↓` to switch presets
- Default expands to Medium

### Execution Flow

1. User presses `Ctrl+Enter` to execute
2. Identify SQL statement at cursor position (semicolon-separated)
3. Create new Tab to display result
4. Editor auto-collapses
5. Focus switches to Data Panel
6. SQL content preserved in editor (not cleared)

### Multi-Statement & Transaction

- Statements separated by semicolon `;`
- Execute single statement at cursor position
- **Same session (connection)**, shares session state like temp tables
- **Default autocommit**, no automatic transaction
- User must manually `BEGIN` to start transaction

## Result Tabs

### Tab Management

- Maximum 10 tabs retained
- Oldest tab auto-closes when exceeded
- No manual close functionality (for now)

### Tab Title Format

`[index] smart_title (row_count)`

### Smart Title Rules (by priority)

1. **Custom comment** - Extract when SQL starts with `-- title`
   - `-- Active Users\nSELECT...` → `[1] Active Users (128 rows)`

2. **Table name extraction**
   - `SELECT * FROM users` → `[1] users (128 rows)`
   - `SELECT * FROM users JOIN orders` → `[1] users(+) (128 rows)`
   - `UPDATE users SET...` → `[1] UPDATE users (5 rows)`
   - `DELETE FROM orders` → `[1] DELETE orders (3 rows)`

3. **SQL truncation** - Show first 20 chars when unrecognizable
   - `WITH cte AS (...)` → `[1] WITH cte AS... (50 rows)`

### Tab Switching

- `[` previous tab
- `]` next tab

### Width Adaptation

- Sufficient space: `[1] users (128 rows)`
- Limited space: `[1] users`
- Extreme case: `[1] use...`

## Editor Features

### Basic Editing

- Multi-line text editing (insert, delete, newline)
- Cursor movement (↑↓←→, Home/End for line start/end, Ctrl+Home/End for document start/end)
- Text selection and deletion
- `Ctrl+U` to clear content

### Line Numbers

- Display line numbers on the left
- Follow theme colors

### Syntax Highlighting

- SQL keywords (SELECT, FROM, WHERE, etc.)
- Strings (single-quoted content)
- Numbers
- Comments (`--` and `/* */`)
- Identifiers (table names, column names)
- Follow current theme colors

### History Navigation

- `Ctrl+↑` previous history
- `Ctrl+↓` next history
- Reuse existing history store

### External Editor

- `Ctrl+O` to open external editor
- Read `$EDITOR` environment variable (default vim)
- Sync content back to SQL Editor after editing

## Keyboard Shortcuts

### Editor Control

| Shortcut | Action |
|----------|--------|
| `Ctrl+E` | Toggle editor expand/collapse |
| `Ctrl+Shift+↑` | Increase editor height preset |
| `Ctrl+Shift+↓` | Decrease editor height preset |
| `Ctrl+O` | Open external editor |

### Execution & History

| Shortcut | Action |
|----------|--------|
| `Ctrl+S` | Execute statement at cursor |
| `Ctrl+↑` | Previous history |
| `Ctrl+↓` | Next history |

> Note: `Ctrl+Enter` produces same key code as `Enter` in terminals. `Alt+Enter` doesn't work on macOS.

### Tabs & Focus

| Shortcut | Action |
|----------|--------|
| `[` | Switch to previous Result Tab |
| `]` | Switch to next Result Tab |
| `Tab` | Cycle focus (Sidebar → Data → Editor) |

### Smart Focus

- `Ctrl+E` expand → auto-focus Editor
- `Ctrl+S` execute → auto-focus Data Panel
- Editor collapse → auto-focus Data Panel

## Implementation Impact

### New Components

- `sql_editor.go` - Multi-line SQL editor with syntax highlighting and line numbers
- `result_tabs.go` - Result Tab management component

### Modified Files

- `app.go` - Layout restructure, integrate new components
- `table_view.go` - Adapt to new Tab container
- Remove `quick_query.go` - Replaced by SQL Editor

### Layout Change

- Current: Sidebar + Main content + Bottom QuickQuery
- New: Sidebar + Right side (Result Tabs + Data Panel + SQL Editor)

### Reuse Existing

- `history/store.go` - Query history storage
- `query/executor.go` - SQL execution engine
- `Theme` - Syntax highlighting colors
