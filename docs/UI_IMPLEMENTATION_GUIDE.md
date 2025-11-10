# UI Implementation Guide

Step-by-step guide to implement the UI/UX improvements for LazyPG.

---

## Phase 1: Foundation (2-3 hours)

### Step 1.1: Update Theme Structure (30 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/theme/theme.go`

Add new color fields to the Theme struct:

```go
type Theme struct {
    Name string

    // Base colors
    Background lipgloss.Color
    Foreground lipgloss.Color

    // NEW: Surface layers
    Mantle   lipgloss.Color
    Crust    lipgloss.Color
    Surface0 lipgloss.Color
    Surface1 lipgloss.Color
    Surface2 lipgloss.Color

    // NEW: Text hierarchy
    Text     lipgloss.Color
    Subtext1 lipgloss.Color
    Subtext0 lipgloss.Color
    Overlay2 lipgloss.Color
    Overlay1 lipgloss.Color
    Overlay0 lipgloss.Color

    // UI elements
    Border        lipgloss.Color
    BorderFocused lipgloss.Color
    Selection     lipgloss.Color
    Cursor        lipgloss.Color

    // Status colors
    Success lipgloss.Color
    Warning lipgloss.Color
    Error   lipgloss.Color
    Info    lipgloss.Color

    // NEW: Full accent palette
    Red      lipgloss.Color
    Peach    lipgloss.Color
    Yellow   lipgloss.Color
    Green    lipgloss.Color
    Teal     lipgloss.Color
    Sky      lipgloss.Color
    Sapphire lipgloss.Color
    Blue     lipgloss.Color
    Mauve    lipgloss.Color
    Pink     lipgloss.Color

    // Syntax highlighting
    Keyword  lipgloss.Color
    String   lipgloss.Color
    Number   lipgloss.Color
    Comment  lipgloss.Color
    Function lipgloss.Color
    Operator lipgloss.Color

    // Table colors
    TableHeader      lipgloss.Color
    TableRowEven     lipgloss.Color
    TableRowOdd      lipgloss.Color
    TableRowSelected lipgloss.Color
    TableBorder      lipgloss.Color  // NEW

    // JSONB colors
    JSONKey     lipgloss.Color
    JSONString  lipgloss.Color
    JSONNumber  lipgloss.Color
    JSONBoolean lipgloss.Color
    JSONNull    lipgloss.Color
}
```

### Step 1.2: Create Catppuccin Mocha Theme (45 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/theme/catppuccin_mocha.go`

```go
package theme

import "github.com/charmbracelet/lipgloss"

// CatppuccinMocha returns the Catppuccin Mocha dark theme
// Based on: https://github.com/catppuccin/catppuccin
func CatppuccinMocha() Theme {
    return Theme{
        Name: "catppuccin-mocha",

        // Base colors
        Background: lipgloss.Color("#1e1e2e"),  // Base
        Foreground: lipgloss.Color("#cdd6f4"),  // Text

        // Surface layers
        Mantle:   lipgloss.Color("#181825"),
        Crust:    lipgloss.Color("#11111b"),
        Surface0: lipgloss.Color("#313244"),
        Surface1: lipgloss.Color("#45475a"),
        Surface2: lipgloss.Color("#585b70"),

        // Text hierarchy
        Text:     lipgloss.Color("#cdd6f4"),
        Subtext1: lipgloss.Color("#bac2de"),
        Subtext0: lipgloss.Color("#a6adc8"),
        Overlay2: lipgloss.Color("#9399b2"),
        Overlay1: lipgloss.Color("#7f849c"),
        Overlay0: lipgloss.Color("#6c7086"),

        // UI elements
        Border:        lipgloss.Color("#45475a"),  // Surface1
        BorderFocused: lipgloss.Color("#89b4fa"),  // Blue
        Selection:     lipgloss.Color("#313244"),  // Surface0
        Cursor:        lipgloss.Color("#f5e0dc"),  // Rosewater

        // Status colors
        Success: lipgloss.Color("#a6e3a1"),  // Green
        Warning: lipgloss.Color("#f9e2af"),  // Yellow
        Error:   lipgloss.Color("#f38ba8"),  // Red
        Info:    lipgloss.Color("#89dceb"),  // Sky

        // Accent colors
        Red:      lipgloss.Color("#f38ba8"),
        Peach:    lipgloss.Color("#fab387"),
        Yellow:   lipgloss.Color("#f9e2af"),
        Green:    lipgloss.Color("#a6e3a1"),
        Teal:     lipgloss.Color("#94e2d5"),
        Sky:      lipgloss.Color("#89dceb"),
        Sapphire: lipgloss.Color("#74c7ec"),
        Blue:     lipgloss.Color("#89b4fa"),
        Mauve:    lipgloss.Color("#cba6f7"),
        Pink:     lipgloss.Color("#f5c2e7"),

        // Syntax highlighting
        Keyword:  lipgloss.Color("#cba6f7"),  // Mauve
        String:   lipgloss.Color("#f5c2e7"),  // Pink
        Number:   lipgloss.Color("#fab387"),  // Peach
        Comment:  lipgloss.Color("#6c7086"),  // Overlay0
        Function: lipgloss.Color("#89b4fa"),  // Blue
        Operator: lipgloss.Color("#94e2d5"),  // Teal

        // Table colors
        TableHeader:      lipgloss.Color("#89b4fa"),  // Blue
        TableRowEven:     lipgloss.Color("#1e1e2e"),  // Base
        TableRowOdd:      lipgloss.Color("#181825"),  // Mantle
        TableRowSelected: lipgloss.Color("#313244"),  // Surface0
        TableBorder:      lipgloss.Color("#45475a"),  // Surface1

        // JSONB colors
        JSONKey:     lipgloss.Color("#89b4fa"),  // Blue
        JSONString:  lipgloss.Color("#f5c2e7"),  // Pink
        JSONNumber:  lipgloss.Color("#fab387"),  // Peach
        JSONBoolean: lipgloss.Color("#a6e3a1"),  // Green
        JSONNull:    lipgloss.Color("#6c7086"),  // Overlay0
    }
}
```

