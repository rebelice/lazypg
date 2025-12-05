# Code Editor Implementation Plan

## Overview

Implement a unified Code Editor component for viewing and editing PostgreSQL database object definitions (functions, procedures, views, triggers, etc.). The component will support both read-only viewing and full editing capabilities.

## Goals

1. Replace the current ugly `renderObjectDetails` with a proper code editor component
2. Support syntax highlighting for SQL/PLpgSQL using Chroma library
3. Provide read-only mode for viewing definitions
4. Support edit mode for modifying definitions
5. Execute changes via `CREATE OR REPLACE` statements
6. Maintain consistency with existing SQL Editor styling

## Architecture

### Component Structure

```
┌─────────────────────────────────────────────────────────────┐
│                      CodeEditor                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                   EditorCore                             ││
│  │  - Text buffer (lines []string)                         ││
│  │  - Cursor management (row, col)                         ││
│  │  - Scrolling (scrollY, scrollX)                         ││
│  │  - Line numbers                                          ││
│  │  - Syntax highlighting (Chroma)                         ││
│  └─────────────────────────────────────────────────────────┘│
│         ▲                              ▲                     │
│         │                              │                     │
│  ┌──────┴──────┐              ┌────────┴────────┐           │
│  │ ReadOnly    │◄── e ───────►│ Editable        │           │
│  │ Mode        │              │ Mode            │           │
│  │ - j/k scroll│              │ - Full cursor   │           │
│  │ - y:copy    │              │ - Text editing  │           │
│  │ - q:close   │              │ - Ctrl+S:save   │           │
│  └─────────────┘              └─────────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Reuse SQLEditor tokenizer**: The existing `sql_editor.go` has a working SQL tokenizer - we'll enhance it to support PLpgSQL keywords
2. **Use Chroma for highlighting**: Add Chroma library for more sophisticated syntax highlighting with theme support
3. **Separate component**: Create `code_editor.go` as a standalone component (not modify SQLEditor) to keep concerns separated
4. **Mode switching**: ReadOnly mode by default, press `e` to enter Edit mode

## Implementation Tasks

### Phase 1: Core CodeEditor Component

#### Task 1.1: Create CodeEditor Base Structure
**File**: `internal/ui/components/code_editor.go`

Create the base component with:
- Struct definition with all necessary fields
- Constructor `NewCodeEditor()`
- Read-only view rendering with line numbers
- Basic scrolling (j/k for vim-style, arrow keys)

```go
type CodeEditor struct {
    // Content
    lines       []string
    cursorRow   int
    cursorCol   int
    scrollY     int

    // Object info
    Title       string      // e.g., "Function: public.get_user_by_id(integer)"
    ObjectType  string      // "function", "procedure", "view", etc.
    ObjectName  string      // "public.get_user_by_id"
    Language    string      // "plpgsql", "sql"

    // State
    Width       int
    Height      int
    ReadOnly    bool        // true = view mode, false = edit mode
    Modified    bool        // true if content changed from original
    Original    string      // Original content for comparison

    // Theme
    Theme       theme.Theme

    // Cached styles
    cachedStyles *codeEditorStyles
}
```

#### Task 1.2: Implement Syntax Highlighting with Chroma
**File**: `internal/ui/components/code_editor.go`

Add Chroma-based syntax highlighting:
- Add dependency: `github.com/alecthomas/chroma/v2`
- Create highlight function using Chroma's PostgreSQL SQL and PLpgSQL lexers
- Map Chroma tokens to theme colors
- Fallback to built-in tokenizer if Chroma fails

```go
func (ce *CodeEditor) highlightLine(line string) string {
    lexer := lexers.Get(ce.Language) // "postgresql" or "plpgsql"
    if lexer == nil {
        lexer = lexers.Get("sql")
    }
    // Use terminal256 formatter
    formatter := formatters.Get("terminal256")
    style := styles.Get("catppuccin-mocha") // or map from theme

    iterator, _ := lexer.Tokenise(nil, line)
    var buf bytes.Buffer
    formatter.Format(&buf, style, iterator)
    return buf.String()
}
```

#### Task 1.3: Read-Only View with Status Bar
**File**: `internal/ui/components/code_editor.go`

Implement the View() method for read-only mode:
```
┌─ Function: public.get_user_by_id(integer) ────────────────┐
│                                           [Read Only]     │
│  1 │ CREATE OR REPLACE FUNCTION get_user_by_id(          │
│  2 │     p_user_id integer                               │
│  3 │ ) RETURNS TABLE(id integer, name text)              │
│  4 │ LANGUAGE plpgsql AS $$                              │
│  5 │ BEGIN                                               │
│  6 │     RETURN QUERY SELECT * FROM users                │
│  7 │     WHERE id = p_user_id;                           │
│  8 │ END;                                                │
│  9 │ $$ SECURITY DEFINER;                                │
├───────────────────────────────────────────────────────────┤
│ e:edit  y:copy  q:close               Line 1/9  Col 1    │
└───────────────────────────────────────────────────────────┘
```

### Phase 2: Edit Mode Support

#### Task 2.1: Mode Switching and Cursor
**File**: `internal/ui/components/code_editor.go`

Add edit mode functionality:
- `EnterEditMode()` / `ExitEditMode()` methods
- Cursor rendering (like SQLEditor)
- Mode indicator in title bar: `[Read Only]` / `[Editing]` / `[Modified]`

#### Task 2.2: Text Editing Operations
**File**: `internal/ui/components/code_editor.go`

Implement text editing (reuse patterns from SQLEditor):
- Character insertion
- Backspace/Delete
- Newline insertion
- Home/End navigation
- Word-based movement (Ctrl+Left/Right)

#### Task 2.3: Update Handler
**File**: `internal/ui/components/code_editor.go`

```go
func (ce *CodeEditor) Update(msg tea.KeyMsg) (*CodeEditor, tea.Cmd) {
    if ce.ReadOnly {
        return ce.handleReadOnlyKeys(msg)
    }
    return ce.handleEditKeys(msg)
}
```

### Phase 3: Save and Execute

#### Task 3.1: Save Dialog Message Types
**File**: `internal/ui/components/code_editor.go`

```go
// SaveObjectMsg is sent when user wants to save changes
type SaveObjectMsg struct {
    ObjectType string // "function", "procedure", "view", etc.
    ObjectName string // "public.get_user_by_id"
    Content    string // New definition
}

