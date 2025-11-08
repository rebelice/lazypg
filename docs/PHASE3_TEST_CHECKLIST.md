# Phase 3 Testing Checklist

## Pre-Test Setup
- [x] PostgreSQL running on localhost:5432
- [x] Application built at `bin/lazypg`
- [ ] Terminal size at least 80x24

## Test 1: Application Launch
**Steps:**
1. Run `./bin/lazypg`
2. Verify UI displays:
   - Top bar: "lazypg" with "⌘K"
   - Left panel: "Navigation" with tree view
   - Right panel: "Content"
   - Bottom bar: keyboard shortcuts

**Expected:**
- Clean UI render without artifacts
- No error messages
- Help text shows "[tab] Switch panel | [q] Quit"

**Status:** [ ]

---

## Test 2: Connection Discovery
**Steps:**
1. Press `c` to open connection dialog
2. Wait 5 seconds for auto-discovery
3. Verify discovered instances appear

**Expected:**
- Connection dialog displays
- Shows "Auto-Discovered Instances" section
- Lists local PostgreSQL on port 5432
- Status shows "Discovering..."

**Status:** [ ]

---

## Test 3: Database Connection (Auto-Discovered)
**Steps:**
1. In connection dialog, select discovered instance with `↓/j`
2. Press `Enter` to connect
3. Observe connection attempt

**Expected:**
- Dialog closes
- Left panel starts loading database tree
- No error overlay (if credentials are correct)

**Alternative - If auth fails:**
- Error overlay appears with clear message
- Press `Esc` to dismiss
- Try manual mode (see Test 4)

**Status:** [ ]

---

## Test 4: Manual Connection
**Steps:**
1. Press `c` for connection dialog
2. Press `m` to toggle manual mode
3. Enter connection details:
   - Host: localhost
   - Port: 5432
   - Database: postgres (or your db name)
   - User: your_username
   - Password: (if needed)
4. Press `Enter`

**Expected:**
- Form shows editable fields
- Can navigate with `Tab`/`Shift+Tab` between fields
- Connection succeeds or shows error

**Status:** [ ]

---

## Test 5: Navigation Tree Display
**Prerequisites:** Connected to database

**Steps:**
1. Observe left panel after connection
2. Verify tree structure shows:
   - Database name (e.g., "postgres")
   - Schema nodes (collapsed by default)

**Expected:**
- Tree renders with proper indentation
- Expand/collapse icons (▶ for collapsed, ▼ for expanded)
- Cursor on first item
- Visual hierarchy clear

**Status:** [ ]

---

## Test 6: Tree Navigation - Keyboard
**Steps:**
1. Press `j` or `↓` - move down
2. Press `k` or `↑` - move up
3. Press `Space` or `Enter` on schema - expand
4. Press `Space` or `Enter` again - collapse
5. Press `g` - jump to top
6. Press `G` (Shift+g) - jump to bottom

**Expected:**
- Cursor moves smoothly
- Selected item highlighted
- Expand/collapse works
- Jump commands work instantly

**Status:** [ ]

---

## Test 7: Schema and Table Display
**Steps:**
1. Navigate to a schema (e.g., "public")
2. Press `Space` or `Enter` to expand
3. Verify tables appear under schema
4. Count visible tables

**Expected:**
- Schema expands showing table nodes
- Tables have table icon (📊)
- Table names match actual database tables
- Indentation shows hierarchy

**Status:** [ ]

---

## Test 8: Load Table Data
**Steps:**
1. Navigate to any table node
2. Press `Enter` to load data
3. Observe right panel

**Expected:**
- Focus switches to right panel (right border highlighted)
- Table header shows column names
- Separator line (────┼────)
- Data rows display
- Bottom shows: "Rows 1-N of Total"

**Status:** [ ]

---

## Test 9: Table Navigation
**Prerequisites:** Table data loaded in right panel

**Steps:**
1. Press `j` or `↓` - move down one row
2. Press `k` or `↑` - move up one row
3. Press `Ctrl+D` - page down
4. Press `Ctrl+U` - page up

**Expected:**
- Selected row highlighted (blue background)
- Smooth scrolling
- Page commands jump by viewport height
- Status bar updates row numbers

**Status:** [ ]