### Step 1.3: Update Default Theme (15 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/theme/default.go`

Update to populate new fields (can map to existing or use Catppuccin):

```go
package theme

import "github.com/charmbracelet/lipgloss"

// DefaultTheme returns the default dark theme
// Now uses Catppuccin Mocha for consistency
func DefaultTheme() Theme {
    return CatppuccinMocha()
}
```

### Step 1.4: Test Theme Update (30 min)

```bash
# Build and run
cd /Users/rebeliceyang/Github/lazypg
go build -o bin/lazypg ./cmd/lazypg
./bin/lazypg

# Verify:
# - Colors look good
# - No compilation errors
# - Existing UI still works
```

---

## Phase 2: Tree View Enhancement (3-4 hours)

### Step 2.1: Add Icon Constants (15 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/components/tree_view.go`

Add at the top after imports:

```go
const (
    // Tree node icons
    IconDatabaseActive   = "●"  // Active database
    IconDatabaseInactive = "○"  // Inactive database
    IconExpanded         = "▾"  // Expanded node
    IconCollapsed        = "▸"  // Collapsed node
    IconTable            = "▦"  // Table
    IconView             = "▤"  // View
    IconFunction         = "ƒ"  // Function
    IconColumn           = "•"  // Column bullet
    IconEmpty            = "∅"  // Empty set
    IconLoading          = "…"  // Loading

    // Column indicators
    IconPrimaryKey = "⚿"  // Primary key
    IconForeignKey = "→"  // Foreign key
    IconIndex      = "⚡"  // Index
    IconNotNull    = "*"  // Not null
)
```

### Step 2.2: Enhance getNodeIcon() (45 min)

Replace the existing `getNodeIcon()` function:

```go
// getNodeIcon returns the appropriate icon for a node with color
func (tv *TreeView) getNodeIcon(node *models.TreeNode) string {
    iconStyle := lipgloss.NewStyle()

    switch node.Type {
    case models.TreeNodeTypeDatabase:
        // Check if this is the active database
        isActive := false
        if meta, ok := node.Metadata.(map[string]interface{}); ok {
            if active, ok := meta["active"].(bool); ok {
                isActive = active
            }
        }

        if isActive {
            return iconStyle.
                Foreground(tv.Theme.Success).
                Bold(true).
                Render(IconDatabaseActive)
        }
        return iconStyle.
            Foreground(tv.Theme.Overlay1).
            Render(IconDatabaseInactive)

    case models.TreeNodeTypeSchema:
        color := tv.Theme.Info
        if node.Expanded {
            return iconStyle.Foreground(color).Render(IconExpanded)
        }
        return iconStyle.Foreground(color).Render(IconCollapsed)

    case models.TreeNodeTypeTable:
        return iconStyle.
            Foreground(tv.Theme.Mauve).
            Render(IconTable)

    case models.TreeNodeTypeView:
        return iconStyle.
            Foreground(tv.Theme.Teal).
            Render(IconView)

    case models.TreeNodeTypeFunction:
        return iconStyle.
            Foreground(tv.Theme.Peach).
            Render(IconFunction)

    case models.TreeNodeTypeColumn:
        // Columns get double indent + bullet
        return "  " + iconStyle.
            Foreground(tv.Theme.Overlay1).
            Render(IconColumn)
    }

    // Default for unknown types
    if node.Expanded {
        return IconExpanded
    }
    return IconCollapsed
}
```

