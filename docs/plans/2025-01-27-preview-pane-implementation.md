# Preview Pane Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a unified preview pane component that shows full content when table cells or JSONB tree nodes are truncated.

**Architecture:** A new `PreviewPane` component in `internal/ui/components/` that can be embedded in both `TableView` and `JSONBViewer`. The pane auto-shows when content is truncated, supports scrolling for large content, and formats JSON automatically.

**Tech Stack:** Go, Bubble Tea, Lipgloss, go-runewidth

---

## Task 1: Create PreviewPane Component Structure

**Files:**
- Create: `internal/ui/components/preview_pane.go`

**Step 1: Write the component structure**

```go
package components

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/jsonb"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// PreviewPane displays full content for truncated values
type PreviewPane struct {
	Width     int
	MaxHeight int    // Maximum height (screen 1/3)
	Content   string // Raw content to display
	Title     string // Title (column name or JSON path)

	// Visibility state
	Visible       bool // Whether pane should be shown
	ForceHidden   bool // User manually hid the pane (overrides auto-show)
	IsTruncated   bool // Whether content was truncated in parent view

	// Scrolling
	scrollY       int
	contentLines  []string // Formatted content split into lines

	// Styling
	Theme theme.Theme
	style lipgloss.Style
}

// NewPreviewPane creates a new preview pane
func NewPreviewPane(th theme.Theme) *PreviewPane {
	return &PreviewPane{
		Width:     80,
		MaxHeight: 10,
		Theme:     th,
		style: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(th.Border).
			Padding(0, 1),
	}
}
```

**Step 2: Run build to verify it compiles**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/preview_pane.go
git commit -m "feat: add PreviewPane component structure"
```

---

## Task 2: Implement SetContent Method

**Files:**
- Modify: `internal/ui/components/preview_pane.go`

**Step 1: Add SetContent method**

Add after NewPreviewPane:

```go
// SetContent sets the content to display
// isTruncated indicates whether the content was truncated in the parent view
func (p *PreviewPane) SetContent(content, title string, isTruncated bool) {
	p.Content = content
	p.Title = title
	p.IsTruncated = isTruncated
	p.scrollY = 0

	// Format content
	p.formatContent()

	// Update visibility (only auto-show if not force hidden)
	if !p.ForceHidden {
		p.Visible = isTruncated && content != "" && content != "NULL"
	}
}

// formatContent formats the raw content for display
func (p *PreviewPane) formatContent() {
	if p.Content == "" {
		p.contentLines = []string{}
		return
	}

	// Calculate available width for content
	contentWidth := p.Width - p.style.GetHorizontalFrameSize()
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Try to format as JSON if it looks like JSONB
	formatted := p.Content
	if jsonb.IsJSONB(p.Content) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(p.Content), &parsed); err == nil {
			if pretty, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				formatted = string(pretty)
			}
		}
	}

	// Wrap lines to fit width
	p.contentLines = p.wrapText(formatted, contentWidth)
}

// wrapText wraps text to fit within maxWidth
func (p *PreviewPane) wrapText(text string, maxWidth int) []string {
	var result []string
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		if runewidth.StringWidth(line) <= maxWidth {
			result = append(result, line)
			continue
		}

		// Wrap long lines
		current := ""
		currentWidth := 0
		for _, r := range line {
			rWidth := runewidth.RuneWidth(r)
			if currentWidth+rWidth > maxWidth {
				result = append(result, current)
				current = string(r)
				currentWidth = rWidth
			} else {
				current += string(r)
				currentWidth += rWidth
			}
		}
		if current != "" {
			result = append(result, current)
		}
	}

	return result
}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/preview_pane.go
git commit -m "feat: add SetContent and formatContent methods to PreviewPane"
```

---

## Task 3: Implement Toggle and Height Methods

**Files:**
- Modify: `internal/ui/components/preview_pane.go`

**Step 1: Add Toggle and Height methods**

Add after wrapText:

```go
// Toggle toggles the preview pane visibility
// When toggled off, sets ForceHidden to prevent auto-show
// When toggled on, clears ForceHidden to allow auto-show
func (p *PreviewPane) Toggle() {
	if p.Visible {
		p.Visible = false
		p.ForceHidden = true
	} else {
		p.ForceHidden = false
		// Only show if content is truncated
		p.Visible = p.IsTruncated && p.Content != "" && p.Content != "NULL"
	}
}