// ObjectSavedMsg is sent after save completes
type ObjectSavedMsg struct {
    Success bool
    Error   error
}
```

#### Task 3.2: Generate SQL Statements
**File**: `internal/db/metadata/objects.go`

Add functions to generate appropriate SQL for each object type:

```go
// GenerateFunctionReplaceSQL generates CREATE OR REPLACE FUNCTION statement
func GenerateFunctionReplaceSQL(source string) string {
    // For functions, the source usually already contains CREATE OR REPLACE
    return source
}

// GenerateViewReplaceSQL generates CREATE OR REPLACE VIEW statement
func GenerateViewReplaceSQL(schema, name, definition string) string {
    return fmt.Sprintf("CREATE OR REPLACE VIEW %s.%s AS\n%s",
        schema, name, definition)
}
```

### Phase 4: Integration with App

#### Task 4.1: Replace renderObjectDetails
**File**: `internal/app/app.go`

Replace the current inline rendering with CodeEditor component:
- Add `codeEditor *components.CodeEditor` field to App struct
- Create/update CodeEditor when `ObjectDetailsLoadedMsg` is received
- Handle CodeEditor key events in App.Update()
- Handle `SaveObjectMsg` and execute SQL

#### Task 4.2: Handle Object Selection
**File**: `internal/app/app.go`

When a database object is selected (Enter key on tree node):
```go
case TreeNodeTypeFunction:
    a.codeEditor = components.NewCodeEditor(a.theme)
    a.codeEditor.SetContent(msg.Content, "function", msg.Title)
    a.codeEditor.Language = "plpgsql"
```

#### Task 4.3: Execute Save Command
**File**: `internal/app/app.go`

```go
case components.SaveObjectMsg:
    return a, func() tea.Msg {
        _, err := a.connectionManager.ActivePool().Execute(ctx, msg.Content)
        if err != nil {
            return components.ObjectSavedMsg{Success: false, Error: err}
        }
        return components.ObjectSavedMsg{Success: true}
    }