### Step 2.3: Enhance buildNodeLabel() (60 min)

Replace the existing `buildNodeLabel()` function:

```go
// buildNodeLabel builds the display label for a node, including metadata
func (tv *TreeView) buildNodeLabel(node *models.TreeNode) string {
    label := node.Label

    switch node.Type {
    case models.TreeNodeTypeDatabase:
        // Active indicator is in icon now, but can add metadata
        if meta, ok := node.Metadata.(map[string]interface{}); ok {
            if isActive, ok := meta["active"].(bool); ok && isActive {
                // Bold the label for active database
                activeStyle := lipgloss.NewStyle().
                    Foreground(tv.Theme.Success).
                    Bold(true)
                return activeStyle.Render(label)
            }
        }

    case models.TreeNodeTypeSchema:
        dimStyle := lipgloss.NewStyle().Foreground(tv.Theme.Overlay1)

        if !node.Loaded {
            return label + " " + dimStyle.Render(IconLoading)
        }

        childCount := len(node.Children)
        if childCount == 0 {
            return label + " " + dimStyle.Render(IconEmpty)
        }
        return label + " " + dimStyle.Render(fmt.Sprintf("(%d)", childCount))

    case models.TreeNodeTypeTable:
        // Add row count if available
        if meta, ok := node.Metadata.(map[string]interface{}); ok {
            if rowCount, ok := meta["row_count"].(int64); ok {
                countStyle := lipgloss.NewStyle().Foreground(tv.Theme.Teal)
                return label + " " + countStyle.Render(tv.formatRowCount(rowCount))
            }
        }

    case models.TreeNodeTypeView:
        // Views could show row count too
        dimStyle := lipgloss.NewStyle().Foreground(tv.Theme.Overlay1)
        return label + " " + dimStyle.Render("(view)")

    case models.TreeNodeTypeFunction:
        // Functions could show parameter count
        dimStyle := lipgloss.NewStyle().Foreground(tv.Theme.Overlay1)
        return label + " " + dimStyle.Render("()")

    case models.TreeNodeTypeColumn:
        // Enhanced column display
        return tv.buildColumnLabel(node)
    }

    return label
}

// buildColumnLabel creates detailed column labels with type and indicators
func (tv *TreeView) buildColumnLabel(node *models.TreeNode) string {
    label := node.Label

    colInfo, ok := node.Metadata.(models.ColumnInfo)
    if !ok {
        return label
    }

    // Add data type
    typeStyle := lipgloss.NewStyle().Foreground(tv.Theme.Overlay1)
    label += " " + typeStyle.Render(colInfo.DataType)

    // Build indicators
    var indicators []string

    if colInfo.PrimaryKey {
        pkStyle := lipgloss.NewStyle().
            Foreground(tv.Theme.Warning).
            Bold(true)
        indicators = append(indicators, pkStyle.Render(IconPrimaryKey))
    }

    if colInfo.ForeignKey {
        fkStyle := lipgloss.NewStyle().Foreground(tv.Theme.Info)
        indicators = append(indicators, fkStyle.Render(IconForeignKey))
    }

    if !colInfo.Nullable {
        nnStyle := lipgloss.NewStyle().Foreground(tv.Theme.Success)
        indicators = append(indicators, nnStyle.Render(IconNotNull))
    }

    if colInfo.Indexed {
        idxStyle := lipgloss.NewStyle().Foreground(tv.Theme.Mauve)
        indicators = append(indicators, idxStyle.Render(IconIndex))
    }

    if len(indicators) > 0 {
        label += " " + strings.Join(indicators, "")
    }

    return label
}

// formatRowCount formats row counts with k/M suffixes
func (tv *TreeView) formatRowCount(count int64) string {
    if count < 1000 {
        return fmt.Sprintf("%d", count)
    } else if count < 10000 {
        k := float64(count) / 1000.0
        if k == float64(int(k)) {
            return fmt.Sprintf("%.0fk", k)
        }
        return fmt.Sprintf("%.1fk", k)
    } else if count < 1000000 {
        return fmt.Sprintf("%.0fk", float64(count)/1000.0)
    } else {
        return fmt.Sprintf("%.1fM", float64(count)/1000000.0)
    }
}
```