// Height returns the actual rendered height including borders
// Returns 0 if not visible
func (p *PreviewPane) Height() int {
	if !p.Visible {
		return 0
	}

	// Calculate content height
	contentHeight := len(p.contentLines)
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Apply max height constraint
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	if maxContentHeight < 1 {
		maxContentHeight = 1
	}
	if contentHeight > maxContentHeight {
		contentHeight = maxContentHeight
	}

	// Add frame size for total height
	return contentHeight + p.style.GetVerticalFrameSize()
}

// IsScrollable returns true if content exceeds visible area
func (p *PreviewPane) IsScrollable() bool {
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	return len(p.contentLines) > maxContentHeight
}

// ScrollUp scrolls content up
func (p *PreviewPane) ScrollUp() {
	if p.scrollY > 0 {
		p.scrollY--
	}
}

// ScrollDown scrolls content down
func (p *PreviewPane) ScrollDown() {
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	maxScroll := len(p.contentLines) - maxContentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scrollY < maxScroll {
		p.scrollY++
	}
}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/preview_pane.go
git commit -m "feat: add Toggle, Height, and scroll methods to PreviewPane"
```

---

## Task 4: Implement View Method

**Files:**
- Modify: `internal/ui/components/preview_pane.go`

**Step 1: Add View method**

Add after ScrollDown:

```go
// View renders the preview pane
func (p *PreviewPane) View() string {
	if !p.Visible {
		return ""
	}

	// Calculate dimensions
	contentWidth := p.Width - p.style.GetHorizontalFrameSize()
	maxContentHeight := p.MaxHeight - p.style.GetVerticalFrameSize()
	if maxContentHeight < 1 {
		maxContentHeight = 1
	}

	// Build header
	titleStyle := lipgloss.NewStyle().
		Foreground(p.Theme.Info).
		Bold(true)

	header := titleStyle.Render("Preview")
	if p.Title != "" {
		header = titleStyle.Render("Preview: " + p.Title)
	}

	// Truncate header if too long
	if runewidth.StringWidth(header) > contentWidth-4 {
		header = runewidth.Truncate(header, contentWidth-4, "...")
	}

	// Get visible content lines
	startLine := p.scrollY
	endLine := startLine + maxContentHeight - 1 // -1 for header
	if endLine > len(p.contentLines) {
		endLine = len(p.contentLines)
	}

	var contentParts []string
	contentParts = append(contentParts, header)

	// Add content lines
	contentStyle := lipgloss.NewStyle().Foreground(p.Theme.Foreground)
	for i := startLine; i < endLine; i++ {
		line := p.contentLines[i]
		// Truncate line if too long
		if runewidth.StringWidth(line) > contentWidth {
			line = runewidth.Truncate(line, contentWidth, "...")
		}
		contentParts = append(contentParts, contentStyle.Render(line))
	}

	// Build help text
	helpParts := []string{}
	if p.IsScrollable() {
		helpParts = append(helpParts, "↑↓: Scroll")
	}
	helpParts = append(helpParts, "p: Toggle")

	// Add JSONB hint if content is JSON
	if jsonb.IsJSONB(p.Content) {
		helpParts = append(helpParts, "J: Tree")
	}

	helpText := strings.Join(helpParts, " │ ")
	helpStyle := lipgloss.NewStyle().
		Foreground(p.Theme.Metadata).
		Italic(true)

	// Build footer with right-aligned help
	footerPadding := contentWidth - runewidth.StringWidth(helpText)
	if footerPadding < 0 {
		footerPadding = 0
	}
	footer := strings.Repeat(" ", footerPadding) + helpStyle.Render(helpText)
	contentParts = append(contentParts, footer)

	// Join content
	content := strings.Join(contentParts, "\n")

	// Apply container style
	containerStyle := p.style.Copy().
		Width(p.Width).
		MaxHeight(p.MaxHeight)

	return containerStyle.Render(content)
}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/preview_pane.go
git commit -m "feat: add View method to PreviewPane"
```

---

## Task 5: Add PreviewPane to TableView

**Files:**
- Modify: `internal/ui/components/table_view.go`

**Step 1: Add PreviewPane field to TableView struct**

In `table_view.go`, add to the TableView struct (after line 47):

```go
	// Preview pane for truncated content
	PreviewPane *PreviewPane
