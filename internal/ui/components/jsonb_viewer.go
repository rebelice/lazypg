package components

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// NodeType represents the type of a JSON node
type NodeType int

const (
	NodeObject NodeType = iota
	NodeArray
	NodeString
	NodeNumber
	NodeBoolean
	NodeNull
)

// TreeNode represents a node in the JSON tree
type TreeNode struct {
	Key        string      // Key name (for object properties)
	Value      interface{} // Raw value
	Type       NodeType    // Type of this node
	IsExpanded bool        // Whether this node is expanded (for objects/arrays)
	Children   []*TreeNode // Child nodes
	Parent     *TreeNode   // Parent node
	Path       []string    // Full path from root to this node
	Level      int         // Indentation level (depth in tree)
}

// CloseJSONBViewerMsg is sent when viewer should close
type CloseJSONBViewerMsg struct{}

// JSONBViewer displays JSONB data as an interactive collapsible tree
type JSONBViewer struct {
	Width  int
	Height int
	Theme  theme.Theme

	// Tree structure
	root *TreeNode

	// Flattened list of visible nodes (for rendering and navigation)
	visibleNodes []*TreeNode

	// Navigation state
	selectedIndex int // Index in visibleNodes
	scrollOffset  int // Scroll offset for viewport

	// Search state
	searchMode        bool
	searchQuery       string
	searchResults     []*TreeNode // All nodes matching search (including collapsed)
	currentMatchIndex int         // Current position in searchResults

	// Quick jump state
	quickJumpMode   bool
	quickJumpBuffer string

	// Marks/bookmarks (a-z)
	marks map[rune]*TreeNode

	// Help mode
	helpMode bool

	// Status message (e.g., "Path copied!")
	statusMessage string

	// Preview pane for truncated string values
	previewPane *PreviewPane
}

// NewJSONBViewer creates a new tree-based JSONB viewer
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

// SetValue parses JSON and builds the tree structure
func (jv *JSONBViewer) SetValue(value interface{}) error {
	// Parse JSON if it's a string
	var parsed interface{}
	switch v := value.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(v, &parsed); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	default:
		parsed = v
	}

	// Build tree
	jv.root = jv.buildTree("root", parsed, nil, []string{}, 0)
	jv.root.IsExpanded = true // Root is always expanded

	// Flatten tree to get visible nodes
	jv.rebuildVisibleNodes()

	// Reset navigation
	jv.selectedIndex = 0
	jv.scrollOffset = 0

	return nil
}

// buildTree recursively builds the tree structure from JSON
func (jv *JSONBViewer) buildTree(key string, value interface{}, parent *TreeNode, path []string, level int) *TreeNode {
	node := &TreeNode{
		Key:        key,
		Value:      value,
		Parent:     parent,
		Path:       path,
		Level:      level,
		IsExpanded: false, // Collapsed by default
	}

	// Determine type and build children
	if value == nil {
		node.Type = NodeNull
		return node
	}

	switch v := value.(type) {
	case map[string]interface{}:
		node.Type = NodeObject
		node.Children = make([]*TreeNode, 0, len(v))
		for childKey, childValue := range v {
			childPath := append([]string{}, path...)
			childPath = append(childPath, childKey)
			childNode := jv.buildTree(childKey, childValue, node, childPath, level+1)
			node.Children = append(node.Children, childNode)
		}

	case []interface{}:
		node.Type = NodeArray
		node.Children = make([]*TreeNode, 0, len(v))
		for i, childValue := range v {
			childKey := fmt.Sprintf("[%d]", i)
			childPath := append([]string{}, path...)
			childPath = append(childPath, fmt.Sprintf("%d", i))
			childNode := jv.buildTree(childKey, childValue, node, childPath, level+1)
			node.Children = append(node.Children, childNode)
		}

	case string:
		node.Type = NodeString

	case float64:
		node.Type = NodeNumber

	case bool:
		node.Type = NodeBoolean

	default:
		node.Type = NodeNull
	}

	return node
}