### Step 2.4: Improve Empty State (15 min)

Update the `emptyState()` function:

```go
// emptyState returns the empty state view
func (tv *TreeView) emptyState() string {
    iconStyle := lipgloss.NewStyle().
        Foreground(tv.Theme.Overlay1).
        Bold(true).
        Width(tv.Width - 2).
        Align(lipgloss.Center)

    messageStyle := lipgloss.NewStyle().
        Foreground(tv.Theme.Overlay1).
        Italic(true).
        Width(tv.Width - 2).
        Align(lipgloss.Center)

    icon := iconStyle.Render("⚠")
    message := messageStyle.Render("No database connection\nPress 'c' to connect")

    return icon + "\n" + message
}
```

### Step 2.5: Test Tree View (30 min)

```bash
# Build and test
go build -o bin/lazypg ./cmd/lazypg
./bin/lazypg

# Verify:
# - Icons appear correctly
# - Colors are applied
# - Row counts show with k/M suffixes
# - Column indicators work
# - Empty state looks good
```

---

## Phase 3: Status Bars (1-2 hours)

### Step 3.1: Extract Status Bar Rendering (45 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/components/status_bar.go`

Create a new file:

```go
package components

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// TopBar renders the application top status bar
func TopBar(appName, version string, connected bool, connInfo string, theme theme.Theme, width int) string {
    // Left side: App name + version
    leftStyle := lipgloss.NewStyle().
        Foreground(theme.Info).
        Background(theme.Surface1).
        Bold(true).
        Padding(0, 1)

    left := leftStyle.Render(fmt.Sprintf("%s v%s", appName, version))

    // Right side: Connection status
    var right string
    if connected {
        connStyle := lipgloss.NewStyle().
            Foreground(theme.Success).
            Background(theme.Surface1).
            Padding(0, 1)
        right = connStyle.Render(" " + connInfo)
    } else {
        disconnStyle := lipgloss.NewStyle().
            Foreground(theme.Error).
            Background(theme.Surface1).
            Padding(0, 1)
        right = disconnStyle.Render("⚠ Not Connected")
    }

    // Calculate spacing
    leftWidth := lipgloss.Width(left)
    rightWidth := lipgloss.Width(right)
    spacing := width - leftWidth - rightWidth
    if spacing < 0 {
        spacing = 0
    }

    // Bar background
    barStyle := lipgloss.NewStyle().
        Background(theme.Surface1).
        Width(width)

    return barStyle.Render(left + strings.Repeat(" ", spacing) + right)
}

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
    Key  string
    Desc string
}

// BottomBar renders the application bottom status bar with keybindings
func BottomBar(keys []KeyBinding, theme theme.Theme, width int) string {
    keyStyle := lipgloss.NewStyle().
        Foreground(theme.Info).
        Bold(true)

    descStyle := lipgloss.NewStyle().
        Foreground(theme.Overlay1)

    var parts []string
    for _, kb := range keys {
        part := keyStyle.Render(kb.Key) + descStyle.Render(" " + kb.Desc)
        parts = append(parts, part)
    }

    content := strings.Join(parts, "  •  ")

    barStyle := lipgloss.NewStyle().
        Background(theme.Mantle).
        Foreground(theme.Text).
        Width(width).
        Padding(0, 1)

    return barStyle.Render(content)
}
```

### Step 3.2: Update App to Use Status Bars (30 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/app/app.go`

Update the view rendering to use the new status bar components:

```go
import (
    "github.com/rebeliceyang/lazypg/internal/ui/components"
)

// In the View() function, replace existing status bar code:

// Top bar
topBar := components.TopBar(
    "LazyPG",
    "0.1.0",
    a.state.ActiveConnection != nil,
    a.getConnectionInfo(),
    a.theme,
    a.state.Width,
)

// Bottom bar
bottomKeys := a.getContextKeys()
bottomBar := components.BottomBar(
    bottomKeys,
    a.theme,
    a.state.Width,
)

// Helper function to get connection info
func (a *App) getConnectionInfo() string {
    if a.state.ActiveConnection == nil {
        return ""
    }

    return fmt.Sprintf(
        "%s@%s:%d/%s",
        a.state.ActiveConnection.User,
        a.state.ActiveConnection.Host,
        a.state.ActiveConnection.Port,
        a.state.CurrentDatabase,
    )
}

// Helper to get context-sensitive keys
func (a *App) getContextKeys() []components.KeyBinding {
    keys := []components.KeyBinding{
        {"tab", "switch"},
        {"↑↓", "navigate"},
        {"→←", "expand/collapse"},
    }

    if a.state.ActiveConnection == nil {
        keys = append(keys, components.KeyBinding{"c", "connect"})
    } else {
        keys = append(keys, components.KeyBinding{"r", "refresh"})
    }

    keys = append(keys, components.KeyBinding{"?", "help"})
    keys = append(keys, components.KeyBinding{"q", "quit"})

    return keys
}
```

### Step 3.3: Test Status Bars (15 min)

```bash
go build -o bin/lazypg ./cmd/lazypg
./bin/lazypg

# Verify:
# - Top bar shows app name and connection status
# - Bottom bar shows relevant keybindings
# - Colors are correct
# - Connection status changes color when connected/disconnected
```

---

## Phase 4: Table View Enhancement (2-3 hours)

### Step 4.1: Create Enhanced Table Component (90 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/components/table_view_enhanced.go`

```go
package components

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/lipgloss/table"
    "github.com/rebeliceyang/lazypg/internal/ui/theme"
)

type TableViewEnhanced struct {
    theme    theme.Theme
    columns  []ColumnMeta
    rows     [][]string
    selected int
    width    int
    height   int
}

type ColumnMeta struct {
    Name       string
    Type       string
    PrimaryKey bool
    ForeignKey bool
    Nullable   bool
    Width      int
}

func NewTableViewEnhanced(columns []ColumnMeta, rows [][]string, theme theme.Theme) *TableViewEnhanced {
    return &TableViewEnhanced{
        theme:    theme,
        columns:  columns,
        rows:     rows,
        selected: 0,
    }
}

func (tv *TableViewEnhanced) View() string {
    if len(tv.rows) == 0 {
        return tv.emptyState()
    }

    // Prepare headers
    headers := make([]string, len(tv.columns))
    for i, col := range tv.columns {
        headers[i] = tv.formatHeader(col)
    }

    // Create table
    t := table.New().
        Border(lipgloss.RoundedBorder()).
        BorderStyle(lipgloss.NewStyle().Foreground(tv.theme.TableBorder)).
        Headers(headers...).
        Rows(tv.rows...).
        StyleFunc(tv.cellStyleFunc)

    content := t.String()

    // Add footer
    footer := tv.renderFooter()

    return content + "\n" + footer
}

func (tv *TableViewEnhanced) formatHeader(col ColumnMeta) string {
    label := col.Name

    // Add indicators
    indicators := []string{}

    if col.PrimaryKey {
        pkStyle := lipgloss.NewStyle().Foreground(tv.theme.Warning)
        indicators = append(indicators, pkStyle.Render("⚿"))
    }

    if col.ForeignKey {
        fkStyle := lipgloss.NewStyle().Foreground(tv.theme.Info)
        indicators = append(indicators, fkStyle.Render("→"))
    }

    if !col.Nullable {
        nnStyle := lipgloss.NewStyle().Foreground(tv.theme.Success)
        indicators = append(indicators, nnStyle.Render("*"))
    }

    if len(indicators) > 0 {
        label += " " + strings.Join(indicators, "")
    }

    return label
}

func (tv *TableViewEnhanced) cellStyleFunc(row, col int) lipgloss.Style {
    // Header row
    if row == table.HeaderRow {
        return lipgloss.NewStyle().
            Foreground(tv.theme.TableHeader).
            Background(tv.theme.Surface1).
            Bold(true).
            Align(lipgloss.Center).
            Padding(0, 1)
    }

    // Selected row
    if row == tv.selected {
        return lipgloss.NewStyle().
            Foreground(tv.theme.Foreground).
            Background(tv.theme.TableRowSelected).
            Bold(true).
            Padding(0, 1)
    }

    // Alternating rows
    baseStyle := lipgloss.NewStyle().Padding(0, 1)
    if row%2 == 0 {
        return baseStyle.
            Foreground(tv.theme.Foreground).
            Background(tv.theme.TableRowEven)
    }
    return baseStyle.
        Foreground(tv.theme.Foreground).
        Background(tv.theme.TableRowOdd)
}

func (tv *TableViewEnhanced) renderFooter() string {
    footerStyle := lipgloss.NewStyle().
        Foreground(tv.theme.Overlay1).
        Italic(true)

    info := fmt.Sprintf(
        "%s rows • %s columns",
        formatNumber(int64(len(tv.rows))),
        formatNumber(int64(len(tv.columns))),
    )

    return footerStyle.Render(info)
}

func (tv *TableViewEnhanced) emptyState() string {
    iconStyle := lipgloss.NewStyle().
        Foreground(tv.theme.Overlay1).
        Bold(true).
        Align(lipgloss.Center)

    messageStyle := lipgloss.NewStyle().
        Foreground(tv.theme.Overlay1).
        Italic(true).
        Align(lipgloss.Center)

    icon := iconStyle.Render("∅")
    message := messageStyle.Render("No data to display")

    return icon + "\n" + message
}

func formatNumber(n int64) string {
    if n < 1000 {
        return fmt.Sprintf("%d", n)
    }

    // Add thousands separators
    s := fmt.Sprintf("%d", n)
    var result []rune
    for i, c := range s {
        if i > 0 && (len(s)-i)%3 == 0 {
            result = append(result, ',')
        }
        result = append(result, c)
    }
    return string(result)
}

// Navigation methods
func (tv *TableViewEnhanced) MoveUp() {
    if tv.selected > 0 {
        tv.selected--
    }
}

func (tv *TableViewEnhanced) MoveDown() {
    if tv.selected < len(tv.rows)-1 {
        tv.selected++
    }
}

func (tv *TableViewEnhanced) GetSelected() int {
    return tv.selected
}
```

