package components

// TreeView component provides a visual representation of hierarchical tree data
// with keyboard navigation, expand/collapse functionality, and viewport scrolling.
//
// Features:
//   - Visual tree rendering with Unicode icons (▾ expanded, ▸ collapsed, • leaf)
//   - Keyboard navigation (↑↓/jk, →←/hl, g/G, space, enter)
//   - Automatic viewport scrolling for large trees
//   - Cursor highlighting with theme colors
//   - Active database highlighting
//   - Row count display for tables
//   - Primary key indicators for columns
//   - Empty state handling
//
// Usage:
//
//	root := models.BuildDatabaseTree(databases, activeDB)
//	treeView := components.NewTreeView(root, theme)
//	treeView.Width = 40
//	treeView.Height = 20
//
//	// In your Update method:
//	treeView, cmd := treeView.Update(msg)
//
//	// In your View method:
//	content := treeView.View()

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// TreeView represents a visual tree component for displaying hierarchical data
type TreeView struct {
	Root         *models.TreeNode // Root node of the tree
	CursorIndex  int              // Current cursor position in the flattened list
	Width        int              // Display width
	Height       int              // Display height
	Theme        theme.Theme      // Color theme
	ScrollOffset int              // Vertical scroll offset for viewport
}

// TreeNodeSelectedMsg is sent when a node is selected (Enter key)
type TreeNodeSelectedMsg struct {
	Node *models.TreeNode
}

// TreeNodeExpandedMsg is sent when a node is expanded/collapsed
type TreeNodeExpandedMsg struct {
	Node     *models.TreeNode
	Expanded bool // true if expanded, false if collapsed
}

// NewTreeView creates a new tree view component
func NewTreeView(root *models.TreeNode, theme theme.Theme) *TreeView {
	return &TreeView{
		Root:         root,
		CursorIndex:  0,
		Width:        40,
		Height:       20,
		Theme:        theme,
		ScrollOffset: 0,
	}
}

// View renders the tree as a string
func (tv *TreeView) View() string {
	if tv.Root == nil {
		return tv.emptyState()
	}

	// Get flattened list of visible nodes
	visibleNodes := tv.Root.Flatten()

	if len(visibleNodes) == 0 {
		return tv.emptyState()
	}

	// Ensure cursor is within bounds
	if tv.CursorIndex < 0 {
		tv.CursorIndex = 0
	}
	if tv.CursorIndex >= len(visibleNodes) {
		tv.CursorIndex = len(visibleNodes) - 1
	}

	// Calculate viewport dimensions
	// Subtract 2 for borders, 2 for title/help
	viewHeight := tv.Height - 4
	if viewHeight < 1 {
		viewHeight = 1
	}

	// Auto-scroll to keep cursor visible
	tv.adjustScrollOffset(len(visibleNodes), viewHeight)

	// Build the tree view
	var lines []string

	// Calculate visible range
	startIdx := tv.ScrollOffset
	endIdx := tv.ScrollOffset + viewHeight
	if endIdx > len(visibleNodes) {
		endIdx = len(visibleNodes)
	}

	// Render visible nodes
	for i := startIdx; i < endIdx; i++ {
		node := visibleNodes[i]
		line := tv.renderNode(node, i == tv.CursorIndex)
		lines = append(lines, line)
	}

	// Fill remaining space if needed
	for len(lines) < viewHeight {
		lines = append(lines, "")
	}

	// Join lines
	content := strings.Join(lines, "\n")

	// Add scroll indicators if needed
	if tv.ScrollOffset > 0 || endIdx < len(visibleNodes) {
		content = tv.addScrollIndicators(content, startIdx, endIdx, len(visibleNodes))
	}

	return content
}