// rebuildVisibleNodes flattens the tree into a list of visible nodes (respecting collapse state)
func (jv *JSONBViewer) rebuildVisibleNodes() {
	jv.visibleNodes = []*TreeNode{}
	if jv.root != nil {
		jv.flattenTree(jv.root)
	}
}

// flattenTree recursively flattens the tree into visibleNodes
func (jv *JSONBViewer) flattenTree(node *TreeNode) {
	jv.visibleNodes = append(jv.visibleNodes, node)

	// Only recurse into children if node is expanded
	if node.IsExpanded && len(node.Children) > 0 {
		for _, child := range node.Children {
			jv.flattenTree(child)
		}
	}
}

// Update handles keyboard input
func (jv *JSONBViewer) Update(msg tea.KeyMsg) (*JSONBViewer, tea.Cmd) {
	// Clear status message on any key press (except when copying)
	if msg.String() != "y" && msg.String() != "Y" {
		jv.statusMessage = ""
	}

	// Handle help mode
	if jv.helpMode {
		// Any key exits help mode
		jv.helpMode = false
		return jv, nil
	}

	// Handle search mode
	if jv.searchMode {
		switch msg.String() {
		case "esc":
			jv.searchMode = false
			jv.searchQuery = ""
			jv.searchResults = nil
			jv.currentMatchIndex = 0
			return jv, nil
		case "enter":
			// Confirm search and stay in results navigation mode
			jv.searchMode = false
			if len(jv.searchResults) > 0 {
				jv.jumpToMatch(0)
			}
			return jv, nil
		case "backspace":
			if len(jv.searchQuery) > 0 {
				jv.searchQuery = jv.searchQuery[:len(jv.searchQuery)-1]
				jv.performSearch()
			}
			return jv, nil
		default:
			// Append character to search query
			if len(msg.String()) == 1 {
				jv.searchQuery += msg.String()
				jv.performSearch()
			}
			return jv, nil
		}
	}

	// Normal navigation mode
	switch msg.String() {
	case "esc", "q":
		// If we have active search results, clear them first
		if len(jv.searchResults) > 0 {
			jv.searchResults = nil
			jv.searchQuery = ""
			jv.currentMatchIndex = 0
			return jv, nil
		}
		// Otherwise close viewer
		return jv, func() tea.Msg {
			return CloseJSONBViewerMsg{}
		}

	case "up", "k":
		if jv.selectedIndex > 0 {
			jv.selectedIndex--
			jv.adjustScroll()
		}

	case "down", "j":
		if jv.selectedIndex < len(jv.visibleNodes)-1 {
			jv.selectedIndex++
			jv.adjustScroll()
		}

	case " ", "enter":
		// Toggle expand/collapse
		if jv.selectedIndex < len(jv.visibleNodes) {
			node := jv.visibleNodes[jv.selectedIndex]
			if node.Type == NodeObject || node.Type == NodeArray {
				node.IsExpanded = !node.IsExpanded
				jv.rebuildVisibleNodes()
				// Keep selection within bounds after collapse/expand
				if jv.selectedIndex >= len(jv.visibleNodes) {
					jv.selectedIndex = len(jv.visibleNodes) - 1
				}
				if jv.selectedIndex < 0 {
					jv.selectedIndex = 0
				}
				jv.adjustScroll()
			}
		}

	case "E":
		// Expand all
		jv.expandAll(jv.root)
		jv.rebuildVisibleNodes()
		// Keep selection within bounds
		if jv.selectedIndex >= len(jv.visibleNodes) {
			jv.selectedIndex = len(jv.visibleNodes) - 1
		}
		if jv.selectedIndex < 0 {
			jv.selectedIndex = 0
		}
		jv.adjustScroll()

	case "C":
		// Collapse all
		jv.collapseAll(jv.root)
		jv.rebuildVisibleNodes()
		// Keep selection within bounds
		if jv.selectedIndex >= len(jv.visibleNodes) {
			jv.selectedIndex = len(jv.visibleNodes) - 1
		}
		if jv.selectedIndex < 0 {
			jv.selectedIndex = 0
		}
		jv.adjustScroll()

	case "/":
		// Enter search mode
		jv.searchMode = true
		jv.searchQuery = ""
		jv.searchResults = nil
		jv.currentMatchIndex = 0

	case "n":
		// Next search result
		if len(jv.searchResults) > 0 {
			jv.currentMatchIndex++
			if jv.currentMatchIndex >= len(jv.searchResults) {
				jv.currentMatchIndex = 0 // Wrap around
			}
			jv.jumpToMatch(jv.currentMatchIndex)
		}

	case "N":
		// Previous search result
		if len(jv.searchResults) > 0 {
			jv.currentMatchIndex--
			if jv.currentMatchIndex < 0 {
				jv.currentMatchIndex = len(jv.searchResults) - 1 // Wrap around
			}
			jv.jumpToMatch(jv.currentMatchIndex)
		}

	// === Phase 1: Basic Navigation ===
	case "ctrl+f", "pgdown":
		// Page down
		jv.pageDown()

	case "ctrl+b", "pgup":
		// Page up
		jv.pageUp()

	case "ctrl+d":
		// Half page down
		jv.halfPageDown()

	case "ctrl+u":
		// Half page up
		jv.halfPageUp()

	case "g", "home":
		// Jump to first node
		jv.selectedIndex = 0
		jv.adjustScroll()

	case "G", "end":
		// Jump to last node
		if len(jv.visibleNodes) > 0 {
			jv.selectedIndex = len(jv.visibleNodes) - 1
			jv.adjustScroll()
		}

	case "J":
		// Jump to next sibling
		jv.jumpToNextSibling()

	case "K":
		// Jump to previous sibling
		jv.jumpToPrevSibling()

	case "p":
		// Jump to parent
		jv.jumpToParent()

	case "P":
		// Toggle preview pane
		if jv.previewPane != nil {
			jv.previewPane.Toggle()
		}

	// === Phase 2: JSON-specific Navigation ===
	case "]":
		// Jump to next array
		jv.jumpToNextOfType(NodeArray)

	case "[":
		// Jump to previous array
		jv.jumpToPrevOfType(NodeArray)

	case "}":
		// Jump to next object
		jv.jumpToNextOfType(NodeObject)

	case "{":
		// Jump to previous object
		jv.jumpToPrevOfType(NodeObject)

	case "\"":
		// Jump to next string value
		jv.jumpToNextOfType(NodeString)

	case "#":
		// Jump to next number value
		jv.jumpToNextOfType(NodeNumber)

	case "]a":
		// Jump to next array item (within same array)
		jv.jumpToNextArrayItem()

	case "[a":
		// Jump to previous array item (within same array)
		jv.jumpToPrevArrayItem()

	case "y":
		// Copy JSON path to clipboard (yank)
		jv.copyCurrentPath()

	case "Y":
		// Copy current node value to clipboard
		jv.copyCurrentValue()

	// === Phase 3: Advanced Features ===
	case "m":
		// Enter mark mode (next key will be the mark name)
		jv.quickJumpMode = true
		jv.quickJumpBuffer = "m"

	case "'":
		// Enter jump to mark mode
		jv.quickJumpMode = true
		jv.quickJumpBuffer = "'"

	case "?":
		// Toggle help mode
		jv.helpMode = !jv.helpMode

	default:
		// Handle quick jump mode
		if jv.quickJumpMode && len(msg.String()) == 1 {
			jv.handleQuickJump(msg.String())
			return jv, nil
		}

		// Handle first-letter quick jump (when not in any special mode)
		if len(msg.String()) == 1 && msg.String() >= "a" && msg.String() <= "z" {
			jv.jumpToKeyStartingWith(msg.String())
		}
	}

	return jv, nil
}

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

