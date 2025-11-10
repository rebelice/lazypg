# UI Design Research Summary

## Quick Reference Guide

This is a condensed summary of the full [UI/UX Design Specification](./UI_UX_DESIGN_SPECIFICATION.md).

---

## Key Findings

### 1. Color Palette - Catppuccin Mocha (Recommended)

**Why Catppuccin?**
- Most popular modern TUI theme (300+ app ports)
- Excellent contrast and readability
- Soothing pastel colors reduce eye strain
- Proven in production TUI apps

**Essential Colors:**
```
Background:  #1e1e2e  (Base)
Foreground:  #cdd6f4  (Text)
Border:      #45475a  (Surface1)
Focus:       #89b4fa  (Blue)
Selection:   #313244  (Surface0)

Success:     #a6e3a1  (Green)
Warning:     #f9e2af  (Yellow)
Error:       #f38ba8  (Red)
Info:        #89dceb  (Sky)

Table Header: #89b4fa  (Blue)
Muted Text:   #6c7086  (Overlay0)
```

### 2. Layout Principles

**Spacing System:** 4px/8px base units
- Compact spacing: 2-4px (related items)
- Standard padding: 8px (component internal)
- Section margins: 12-16px (separation)

**Panel Layout:**
```
┌─ Top Bar (1 line) ─────────────────────────────┐
│ App Name                    Connection Status  │
├─ Left Panel (25-35%) ─┬─ Right Panel (65-75%) ─┤
│ Database Navigator     │ Data Display          │
│                        │                       │
│ ▾ postgres (●)         │ ┌─ Table: users ───┐ │
│   ▾ public             │ │ id  name   email │ │
│     ▸ tables (15)      │ │ 1   Alice  a@... │ │
│     ▸ views (3)        │ │ 2   Bob    b@... │ │
│                        │ └──────────────────┘ │
├────────────────────────┴───────────────────────┤
│ [tab] switch • [↑↓] navigate • [q] quit        │
└────────────────────────────────────────────────┘
```

### 3. Visual Elements

**Icons (Unicode):**
```
Database:  ◉ (inactive)  ● (active)
Schema:    ▾ (expanded)  ▸ (collapsed)
Table:     ▦
View:      ▤
Function:  ƒ
Column:    •
PK:        ⚿
FK:        →
Index:     ⚡
Not Null:  *
Empty:     ∅
Loading:   …
```

**Typography:**
- Bold: Selected items, headers, emphasis
- Italic: Empty states, help text, metadata
- Dimmed: Secondary info, row counts, comments
- Color > Weight: Prefer color for hierarchy

### 4. Component Patterns

**Tree View:**
```go
// Node rendering pattern
indent + icon + " " + label + " " + metadata

// Example output:
  ▦ users 1.2k rows
    • id integer ⚿ *
    • name varchar
    • email varchar → FK
```

**Table View:**
```go
// Use lipgloss table component
table.New().
    Border(lipgloss.RoundedBorder()).
    BorderStyle(BorderColor).
    Headers(...).
    Rows(...).
    StyleFunc(cellStyler)

// Alignment:
// - Text: Left
// - Numbers: Right
// - Boolean: Center
```

**Status Bars:**
```
Top:    App Name (left) + Connection Status (right)
Bottom: Context Keys (left) + Info/Help (right)
```

---

## Immediate Action Items

### Priority 1: Colors (1-2 hours)

1. Create `internal/ui/theme/catppuccin_mocha.go`
2. Update theme structure with additional colors
3. Switch default theme to Catppuccin
4. Test with existing components

**Impact:** High visual improvement with minimal code changes

### Priority 2: Tree Icons (2-3 hours)

1. Add icon constants to tree_view.go
2. Implement `getEnhancedIcon()` with colors
3. Update `buildNodeLabel()` for better metadata
4. Add row count formatting

**Impact:** Much clearer visual hierarchy

### Priority 3: Empty/Loading States (1 hour)

1. Add proper empty state messages
2. Implement loading spinner
3. Add icons to states

**Impact:** Better UX feedback

### Priority 4: Table Improvements (3-4 hours)

1. Switch to lipgloss table component
2. Add borders and proper styling
3. Implement row selection highlighting
4. Add footer with metadata

**Impact:** Professional data display

---

## Design Rules of Thumb

### Colors
1. Use accent colors semantically (green=success, red=error, etc.)
2. Dim secondary information (gray/overlay colors)
3. Bold + color for selected/focused items
4. Keep background subtle (one shade darker for selection)

### Spacing
1. Related items: 0-4px apart
2. Component padding: 8px (2 spaces)
3. Section separation: 16px+ (4 spaces)
4. More space = less related

### Typography
1. Bold for selection and headers
2. Italic for help/empty states only
3. Dimmed colors for metadata
4. Mono-space font always (terminal default)

### Data Display
1. Left-align text
2. Right-align numbers
3. Center-align booleans
4. Truncate with … if too long
5. Show metadata in dimmed color

### Feedback
1. Always show loading states
2. Provide helpful empty states
3. Use icons for quick recognition
4. Context-sensitive help in status bar

---

## Example: Before & After

### Before (Current)
```
┌────────────────────┐
│ postgres           │
│   public           │
│     users          │
│     posts          │
└────────────────────┘
```

### After (Enhanced)
```
┌─ Databases ────────┐
│ ● postgres         │
│   ▾ public (12)    │
│     ▦ users 1.2k   │
│     ▦ posts 8.4k   │
└────────────────────┘
```

**Improvements:**
- Color-coded icons (● green for active)
- Expansion indicators (▾)
- Type icons (▦ for tables)
- Metadata (row counts)
- Better border with title

---

## Code Snippets

### Quick Color Update
```go
// OLD
Border: lipgloss.Color("240")

// NEW
Border: lipgloss.Color("#45475a")
```

### Icon Usage
```go
const (
    IconDatabase = "●"
    IconTable    = "▦"
    IconColumn   = "•"
    IconPK       = "⚿"
)

iconStyle := lipgloss.NewStyle().
    Foreground(theme.Success).
    Bold(true)
icon := iconStyle.Render(IconDatabase)
```

### Row Count Formatting
```go
func formatRowCount(n int64) string {
    if n < 1000 {
        return fmt.Sprintf("%d", n)
    } else if n < 10000 {
        return fmt.Sprintf("%.1fk", float64(n)/1000)
    }
    return fmt.Sprintf("%.0fk", float64(n)/1000)
}
```

---

## Resources

**Full Documentation:**
- [Complete UI/UX Specification](./UI_UX_DESIGN_SPECIFICATION.md)

**External References:**
- [Catppuccin Palette](https://catppuccin.com/palette)
- [LazyGit UI](https://github.com/jesseduffield/lazygit)
- [k9s Skins](https://k9scli.io/topics/skins/)
- [Lip Gloss Docs](https://github.com/charmbracelet/lipgloss)

**Color Tools:**
- [Catppuccin Ports](https://github.com/catppuccin/catppuccin)
- [Terminal Colors Preview](https://terminalcolors.com/)

---

## Next Steps

1. Review full specification document
2. Implement Priority 1 (colors) first
3. Test changes with real database
4. Iterate based on visual feedback
5. Consider adding theme switcher later

**Estimated Total Time:** 8-12 hours for all priority items

---

**Last Updated:** 2025-11-10
