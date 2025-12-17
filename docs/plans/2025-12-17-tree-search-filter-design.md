# TreeView Search/Filter Design

## Overview

Add `/` key triggered search filtering for TreeView (Explorer), supporting:
- Fuzzy matching
- `!` prefix negation
- Type prefix filtering (e.g., `t:plan` or `table:plan`)
- Search all nodes (including collapsed ones)
- Filter mode display (hide non-matches, auto-expand parent paths of matches)

## Interaction Flow

```
Normal State ──[/]──► Search Input State
                       │
                       ├── Type chars → Real-time filter tree
                       ├── Backspace → Delete char, update filter
                       ├── Enter → Exit input, keep filter, cursor to first match
                       └── Esc → Exit input, keep filter results
                                 │
              Filter Active ◄────┘
                       │
                       ├── j/k/↑/↓ → Navigate within filtered results
                       ├── Enter → Select current node
                       ├── / → Re-enter search input
                       └── Esc → Clear filter, restore full tree
                                 │
               Normal State ◄────┘
```

## Search Syntax

| Input | Meaning |
|-------|---------|
| `plan` | Fuzzy match nodes containing "plan" |
| `!test` | Exclude nodes matching "test" |
| `t:plan` | Only tables matching "plan" |
| `table:plan` | Same as above, full syntax |
| `f:get` | Only functions matching "get" |
| `!t:` | Exclude all tables, show other types |

## Type Prefix Mapping

| Short | Long | TreeNodeType |
|-------|------|--------------|
| `t:` | `table:` | `TreeNodeTypeTable` |
| `v:` | `view:` | `TreeNodeTypeView` |
| `f:` | `func:` / `function:` | `TreeNodeTypeFunction` |
| `s:` | `schema:` | `TreeNodeTypeSchema` |
| `seq:` | `sequence:` | `TreeNodeTypeSequence` |
| `ext:` | `extension:` | `TreeNodeTypeExtension` |
| `col:` | `column:` | `TreeNodeTypeColumn` |
| `idx:` | `index:` | `TreeNodeTypeIndex` |

## Fuzzy Matching Algorithm

Simple subsequence matching (no external dependencies):

```
Input: "pck"
Target: "plan_check_run"
Match: p___c____k______ ✓

Algorithm: Each character in input must appear in order in target
```

### Match Priority (for sorting)

1. **Exact prefix match** - `plan` matches start of `plan_check_run`
2. **Contiguous substring** - `check` appears contiguously in `plan_check_run`
3. **Fuzzy subsequence** - `pcr` matches `plan_check_run`

Items sorted by priority, then alphabetically within same priority.

### Negation Logic

- `!plan` → Show all nodes that do NOT match `plan`
- `!t:` → Show all NON-table type nodes
- `!t:plan` → Show non-table nodes + tables not matching "plan"

## Filtered Tree Structure

### Problem

After searching all nodes (including collapsed), how to display filtered results?

### Solution: Build Filtered View Tree

```
Original tree:                 After searching "plan":

bbdev                         bbdev
├─ Extensions (1)             └─ public
│  └─ plpgsql v1.0               └─ Tables (2 matches)
└─ public                           ├─ plan ✓
   ├─ Tables (39)                   └─ plan_check_run ✓
   │  ├─ audit_log
   │  ├─ plan ✓
   │  ├─ plan_check_run ✓
   │  └─ ...
   └─ Functions (1)
```

### Processing Rules

1. **Matching leaf nodes** → Display
2. **Ancestors of matches** → Display (as path, auto-expand)
3. **Container nodes** (e.g., "Tables") → Display if has matching children, update count to match count
4. **Non-matching with no matching descendants** → Hide

### Data Structure

Don't modify original tree, create filtered view:

```go
type TreeView struct {
    Root         *models.TreeNode // Original complete tree
    FilteredRoot *models.TreeNode // Filtered view tree (nil = no filter)

    // Search state
    SearchMode    SearchModeState  // Off / Inputting / FilterActive
    SearchQuery   string
    SearchMatches []*models.TreeNode // Matched nodes list for quick navigation
}

type SearchModeState int
const (
    SearchOff SearchModeState = iota
    SearchInputting   // Typing, show input box
    SearchFilterActive // Input done, filter active, can navigate
)
```

### Display Logic

```go
func (tv *TreeView) getDisplayRoot() *models.TreeNode {
    if tv.FilteredRoot != nil {
        return tv.FilteredRoot
    }
    return tv.Root
}
```

## UI Display

### Search Input Position

Display search bar at top of TreeView (like lazygit):

```
┌─ Explorer ─────────────────┐
│ / plan_                    │  ← Shown during search input
├────────────────────────────┤
│ ● bbdev                    │
│   └─ public                │
│      └─ Tables (2)         │
│         ├─ plan            │
│         └─ plan_check_run  │
└────────────────────────────┘
```

### Status Indicators

| State | Display |
|-------|---------|
| `SearchOff` | Normal title "Explorer" |
| `SearchInputting` | `/ {query}_` with cursor |
| `SearchFilterActive` | `[{n} matches] / {query}` showing match count and query |

### Match Highlighting

In filtered node names, highlight matching characters:

```
plan_check_run   (search "pcr")
↓
plan_check_run   (p, c, r highlighted)
```

### No Matches

```
┌─ Explorer ─────────────────┐
│ / xyz_                     │
├────────────────────────────┤
│                            │
│     No matches found       │
│                            │
└────────────────────────────┘
```

### Key Hints

In `SearchFilterActive` state, status bar shows:
```
/ search │ Esc clear │ Enter select
```

## Implementation

### File Changes

| File | Changes |
|------|---------|
| `internal/ui/components/tree_view.go` | Main impl: search state, filter logic, UI rendering |
| `internal/ui/components/tree_filter.go` | New file: filter algorithm, fuzzy match, type parsing |
| `internal/models/tree.go` | May need `Clone()` method for creating filtered view |
| `internal/app/app.go` | Route `/` key to TreeView |

### Core Functions

```go
// tree_filter.go

// ParseSearchQuery parses search syntax
// Input: "!t:plan" → {Negate: true, TypeFilter: "table", Pattern: "plan"}
func ParseSearchQuery(query string) SearchQuery

// FuzzyMatch performs fuzzy matching
// Returns whether matched and match positions (for highlighting)
func FuzzyMatch(pattern, target string) (bool, []int)

// FilterTree builds filtered tree view
func FilterTree(root *models.TreeNode, query SearchQuery) (*models.TreeNode, []*models.TreeNode)
```

### Performance Considerations

1. **Debounce** - 50ms debounce during input to avoid rebuilding filter tree on every keystroke
2. **Cache** - Don't recalculate for same query
3. **Lazy expand** - Filtered tree only contains necessary nodes, don't copy entire tree structure

### Test Cases

1. Basic fuzzy: `pcr` matches `plan_check_run`
2. Negation: `!plan` doesn't show nodes containing plan
3. Type filter: `t:plan` only matches tables
4. Combination: `!t:test` excludes tables containing test
5. Empty query: show full tree
6. No matches: show "No matches found"
7. Two-stage Esc: first keeps filter, second clears