```

**Step 2: Initialize PreviewPane in NewTableView**

Modify NewTableView function to initialize the preview pane:

```go
func NewTableView(th theme.Theme) *TableView {
	return &TableView{
		Columns:       []string{},
		Rows:          [][]string{},
		ColumnWidths:  []int{},
		Theme:         th,
		SortColumn:    -1,
		SortDirection: "ASC",
		NullsFirst:    false,
		PreviewPane:   NewPreviewPane(th),
	}
}
```

**Step 3: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/ui/components/table_view.go
git commit -m "feat: add PreviewPane to TableView"
```

---

## Task 6: Add Truncation Detection to TableView

**Files:**
- Modify: `internal/ui/components/table_view.go`

**Step 1: Add method to check if current cell is truncated**

Add after GetMatchInfo method:

```go
// IsCellTruncated checks if the currently selected cell content is truncated
func (tv *TableView) IsCellTruncated() bool {
	if tv.SelectedRow < 0 || tv.SelectedRow >= len(tv.Rows) {
		return false
	}
	if tv.SelectedCol < 0 || tv.SelectedCol >= len(tv.ColumnWidths) {
		return false
	}
	if tv.SelectedCol >= len(tv.Rows[tv.SelectedRow]) {
		return false
	}

	cellValue := tv.Rows[tv.SelectedRow][tv.SelectedCol]
	colWidth := tv.ColumnWidths[tv.SelectedCol]

	// Check if cell content width exceeds column width
	return runewidth.StringWidth(cellValue) > colWidth
}

// GetSelectedCellContent returns the full content of the selected cell
func (tv *TableView) GetSelectedCellContent() string {
	if tv.SelectedRow < 0 || tv.SelectedRow >= len(tv.Rows) {
		return ""
	}
	if tv.SelectedCol < 0 || tv.SelectedCol >= len(tv.Rows[tv.SelectedRow]) {
		return ""
	}
	return tv.Rows[tv.SelectedRow][tv.SelectedCol]
}

// GetSelectedColumnName returns the name of the currently selected column
func (tv *TableView) GetSelectedColumnName() string {
	if tv.SelectedCol < 0 || tv.SelectedCol >= len(tv.Columns) {
		return ""
	}
	return tv.Columns[tv.SelectedCol]
}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/table_view.go
git commit -m "feat: add truncation detection methods to TableView"
```

---

## Task 7: Add UpdatePreviewPane Method to TableView

**Files:**
- Modify: `internal/ui/components/table_view.go`

**Step 1: Add method to update preview pane based on selection**

Add after GetSelectedColumnName:

```go
// UpdatePreviewPane updates the preview pane with current selection
func (tv *TableView) UpdatePreviewPane() {
	if tv.PreviewPane == nil {
		return
	}

	content := tv.GetSelectedCellContent()
	title := tv.GetSelectedColumnName()
	isTruncated := tv.IsCellTruncated()

	tv.PreviewPane.SetContent(content, title, isTruncated)
}

// SetPreviewPaneDimensions sets the dimensions for the preview pane
func (tv *TableView) SetPreviewPaneDimensions(width, maxHeight int) {
	if tv.PreviewPane != nil {
		tv.PreviewPane.Width = width
		tv.PreviewPane.MaxHeight = maxHeight
	}
}

// TogglePreviewPane toggles the preview pane visibility
func (tv *TableView) TogglePreviewPane() {
	if tv.PreviewPane != nil {
		tv.PreviewPane.Toggle()
	}
}

// GetPreviewPaneHeight returns the current preview pane height
func (tv *TableView) GetPreviewPaneHeight() int {
	if tv.PreviewPane != nil {
		return tv.PreviewPane.Height()
	}
	return 0
}
```

**Step 2: Call UpdatePreviewPane in movement methods**

Modify MoveSelection method to call UpdatePreviewPane at the end:

```go
// MoveSelection moves the selection up or down
func (tv *TableView) MoveSelection(delta int) {
	tv.SelectedRow += delta

	// Bounds checking
	if tv.SelectedRow < 0 {
		tv.SelectedRow = 0
	}
	if tv.SelectedRow >= len(tv.Rows) {
		tv.SelectedRow = len(tv.Rows) - 1
	}

	// Adjust visible window if needed
	if tv.SelectedRow < tv.TopRow {
		tv.TopRow = tv.SelectedRow
	}
	if tv.SelectedRow >= tv.TopRow+tv.VisibleRows {
		tv.TopRow = tv.SelectedRow - tv.VisibleRows + 1
	}

	// Update preview pane
	tv.UpdatePreviewPane()
}
```

