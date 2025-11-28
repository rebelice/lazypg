# Column Sort Shortcut Design

## Overview

Add a keyboard shortcut to quickly reverse the sort direction of the current sorted column.

## Current State

- `s` - Sort by current selected column (toggles ASC → DESC)
- `S` - Toggle NULLS FIRST/LAST
- Sorting uses server-side ORDER BY (re-queries database)

## Design

### Keyboard Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `s` | Sort current column | Existing: first press ASC, second press DESC |
| `S` | Toggle NULLS FIRST/LAST | Existing |
| `r` | Reverse sort direction | New: quickly toggle ASC ↔ DESC |

### Behavior

**`r` key (reverse sort):**
1. If a sort column is active → toggle ASC/DESC and reload data
2. If no sort is active → no-op (ignore the keypress)

**Difference between `s` and `r`:**
- `s` = sort by **selected column** (may change which column is sorted)
- `r` = reverse **current sort column** direction (does not change sort column)

### Implementation

1. Add `r` key handler in `internal/app/app.go` under Data tab key handling
2. Check if `tableView.SortColumn >= 0` before reversing
3. Call `tableView.ReverseSortDirection()` (new method)
4. Trigger data reload with updated sort parameters

### New Method in TableView

```go
// ReverseSortDirection reverses the current sort direction
// Returns true if there was an active sort to reverse
func (tv *TableView) ReverseSortDirection() bool {
    if tv.SortColumn < 0 {
        return false
    }
    if tv.SortDirection == "ASC" {
        tv.SortDirection = "DESC"
    } else {
        tv.SortDirection = "ASC"
    }
    return true
}
```

### Files to Modify

1. `internal/ui/components/table_view.go` - Add `ReverseSortDirection()` method
2. `internal/app/app.go` - Add `r` key handler in Data tab section

## Research Summary

### How Other Tools Handle Sorting

| Tool | Sorting Method | Notes |
|------|----------------|-------|
| DBeaver | Hybrid | Click header = client-side; refresh = server-side ORDER BY |
| DataGrip | Server-side | "Sort via ORDER BY" option sends new query |
| lazysql | Unknown | `K`/`J` for ASC/DESC |
| htop | Client-side | `Shift+I` to reverse sort |

### Why Server-Side Sorting (Current Approach)

- Correctly sorts entire table (not just loaded rows)
- Works with pagination (LIMIT/OFFSET)
- Uses database collation for accurate string sorting
- No memory pressure for large tables

## Sources

- [DBeaver Data Ordering](https://github.com/dbeaver/cloudbeaver/wiki/Data-Ordering)
- [DataGrip Sort Data](https://www.jetbrains.com/help/datagrip/tables-sort.html)
- [lazysql GitHub](https://github.com/jorgerojas26/lazysql)
- [htop sorting shortcuts](https://www.howtogeek.com/how-to-use-linux-htop-command/)
