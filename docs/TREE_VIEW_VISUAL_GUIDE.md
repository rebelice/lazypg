# TreeView Visual Guide

## Overview

This document shows visual examples of the TreeView component in different states.

## Basic Tree Structure

### Fully Collapsed
```
▸ postgres (active)
▸ myapp_db
▸ template1
```

### Single Database Expanded
```
▾ postgres (active)
  ▸ public
  ▸ information_schema
  ▸ pg_catalog
▸ myapp_db
▸ template1
```

### Schema Expanded
```
▾ postgres (active)
  ▾ public
    ▸ users (250 rows)
    ▸ posts (1.2k rows)
    ▸ comments (5.3k rows)
  ▸ information_schema
  ▸ pg_catalog
▸ myapp_db
▸ template1
```

### Table Expanded (with columns)
```
▾ postgres (active)
  ▾ public
    ▾ users (250 rows)
      • id (integer) PK
      • email (varchar(255))
      • name (varchar(100))
      • created_at (timestamp)
    ▸ posts (1.2k rows)
    ▸ comments (5.3k rows)
  ▸ information_schema
▸ myapp_db
▸ template1
```

## Visual Indicators

### Icons

| Icon | Meaning | Node State |
|------|---------|------------|
| `▾` | Expanded | Node is open, showing children |
| `▸` | Collapsed | Node has children but is closed |
| `•` | Leaf | Column node (no children possible) |

### Highlighting

#### Selected Row (cursor on "postgres")
```
┌─ Databases ────────────────────────┐
│ [▾ postgres (active)]             │ ← Selected (highlighted)
│   ▸ public                         │
│   ▸ information_schema             │
│ ▸ myapp_db                         │
│ ▸ template1                        │
└────────────────────────────────────┘
```

#### Active Database Marker
```
▾ postgres (active)    ← Green "(active)" badge
  ▸ public
  ▸ information_schema
▸ myapp_db
▸ template1
```

#### Primary Key Indicator
```
▾ users (250 rows)
  • id (integer) PK     ← Yellow "PK" badge
  • email (varchar(255))
  • name (varchar(100))
```

## Row Count Formatting

The component intelligently formats row counts:

```
▸ small_table (150 rows)        ← Under 1k: exact number
▸ medium_table (1.5k rows)      ← 1k-10k: one decimal
▸ large_table (250k rows)       ← 10k-1M: no decimal
▸ huge_table (1.5M rows)        ← 1M+: one decimal with M
```

## Indentation

Each level of hierarchy is indented by 2 spaces:

```
Level 0: ▾ postgres (active)
Level 1:   ▾ public
Level 2:     ▾ users (250 rows)
Level 3:       • id (integer) PK
Level 3:       • email (varchar(255))
```

## Viewport Scrolling

When the tree is larger than the display area, scroll indicators appear:

### Scrolled Down (more content above)
```
┌─ Databases ────────────────────────┐
│ ↑ ▸ information_schema            │ ← Up arrow
│   ▸ public                         │
│   ▸ pg_catalog                     │
│ ▸ template1                        │
│ ▸ test_db                          │
└────────────────────────────────────┘
```

### Scrolled Up (more content below)
```
┌─ Databases ────────────────────────┐
│ ▸ postgres (active)                │
│ ▸ myapp_db                         │
│ ▸ template1                        │
│ ▸ test_db                          │
│ ↓ ▸ another_db                     │ ← Down arrow
└────────────────────────────────────┘
```

### Scrolled Middle (content both above and below)
```
┌─ Databases ────────────────────────┐
│ ↑ ▸ postgres (active)              │ ← Up arrow
│   ▸ myapp_db                       │
│   ▸ template1                      │
│   ▸ test_db                        │
│ ↓ ▸ another_db                     │ ← Down arrow
└────────────────────────────────────┘
```

## Empty State

When no databases are connected:

```
┌─ Databases ────────────────────────┐
│                                    │
│                                    │
│     No databases connected         │
│                                    │
│                                    │
└────────────────────────────────────┘
```