### Step 4.2: Integrate Enhanced Table (30 min)

Update the app to use the enhanced table component when displaying data.

### Step 4.3: Test Table View (30 min)

```bash
go build -o bin/lazypg ./cmd/lazypg
./bin/lazypg

# Connect to database
# Select a table
# Verify:
# - Borders appear correctly
# - Headers show indicators
# - Row selection works
# - Alternating colors visible
# - Footer displays metadata
```

---

## Phase 5: Loading & Empty States (1 hour)

### Step 5.1: Create Loading Component (30 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/components/loading.go`

```go
package components

import (
    "github.com/charmbracelet/lipgloss"
    "github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// LoadingSpinner represents an animated loading indicator
type LoadingSpinner struct {
    frames []string
    index  int
    theme  theme.Theme
}

// NewLoadingSpinner creates a new loading spinner
func NewLoadingSpinner(theme theme.Theme) *LoadingSpinner {
    return &LoadingSpinner{
        frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
        index:  0,
        theme:  theme,
    }
}

// View returns the current frame
func (ls *LoadingSpinner) View(message string) string {
    spinnerStyle := lipgloss.NewStyle().
        Foreground(ls.theme.Info).
        Bold(true)

    messageStyle := lipgloss.NewStyle().
        Foreground(ls.theme.Text)

    frame := ls.frames[ls.index%len(ls.frames)]
    return spinnerStyle.Render(frame) + " " + messageStyle.Render(message)
}

// Tick advances the spinner animation
func (ls *LoadingSpinner) Tick() {
    ls.index++
}

// EmptyState renders an empty state message
func EmptyState(icon, message string, theme theme.Theme) string {
    iconStyle := lipgloss.NewStyle().
        Foreground(theme.Overlay1).
        Bold(true).
        Align(lipgloss.Center)

    messageStyle := lipgloss.NewStyle().
        Foreground(theme.Overlay1).
        Italic(true).
        Align(lipgloss.Center)

    return iconStyle.Render(icon) + "\n" + messageStyle.Render(message)
}
```

### Step 5.2: Add Loading States (30 min)

Update components to show loading spinners during async operations.

---

## Phase 6: Help Modal (1 hour)

### Step 6.1: Create Help Component (45 min)

**File:** `/Users/rebeliceyang/Github/lazypg/internal/ui/components/help_modal.go`