Similarly modify MoveSelectionHorizontal:

```go
// MoveSelectionHorizontal moves the selected column left or right with auto-scroll
func (tv *TableView) MoveSelectionHorizontal(delta int) {
	tv.SelectedCol += delta

	// Bounds checking
	if tv.SelectedCol < 0 {
		tv.SelectedCol = 0
	}
	if tv.SelectedCol >= len(tv.Columns) {
		tv.SelectedCol = len(tv.Columns) - 1
	}

	// Auto-scroll to keep selected column visible
	if tv.SelectedCol < tv.LeftColOffset {
		tv.LeftColOffset = tv.SelectedCol
	}
	if tv.SelectedCol >= tv.LeftColOffset+tv.VisibleCols {
		tv.LeftColOffset = tv.SelectedCol - tv.VisibleCols + 1
	}

	// Bounds check LeftColOffset
	if tv.LeftColOffset < 0 {
		tv.LeftColOffset = 0
	}
	maxOffset := len(tv.Columns) - tv.VisibleCols
	if maxOffset < 0 {
		maxOffset = 0
	}
	if tv.LeftColOffset > maxOffset {
		tv.LeftColOffset = maxOffset
	}

	// Update preview pane
	tv.UpdatePreviewPane()
}
```

**Step 3: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/ui/components/table_view.go
git commit -m "feat: add UpdatePreviewPane and integrate with movement methods"
```

---

## Task 8: Integrate PreviewPane into App Right Panel

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Handle 'p' key for preview pane toggle**

In the Update method, under the right panel key handling (around line 839), add a case for 'p':

After the `case "/":` block (around line 953), add:

```go
				case "p":
					// Toggle preview pane
					a.tableView.TogglePreviewPane()
					return a, nil
```

**Step 2: Update renderRightPanel to include preview pane**

Modify the renderRightPanel method to render the preview pane:

```go
// renderRightPanel renders the right panel content based on current state
func (a *App) renderRightPanel(width, height int) string {
	// If table is selected, show structure view with tabs
	if a.currentTable != "" {
		// Calculate preview pane height
		previewHeight := 0
		if a.currentTab == 0 && a.tableView.PreviewPane != nil {
			// Set preview pane dimensions (max 1/3 of available height)
			maxPreviewHeight := height / 3
			if maxPreviewHeight < 5 {
				maxPreviewHeight = 5
			}
			a.tableView.SetPreviewPaneDimensions(width, maxPreviewHeight)
			a.tableView.UpdatePreviewPane()
			previewHeight = a.tableView.GetPreviewPaneHeight()
		}

		// Calculate main content height (subtract preview pane height)
		mainHeight := height - previewHeight
		if mainHeight < 5 {
			mainHeight = 5
		}

		// Update structure view dimensions
		a.structureView.Width = width
		a.structureView.Height = mainHeight

		// Load table structure if needed (when table changes)
		conn, err := a.connectionManager.GetActive()
		if err == nil && conn != nil && conn.Pool != nil {
			parts := strings.Split(a.currentTable, ".")
			if len(parts) == 2 {
				// Only load if we haven't loaded this table yet
				if !a.structureView.HasTableLoaded(parts[0], parts[1]) {
					ctx := context.Background()
					err := a.structureView.SetTable(ctx, conn.Pool, parts[0], parts[1])
					if err != nil {
						log.Printf("Failed to load structure: %v", err)
					}
				}
			}
		}

		// Render main content
		mainContent := a.structureView.View()

		// If on Data tab and preview pane is visible, append it
		if a.currentTab == 0 && previewHeight > 0 {
			previewContent := a.tableView.PreviewPane.View()
			return lipgloss.JoinVertical(lipgloss.Left, mainContent, previewContent)
		}

		return mainContent
	}

	// No table selected - show table view (will display empty state)
	a.tableView.Width = width
	a.tableView.Height = height
	return a.tableView.View()
}
```

**Step 3: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: integrate PreviewPane into right panel rendering"
```

---

## Task 9: Add PreviewPane to JSONBViewer