```

### Phase 5: Object-Specific Editing Support

#### Task 5.1: Support Different Object Types

| Object Type | Edit Method | SQL Template |
|-------------|-------------|--------------|
| Function | CREATE OR REPLACE | Source contains full definition |
| Procedure | CREATE OR REPLACE | Source contains full definition |
| View | CREATE OR REPLACE | `CREATE OR REPLACE VIEW x AS ...` |
| Materialized View | DROP + CREATE | Requires confirmation dialog |
| Trigger | DROP + CREATE | Requires confirmation dialog |
| Sequence | ALTER SEQUENCE | Property-based changes |

#### Task 5.2: Confirmation Dialog for Destructive Operations
**File**: `internal/ui/components/confirm_dialog.go` (new)

For operations that require DROP (materialized views, triggers):
```
┌─ Confirm Changes ─────────────────────────────┐
│                                               │
│  Updating this object requires dropping and   │
│  recreating it. This may fail if there are    │
│  dependent objects.                           │
│                                               │
│  Object: public.monthly_stats (mat. view)     │
│                                               │
│           [Cancel]  [Proceed]                 │
└───────────────────────────────────────────────┘
```

### Phase 6: Enhanced Features

#### Task 6.1: Diff View Before Save
Show changes before saving:
```
┌─ Changes to public.get_user_by_id ────────────┐
│   5     RETURN QUERY SELECT * FROM users      │
│ - 6         WHERE id = p_user_id;             │
│ + 6         WHERE id = p_user_id              │
│ + 7         AND active = true;                │
│   8     END;                                  │
├───────────────────────────────────────────────┤
│           [Cancel]  [Save Changes]            │
└───────────────────────────────────────────────┘
```

#### Task 6.2: Undo/Redo Support
- Implement simple undo stack
- Ctrl+Z for undo, Ctrl+Y or Ctrl+Shift+Z for redo

#### Task 6.3: Copy Full Definition
- `y` key copies entire definition to clipboard
- Status message: "Definition copied to clipboard"

## File Changes Summary

### New Files
1. `internal/ui/components/code_editor.go` - Main CodeEditor component
2. `internal/ui/components/confirm_dialog.go` - Confirmation dialog for destructive ops (Phase 5)

### Modified Files
1. `internal/app/app.go` - Integration with CodeEditor
2. `internal/db/metadata/objects.go` - Add SQL generation functions
3. `go.mod` - Add Chroma dependency

## Dependencies

Add to `go.mod`:
```
github.com/alecthomas/chroma/v2 v2.x.x
```

## UI Mockups

### Read-Only Mode (Default)
```
┌─ Function: public.get_user_by_id(integer) ─────── [Read Only] ─┐
│  1 │ CREATE OR REPLACE FUNCTION get_user_by_id(               │
│  2 │     p_user_id integer                                    │
│  3 │ ) RETURNS TABLE(id integer, name text)                   │
│  4 │ LANGUAGE plpgsql AS $$                                   │
│  5 │ BEGIN                                                    │
│  6 │     RETURN QUERY SELECT id, name                         │
│  7 │     FROM users WHERE id = p_user_id;                     │
│  8 │ END;                                                     │
│  9 │ $$;                                                      │
├────────────────────────────────────────────────────────────────┤
│ e:edit  y:copy  q:close                       Line 1/9  100%  │
└────────────────────────────────────────────────────────────────┘
```

### Edit Mode
```
┌─ Function: public.get_user_by_id(integer) ───── [Modified *] ──┐
│  1 │ CREATE OR REPLACE FUNCTION get_user_by_id(               │
│  2 │     p_user_id integer                                    │
│  3 │ ) RETURNS TABLE(id integer, name text, email text)       │
│  4 │ LANGUAGE plpgsql AS $$                                   │
│  5 │ BEGIN                                                    │
│  6 │     RETURN QUERY SELECT id, name, email█                 │
│  7 │     FROM users WHERE id = p_user_id;                     │
│  8 │ END;                                                     │
│  9 │ $$;                                                      │
├────────────────────────────────────────────────────────────────┤
│ Ctrl+S:save  Esc:cancel                       Ln 6, Col 45    │
└────────────────────────────────────────────────────────────────┘
```

## Key Bindings

### Read-Only Mode
| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `g` | Go to top |
| `G` | Go to bottom |
| `y` | Copy definition |
| `e` | Enter edit mode |
| `q` / `Esc` | Close and return to tree |

### Edit Mode
| Key | Action |
|-----|--------|
| Arrow keys | Move cursor |
| `Home` | Start of line |
| `End` | End of line |
| `Ctrl+Home` | Start of file |
| `Ctrl+End` | End of file |
| `Backspace` | Delete before cursor |
| `Delete` | Delete after cursor |
| `Enter` | New line |
| `Ctrl+S` | Save changes |
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |
| `Esc` | Cancel and return to read-only |

## Implementation Order

1. **Phase 1** (Core): Create basic CodeEditor with read-only view and syntax highlighting
2. **Phase 4.1** (Integration): Replace `renderObjectDetails` with CodeEditor
3. **Phase 2** (Edit Mode): Add edit mode support
4. **Phase 3** (Save): Implement save functionality
5. **Phase 5** (Objects): Support different object types
6. **Phase 6** (Enhanced): Add diff view, undo/redo

## Testing

1. Unit tests for CodeEditor component
2. Test syntax highlighting for SQL, PLpgSQL, SQL with dollar-quotes
3. Test edit mode operations
4. Integration tests for save/execute flow
5. Test with various object types (functions, views, triggers)

## Verification

After implementation, verify:
- [ ] Functions display with proper syntax highlighting
- [ ] Scrolling works correctly for long definitions
- [ ] Edit mode allows text modification
- [ ] Save executes correct SQL
- [ ] Error handling for failed saves
- [ ] Copy to clipboard works
- [ ] Proper styling matches existing SQL Editor