// expandAll recursively expands all nodes
func (jv *JSONBViewer) expandAll(node *TreeNode) {
	if node.Type == NodeObject || node.Type == NodeArray {
		node.IsExpanded = true
		for _, child := range node.Children {
			jv.expandAll(child)
		}
	}
}

// collapseAll recursively collapses all nodes
func (jv *JSONBViewer) collapseAll(node *TreeNode) {
	if node.Type == NodeObject || node.Type == NodeArray {
		node.IsExpanded = false
		for _, child := range node.Children {
			jv.collapseAll(child)
		}
	}
}

// performSearch searches for nodes matching the query (searches ALL nodes, not just visible ones)
func (jv *JSONBViewer) performSearch() {
	jv.searchResults = []*TreeNode{}
	jv.currentMatchIndex = 0

	if jv.searchQuery == "" {
		return
	}

	query := strings.ToLower(jv.searchQuery)
	jv.searchInTree(jv.root, query)

	// If we found results, jump to the first one
	if len(jv.searchResults) > 0 {
		jv.jumpToMatch(0)
	}
}

// searchInTree recursively searches all nodes in the tree
func (jv *JSONBViewer) searchInTree(node *TreeNode, query string) {
	if node == nil {
		return
	}

	// Search in key name
	matchesKey := strings.Contains(strings.ToLower(node.Key), query)

	// Search in value (for primitives)
	matchesValue := false
	if node.Type == NodeString || node.Type == NodeNumber || node.Type == NodeBoolean {
		valueStr := fmt.Sprintf("%v", node.Value)
		matchesValue = strings.Contains(strings.ToLower(valueStr), query)
	}

	// Add to results if matches
	if matchesKey || matchesValue {
		jv.searchResults = append(jv.searchResults, node)
	}

	// Recursively search children
	for _, child := range node.Children {
		jv.searchInTree(child, query)
	}
}

