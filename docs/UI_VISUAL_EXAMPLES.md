# LazyPG Visual Design Examples

This document provides visual examples of the recommended UI improvements.

---

## 1. Database Navigator - Before & After

### Current (Before)
```
┌────────────────────────────┐
│ postgres                   │
│   public                   │
│     users                  │
│     posts                  │
│     comments               │
│   pg_catalog               │
│ template1                  │
└────────────────────────────┘
```

### Enhanced (After)
```
┌─ Database Navigator ───────┐
│ ● postgres (active)        │
│   ▾ public (12)            │
│     ▦ users 1.2k           │
│     ▦ posts 8.4k           │
│     ▦ comments 24.5k       │
│   ▸ pg_catalog (287)       │
│ ○ template1                │
└────────────────────────────┘
```

**Improvements:**
- Title in border
- Active database indicator (●)
- Expansion state (▾ expanded, ▸ collapsed)
- Type icons (▦ for tables)
- Row counts with k/M formatting
- Schema item counts

---

## 2. Column Display - Before & After

### Current (Before)
```
• id
• name
• email
• created_at
• is_active
```

### Enhanced (After)
```
  • id integer ⚿ *
  • name varchar(255)
  • email varchar(255) → *
  • created_at timestamp
  • is_active boolean
```

**Improvements:**
- Data types shown
- Primary key indicator (⚿)
- Foreign key indicator (→)
- Not null indicator (*)
- Proper indentation

---

## 3. Table View - Before & After

### Current (Basic)
```
id    name     email          active
1     Alice    a@example.com  true
2     Bob      b@example.com  false
3     Charlie  c@example.com  true
```

### Enhanced (With Borders & Styling)
```
┌────┬─────────┬───────────────┬────────┐
│ id │ name    │ email         │ active │
├────┼─────────┼───────────────┼────────┤
│ 1  │ Alice   │ a@example.com │ ✓      │
│ 2  │ Bob     │ b@example.com │ ✗      │
│ 3  │ Charlie │ c@example.com │ ✓      │
└────┴─────────┴───────────────┴────────┘

3 rows • 4 columns
```

**Improvements:**
- Rounded borders
- Header row separation
- Boolean symbols (✓/✗) or colors
- Footer with metadata
- Proper column alignment

---

## 4. Full Application Layout