**Files:**
- Modify: `internal/ui/components/jsonb_viewer.go`

**Step 1: Add PreviewPane field to JSONBViewer struct**

In the JSONBViewer struct (around line 42), add after `statusMessage`:

```go
	// Preview pane for truncated string values
	previewPane *PreviewPane
```

**Step 2: Initialize PreviewPane in NewJSONBViewer**

Modify NewJSONBViewer:

```go
func NewJSONBViewer(th theme.Theme) *JSONBViewer {
	return &JSONBViewer{
		Width:         80,
		Height:        30,
		Theme:         th,
		selectedIndex: 0,
		scrollOffset:  0,
		marks:         make(map[rune]*TreeNode),
		previewPane:   NewPreviewPane(th),
	}
}
```

**Step 3: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/ui/components/jsonb_viewer.go
git commit -m "feat: add PreviewPane to JSONBViewer"
```

---

## Task 10: Integrate PreviewPane into JSONBViewer Navigation

**Files:**
- Modify: `internal/ui/components/jsonb_viewer.go`

**Step 1: Add method to update preview pane**

Add after the copyCurrentValue method:

```go
// updatePreviewPane updates the preview pane with current node info
func (jv *JSONBViewer) updatePreviewPane() {
	if jv.previewPane == nil || jv.selectedIndex >= len(jv.visibleNodes) {
		return
	}

	node := jv.visibleNodes[jv.selectedIndex]

	// Build JSON path
	var path string
	if len(node.Path) > 0 {
		path = "$." + strings.Join(node.Path, ".")
	} else {
		path = "$"
	}

	// Check if value is truncated (string > 50 chars)
	isTruncated := false
	content := ""

	switch node.Type {
	case NodeString:
		str := fmt.Sprintf("%v", node.Value)
		content = str
		isTruncated = len(str) > 50
	case NodeObject, NodeArray:
		// Format as JSON
		if jsonBytes, err := json.MarshalIndent(node.Value, "", "  "); err == nil {
			content = string(jsonBytes)
			isTruncated = true // Always show for objects/arrays
		}
	case NodeNumber, NodeBoolean:
		content = fmt.Sprintf("%v", node.Value)
		isTruncated = false
	case NodeNull:
		content = "null"
		isTruncated = false
	}

	jv.previewPane.SetContent(content, path, isTruncated)
}
```

**Step 2: Call updatePreviewPane in adjustScroll**

Modify the adjustScroll method to update preview pane:

```go
// adjustScroll adjusts scroll offset to keep selected node visible
func (jv *JSONBViewer) adjustScroll() {
	contentHeight := jv.Height - 5 // Account for header and footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Scroll up if selected is above viewport
	if jv.selectedIndex < jv.scrollOffset {
		jv.scrollOffset = jv.selectedIndex
	}

	// Scroll down if selected is below viewport
	if jv.selectedIndex >= jv.scrollOffset+contentHeight {
		jv.scrollOffset = jv.selectedIndex - contentHeight + 1
	}

	// Update preview pane
	jv.updatePreviewPane()
}
```

**Step 3: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/ui/components/jsonb_viewer.go
git commit -m "feat: integrate PreviewPane with JSONBViewer navigation"
```

---

## Task 11: Add 'p' Key Handler to JSONBViewer

**Files:**
- Modify: `internal/ui/components/jsonb_viewer.go`

**Step 1: Modify the Update method case "p"**

The existing 'p' key is used for "jump to parent". We need to use a different key or add preview toggle functionality. Since 'p' is already taken, we'll keep preview toggle on a different key in JSONBViewer. Let's check if there's a conflict.

Looking at line 376-378, 'p' is used for "Jump to parent". We should NOT change this since it's documented and useful.

For JSONBViewer, since it's a modal that already shows content, we can skip the preview pane integration and just show full content inline. However, for long strings that are truncated, we can show preview pane with 'P' (uppercase).

Modify the Update method to add case for "P":

In the switch statement around line 242, add after the "p" case:

```go
	case "P":
		// Toggle preview pane
		if jv.previewPane != nil {
			jv.previewPane.Toggle()
		}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/jsonb_viewer.go
git commit -m "feat: add P key to toggle preview pane in JSONBViewer"
```

---

## Task 12: Integrate PreviewPane into JSONBViewer View

**Files:**
- Modify: `internal/ui/components/jsonb_viewer.go`