## Complete Example

A fully expanded tree showing all features:

```
┌─ Database Navigator ───────────────┐
│ ▾ postgres (active)                │ ← Active DB (green)
│   ▾ public                         │
│     ▾ users (250 rows)             │ ← Row count
│       • id (integer) PK            │ ← PK indicator (yellow)
│       • email (varchar(255))       │
│       • name (varchar(100))        │
│       • created_at (timestamp)     │
│     ▸ posts (1.2k rows)            │ ← Formatted count
│     ▸ comments (15k rows)          │
│   ▸ information_schema             │
│ ▸ myapp_db                         │
│ ▸ template1                        │
│                                    │
│ Current: postgres > public > users │
│ [g]top [G]bottom [↑↓]navigate     │
└────────────────────────────────────┘
```

## Keyboard Navigation Visual Flow

### Moving Down
```
Step 1:                Step 2:                Step 3:
[▾ postgres]           ▾ postgres             ▾ postgres
  ▸ public               [▸ public]             ▸ public
  ▸ info_schema          ▸ info_schema          [▸ info_schema]
▸ myapp_db             ▸ myapp_db             ▸ myapp_db
```

### Expanding a Node
```
Before (press →/l/Space):    After:
▾ postgres                   ▾ postgres
  [▸ public]                   [▾ public]
  ▸ info_schema                  ▸ users
▸ myapp_db                       ▸ posts
                                 ▸ info_schema
                               ▸ myapp_db
```

### Collapsing a Node
```
Before (press ←/h):          After:
▾ postgres                   ▾ postgres
  [▾ public]                   [▸ public]
    ▸ users                  ▸ info_schema
    ▸ posts                ▸ myapp_db
  ▸ info_schema
▸ myapp_db
```

### Navigate to Parent
```
Before (press ←/h):          After:
▾ postgres                   ▾ postgres
  [▸ public]                   [▸ public]
  ▸ info_schema                ▸ info_schema
▸ myapp_db                   ▸ myapp_db

(public is collapsed,        (cursor moved up to
 so move to parent)           postgres database)

                             [▾ postgres]
                               ▸ public
                               ▸ info_schema
                             ▸ myapp_db
```

### Jump to Top/Bottom
```
Jump to Top (g):             Jump to Bottom (G):
[▾ postgres]                 ▾ postgres
  ▸ public                     ▸ public
  ▸ info_schema                ▸ info_schema
▸ myapp_db                   ▸ myapp_db
▸ template1                  [▸ template1]
```

## Color Scheme (Default Theme)

- **Selected Row**: Gray background (#444444), white text
- **Normal Text**: Light gray (#d0d0d0)
- **Active Database**: Green "(active)" badge (#00d700)
- **Primary Key**: Yellow "PK" badge (#ffaf00)
- **Scroll Indicators**: Blue arrows (#5fafff)
- **Empty State**: Dim gray, italic (#808080)
- **Border**: Purple when focused (#5f5faf), gray otherwise (#585858)

## Responsive Behavior

The tree adapts to available space:

### Wide Panel (60 columns)
```
▾ postgres (active)
  ▾ public
    ▸ users (250 rows)
    ▸ posts (1.2k rows)
    ▸ comments (5.3k rows)
```

### Narrow Panel (30 columns)
```
▾ postgres (active)
  ▾ public
    ▸ users (250 rows)
    ▸ posts (1.2k ro…
    ▸ comments (5.3…
```

Long labels are truncated with ellipsis (…) to fit the available width.

## Accessibility Features

1. **Clear Visual Hierarchy**: Indentation shows parent-child relationships
2. **Multiple Navigation Methods**: Both arrow keys and vi keybindings (hjkl)
3. **Visual Feedback**: Cursor highlighting shows current position
4. **Status Indicators**: Icons show node state at a glance
5. **Scroll Indicators**: Arrows show when content extends beyond viewport
6. **Descriptive Labels**: Full context in column descriptions (name + type)
7. **Empty State Message**: Clear feedback when no data available