// jumpToMatch navigates to a specific search result
func (jv *JSONBViewer) jumpToMatch(matchIndex int) {
	if matchIndex < 0 || matchIndex >= len(jv.searchResults) {
		return
	}

	targetNode := jv.searchResults[matchIndex]
	jv.currentMatchIndex = matchIndex

	// Expand all parent nodes to make target visible
	jv.expandPathToNode(targetNode)

	// Rebuild visible nodes
	jv.rebuildVisibleNodes()

	// Find the target node in visible nodes and select it
	for i, node := range jv.visibleNodes {
		if node == targetNode {
			jv.selectedIndex = i
			jv.adjustScroll()
			break
		}
	}
}

// expandPathToNode expands all parent nodes leading to the target node
func (jv *JSONBViewer) expandPathToNode(target *TreeNode) {
	if target == nil {
		return
	}

	// Walk up the tree and expand all parents
	current := target.Parent
	for current != nil {
		current.IsExpanded = true
		current = current.Parent
	}
}

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

// renderTree renders the visible portion of the tree
func (jv *JSONBViewer) renderTree(height int) string {
	if len(jv.visibleNodes) == 0 {
		return lipgloss.NewStyle().
			Foreground(jv.Theme.Metadata).
			Italic(true).
			Render("No data")
	}

	var lines []string

	endIndex := jv.scrollOffset + height
	if endIndex > len(jv.visibleNodes) {
		endIndex = len(jv.visibleNodes)
	}

	for i := jv.scrollOffset; i < endIndex; i++ {
		node := jv.visibleNodes[i]
		isSelected := i == jv.selectedIndex
		isSearchMatch := jv.isSearchMatch(node)
		line := jv.renderNode(node, isSelected, isSearchMatch)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// isSearchMatch checks if a node is in the current search results
func (jv *JSONBViewer) isSearchMatch(node *TreeNode) bool {
	for _, match := range jv.searchResults {
		if match == node {
			return true
		}
	}
	return false
}

// renderNode renders a single tree node with proper indentation and styling
func (jv *JSONBViewer) renderNode(node *TreeNode, isSelected bool, isSearchMatch bool) string {
	// Indentation
	indent := strings.Repeat("  ", node.Level)

	// Expand/collapse indicator
	var indicator string
	if node.Type == NodeObject || node.Type == NodeArray {
		if node.IsExpanded {
			indicator = "▼ "
		} else {
			indicator = "▶ "
		}
	} else {
		indicator = "  "
	}

	// Key with syntax highlighting
	keyStyle := lipgloss.NewStyle().Foreground(jv.Theme.Info) // Blue for keys
	keyPart := keyStyle.Render(node.Key)

	// Value with syntax highlighting
	var valuePart string
	switch node.Type {
	case NodeObject:
		count := len(node.Children)
		valuePart = lipgloss.NewStyle().
			Foreground(jv.Theme.Metadata).
			Render(fmt.Sprintf(" { %d properties }", count))

	case NodeArray:
		count := len(node.Children)
		valuePart = lipgloss.NewStyle().
			Foreground(jv.Theme.Metadata).
			Render(fmt.Sprintf(" [ %d items ]", count))

	case NodeString:
		str := fmt.Sprintf("%v", node.Value)
		if len(str) > 50 {
			str = str[:47] + "..."
		}
		valuePart = lipgloss.NewStyle().
			Foreground(jv.Theme.Success). // Green for strings
			Render(fmt.Sprintf(": \"%s\"", str))

	case NodeNumber:
		valuePart = lipgloss.NewStyle().
			Foreground(jv.Theme.Warning). // Yellow/orange for numbers
			Render(fmt.Sprintf(": %v", node.Value))

	case NodeBoolean:
		valuePart = lipgloss.NewStyle().
			Foreground(jv.Theme.Error). // Red for booleans
			Render(fmt.Sprintf(": %v", node.Value))

	case NodeNull:
		valuePart = lipgloss.NewStyle().
			Foreground(jv.Theme.Metadata).
			Italic(true).
			Render(": null")
	}

	// Add search match indicator (only if not selected)
	var searchIndicator string
	if isSearchMatch && !isSelected {
		searchIndicator = lipgloss.NewStyle().
			Foreground(jv.Theme.Warning).
			Render(" 🔍")
	}

	line := indent + indicator + keyPart + valuePart + searchIndicator

	// Priority 1: Highlight selected row (most prominent)
	if isSelected {
		style := lipgloss.NewStyle().
			Background(jv.Theme.BorderFocused). // Bright blue background
			Foreground(jv.Theme.Background).    // Dark text for contrast
			Bold(true).
			Width(jv.Width - 6) // Account for container padding

		// If selected row is also a search match, add indicator
		if isSearchMatch {
			matchIndicator := lipgloss.NewStyle().
				Foreground(jv.Theme.Warning).
				Bold(true).
				Render(" ⭐")
			return style.Render(line + matchIndicator)
		}

		return style.Render(line)
	}

	// Priority 2: Highlight search matches with subtle background (less prominent)
	if isSearchMatch {
		return lipgloss.NewStyle().
			Background(jv.Theme.Selection). // Subtle gray background
			Foreground(jv.Theme.Foreground).
			Width(jv.Width - 6).
			Render(line)
	}

	return line
}

// renderStatus renders the status bar at the bottom
func (jv *JSONBViewer) renderStatus() string {
	totalNodes := len(jv.visibleNodes)
	currentPos := jv.selectedIndex + 1

	var pathStr string
	if jv.selectedIndex < len(jv.visibleNodes) {
		node := jv.visibleNodes[jv.selectedIndex]
		if len(node.Path) > 0 {
			pathStr = "Path: $." + strings.Join(node.Path, ".")
		} else {
			pathStr = "Path: $"
		}

		// Truncate if too long
		maxPathLen := jv.Width - 30
		if len(pathStr) > maxPathLen {
			pathStr = pathStr[:maxPathLen-3] + "..."
		}
	}

	status := fmt.Sprintf(" %d/%d  %s", currentPos, totalNodes, pathStr)

	// Add status message if present
	if jv.statusMessage != "" {
		status = fmt.Sprintf("%s  |  %s", status, jv.statusMessage)
	}

	return lipgloss.NewStyle().
		Foreground(jv.Theme.Metadata).
		Italic(true).
		Render(status)
}

// ============================================================================
// Phase 1: Basic Navigation Methods
// ============================================================================

// pageDown scrolls down one full page
func (jv *JSONBViewer) pageDown() {
	contentHeight := jv.Height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	jv.selectedIndex += contentHeight
	if jv.selectedIndex >= len(jv.visibleNodes) {
		jv.selectedIndex = len(jv.visibleNodes) - 1
	}
	jv.adjustScroll()
}

// pageUp scrolls up one full page
func (jv *JSONBViewer) pageUp() {
	contentHeight := jv.Height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	jv.selectedIndex -= contentHeight
	if jv.selectedIndex < 0 {
		jv.selectedIndex = 0
	}
	jv.adjustScroll()
}

// halfPageDown scrolls down half a page
func (jv *JSONBViewer) halfPageDown() {
	contentHeight := jv.Height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	jv.selectedIndex += contentHeight / 2
	if jv.selectedIndex >= len(jv.visibleNodes) {
		jv.selectedIndex = len(jv.visibleNodes) - 1
	}
	jv.adjustScroll()
}

// halfPageUp scrolls up half a page
func (jv *JSONBViewer) halfPageUp() {
	contentHeight := jv.Height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}
	jv.selectedIndex -= contentHeight / 2
	if jv.selectedIndex < 0 {
		jv.selectedIndex = 0
	}
	jv.adjustScroll()
}

// jumpToNextSibling jumps to the next sibling node (same parent, same level)
func (jv *JSONBViewer) jumpToNextSibling() {
	if jv.selectedIndex >= len(jv.visibleNodes) {
		return
	}

	currentNode := jv.visibleNodes[jv.selectedIndex]
	currentParent := currentNode.Parent

	// Find next sibling
	for i := jv.selectedIndex + 1; i < len(jv.visibleNodes); i++ {
		node := jv.visibleNodes[i]
		
		// Stop if we've gone past siblings (reached parent's sibling or higher level)
		if node.Level < currentNode.Level {
			break
		}
		
		// Found next sibling
		if node.Parent == currentParent && node.Level == currentNode.Level {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}
}

// jumpToPrevSibling jumps to the previous sibling node
func (jv *JSONBViewer) jumpToPrevSibling() {
	if jv.selectedIndex >= len(jv.visibleNodes) || jv.selectedIndex == 0 {
		return
	}

	currentNode := jv.visibleNodes[jv.selectedIndex]
	currentParent := currentNode.Parent

	// Find previous sibling
	for i := jv.selectedIndex - 1; i >= 0; i-- {
		node := jv.visibleNodes[i]
		
		// Stop if we've gone past siblings
		if node.Level < currentNode.Level {
			break
		}
		
		// Found previous sibling
		if node.Parent == currentParent && node.Level == currentNode.Level {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}
}

// jumpToParent jumps to the parent node
func (jv *JSONBViewer) jumpToParent() {
	if jv.selectedIndex >= len(jv.visibleNodes) {
		return
	}

	currentNode := jv.visibleNodes[jv.selectedIndex]
	if currentNode.Parent == nil {
		return // Already at root
	}

	// Find parent in visible nodes
	for i, node := range jv.visibleNodes {
		if node == currentNode.Parent {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}
}

// ============================================================================
// Phase 2: JSON-specific Navigation Methods
// ============================================================================

// jumpToNextOfType jumps to the next node of specific type
func (jv *JSONBViewer) jumpToNextOfType(nodeType NodeType) {
	if jv.selectedIndex >= len(jv.visibleNodes)-1 {
		return
	}

	for i := jv.selectedIndex + 1; i < len(jv.visibleNodes); i++ {
		if jv.visibleNodes[i].Type == nodeType {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}
}

// jumpToPrevOfType jumps to the previous node of specific type
func (jv *JSONBViewer) jumpToPrevOfType(nodeType NodeType) {
	if jv.selectedIndex == 0 {
		return
	}

	for i := jv.selectedIndex - 1; i >= 0; i-- {
		if jv.visibleNodes[i].Type == nodeType {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}
}

// jumpToNextArrayItem jumps to next item in the same array
func (jv *JSONBViewer) jumpToNextArrayItem() {
	if jv.selectedIndex >= len(jv.visibleNodes) {
		return
	}

	currentNode := jv.visibleNodes[jv.selectedIndex]
	
	// Check if current node is in an array
	if currentNode.Parent == nil || currentNode.Parent.Type != NodeArray {
		return
	}

	// Find next sibling in the same array
	jv.jumpToNextSibling()
}

// jumpToPrevArrayItem jumps to previous item in the same array
func (jv *JSONBViewer) jumpToPrevArrayItem() {
	if jv.selectedIndex >= len(jv.visibleNodes) {
		return
	}

	currentNode := jv.visibleNodes[jv.selectedIndex]
	
	// Check if current node is in an array
	if currentNode.Parent == nil || currentNode.Parent.Type != NodeArray {
		return
	}

	// Find previous sibling in the same array
	jv.jumpToPrevSibling()
}

// copyCurrentPath copies the JSON path of current node to clipboard
func (jv *JSONBViewer) copyCurrentPath() {
	if jv.selectedIndex >= len(jv.visibleNodes) {
		jv.statusMessage = "⚠ No node selected"
		return
	}

	node := jv.visibleNodes[jv.selectedIndex]
	var pathStr string
	if len(node.Path) > 0 {
		pathStr = "$." + strings.Join(node.Path, ".")
	} else {
		pathStr = "$"
	}

	// Copy to clipboard
	err := clipboard.WriteAll(pathStr)
	if err != nil {
		jv.statusMessage = fmt.Sprintf("⚠ Failed to copy: %v", err)
		return
	}

	jv.statusMessage = fmt.Sprintf("✓ Copied: %s", pathStr)
}

// ============================================================================
// Phase 3: Advanced Navigation Methods
// ============================================================================

// handleQuickJump handles mark setting and jumping
func (jv *JSONBViewer) handleQuickJump(char string) {
	defer func() {
		jv.quickJumpMode = false
		jv.quickJumpBuffer = ""
	}()

	if len(char) != 1 {
		return
	}

	r := rune(char[0])
	
	// Check if it's a valid mark character (a-z)
	if r < 'a' || r > 'z' {
		return
	}

	if jv.quickJumpBuffer == "m" {
		// Set mark
		if jv.selectedIndex < len(jv.visibleNodes) {
			jv.marks[r] = jv.visibleNodes[jv.selectedIndex]
		}
	} else if jv.quickJumpBuffer == "'" {
		// Jump to mark
		if targetNode, ok := jv.marks[r]; ok {
			// Find the node in visible nodes
			for i, node := range jv.visibleNodes {
				if node == targetNode {
					jv.selectedIndex = i
					jv.adjustScroll()
					return
				}
			}
			// If not visible, expand path to it
			jv.expandPathToNode(targetNode)
			jv.rebuildVisibleNodes()
			for i, node := range jv.visibleNodes {
				if node == targetNode {
					jv.selectedIndex = i
					jv.adjustScroll()
					return
				}
			}
		}
	}
}

// jumpToKeyStartingWith jumps to next node whose key starts with the given letter
func (jv *JSONBViewer) jumpToKeyStartingWith(letter string) {
	if jv.selectedIndex >= len(jv.visibleNodes)-1 {
		// Try from beginning
		for i := 0; i < len(jv.visibleNodes); i++ {
			node := jv.visibleNodes[i]
			if len(node.Key) > 0 && strings.HasPrefix(strings.ToLower(node.Key), strings.ToLower(letter)) {
				jv.selectedIndex = i
				jv.adjustScroll()
				return
			}
		}
		return
	}

	// Search from current position forward
	for i := jv.selectedIndex + 1; i < len(jv.visibleNodes); i++ {
		node := jv.visibleNodes[i]
		if len(node.Key) > 0 && strings.HasPrefix(strings.ToLower(node.Key), strings.ToLower(letter)) {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}

	// Wrap around to beginning
	for i := 0; i <= jv.selectedIndex; i++ {
		node := jv.visibleNodes[i]
		if len(node.Key) > 0 && strings.HasPrefix(strings.ToLower(node.Key), strings.ToLower(letter)) {
			jv.selectedIndex = i
			jv.adjustScroll()
			return
		}
	}
}

// renderHelp renders the help documentation
func (jv *JSONBViewer) renderHelp() string {
	helpText := `
JSONB Viewer - Keyboard Shortcuts

Basic Navigation:
  ↑/k          Move up one line
  ↓/j          Move down one line
  g / Home     Jump to first item
  G / End      Jump to last item
  Ctrl-f/PgDn  Page down
  Ctrl-b/PgUp  Page up
  Ctrl-d       Half page down
  Ctrl-u       Half page up

Tree Navigation:
  Space/Enter  Expand/collapse current node
  E            Expand all nodes
  C            Collapse all nodes
  J            Jump to next sibling
  K            Jump to previous sibling
  p            Jump to parent node

JSON Type Navigation:
  ]            Jump to next Array
  [            Jump to previous Array
  }            Jump to next Object
  {            Jump to previous Object
  "            Jump to next String value
  #            Jump to next Number value
  ]a           Jump to next array item
  [a           Jump to previous array item

Search:
  /            Enter search mode
  n            Next search result
  N            Previous search result
  Esc          Clear search results

Advanced:
  y            Copy JSON path (yank)
  Y            Copy current node value
  m{a-z}       Set mark at current position
  '{a-z}       Jump to mark
  a-z          Quick jump to key starting with letter

Other:
  ?            Toggle this help
  q / Esc      Close viewer

Press any key to close help...
`

	return lipgloss.NewStyle().
		Foreground(jv.Theme.Foreground).
		Render(helpText)
}

// copyCurrentValue copies the current node's value to clipboard
func (jv *JSONBViewer) copyCurrentValue() {
	if jv.selectedIndex >= len(jv.visibleNodes) {
		jv.statusMessage = "⚠ No node selected"
		return
	}

	node := jv.visibleNodes[jv.selectedIndex]
	
	// Marshal the value to JSON
	var valueStr string
	if node.Type == NodeObject || node.Type == NodeArray {
		// For objects and arrays, marshal to pretty JSON
		jsonBytes, err := json.MarshalIndent(node.Value, "", "  ")
		if err != nil {
			jv.statusMessage = fmt.Sprintf("⚠ Failed to marshal: %v", err)
			return
		}
		valueStr = string(jsonBytes)
	} else {
		// For primitives, use simple string representation
		switch node.Type {
		case NodeString:
			valueStr = fmt.Sprintf("%v", node.Value)
		case NodeNumber, NodeBoolean:
			valueStr = fmt.Sprintf("%v", node.Value)
		case NodeNull:
			valueStr = "null"
		default:
			valueStr = fmt.Sprintf("%v", node.Value)
		}
	}

	// Copy to clipboard
	err := clipboard.WriteAll(valueStr)
	if err != nil {
		jv.statusMessage = fmt.Sprintf("⚠ Failed to copy: %v", err)
		return
	}

	// Show preview of copied content
	preview := valueStr
	if len(preview) > 50 {
		preview = preview[:47] + "..."
	}
	jv.statusMessage = fmt.Sprintf("✓ Copied value: %s", preview)
}

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