**Step 1: Modify the View method to include preview pane**

In the View method (around line 574), modify to include preview pane:

```go
// View renders the JSONB viewer
func (jv *JSONBViewer) View() string {
	var sections []string

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Foreground(jv.Theme.Background).
		Background(jv.Theme.Info).
		Padding(0, 1).
		Bold(true)

	title := " JSONB Tree Viewer"
	sections = append(sections, titleStyle.Render(title))

	// Instructions or search bar
	instrStyle := lipgloss.NewStyle().
		Foreground(jv.Theme.Metadata).
		Padding(0, 1)

	if jv.searchMode {
		searchBar := fmt.Sprintf("Search: %s_", jv.searchQuery)
		if len(jv.searchResults) > 0 {
			searchBar += fmt.Sprintf("  (%d matches)", len(jv.searchResults))
		}
		sections = append(sections, instrStyle.Render(searchBar))
	} else if jv.quickJumpMode {
		// Show mark/jump mode
		var modeInfo string
		if jv.quickJumpBuffer == "m" {
			modeInfo = "Mark mode: Press a-z to set mark"
		} else if jv.quickJumpBuffer == "'" {
			modeInfo = "Jump mode: Press a-z to jump to mark"
		}
		sections = append(sections, instrStyle.Render(modeInfo))
	} else if len(jv.searchResults) > 0 {
		// Show search results navigation info
		searchInfo := fmt.Sprintf("Search: \"%s\" (%d/%d)  n: Next  N: Prev  Esc: Clear",
			jv.searchQuery, jv.currentMatchIndex+1, len(jv.searchResults))
		sections = append(sections, instrStyle.Render(searchInfo))
	} else {
		// Show help text - rotate through different hints
		instr := "↑↓/jk: Move  g/G: Top/Bottom  Ctrl-f/b: Page  JK: Sibling  p: Parent  ]/[: Jump Type  y: Copy Path  m/': Mark  /: Search  ?: Help"
		sections = append(sections, instrStyle.Render(instr))
	}

	// Calculate preview pane height
	previewHeight := 0
	if jv.previewPane != nil && jv.previewPane.Visible {
		// Set preview pane dimensions
		maxPreviewHeight := jv.Height / 4
		if maxPreviewHeight < 4 {
			maxPreviewHeight = 4
		}
		jv.previewPane.Width = jv.Width - 4 // Account for container padding
		jv.previewPane.MaxHeight = maxPreviewHeight
		previewHeight = jv.previewPane.Height()
	}

	// Content (tree view or help) - adjust height for preview pane
	contentHeight := jv.Height - 5 - previewHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string
	if jv.helpMode {
		content = jv.renderHelp()
	} else {
		content = jv.renderTree(contentHeight)
	}
	sections = append(sections, content)

	// Preview pane (if visible)
	if jv.previewPane != nil && previewHeight > 0 {
		sections = append(sections, jv.previewPane.View())
	}

	// Status bar
	statusBar := jv.renderStatus()
	sections = append(sections, statusBar)

	// Container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(jv.Theme.Border).
		Width(jv.Width).
		Padding(1)

	return lipgloss.Place(
		jv.Width,
		jv.Height,
		lipgloss.Center,
		lipgloss.Center,
		containerStyle.Render(strings.Join(sections, "\n")),
	)
}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/ui/components/jsonb_viewer.go
git commit -m "feat: integrate PreviewPane into JSONBViewer View"
```

---

## Task 13: Add Scroll Handling for PreviewPane

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add scroll key handling when preview pane is visible**

This is tricky because we need to handle scroll keys for preview pane without conflicting with table navigation. Since the preview pane doesn't have focus, we'll use modifier keys.

For simplicity, let's use `ctrl+up` and `ctrl+down` for preview pane scrolling when it's visible.

In the Update method, in the right panel handling section (around line 839), add before the existing key handlers:

```go
			// Handle preview pane scrolling (when visible)
			if a.tableView.PreviewPane != nil && a.tableView.PreviewPane.Visible {
				switch msg.String() {
				case "ctrl+up":
					a.tableView.PreviewPane.ScrollUp()
					return a, nil
				case "ctrl+down":
					a.tableView.PreviewPane.ScrollDown()
					return a, nil
				}
			}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: add ctrl+up/down for preview pane scrolling"
```

---