// Update handles keyboard input for tree navigation
func (tv *TreeView) Update(msg tea.KeyMsg) (*TreeView, tea.Cmd) {
	if tv.Root == nil {
		return tv, nil
	}

	visibleNodes := tv.Root.Flatten()
	if len(visibleNodes) == 0 {
		return tv, nil
	}

	var cmd tea.Cmd

	switch msg.String() {
	case "up", "k":
		// Move cursor up
		if tv.CursorIndex > 0 {
			tv.CursorIndex--
		}

	case "down", "j":
		// Move cursor down
		if tv.CursorIndex < len(visibleNodes)-1 {
			tv.CursorIndex++
		}

	case "g":
		// Jump to top
		tv.CursorIndex = 0
		tv.ScrollOffset = 0

	case "G":
		// Jump to bottom
		tv.CursorIndex = len(visibleNodes) - 1

	case "right", "l", " ":
		// Expand node or move into expanded node
		currentNode := visibleNodes[tv.CursorIndex]
		if currentNode != nil {
			wasExpanded := currentNode.Expanded
			currentNode.Toggle()

			// Send expand/collapse message
			if currentNode.Expanded != wasExpanded {
				cmd = func() tea.Msg {
					return TreeNodeExpandedMsg{
						Node:     currentNode,
						Expanded: currentNode.Expanded,
					}
				}
			}
		}

	case "left", "h":
		// Collapse node or move to parent
		currentNode := visibleNodes[tv.CursorIndex]
		if currentNode != nil {
			if currentNode.Expanded {
				// Collapse if expanded
				currentNode.Toggle()
				cmd = func() tea.Msg {
					return TreeNodeExpandedMsg{
						Node:     currentNode,
						Expanded: false,
					}
				}
			} else if currentNode.Parent != nil && currentNode.Parent.Type != models.TreeNodeTypeRoot {
				// Move to parent if collapsed
				parentIndex := tv.findNodeIndex(visibleNodes, currentNode.Parent)
				if parentIndex >= 0 {
					tv.CursorIndex = parentIndex
				}
			}
		}

	case "enter":
		// Select node
		currentNode := visibleNodes[tv.CursorIndex]
		if currentNode != nil && currentNode.Selectable {
			cmd = func() tea.Msg {
				return TreeNodeSelectedMsg{Node: currentNode}
			}
		}
	}

	return tv, cmd
}

// renderNode renders a single tree node with appropriate styling
func (tv *TreeView) renderNode(node *models.TreeNode, selected bool) string {
	if node == nil {
		return ""
	}

	// Calculate indentation based on depth
	// Root is depth 0, but we don't render root, so subtract 1
	depth := node.GetDepth() - 1
	if depth < 0 {
		depth = 0
	}
	indent := strings.Repeat("  ", depth)

	// Choose icon based on node state
	icon := tv.getNodeIcon(node)

	// Build label with metadata
	label := tv.buildNodeLabel(node)

	// Combine parts
	content := fmt.Sprintf("%s%s %s", indent, icon, label)

	// Truncate if too long
	maxWidth := tv.Width - 2 // Account for padding
	if len(content) > maxWidth {
		content = content[:maxWidth-1] + "…"
	}

	// Apply styling
	var style lipgloss.Style
	if selected {
		style = lipgloss.NewStyle().
			Background(tv.Theme.Selection).
			Foreground(tv.Theme.Foreground).
			Bold(true).
			Width(maxWidth)
	} else {
		style = lipgloss.NewStyle().
			Foreground(tv.Theme.Foreground).
			Width(maxWidth)
	}

	return style.Render(content)
}

// getNodeIcon returns the appropriate icon for a node
func (tv *TreeView) getNodeIcon(node *models.TreeNode) string {
	if node.Type == models.TreeNodeTypeColumn {
		// Columns are leaf nodes
		return "•"
	}

	if node.Expanded {
		// Expanded node
		return "▾"
	}

	// Collapsed node (or not yet loaded)
	if len(node.Children) > 0 || !node.Loaded {
		return "▸"
	}

	// Empty non-leaf node
	return "▸"
}