### Enhanced Layout
```
┌─ LazyPG ─────────────────────────────────────┬─ postgres@localhost:5432/mydb ─┐
│                                                                                 │
│  ┌─ Database Navigator ───┐  ┌─ Table: users (1,234 rows) ─────────────────┐  │
│  │                         │  │                                              │  │
│  │ ● postgres              │  │ ┌────┬─────────┬───────────────┬────────┐   │  │
│  │   ▾ public (12)         │  │ │ id │ name    │ email         │ active │   │  │
│  │     ▦ users 1.2k        │  │ ├────┼─────────┼───────────────┼────────┤   │  │
│  │     ▦ posts 8.4k        │  │ │ 1  │ Alice   │ a@example.com │   ✓    │   │  │
│  │     ▦ comments 24.5k    │  │ │ 2  │ Bob     │ b@example.com │   ✗    │   │  │
│  │     ▤ recent_users      │  │ │ 3  │ Charlie │ c@example.com │   ✓    │   │  │
│  │     ▤ active_posts      │  │ │ 4  │ Diana   │ d@example.com │   ✓    │   │  │
│  │     ƒ get_user_stats    │  │ │ 5  │ Eve     │ e@example.com │   ✗    │   │  │
│  │   ▸ pg_catalog (287)    │  │ └────┴─────────┴───────────────┴────────┘   │  │
│  │ ○ template1             │  │                                              │  │
│  │ ○ template0             │  │ 1,234 rows • 4 columns • 128 KB              │  │
│  │                         │  │                                              │  │
│  └─────────────────────────┘  └──────────────────────────────────────────────┘  │
│                                                                                  │
│  tab switch • ↑↓ navigate • →← expand • enter select • r refresh • ? help     │
└──────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- Top bar: App name + connection info
- Left panel: Database navigator (25-35% width)
- Right panel: Data display (65-75% width)
- Bottom bar: Context-sensitive keybindings
- Panel titles in borders
- Visual feedback for active panel

---

## 5. Color Coding Examples

### Status Colors (Catppuccin Mocha)
```
✓ Success Message     (#a6e3a1 - Green)
⚠ Warning Message     (#f9e2af - Yellow)
✗ Error Message       (#f38ba8 - Red)
ℹ Info Message        (#89dceb - Sky)
```

### Node Type Colors
```
● Database (Active)   (#a6e3a1 - Green)
○ Database (Inactive) (#6c7086 - Gray)
▦ Table               (#cba6f7 - Mauve)
▤ View                (#94e2d5 - Teal)
ƒ Function            (#fab387 - Peach)
• Column              (#cdd6f4 - Text)
```

### Data Type Colors
```
123       Integer     (#fab387 - Peach)
"text"    String      (#f5c2e7 - Pink)
true      Boolean     (#a6e3a1 - Green)
NULL      Null        (#6c7086 - Gray, Italic)
```

---

## 6. Loading States

### Loading Databases
```
┌─ Database Navigator ────┐
│                          │
│    ⠋ Loading databases…  │
│                          │
└──────────────────────────┘
```

### Loading Tables
```
┌─ Database Navigator ────┐
│ ● postgres               │
│   ▾ public …             │
│     ⠙ Loading tables…    │
└──────────────────────────┘
```

**Spinner frames:** ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏

---

## 7. Empty States

### No Connection
```
┌─ Database Navigator ────┐
│                          │
│          ⚠               │
│   No database connection │
│   Press 'c' to connect   │
│                          │
└──────────────────────────┘
```

### No Tables
```
┌─ Database Navigator ────┐
│ ● postgres               │
│   ▾ public               │
│                          │
│          ∅               │
│     No tables found      │
│                          │
└──────────────────────────┘
```

### No Results
```
┌─ Query Results ──────────┐
│                          │
│          ∅               │
│  Query returned no rows  │
│                          │
└──────────────────────────┘
```

---

## 8. Help Modal

### Full Help Screen
```
╭────────────────────────────────────────────────────────────────╮
│                                                                │
│                         LazyPG Help                            │
│                                                                │
│  Navigation                                                    │
│    ↑/k          Move up                                        │
│    ↓/j          Move down                                      │
│    →/l          Expand / Go right                              │
│    ←/h          Collapse / Go left                             │
│    g            Jump to top                                    │
│    G            Jump to bottom                                 │
│                                                                │
│  Actions                                                       │
│    enter        Select / Execute                               │
│    space        Toggle expand                                  │
│    tab          Switch panel                                   │
│    r            Refresh                                        │
│    c            New connection                                 │
│                                                                │
│  Application                                                   │
│    ?            Toggle help                                    │
│    q            Quit                                           │
│    ctrl+c       Force quit                                     │
│                                                                │
│                   Press ? or Esc to close                      │
│                                                                │
╰────────────────────────────────────────────────────────────────╯
```

---

## 9. Connection Dialog

### New Connection Form
```
╭──────────────────────────────────────────────╮
│                                              │
│          🔌 New Connection                   │
│                                              │
│  Host:      ╭──────────────────────────────╮ │
│             │ localhost                    │ │
│             ╰──────────────────────────────╯ │
│                                              │
│  Port:      ╭──────────────────────────────╮ │
│             │ 5432                         │ │
│             ╰──────────────────────────────╯ │
│                                              │
│  Database:  ╭──────────────────────────────╮ │
│             │ postgres                     │ │
│             ╰──────────────────────────────╯ │
│                                              │
│  Username:  ╭──────────────────────────────╮ │
│             │ postgres                     │ │
│             ╰──────────────────────────────╯ │
│                                              │
│  Password:  ╭──────────────────────────────╮ │
│             │ ••••••••                     │ │
│             ╰──────────────────────────────╯ │
│                                              │
│   ↑↓ navigate • enter connect • esc cancel  │
│                                              │
╰──────────────────────────────────────────────╯
```

---

## 10. Error Overlay

### Database Error
```
╭──────────────────────────────────────────────╮
│                                              │
│               ✗ Connection Error             │
│                                              │
│  Failed to connect to database:              │
│                                              │
│  FATAL: password authentication failed       │
│  for user "postgres"                         │
│                                              │
│  Details:                                    │
│  - Host: localhost:5432                      │
│  - Database: mydb                            │
│  - User: postgres                            │
│                                              │
│           Press any key to dismiss           │
│                                              │
╰──────────────────────────────────────────────╯
```

---

## 11. Border Styles Comparison

### Normal Border
```
┌────────┐
│ Normal │
└────────┘
```

### Rounded Border (Recommended)
```
╭─────────╮
│ Rounded │
╰─────────╯
```

### Thick Border
```
┏━━━━━━━┓
┃ Thick ┃
┗━━━━━━━┛
```

### Double Border
```
╔════════╗
║ Double ║
╚════════╝
```

**Recommendation:** Use Rounded for modern look, Normal for classic feel.

---

## 12. Data Type Visualization

### JSONB Display
```
┌─ users.metadata (JSONB) ──────────────────┐
│ {                                         │
│   "role": "admin",                        │
│   "permissions": [                        │
│     "read",                               │
│     "write",                              │
│     "delete"                              │
│   ],                                      │
│   "active": true,                         │
│   "login_count": 42                       │
│ }                                         │
└───────────────────────────────────────────┘
```

**Color Coding:**
- Keys: Blue
- Strings: Pink
- Numbers: Peach
- Booleans: Green
- null: Gray italic

### Array Display
```
tags: ["postgresql", "database", "admin"]
```

### NULL vs Empty
```
NULL      (gray, italic)
""        (empty string, shown as quotes)
0         (zero, shown as number)
```

---

## 13. Metadata Display Patterns

### Table Metadata
```
▦ users 1,234 rows

When expanded:
  • id integer ⚿ * ⚡
  • name varchar(255) *
  • email varchar(255) → ⚡
  • created_at timestamp
  • is_active boolean
```

**Symbols:**
- ⚿ = Primary Key
- → = Foreign Key
- * = Not Null
- ⚡ = Indexed

### Schema Metadata
```
▾ public (12)           → 12 tables/views
▸ pg_catalog (287)      → 287 objects
▸ information_schema ∅  → Empty
```

---

## 14. Selection States

### Normal Row
```
│ 1  │ Alice   │ a@example.com │ ✓      │
```

### Selected Row (Highlighted Background)
```
│ 2  │ Bob     │ b@example.com │ ✗      │  ← Background: #313244, Bold
```

### Hover/Focus Effect
```
  ▦ users 1.2k                    (normal)
> ▦ posts 8.4k                    (cursor, with arrow)
  ▦ comments 24.5k                (normal)
```

---

## 15. Responsive Layout

### Narrow Terminal (< 80 cols)
```
┌─ LazyPG ──────────────┬─ postgres ─┐
│ ┌─ DB ──────────────┐ │ ┌─ Data ──┐│
│ │ ● postgres        │ │ │ id name ││
│ │   ▾ public (12)   │ │ │ 1  Alice││
│ │     ▦ users 1.2k  │ │ │ 2  Bob  ││
│ └───────────────────┘ │ └─────────┘│
│ tab • ↑↓ • ? help • q quit         │
└────────────────────────────────────┘
```

### Wide Terminal (> 120 cols)
```
┌─ LazyPG ──────────────────────────────────────────────────────┬─ Connection: postgres@localhost:5432/mydb ─────────┐
│  ┌─ Database Navigator ─────────────┐  ┌─ Table: users (1,234 rows) ─────────────────────────────────────────┐  │
│  │ ● postgres                        │  │ ┌────┬─────────┬───────────────┬────────────────────┬────────┐      │  │
│  │   ▾ public (12)                   │  │ │ id │ name    │ email         │ created_at         │ active │      │  │
│  │     ▦ users 1.2k                  │  │ ├────┼─────────┼───────────────┼────────────────────┼────────┤      │  │
│  │     ▦ posts 8.4k                  │  │ │ 1  │ Alice   │ a@example.com │ 2024-01-15 10:30   │   ✓    │      │  │
│  │     ▦ comments 24.5k              │  │ │ 2  │ Bob     │ b@example.com │ 2024-01-16 14:22   │   ✗    │      │  │
│  │       • id integer ⚿ * ⚡          │  │ │ 3  │ Charlie │ c@example.com │ 2024-01-17 09:15   │   ✓    │      │  │
│  │       • name varchar(255) *       │  │ └────┴─────────┴───────────────┴────────────────────┴────────┘      │  │
│  └───────────────────────────────────┘  └──────────────────────────────────────────────────────────────────────┘  │
│  tab switch panel • ↑↓jk navigate • →←hl expand/collapse • enter select • r refresh • ? help • q quit            │
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 16. Progress Indicators

### Indeterminate Progress
```
⠋ Loading...
⠙ Loading...
⠹ Loading...
⠸ Loading...
⠼ Loading...
⠴ Loading...
```

### Determinate Progress
```
Loading table data... ████████████░░░░░░░░ 60%
```

---

## Color Palette Reference

### Catppuccin Mocha Colors in Context

```
Background Layers:
█ #1e1e2e  Base (main background)
█ #181825  Mantle (darker background)
█ #11111b  Crust (darkest background)
█ #313244  Surface0 (selection)
█ #45475a  Surface1 (borders)
█ #585b70  Surface2 (elevated)

Text Hierarchy:
█ #cdd6f4  Text (primary)
█ #bac2de  Subtext1
█ #a6adc8  Subtext0
█ #9399b2  Overlay2
█ #7f849c  Overlay1
█ #6c7086  Overlay0 (dimmed)

Accent Colors:
█ #f38ba8  Red (errors)
█ #fab387  Peach (numbers)
█ #f9e2af  Yellow (warnings)
█ #a6e3a1  Green (success)
█ #94e2d5  Teal (operators)
█ #89dceb  Sky (info)
█ #74c7ec  Sapphire (links)
█ #89b4fa  Blue (keywords, focus)
█ #cba6f7  Mauve (special)
█ #f5c2e7  Pink (strings)
```

---

## Typography Examples

### Font Weight Usage
```
Normal:   Regular database text
Bold:     Selected items and headers
Italic:   Help text and empty states
Dim:      Secondary metadata
```

### Practical Examples
```
Normal:   postgres
Bold:     postgres  (when selected)
Italic:   No databases connected
Dim:      (12 tables)
```

---

## Icon Usage Guide

| Context | Icon | Color | Meaning |
|---------|------|-------|---------|
| Active DB | ● | Green | Currently connected |
| Inactive DB | ○ | Gray | Available but not connected |
| Expanded Schema | ▾ | Blue | Schema is open |
| Collapsed Schema | ▸ | Blue | Schema is closed |
| Table | ▦ | Purple | Database table |
| View | ▤ | Teal | Database view |
| Function | ƒ | Orange | Database function |
| Column | • | Gray | Table column |
| Primary Key | ⚿ | Yellow | PK constraint |
| Foreign Key | → | Blue | FK constraint |
| Index | ⚡ | Purple | Indexed column |
| Not Null | * | Green | Required field |
| Empty | ∅ | Gray | No items |
| Loading | … | Blue | In progress |
| Success | ✓ | Green | Positive state |
| Error | ✗ | Red | Negative state |
| Warning | ⚠ | Yellow | Caution |
| Info | ℹ | Blue | Information |

---

## Best Practices Summary

1. **Consistency**: Use the same icons and colors for the same concepts throughout
2. **Hierarchy**: More important = more contrast/bold/color
3. **Spacing**: Related items closer, unrelated items farther apart
4. **Feedback**: Always show loading/empty/error states
5. **Color**: Semantic colors (green=good, red=bad, yellow=warning)
6. **Icons**: Use Unicode symbols for quick visual recognition
7. **Truncation**: Always truncate long text with … to maintain layout
8. **Metadata**: Show in dimmed color in parentheses
9. **Alignment**: Text left, numbers right, booleans center
10. **Borders**: Rounded for modern feel, focus color for active panel

---

**End of Visual Examples**