```go
package components

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// HelpModal renders a full-screen help overlay
func HelpModal(width, height int, theme theme.Theme) string {
    // Calculate modal size (80% of screen)
    modalWidth := min(int(float64(width)*0.8), 100)
    modalHeight := min(int(float64(height)*0.8), 30)

    var b strings.Builder

    // Title
    titleStyle := lipgloss.NewStyle().
        Foreground(theme.Info).
        Bold(true).
        Width(modalWidth).
        Align(lipgloss.Center)
    b.WriteString(titleStyle.Render("LazyPG Help"))
    b.WriteString("\n\n")

    // Sections
    sections := []struct {
        Title string
        Keys  []KeyBinding
    }{
        {
            Title: "Navigation",
            Keys: []KeyBinding{
                {"↑/k", "Move up"},
                {"↓/j", "Move down"},
                {"→/l", "Expand / Go right"},
                {"←/h", "Collapse / Go left"},
                {"g", "Jump to top"},
                {"G", "Jump to bottom"},
            },
        },
        {
            Title: "Actions",
            Keys: []KeyBinding{
                {"enter", "Select / Execute"},
                {"space", "Toggle expand"},
                {"tab", "Switch panel"},
                {"r", "Refresh"},
                {"c", "New connection"},
            },
        },
        {
            Title: "Application",
            Keys: []KeyBinding{
                {"?", "Toggle help"},
                {"q", "Quit"},
                {"ctrl+c", "Force quit"},
            },
        },
    }

    sectionStyle := lipgloss.NewStyle().
        Foreground(theme.Mauve).
        Bold(true).
        Underline(true)

    keyStyle := lipgloss.NewStyle().
        Foreground(theme.Peach).
        Bold(true).
        Width(12)

    descStyle := lipgloss.NewStyle().
        Foreground(theme.Text)

    for i, section := range sections {
        if i > 0 {
            b.WriteString("\n")
        }
        b.WriteString(sectionStyle.Render(section.Title) + "\n")

        for _, kb := range section.Keys {
            line := "  " +
                keyStyle.Render(kb.Key) +
                descStyle.Render(kb.Desc)
            b.WriteString(line + "\n")
        }
    }

    // Footer
    footerStyle := lipgloss.NewStyle().
        Foreground(theme.Overlay1).
        Italic(true).
        Width(modalWidth).
        Align(lipgloss.Center)
    b.WriteString("\n" + footerStyle.Render("Press ? or Esc to close"))

    // Box
    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(theme.BorderFocused).
        Padding(1, 2).
        Width(modalWidth)

    content := boxStyle.Render(b.String())

    // Center on screen
    vPadding := (height - lipgloss.Height(content)) / 2
    hPadding := (width - lipgloss.Width(content)) / 2

    return strings.Repeat("\n", vPadding) +
        lipgloss.NewStyle().PaddingLeft(hPadding).Render(content)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Step 6.2: Integrate Help Modal (15 min)

Update app to show help modal when '?' is pressed.

---

## Testing Checklist

After each phase, verify:

- [ ] No compilation errors
- [ ] Colors render correctly
- [ ] Icons display properly (not boxes/question marks)
- [ ] Layout is clean and aligned
- [ ] Navigation still works
- [ ] Performance is good
- [ ] Terminal resize works

---

## Rollback Plan

If issues occur:

```bash
# Revert to previous commit
git diff HEAD
git checkout -- <file>

# Or create a branch before starting
git checkout -b ui-enhancement
# Work on changes
# If issues: git checkout main
```

---

## Expected Timeline

| Phase | Task | Time | Cumulative |
|-------|------|------|------------|
| 1 | Foundation | 2-3h | 2-3h |
| 2 | Tree View | 3-4h | 5-7h |
| 3 | Status Bars | 1-2h | 6-9h |
| 4 | Table View | 2-3h | 8-12h |
| 5 | Loading States | 1h | 9-13h |
| 6 | Help Modal | 1h | 10-14h |

**Total: 10-14 hours** for complete implementation

---

## Quick Start (Minimum Viable)

If time is limited, do these in order:
1. **Phase 1** - Foundation (colors) - 2-3h
   - Biggest visual impact
   - Low effort

2. **Phase 2 - Step 2.1-2.3** - Basic tree icons - 1-2h
   - Good improvement
   - Medium effort

3. **Phase 3** - Status bars - 1-2h
   - Professional look
   - Medium effort

**Total: 4-7 hours for 80% of visual improvement**

---

**End of Implementation Guide**