---

## Test 10: Lazy Loading (Large Tables)
**Prerequisites:** Table with >100 rows

**Steps:**
1. Load a large table
2. Verify initial load shows "Rows 1-100 of XXX"
3. Press `j` repeatedly to scroll down
4. When reaching row ~90, continue scrolling
5. Observe status bar

**Expected:**
- First load: 100 rows maximum
- As you approach bottom (~10 rows from end)
- Next 100 rows load automatically
- Status updates: "Rows 1-200 of XXX"
- No visible lag/stutter

**Status:** [ ]

---

## Test 11: Virtual Scrolling Performance
**Prerequisites:** Large table loaded

**Steps:**
1. Hold down `j` key
2. Observe scrolling smoothness
3. Try `Ctrl+D` multiple times rapidly
4. Monitor memory (optional: `ps aux | grep lazypg`)

**Expected:**
- Smooth scrolling, no lag
- UI remains responsive
- Memory usage stays reasonable
- Only visible rows + buffer rendered

**Status:** [ ]

---

## Test 12: Panel Switching
**Steps:**
1. Press `Tab` from right panel
2. Verify focus moves to left panel
3. Press `Tab` again
4. Verify focus returns to right panel

**Expected:**
- Border color changes (focused = bright, unfocused = dim)
- Keyboard commands work in focused panel
- Smooth transition

**Status:** [ ]

---

## Test 13: Help System
**Steps:**
1. Press `?` to open help
2. Review keyboard shortcuts
3. Press `?` or `Esc` to close

**Expected:**
- Help overlay centers on screen
- Shows all key bindings organized by category
- Help overlay dismisses cleanly

**Status:** [ ]

---

## Test 14: Error Handling - Connection Failure
**Steps:**
1. Press `c` for connection dialog
2. Press `m` for manual mode
3. Enter invalid host: "nonexistent.host"
4. Press `Enter`

**Expected:**
- Error overlay appears
- Clear error message
- "Press ESC to dismiss" shown
- Dialog doesn't close (stays for retry)

**Status:** [ ]

---

## Test 15: Error Handling - Invalid Table
**Steps:**
1. Manually trigger table load on non-existent table (if possible)
   OR wait for any database error

**Expected:**
- Error overlay appears
- Error message shown
- Can dismiss with `Esc`
- Application doesn't crash

**Status:** [ ]

---

## Test 16: Empty Table
**Prerequisites:** Database with an empty table

**Steps:**
1. Load an empty table (0 rows)
2. Observe display

**Expected:**
- Header still shows column names
- "Rows 0-0 of 0" or similar
- No crash
- Clean empty state

**Status:** [ ]

---

## Test 17: NULL Values
**Prerequisites:** Table with NULL values

**Steps:**
1. Load table containing NULLs
2. Observe NULL cells

**Expected:**
- NULL values display as "NULL"
- Distinguishable from empty strings
- Proper alignment maintained

**Status:** [ ]

---

## Test 18: Wide Tables (Many Columns)
**Prerequisites:** Table with 10+ columns

**Steps:**
1. Load wide table
2. Observe column rendering

**Expected:**
- Columns truncate with "..." if too wide
- All columns visible (horizontal scroll not in Phase 3)
- Layout doesn't break
- Text remains readable

**Status:** [ ]

---

## Test 19: Special Characters in Data
**Prerequisites:** Table with special chars (emoji, unicode, etc.)

**Steps:**
1. Load table with special characters
2. Verify display

**Expected:**
- Special characters render correctly
- No garbled text
- Alignment maintained

**Status:** [ ]

---

## Test 20: Application Exit
**Steps:**
1. Press `q` to quit
2. Verify clean exit

**Alternative:**
1. Press `Ctrl+C`
2. Verify clean exit

**Expected:**
- Application exits immediately
- No errors printed
- Terminal restored to normal state
- No zombie processes

**Status:** [ ]

---

## Summary

**Tests Passed:** __ / 20
**Tests Failed:** __ / 20
**Tests Skipped:** __ / 20

### Issues Found
1.
2.
3.

### Notes
-
-
-

### Next Steps
- [ ] Fix any critical bugs found
- [ ] Document known limitations
- [ ] Plan Phase 4 features