## Task 14: Update Bottom Bar Help Text

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Update bottom bar to show preview pane hint**

In the renderNormalView method, update the bottom bar help text for right panel (around line 1175):

```go
		// Table navigation keys
		bottomBarLeft = keyStyle.Render("↑↓") + dimStyle.Render(" navigate") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("Ctrl+D/U") + dimStyle.Render(" page") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("p") + dimStyle.Render(" preview") +
			separatorStyle.Render(" │ ") +
			keyStyle.Render("J") + dimStyle.Render(" jsonb")
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: update bottom bar to show preview pane hint"
```

---

## Task 15: Add Copy Functionality to PreviewPane

**Files:**
- Modify: `internal/ui/components/preview_pane.go`

**Step 1: Add Copy method**

Add import for clipboard and add Copy method:

At the top, add import:
```go
import (
	"encoding/json"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rebeliceyang/lazypg/internal/jsonb"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)
```

Add method after ScrollDown:

```go
// CopyContent copies the preview content to clipboard
func (p *PreviewPane) CopyContent() error {
	return clipboard.WriteAll(p.Content)
}
```

**Step 2: Add 'y' key handling to update help text**

Update the View method to include 'y' in help text:

```go
	// Build help text
	helpParts := []string{}
	if p.IsScrollable() {
		helpParts = append(helpParts, "↑↓: Scroll")
	}
	helpParts = append(helpParts, "p: Toggle")
	helpParts = append(helpParts, "y: Copy")

	// Add JSONB hint if content is JSON
	if jsonb.IsJSONB(p.Content) {
		helpParts = append(helpParts, "J: Tree")
	}
```

**Step 3: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/ui/components/preview_pane.go
git commit -m "feat: add copy functionality to PreviewPane"
```

---

## Task 16: Handle 'y' Key for Preview Copy in App

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add 'y' key handling when preview pane is visible**

In the Update method, in the right panel handling section, modify the 'y' key handling to check for preview pane:

Find the existing 'y' key handling (around line 794) and modify it:

```go
		case "y":
			// Copy functionality
			if a.state.FocusedPanel == models.RightPanel {
				// If preview pane is visible, copy its content
				if a.tableView.PreviewPane != nil && a.tableView.PreviewPane.Visible {
					if err := a.tableView.PreviewPane.CopyContent(); err != nil {
						log.Printf("Failed to copy: %v", err)
					} else {
						log.Println("Copied preview content")
					}
					return a, nil
				}
			}
			// Original structure view copy (copy name)
			if a.currentTab > 0 {
				statusMsg := a.structureView.CopyCurrentName()
				if statusMsg != "" {
					log.Println(statusMsg)
				}
				return a, nil
			}
```

**Step 2: Run build to verify**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: add y key to copy preview pane content"
```

---

## Task 17: Final Build and Test

**Files:**
- All modified files

**Step 1: Run full build**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go build ./...`
Expected: Build succeeds with no errors

**Step 2: Run tests**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go test ./... 2>&1 | grep -E '(PASS|FAIL|ok|---)'`
Expected: Tests pass (note: some existing tests may fail as noted earlier)

**Step 3: Manual test**

Run: `cd /Users/rebeliceyang/Github/lazypg/.worktrees/preview-pane && go run main.go`
Expected: Application launches, connect to a database, select a table with long content, preview pane should appear automatically

**Step 4: Commit final changes if any fixes needed**

```bash
git add -A
git commit -m "feat: complete PreviewPane implementation"
```

---

## Summary

This implementation plan creates:

1. **PreviewPane component** (`internal/ui/components/preview_pane.go`)
   - Auto-show when content is truncated
   - Smart JSON formatting
   - Scrolling support
   - Copy functionality

2. **TableView integration**
   - Truncation detection
   - Preview pane updates on selection change
   - 'p' key to toggle

3. **JSONBViewer integration**
   - Preview pane for truncated strings
   - 'P' key to toggle (since 'p' is already used for parent)

4. **App integration**
   - Right panel layout with preview pane
   - Key bindings for scroll and copy
   - Updated help text

Key bindings:
- `p` - Toggle preview pane (in table view)
- `P` - Toggle preview pane (in JSONB viewer)
- `ctrl+up/down` - Scroll preview pane
- `y` - Copy preview content
- `J` - Open JSONB Tree Viewer (hint shown in preview)