// buildNodeLabel builds the display label for a node, including metadata
func (tv *TreeView) buildNodeLabel(node *models.TreeNode) string {
	label := node.Label

	// Add metadata based on node type
	switch node.Type {
	case models.TreeNodeTypeDatabase:
		// Check if this is the active database
		if meta, ok := node.Metadata.(map[string]interface{}); ok {
			if isActive, ok := meta["active"].(bool); ok && isActive {
				activeStyle := lipgloss.NewStyle().Foreground(tv.Theme.Success)
				label += " " + activeStyle.Render("(active)")
			}
		}

	case models.TreeNodeTypeSchema:
		// Show table count or "empty" for schemas
		if node.Loaded {
			childCount := len(node.Children)
			dimStyle := lipgloss.NewStyle().Foreground(tv.Theme.Comment)
			if childCount == 0 {
				label += " " + dimStyle.Render("(empty)")
			} else {
				label += " " + dimStyle.Render(fmt.Sprintf("(%d)", childCount))
			}
		}

	case models.TreeNodeTypeTable:
		// Add row count if available
		if meta, ok := node.Metadata.(map[string]interface{}); ok {
			if rowCount, ok := meta["row_count"].(int64); ok {
				label += fmt.Sprintf(" (%s rows)", formatNumber(rowCount))
			}
		}

	case models.TreeNodeTypeColumn:
		// Column label already includes type from BuildColumnNodes
		// Optionally add PK indicator
		if meta, ok := node.Metadata.(models.ColumnInfo); ok {
			if meta.PrimaryKey {
				pkStyle := lipgloss.NewStyle().Foreground(tv.Theme.Warning)
				label += " " + pkStyle.Render("PK")
			}
		}
	}

	return label
}

// adjustScrollOffset adjusts the scroll offset to keep the cursor visible
func (tv *TreeView) adjustScrollOffset(totalNodes, viewHeight int) {
	// Ensure cursor is visible in viewport
	if tv.CursorIndex < tv.ScrollOffset {
		tv.ScrollOffset = tv.CursorIndex
	}
	if tv.CursorIndex >= tv.ScrollOffset+viewHeight {
		tv.ScrollOffset = tv.CursorIndex - viewHeight + 1
	}

	// Ensure scroll offset is within bounds
	if tv.ScrollOffset < 0 {
		tv.ScrollOffset = 0
	}
	maxScroll := totalNodes - viewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if tv.ScrollOffset > maxScroll {
		tv.ScrollOffset = maxScroll
	}
}

// addScrollIndicators adds visual indicators for scrollable content
func (tv *TreeView) addScrollIndicators(content string, startIdx, endIdx, total int) string {
	// This is a simple implementation - could be enhanced with actual scroll bar
	lines := strings.Split(content, "\n")

	if startIdx > 0 && len(lines) > 0 {
		// Add up arrow indicator
		indicator := lipgloss.NewStyle().Foreground(tv.Theme.Info).Render("↑")
		lines[0] = indicator + " " + lines[0][2:]
	}

	if endIdx < total && len(lines) > 0 {
		// Add down arrow indicator
		lastIdx := len(lines) - 1
		indicator := lipgloss.NewStyle().Foreground(tv.Theme.Info).Render("↓")
		lines[lastIdx] = indicator + " " + lines[lastIdx][2:]
	}

	return strings.Join(lines, "\n")
}

// emptyState returns the empty state view
func (tv *TreeView) emptyState() string {
	style := lipgloss.NewStyle().
		Foreground(tv.Theme.Comment).
		Italic(true).
		Width(tv.Width - 2).
		Align(lipgloss.Center)

	return style.Render("No databases connected")
}

// findNodeIndex finds the index of a node in the flattened list
func (tv *TreeView) findNodeIndex(nodes []*models.TreeNode, target *models.TreeNode) int {
	for i, node := range nodes {
		if node == target {
			return i
		}
	}
	return -1
}

// formatNumber formats a number with commas for readability
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 10000 {
		// For 1k-10k, show one decimal place unless it's a round number
		k := float64(n) / 1000.0
		if k == float64(int(k)) {
			return fmt.Sprintf("%.0fk", k)
		}
		return fmt.Sprintf("%.1fk", k)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.0fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000.0)
}

// GetCurrentNode returns the currently selected node
func (tv *TreeView) GetCurrentNode() *models.TreeNode {
	if tv.Root == nil {
		return nil
	}

	visibleNodes := tv.Root.Flatten()
	if tv.CursorIndex < 0 || tv.CursorIndex >= len(visibleNodes) {
		return nil
	}

	return visibleNodes[tv.CursorIndex]
}

// SetCursorToNode sets the cursor to a specific node (by ID)
func (tv *TreeView) SetCursorToNode(nodeID string) bool {
	if tv.Root == nil {
		return false
	}

	visibleNodes := tv.Root.Flatten()
	for i, node := range visibleNodes {
		if node.ID == nodeID {
			tv.CursorIndex = i
			return true
		}
	}

	return false
}
