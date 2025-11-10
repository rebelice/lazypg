# UI/UX Design Specification for LazyPG

**Version:** 1.0
**Date:** 2025-11-10
**Author:** Research & Analysis of Modern TUI/SQL Tools

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Research Findings](#research-findings)
3. [Color Palette Recommendations](#color-palette-recommendations)
4. [Layout & Spacing Guidelines](#layout--spacing-guidelines)
5. [Component Design Specifications](#component-design-specifications)
6. [Implementation Examples](#implementation-examples)
7. [Recommended Improvements](#recommended-improvements)

---

## Executive Summary

This document provides comprehensive UI/UX design guidelines for LazyPG based on analysis of modern TUI and SQL editor tools including:
- **LazyGit** - Git TUI gold standard
- **k9s** - Kubernetes TUI with excellent theming
- **pgcli** - PostgreSQL CLI with table rendering
- **TablePlus/DBeaver** - Modern SQL GUI references
- **Bubble Tea ecosystem** - Go TUI framework best practices

### Key Design Philosophy

1. **Colorful is better than colorless** - Use color to create distinction and hierarchy
2. **Balance over extremes** - Middle ground between low and high contrast
3. **Harmony over dissonance** - Complementary color relationships
4. **Information density** - Maximize information while maintaining readability
5. **Consistent spacing** - 4px/8px base unit system for terminal UIs

---

## Research Findings

### 1. LazyGit Design Analysis

**Panel Layout:**
- Multi-panel layout with 4-5 main panels (Status, Files, Branches, Commits, Stash)
- Hotkeys (1-5) for direct panel navigation
- Clear visual focus indication with border highlighting
- Vertical/horizontal split modes configurable
- Status panel at top-left showing current state

**Color Scheme:**
- Uses terminal background by default (transparent)
- Configurable theme system with YAML files
- Active border in accent color (blue/purple)
- Inactive border in muted color
- Selection backgrounds subtle (one shade darker)
- Status indicators: green (success), red (error), yellow (warning), blue (info)

**Visual Hierarchy:**
- Bold text for focused elements
- Dimmed/faint text for secondary information
- Icon usage: ▾ (expanded), ▸ (collapsed), • (leaf)
- Metadata shown in parentheses with dimmed styling

**Typography:**
- Mono-spaced font (terminal default)
- Bold for selected/focused items
- Italic rarely used
- Color used more than font weight for hierarchy

### 2. k9s Design Patterns

**Skin Configuration Structure:**
```yaml
k9s:
  body:
    fgColor: "#ffffff"
    bgColor: "default"
    logoColor: "#856cc4"

  frame:
    border:
      fgColor: "#666666"
      focusColor: "#69d9ed"
    title:
      fgColor: "#ffffff"
      bgColor: "#333333"

  table:
    fgColor: "#ffffff"
    bgColor: "default"
    cursorFgColor: "#000000"
    cursorBgColor: "#69d9ed"

  status:
    newColor: "#69d9ed"          # Blue
    modifyColor: "#856cc4"       # Purple
    addColor: "#a7e24c"          # Green
    errorColor: "#f72972"        # Magenta
    highlightColor: "#e47c20"    # Orange
    completeColor: "#a7e24c"     # Green
```

**Key Features:**
- Environment-based coloring (production=red, staging=yellow, dev=green backgrounds)
- High contrast focused cursor with inverted colors
- 256-color mode support
- Named colors and hex codes supported
- Transparent "default" for terminal background

### 3. Catppuccin Color System

The most popular modern TUI theme with 4 flavors and 26 colors each.

#### Mocha Flavor (Dark - Recommended for LazyPG)

**Base Colors:**
- `#1e1e2e` - Base (darkest background)
- `#181825` - Mantle (darker background)
- `#11111b` - Crust (UI background)
- `#313244` - Surface0 (subtle backgrounds)
- `#45475a` - Surface1 (elevated surfaces)
- `#585b70` - Surface2 (more elevated)

**Text Colors:**
- `#cdd6f4` - Text (primary text)
- `#bac2de` - Subtext1 (secondary text)
- `#a6adc8` - Subtext0 (tertiary text)
- `#9399b2` - Overlay2 (muted text)
- `#7f849c` - Overlay1 (more muted)
- `#6c7086` - Overlay0 (most muted)

**Accent Colors:**
- `#f38ba8` - Red (errors, deletions)
- `#fab387` - Peach (warnings, attention)
- `#f9e2af` - Yellow (cautions, searching)
- `#a6e3a1` - Green (success, additions)
- `#94e2d5` - Teal (info, neutral actions)
- `#89dceb` - Sky (highlights)
- `#74c7ec` - Sapphire (primary actions)
- `#89b4fa` - Blue (links, focused borders)
- `#cba6f7` - Mauve (special, keywords)
- `#f5c2e7` - Pink (strings, literals)
- `#f2cdcd` - Flamingo (secondary accents)
- `#f5e0dc` - Rosewater (tertiary accents)

#### Latte Flavor (Light - Optional)

**Base Colors:**
- `#eff1f5` - Base (lightest background)
- `#e6e9ef` - Mantle (lighter background)
- `#dce0e8` - Crust (UI background)

**Text Colors:**
- `#4c4f69` - Text (primary text)
- `#5c5f77` - Subtext1 (secondary text)
- `#6c6f85` - Subtext0 (tertiary text)

**Accent Colors:** (Same structure, different saturation for light background)
- `#d20f39` - Red
- `#fe640b` - Peach
- `#df8e1d` - Yellow
- `#40a02b` - Green
- `#179299` - Teal
- `#04a5e5` - Sky
- `#209fb5` - Sapphire
- `#1e66f5` - Blue
- `#8839ef` - Mauve
- `#ea76cb` - Pink

### 4. pgcli Table Rendering

**Table Formats Supported:**
- `psql` - PostgreSQL default (recommended)
- `grid` - ASCII grid with borders
- `fancy_grid` - Unicode box-drawing characters
- `simple` - Minimal, no borders
- `vertical` - Key-value pairs (for wide tables)

**Features:**
- Auto-switch to vertical when exceeding terminal width
- Configurable row limit prompts
- Syntax highlighting for data types
- Smart pager selection (less/more)
- Color-coded NULL values
- Multi-line cell support

**Column Alignment:**
- Text: Left-aligned
- Numbers: Right-aligned (for decimal comparison)
- Boolean: Center-aligned
- NULL: Dimmed/italic

### 5. Bubble Tea / Lip Gloss Patterns

**Border Styles:**
```go
// Normal Border (most common)
lipgloss.NormalBorder()
// ┌──────┐
// │      │
// └──────┘

// Rounded Border (modern look)
lipgloss.RoundedBorder()
// ╭──────╮
// │      │
// ╰──────╯

// Thick Border (emphasis)
lipgloss.ThickBorder()
// ┏━━━━━━┓
// ┃      ┃
// ┗━━━━━━┛

// Double Border (special panels)
lipgloss.DoubleBorder()
// ╔══════╗
// ║      ║
// ╚══════╝
```

**Table Rendering Pattern:**
```go
var (
    // Color definitions
    purple    = lipgloss.Color("99")
    gray      = lipgloss.Color("245")
    lightGray = lipgloss.Color("241")

    // Header style
    headerStyle = lipgloss.NewStyle().
        Foreground(purple).
        Bold(true).
        Align(lipgloss.Center)

    // Cell styles
    cellStyle    = lipgloss.NewStyle().Padding(0, 1).Width(14)
    oddRowStyle  = cellStyle.Foreground(gray)
    evenRowStyle = cellStyle.Foreground(lightGray)
)

t := table.New().
    Border(lipgloss.NormalBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(purple)).
    StyleFunc(func(row, col int) lipgloss.Style {
        switch {
        case row == table.HeaderRow:
            return headerStyle
        case row%2 == 0:
            return evenRowStyle
        default:
            return oddRowStyle
        }
    }).
    Headers("COLUMN1", "COLUMN2", "COLUMN3").
    Rows(rows...)
```

---

## Color Palette Recommendations

### Primary Palette (Catppuccin Mocha-Inspired)

Based on research, here's the recommended color scheme for LazyPG:

```go
// theme/catppuccin_mocha.go
package theme

import "github.com/charmbracelet/lipgloss"

func CatppuccinMocha() Theme {
    return Theme{
        Name: "catppuccin-mocha",

        // Base colors
        Background: lipgloss.Color("#1e1e2e"),  // Base
        Foreground: lipgloss.Color("#cdd6f4"),  // Text

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

        // Syntax highlighting (SQL)
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

        // Database-specific colors
        PrimaryKey:    lipgloss.Color("#f9e2af"),  // Yellow
        ForeignKey:    lipgloss.Color("#74c7ec"),  // Sapphire
        Nullable:      lipgloss.Color("#9399b2"),  // Overlay2
        NotNull:       lipgloss.Color("#a6e3a1"),  // Green
        Index:         lipgloss.Color("#cba6f7"),  // Mauve

        // JSONB colors
        JSONKey:     lipgloss.Color("#89b4fa"),  // Blue
        JSONString:  lipgloss.Color("#f5c2e7"),  // Pink
        JSONNumber:  lipgloss.Color("#fab387"),  // Peach
        JSONBoolean: lipgloss.Color("#a6e3a1"),  // Green
        JSONNull:    lipgloss.Color("#6c7086"),  // Overlay0

        // Special UI elements
        ActiveDatabase: lipgloss.Color("#a6e3a1"),  // Green
        InactiveDatabase: lipgloss.Color("#6c7086"),  // Overlay0
        RowCount: lipgloss.Color("#94e2d5"),  // Teal
        ColumnType: lipgloss.Color("#f2cdcd"),  // Flamingo
    }
}
```

### Alternative Palette (Monokai-Inspired for k9s users)

```go
func MonokaiTheme() Theme {
    return Theme{
        Name: "monokai",

        Background: lipgloss.Color("#272822"),  // Monokai dark
        Foreground: lipgloss.Color("#f8f8f2"),  // Monokai white

        Border:        lipgloss.Color("#75715e"),  // Comment gray
        BorderFocused: lipgloss.Color("#66d9ef"),  // Cyan
        Selection:     lipgloss.Color("#49483e"),  // Selection
        Cursor:        lipgloss.Color("#f92672"),  // Pink

        Success: lipgloss.Color("#a6e22e"),  // Green
        Warning: lipgloss.Color("#e6db74"),  // Yellow
        Error:   lipgloss.Color("#f92672"),  // Pink
        Info:    lipgloss.Color("#66d9ef"),  // Cyan

        Keyword:  lipgloss.Color("#f92672"),  // Pink
        String:   lipgloss.Color("#e6db74"),  // Yellow
        Number:   lipgloss.Color("#ae81ff"),  // Purple
        Comment:  lipgloss.Color("#75715e"),  // Gray
        Function: lipgloss.Color("#a6e22e"),  // Green
        Operator: lipgloss.Color("#f8f8f2"),  // White
    }
}
```

### Current LazyPG Colors (Analysis)

```go
// Current implementation uses ANSI 256 colors
Background: lipgloss.Color("235"),  // #262626 (close to Catppuccin)
Foreground: lipgloss.Color("252"),  // #d0d0d0 (good)
Border:     lipgloss.Color("240"),  // #585858 (reasonable)
BorderFocused: lipgloss.Color("62"), // #5f5fd7 (blue, good choice)
Selection:  lipgloss.Color("237"),  // #3a3a3a (good)

// Status colors are decent
Success: lipgloss.Color("42"),   // #00d787 (bright green, good)
Warning: lipgloss.Color("220"),  // #ffd700 (gold, good)
Error:   lipgloss.Color("196"),  // #ff0000 (bright red, good)
Info:    lipgloss.Color("75"),   // #5fafff (blue, good)
```

**Assessment:** Current colors are functional but could be enhanced with a cohesive palette like Catppuccin for better visual harmony.

---

## Layout & Spacing Guidelines

### Terminal UI Spacing System

Based on research, use a **4px base unit** for terminal UIs:

```
Spacing Scale:
- 0px  - No space (flush elements)
- 2px  - Minimal space (dense lists)
- 4px  - Compact space (related items)
- 8px  - Standard space (component padding)
- 12px - Medium space (component margins)
- 16px - Large space (section separation)
- 24px - Extra large (major sections)
- 32px - Huge (layout separation)
```

**Terminal Character Equivalents:**
- 1 space = ~1 character width
- Standard padding = 2 spaces (8px equivalent)
- Panel margins = 1 space (4px equivalent)

### Panel Layout (LazyGit-inspired)

```
┌─ LazyPG ─────────────────────────────────────┬─ Connection: postgres@localhost ─┐
│                                              │                                   │
│  ┌─ Database Navigator ──────┐  ┌─ Data ───────────────────────────────────┐  │
│  │                            │  │                                          │  │
│  │  ▾ postgres (active)       │  │  Table: users                            │  │
│  │    ▾ public                │  │                                          │  │
│  │      ▸ tables (15)         │  │  ┌──────┬─────────┬──────────┬────────┐ │  │
│  │      ▸ views (3)           │  │  │ id   │ name    │ email    │ active │ │  │
│  │      ▸ functions (8)       │  │  ├──────┼─────────┼──────────┼────────┤ │  │
│  │    ▸ pg_catalog            │  │  │ 1    │ Alice   │ a@ex.com │ true   │ │  │
│  │  ▸ template1               │  │  │ 2    │ Bob     │ b@ex.com │ false  │ │  │
│  │  ▸ template0               │  │  └──────┴─────────┴──────────┴────────┘ │  │
│  │                            │  │                                          │  │
│  │                            │  │  1,234 rows • 4 columns • 128 KB         │  │
│  └────────────────────────────┘  └──────────────────────────────────────────┘  │
│                                                                                  │
│  [tab] switch • [↑↓] navigate • [→←] expand/collapse • [?] help • [q] quit     │
└──────────────────────────────────────────────────────────────────────────────────┘
```

**Layout Specifications:**

1. **Top Bar (Status Bar)**
   - Height: 1 line
   - Padding: 1 space left/right
   - Background: Slightly darker than base
   - Left: App name + version
   - Right: Connection status + current database

2. **Main Content Area**
   - Two panels: Navigator (left) + Data (right)
   - Left panel width: 25-35% (configurable, min 20, max 50)
   - Right panel width: Remaining space
   - Gap between panels: 1 space
   - Vertical padding: 1 line top/bottom

3. **Bottom Bar (Status/Help)**
   - Height: 1 line
   - Padding: 1 space left/right
   - Background: Same as top bar
   - Shows context-sensitive keybindings

4. **Panel Borders**
   - Border style: Rounded (modern) or Normal (classic)
   - Border color: Dim when inactive, accent when focused
   - Title: Centered or left-aligned with 1 space padding
   - Title color: Foreground (normal) or accent (focused)

### Information Density Guidelines

**Database Navigator:**
- Indent per level: 2 spaces
- Icon width: 1 character + 1 space
- Metadata: Parentheses with dimmed color
- Row spacing: None (compact list)
- Empty state: Centered, italic, dimmed

**Data Table:**
- Header: Bold, accent color, centered or left-aligned
- Row height: 1 line (no padding between rows)
- Column padding: 1 space left/right
- Selected row: Full-width highlight, bold text
- Alternating rows: Optional (slight background difference)
- Borders: Grid style for tables, optional for lists

**Metadata Display:**
- Row count: Bottom right, dimmed
- Column types: Small font or dimmed
- Primary key: PK indicator in yellow/warning color
- Nullable: Dimmed or gray
- Foreign key: FK indicator in blue/info color

---

## Component Design Specifications

### 1. Tree View Component (Database Navigator)

**Current Implementation Analysis:**
- Good: Icon usage (▾▸•), cursor handling, viewport scrolling
- Improve: Add more visual hierarchy, better metadata display

**Recommended Enhancements:**

```go
// Tree node icons with color coding
func (tv *TreeView) getNodeIcon(node *models.TreeNode) string {
    iconStyle := lipgloss.NewStyle()

    switch node.Type {
    case models.TreeNodeTypeDatabase:
        if isActive {
            iconStyle = iconStyle.Foreground(tv.Theme.Success)
            return iconStyle.Render("●")  // Filled circle for active
        }
        return iconStyle.Foreground(tv.Theme.Overlay1).Render("○")  // Empty circle

    case models.TreeNodeTypeSchema:
        iconStyle = iconStyle.Foreground(tv.Theme.Info)
        if node.Expanded {
            return iconStyle.Render("▾")
        }
        return iconStyle.Render("▸")

    case models.TreeNodeTypeTable:
        iconStyle = iconStyle.Foreground(tv.Theme.Mauve)
        return iconStyle.Render("▦")  // Table icon

    case models.TreeNodeTypeView:
        iconStyle = iconStyle.Foreground(tv.Theme.Teal)
        return iconStyle.Render("▤")  // View icon

    case models.TreeNodeTypeFunction:
        iconStyle = iconStyle.Foreground(tv.Theme.Peach)
        return iconStyle.Render("ƒ")  // Function icon

    case models.TreeNodeTypeColumn:
        return "  •"  // Bullet for columns
    }

    // Default
    if node.Expanded {
        return "▾"
    }
    return "▸"
}
```

**Metadata Display:**
```go
func (tv *TreeView) buildNodeLabel(node *models.TreeNode) string {
    label := node.Label
    dimStyle := lipgloss.NewStyle().Foreground(tv.Theme.Overlay1)

    switch node.Type {
    case models.TreeNodeTypeDatabase:
        if isActive {
            activeStyle := lipgloss.NewStyle().
                Foreground(tv.Theme.Success).
                Bold(true)
            label += " " + activeStyle.Render("●")
        }

    case models.TreeNodeTypeSchema:
        if node.Loaded {
            count := len(node.Children)
            if count == 0 {
                label += " " + dimStyle.Render("∅")  // Empty set symbol
            } else {
                label += " " + dimStyle.Render(fmt.Sprintf("(%d)", count))
            }
        } else {
            label += " " + dimStyle.Render("…")  // Loading
        }

    case models.TreeNodeTypeTable:
        if rowCount, ok := getRowCount(node); ok {
            countStyle := lipgloss.NewStyle().Foreground(tv.Theme.RowCount)
            label += " " + countStyle.Render(formatRowCount(rowCount))
        }

    case models.TreeNodeTypeColumn:
        // Show data type
        if colInfo, ok := node.Metadata.(models.ColumnInfo); ok {
            typeStyle := lipgloss.NewStyle().Foreground(tv.Theme.ColumnType)
            label += " " + typeStyle.Render(colInfo.DataType)

            // Add indicators
            indicators := []string{}
            if colInfo.PrimaryKey {
                pkStyle := lipgloss.NewStyle().Foreground(tv.Theme.Warning)
                indicators = append(indicators, pkStyle.Render("PK"))
            }
            if colInfo.ForeignKey {
                fkStyle := lipgloss.NewStyle().Foreground(tv.Theme.ForeignKey)
                indicators = append(indicators, fkStyle.Render("FK"))
            }
            if !colInfo.Nullable {
                nnStyle := lipgloss.NewStyle().Foreground(tv.Theme.NotNull)
                indicators = append(indicators, nnStyle.Render("NOT NULL"))
            }
            if colInfo.Indexed {
                idxStyle := lipgloss.NewStyle().Foreground(tv.Theme.Index)
                indicators = append(indicators, idxStyle.Render("IDX"))
            }

            if len(indicators) > 0 {
                label += " " + strings.Join(indicators, " ")
            }
        }
    }

    return label
}
```

### 2. Table View Component (Data Display)

**Design Specification:**

```go
type TableView struct {
    Columns []ColumnDef
    Rows    [][]string

    // Display settings
    Width           int
    Height          int
    Theme           theme.Theme

    // State
    SelectedRow     int
    SelectedCol     int
    ScrollOffsetX   int
    ScrollOffsetY   int

    // Features
    AlternatingRows bool
    ShowBorders     bool
    ShowHeader      bool
    ShowFooter      bool
}

type ColumnDef struct {
    Name      string
    DataType  string
    Width     int
    Alignment lipgloss.Position  // Left, Center, Right
    Nullable  bool
    PrimaryKey bool
    ForeignKey bool
}

func (tv *TableView) View() string {
    var b strings.Builder

    // Render header
    if tv.ShowHeader {
        b.WriteString(tv.renderHeader())
        b.WriteString("\n")
    }

    // Render separator
    if tv.ShowBorders {
        b.WriteString(tv.renderSeparator())
        b.WriteString("\n")
    }

    // Render rows
    visibleRows := tv.getVisibleRows()
    for i, row := range visibleRows {
        rowIndex := tv.ScrollOffsetY + i
        selected := rowIndex == tv.SelectedRow
        b.WriteString(tv.renderRow(row, selected, rowIndex%2 == 0))
        b.WriteString("\n")
    }

    // Render footer
    if tv.ShowFooter {
        b.WriteString(tv.renderFooter())
    }

    return b.String()
}

func (tv *TableView) renderHeader() string {
    headerStyle := lipgloss.NewStyle().
        Foreground(tv.Theme.TableHeader).
        Bold(true).
        Background(tv.Theme.Background)

    cells := make([]string, len(tv.Columns))
    for i, col := range tv.Columns {
        cellStyle := headerStyle.
            Width(col.Width).
            Align(col.Alignment)

        // Add indicators
        label := col.Name
        if col.PrimaryKey {
            label += " ⚿"  // Key symbol
        }
        if col.ForeignKey {
            label += " →"  // Arrow
        }
        if !col.Nullable {
            label += " *"  // Required
        }

        cells[i] = cellStyle.Render(label)
    }

    if tv.ShowBorders {
        return "│ " + strings.Join(cells, " │ ") + " │"
    }
    return strings.Join(cells, "  ")
}

func (tv *TableView) renderRow(row []string, selected bool, even bool) string {
    var rowStyle lipgloss.Style

    if selected {
        rowStyle = lipgloss.NewStyle().
            Background(tv.Theme.TableRowSelected).
            Foreground(tv.Theme.Foreground).
            Bold(true)
    } else if even && tv.AlternatingRows {
        rowStyle = lipgloss.NewStyle().
            Background(tv.Theme.TableRowEven).
            Foreground(tv.Theme.Foreground)
    } else {
        rowStyle = lipgloss.NewStyle().
            Background(tv.Theme.TableRowOdd).
            Foreground(tv.Theme.Foreground)
    }

    cells := make([]string, len(tv.Columns))
    for i, col := range tv.Columns {
        value := row[i]

        // Apply data-type specific styling
        cellStyle := rowStyle.
            Width(col.Width).
            Align(col.Alignment)

        // Special rendering for NULL
        if value == "NULL" {
            cellStyle = cellStyle.
                Foreground(tv.Theme.JSONNull).
                Italic(true)
        } else if col.DataType == "boolean" {
            // Color-code booleans
            if value == "true" {
                cellStyle = cellStyle.Foreground(tv.Theme.Success)
            } else {
                cellStyle = cellStyle.Foreground(tv.Theme.Error)
            }
        } else if isNumeric(col.DataType) {
            // Right-align numbers
            cellStyle = cellStyle.Align(lipgloss.Right)
        }

        cells[i] = cellStyle.Render(value)
    }

    if tv.ShowBorders {
        return "│ " + strings.Join(cells, " │ ") + " │"
    }
    return strings.Join(cells, "  ")
}

func (tv *TableView) renderFooter() string {
    footerStyle := lipgloss.NewStyle().
        Foreground(tv.Theme.Overlay1).
        Italic(true)

    totalRows := len(tv.Rows)
    visibleRows := tv.Height - 2 // Minus header and separator

    info := fmt.Sprintf(
        "%s rows • %s columns • showing %d-%d",
        formatNumber(int64(totalRows)),
        formatNumber(int64(len(tv.Columns))),
        tv.ScrollOffsetY+1,
        min(tv.ScrollOffsetY+visibleRows, totalRows),
    )

    return footerStyle.Render(info)
}
```

### 3. Status Bar Component

**Top Status Bar:**
```go
func renderTopBar(appState AppState, theme Theme, width int) string {
    // Left side: App name + version
    leftStyle := lipgloss.NewStyle().
        Foreground(theme.Info).
        Background(theme.Surface1).
        Bold(true).
        Padding(0, 1)
    left := leftStyle.Render("LazyPG v0.1.0")

    // Right side: Connection status
    var right string
    if appState.ActiveConnection != nil {
        connStyle := lipgloss.NewStyle().
            Foreground(theme.Success).
            Background(theme.Surface1)

        connInfo := fmt.Sprintf(
            "%s@%s/%s",
            appState.ActiveConnection.User,
            appState.ActiveConnection.Host,
            appState.CurrentDatabase,
        )
        right = connStyle.Render(" " + connInfo)
    } else {
        disconnStyle := lipgloss.NewStyle().
            Foreground(theme.Error).
            Background(theme.Surface1)
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

    return barStyle.Render(
        left + strings.Repeat(" ", spacing) + right,
    )
}
```

**Bottom Status Bar:**
```go
func renderBottomBar(contextKeys []KeyBinding, theme Theme, width int) string {
    keyStyle := lipgloss.NewStyle().
        Foreground(theme.Info).
        Bold(true)

    descStyle := lipgloss.NewStyle().
        Foreground(theme.Overlay1)

    var parts []string
    for _, kb := range contextKeys {
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

### 4. Help Overlay

**Full-screen Help Modal:**
```go
func renderHelpOverlay(width, height int, theme Theme) string {
    // Calculate modal size (80% of screen, max 100x30)
    modalWidth := min(int(float64(width)*0.8), 100)
    modalHeight := min(int(float64(height)*0.8), 30)

    // Title
    titleStyle := lipgloss.NewStyle().
        Foreground(theme.Info).
        Bold(true).
        Width(modalWidth).
        Align(lipgloss.Center)
    title := titleStyle.Render("LazyPG Help")

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

    var content strings.Builder
    content.WriteString(title + "\n\n")

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
            content.WriteString("\n")
        }
        content.WriteString(sectionStyle.Render(section.Title) + "\n")

        for _, kb := range section.Keys {
            line := "  " +
                keyStyle.Render(kb.Key) +
                descStyle.Render(kb.Desc)
            content.WriteString(line + "\n")
        }
    }

    // Add footer
    footerStyle := lipgloss.NewStyle().
        Foreground(theme.Overlay1).
        Italic(true).
        Width(modalWidth).
        Align(lipgloss.Center)
    content.WriteString("\n" + footerStyle.Render("Press ? or Esc to close"))

    // Box style
    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(theme.BorderFocused).
        Padding(1, 2).
        Width(modalWidth).
        Height(modalHeight)

    modal := boxStyle.Render(content.String())

    // Center on screen
    return centerModal(modal, width, height, theme)
}
```

### 5. Loading States

```go
type LoadingSpinner struct {
    frames []string
    index  int
    theme  Theme
}

func NewLoadingSpinner(theme Theme) *LoadingSpinner {
    return &LoadingSpinner{
        frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
        index:  0,
        theme:  theme,
    }
}

func (ls *LoadingSpinner) View() string {
    style := lipgloss.NewStyle().
        Foreground(ls.theme.Info).
        Bold(true)

    frame := ls.frames[ls.index%len(ls.frames)]
    return style.Render(frame + " Loading...")
}

func (ls *LoadingSpinner) Tick() {
    ls.index++
}

// Usage in empty state
func renderLoadingState(message string, theme Theme) string {
    spinner := NewLoadingSpinner(theme)

    style := lipgloss.NewStyle().
        Foreground(theme.Info).
        Italic(true).
        Align(lipgloss.Center)

    return style.Render(spinner.View() + "\n" + message)
}
```

### 6. Empty States

```go
func renderEmptyState(context string, theme Theme) string {
    var icon, message string

    switch context {
    case "no_connection":
        icon = "⚠"
        message = "No database connection\nPress 'c' to connect"
    case "no_databases":
        icon = "∅"
        message = "No databases found"
    case "no_tables":
        icon = "▢"
        message = "No tables in this schema"
    case "no_results":
        icon = "∅"
        message = "Query returned no results"
    default:
        icon = "?"
        message = "No data available"
    }

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

---

## Implementation Examples

### Example 1: Enhanced Tree View with Icons

```go
// File: internal/ui/components/tree_view_enhanced.go
package components

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/rebeliceyang/lazypg/internal/models"
    "github.com/rebeliceyang/lazypg/internal/ui/theme"
)

const (
    // Icons
    IconDatabase        = "◉"  // Filled circle
    IconDatabaseActive  = "●"  // Filled dot
    IconSchema          = "▾"  // Triangle down
    IconSchemaCollapsed = "▸"  // Triangle right
    IconTable           = "▦"  // Table
    IconView            = "▤"  // View
    IconFunction        = "ƒ"  // Function
    IconColumn          = "•"  // Bullet
    IconPrimaryKey      = "⚿"  // Key
    IconForeignKey      = "→"  // Arrow
    IconIndex           = "⚡"  // Lightning
    IconNotNull         = "*"  // Asterisk
    IconNullable        = "○"  // Circle
    IconEmpty           = "∅"  // Empty set
    IconLoading         = "…"  // Ellipsis
)

func (tv *TreeView) renderNodeEnhanced(node *models.TreeNode, selected bool) string {
    // Calculate indentation
    depth := node.GetDepth() - 1
    if depth < 0 {
        depth = 0
    }

    // Build components
    indent := strings.Repeat("  ", depth)
    icon := tv.getEnhancedIcon(node)
    label := tv.getEnhancedLabel(node)

    // Combine
    content := indent + icon + " " + label

    // Truncate if needed
    maxWidth := tv.Width - 2
    if lipgloss.Width(content) > maxWidth {
        content = truncateString(content, maxWidth-1) + "…"
    }

    // Apply styling
    if selected {
        return tv.styleSelectedNode(content, maxWidth)
    }
    return tv.styleNormalNode(content, maxWidth)
}

func (tv *TreeView) getEnhancedIcon(node *models.TreeNode) string {
    iconStyle := lipgloss.NewStyle()

    switch node.Type {
    case models.TreeNodeTypeDatabase:
        if tv.isActiveDatabase(node) {
            return iconStyle.
                Foreground(tv.Theme.Success).
                Bold(true).
                Render(IconDatabaseActive)
        }
        return iconStyle.
            Foreground(tv.Theme.Overlay1).
            Render(IconDatabase)

    case models.TreeNodeTypeSchema:
        color := tv.Theme.Info
        if node.Expanded {
            return iconStyle.Foreground(color).Render(IconSchema)
        }
        return iconStyle.Foreground(color).Render(IconSchemaCollapsed)

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
        return "  " + iconStyle.
            Foreground(tv.Theme.Overlay1).
            Render(IconColumn)
    }

    return "?"
}

func (tv *TreeView) getEnhancedLabel(node *models.TreeNode) string {
    label := node.Label

    switch node.Type {
    case models.TreeNodeTypeDatabase:
        if tv.isActiveDatabase(node) {
            activeStyle := lipgloss.NewStyle().
                Foreground(tv.Theme.Success).
                Bold(true)
            return activeStyle.Render(label)
        }
        return label

    case models.TreeNodeTypeSchema:
        dimStyle := lipgloss.NewStyle().Foreground(tv.Theme.Overlay1)
        if !node.Loaded {
            return label + " " + dimStyle.Render(IconLoading)
        }
        count := len(node.Children)
        if count == 0 {
            return label + " " + dimStyle.Render(IconEmpty)
        }
        return label + " " + dimStyle.Render(fmt.Sprintf("(%d)", count))

    case models.TreeNodeTypeTable:
        if rowCount, ok := tv.getRowCount(node); ok {
            countStyle := lipgloss.NewStyle().Foreground(tv.Theme.Teal)
            return label + " " + countStyle.Render(tv.formatRowCount(rowCount))
        }
        return label

    case models.TreeNodeTypeColumn:
        return tv.renderColumnLabel(node)
    }

    return label
}

func (tv *TreeView) renderColumnLabel(node *models.TreeNode) string {
    colInfo, ok := node.Metadata.(models.ColumnInfo)
    if !ok {
        return node.Label
    }

    label := node.Label

    // Add data type
    typeStyle := lipgloss.NewStyle().Foreground(tv.Theme.ColumnType)
    label += " " + typeStyle.Render(colInfo.DataType)

    // Add indicators
    indicators := []string{}

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

func (tv *TreeView) formatRowCount(count int64) string {
    if count < 1000 {
        return fmt.Sprintf("%d", count)
    } else if count < 10000 {
        return fmt.Sprintf("%.1fk", float64(count)/1000)
    } else if count < 1000000 {
        return fmt.Sprintf("%.0fk", float64(count)/1000)
    } else {
        return fmt.Sprintf("%.1fM", float64(count)/1000000)
    }
}
```

### Example 2: Professional Table Rendering

```go
// File: internal/ui/components/table_view_enhanced.go
package components

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/lipgloss/table"
)

type TableViewEnhanced struct {
    table    *table.Table
    theme    theme.Theme
    data     TableData
    selected int
}

type TableData struct {
    Columns []ColumnMeta
    Rows    [][]string
}

type ColumnMeta struct {
    Name       string
    Type       string
    PrimaryKey bool
    ForeignKey bool
    Nullable   bool
    Width      int
}

func NewTableViewEnhanced(data TableData, theme theme.Theme) *TableViewEnhanced {
    tv := &TableViewEnhanced{
        theme:    theme,
        data:     data,
        selected: 0,
    }

    tv.buildTable()
    return tv
}

func (tv *TableViewEnhanced) buildTable() {
    // Prepare headers with metadata
    headers := make([]string, len(tv.data.Columns))
    for i, col := range tv.data.Columns {
        headers[i] = tv.formatColumnHeader(col)
    }

    // Create table
    tv.table = table.New().
        Border(lipgloss.RoundedBorder()).
        BorderStyle(lipgloss.NewStyle().Foreground(tv.theme.TableBorder)).
        Headers(headers...).
        Rows(tv.data.Rows...).
        StyleFunc(tv.cellStyleFunc)
}

func (tv *TableViewEnhanced) formatColumnHeader(col ColumnMeta) string {
    headerStyle := lipgloss.NewStyle().
        Foreground(tv.theme.TableHeader).
        Bold(true)

    label := col.Name

    // Add indicators
    if col.PrimaryKey {
        pkStyle := lipgloss.NewStyle().Foreground(tv.theme.Warning)
        label += " " + pkStyle.Render("⚿")
    }
    if col.ForeignKey {
        fkStyle := lipgloss.NewStyle().Foreground(tv.theme.Info)
        label += " " + fkStyle.Render("→")
    }
    if !col.Nullable {
        nnStyle := lipgloss.NewStyle().Foreground(tv.theme.Success)
        label += " " + nnStyle.Render("*")
    }

    return headerStyle.Render(label)
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

func (tv *TableViewEnhanced) View() string {
    content := tv.table.String()

    // Add footer with metadata
    footer := tv.renderFooter()

    return content + "\n" + footer
}

func (tv *TableViewEnhanced) renderFooter() string {
    footerStyle := lipgloss.NewStyle().
        Foreground(tv.theme.Overlay1).
        Italic(true)

    info := fmt.Sprintf(
        "%s rows • %s columns",
        formatNumber(int64(len(tv.data.Rows))),
        formatNumber(int64(len(tv.data.Columns))),
    )

    return footerStyle.Render(info)
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
```

### Example 3: Connection Dialog with Modern Styling

```go
// File: internal/ui/components/connection_dialog_enhanced.go
package components

import (
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/lipgloss"
)

type ConnectionDialogEnhanced struct {
    inputs []textinput.Model
    theme  theme.Theme
    width  int
    height int

    focused int
}

func NewConnectionDialogEnhanced(theme theme.Theme) *ConnectionDialogEnhanced {
    // Create input fields
    inputs := make([]textinput.Model, 5)

    // Host
    inputs[0] = textinput.New()
    inputs[0].Placeholder = "localhost"
    inputs[0].Focus()
    inputs[0].CharLimit = 256
    inputs[0].Width = 40

    // Port
    inputs[1] = textinput.New()
    inputs[1].Placeholder = "5432"
    inputs[1].CharLimit = 5
    inputs[1].Width = 40

    // Database
    inputs[2] = textinput.New()
    inputs[2].Placeholder = "postgres"
    inputs[2].CharLimit = 128
    inputs[2].Width = 40

    // Username
    inputs[3] = textinput.New()
    inputs[3].Placeholder = "postgres"
    inputs[3].CharLimit = 128
    inputs[3].Width = 40

    // Password
    inputs[4] = textinput.New()
    inputs[4].Placeholder = "password"
    inputs[4].EchoMode = textinput.EchoPassword
    inputs[4].EchoCharacter = '•'
    inputs[4].CharLimit = 256
    inputs[4].Width = 40

    return &ConnectionDialogEnhanced{
        inputs:  inputs,
        theme:   theme,
        focused: 0,
    }
}

func (cd *ConnectionDialogEnhanced) View() string {
    var b strings.Builder

    // Title
    titleStyle := lipgloss.NewStyle().
        Foreground(cd.theme.Info).
        Bold(true).
        Width(48).
        Align(lipgloss.Center)
    b.WriteString(titleStyle.Render("🔌 New Connection"))
    b.WriteString("\n\n")

    // Labels
    labels := []string{
        "Host:",
        "Port:",
        "Database:",
        "Username:",
        "Password:",
    }

    labelStyle := lipgloss.NewStyle().
        Foreground(cd.theme.Text).
        Bold(true).
        Width(12)

    for i, label := range labels {
        // Label
        b.WriteString(labelStyle.Render(label))
        b.WriteString(" ")

        // Input field
        inputStyle := lipgloss.NewStyle().
            BorderStyle(lipgloss.RoundedBorder()).
            BorderForeground(cd.theme.Border)

        if i == cd.focused {
            inputStyle = inputStyle.
                BorderForeground(cd.theme.BorderFocused)
        }

        b.WriteString(inputStyle.Render(cd.inputs[i].View()))
        b.WriteString("\n")
    }

    // Instructions
    b.WriteString("\n")
    instructStyle := lipgloss.NewStyle().
        Foreground(cd.theme.Overlay1).
        Italic(true).
        Width(48).
        Align(lipgloss.Center)
    b.WriteString(instructStyle.Render("↑↓ navigate • enter connect • esc cancel"))

    // Box
    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cd.theme.BorderFocused).
        Padding(1, 2).
        Width(52)

    return boxStyle.Render(b.String())
}
```

---

## Recommended Improvements

### 1. Immediate Improvements (High Impact, Low Effort)

**A. Adopt Catppuccin Mocha Color Palette**
- Replace current ANSI 256 colors with Catppuccin hex codes
- Provides better visual harmony and modern aesthetic
- Easy to implement - just update theme/default.go

**B. Enhanced Tree Icons**
- Use Unicode symbols for different node types
- Color-code icons by type (database=green, table=purple, etc.)
- Add visual indicators for active database

**C. Improve Metadata Display**
- Format row counts with k/M suffixes
- Show column indicators (PK, FK, NOT NULL, IDX)
- Use dimmed colors for secondary information

**D. Better Empty States**
- Add icons and helpful messages
- Suggest next actions
- Use italic, centered text

### 2. Medium-term Improvements

**A. Table View Enhancements**
- Implement proper table component with borders
- Add alternating row colors
- Right-align numeric columns
- Color-code data types (NULL=gray, boolean=green/red, etc.)

**B. Loading States**
- Add spinner animations
- Show progress indicators for long operations
- Provide cancellation option

**C. Status Bar Improvements**
- Context-sensitive keybindings display
- Connection status with color coding
- Current operation feedback

**D. Help System**
- Full-screen modal help
- Categorized keybindings
- Searchable help text

### 3. Long-term Improvements

**A. Theme System**
- Multiple theme support (Catppuccin all flavors, Monokai, Dracula, etc.)
- User-configurable themes via YAML
- Light/dark mode toggle

**B. Advanced Table Features**
- Column resizing
- Horizontal scrolling for wide tables
- Cell selection and copy
- Vertical layout for very wide tables

**C. Query Editor**
- Syntax highlighting for SQL
- Auto-completion
- Query history
- Multi-line editing

**D. Data Visualization**
- Simple charts for numeric data
- JSONB tree viewer
- Array/hstore visualizer

---

## Migration Guide

### Phase 1: Update Color Theme

```bash
# 1. Create new theme file
touch internal/ui/theme/catppuccin_mocha.go

# 2. Add Catppuccin Mocha colors (see palette above)

# 3. Update default theme to use Catppuccin
# Edit internal/ui/theme/default.go
```

### Phase 2: Enhance Tree View

```bash
# 1. Update tree_view.go with enhanced icons
# 2. Add color-coded icons
# 3. Improve metadata formatting
# 4. Test with real database connections
```

### Phase 3: Improve Table Rendering

```bash
# 1. Switch to lipgloss table component
# 2. Add proper borders and headers
# 3. Implement row selection styling
# 4. Add footer with metadata
```

### Phase 4: Status Bars

```bash
# 1. Extract status bar rendering to separate component
# 2. Add connection status indicator
# 3. Implement context-sensitive help
# 4. Add visual feedback for operations
```

---

## Code Examples for LazyPG

### Update Theme File

```go
// File: internal/ui/theme/catppuccin_mocha.go
package theme

import "github.com/charmbracelet/lipgloss"

func CatppuccinMocha() Theme {
    return Theme{
        Name: "catppuccin-mocha",

        // Base colors
        Background: lipgloss.Color("#1e1e2e"),
        Foreground: lipgloss.Color("#cdd6f4"),

        // Surface colors
        Mantle:   lipgloss.Color("#181825"),
        Crust:    lipgloss.Color("#11111b"),
        Surface0: lipgloss.Color("#313244"),
        Surface1: lipgloss.Color("#45475a"),
        Surface2: lipgloss.Color("#585b70"),

        // Text colors
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

### Update Theme Structure

```go
// File: internal/ui/theme/theme.go
package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
    Name string

    // Base colors
    Background lipgloss.Color
    Foreground lipgloss.Color

    // Surface layers (Catppuccin)
    Mantle   lipgloss.Color
    Crust    lipgloss.Color
    Surface0 lipgloss.Color
    Surface1 lipgloss.Color
    Surface2 lipgloss.Color

    // Text hierarchy
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

    // Full accent palette
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
    TableBorder      lipgloss.Color

    // JSONB colors
    JSONKey     lipgloss.Color
    JSONString  lipgloss.Color
    JSONNumber  lipgloss.Color
    JSONBoolean lipgloss.Color
    JSONNull    lipgloss.Color
}
```

---

## Conclusion

This design specification provides a comprehensive guide for modernizing LazyPG's UI/UX based on research of leading TUI and SQL tools. The recommendations prioritize:

1. **Visual Harmony** - Cohesive color palette (Catppuccin)
2. **Information Density** - Maximum information without clutter
3. **Clear Hierarchy** - Color, typography, and spacing for structure
4. **Modern Aesthetics** - Unicode symbols, rounded borders, thoughtful styling
5. **User Experience** - Intuitive navigation, helpful feedback, clear states

### Next Steps

1. Implement Catppuccin Mocha theme
2. Enhance tree view with icons and better metadata
3. Improve table rendering with lipgloss table component
4. Add loading and empty states
5. Enhance status bars with context-sensitive help
6. Consider implementing additional themes (Latte for light mode)

### References

- **LazyGit**: https://github.com/jesseduffield/lazygit
- **k9s**: https://github.com/derailed/k9s
- **Catppuccin**: https://github.com/catppuccin/catppuccin
- **Bubble Tea**: https://github.com/charmbracelet/bubbletea
- **Lip Gloss**: https://github.com/charmbracelet/lipgloss
- **pgcli**: https://www.pgcli.com/

---

**Document End**
