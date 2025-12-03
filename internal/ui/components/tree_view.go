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
	zone "github.com/lrstanley/bubblezone"
	"github.com/rebelice/lazypg/internal/models"
	"github.com/rebelice/lazypg/internal/ui/theme"
)

// Zone ID prefixes for mouse click handling
const (
	ZoneTreeRowPrefix = "tree-row-"
)

// TreeView represents a visual tree component for displaying hierarchical data
type TreeView struct {
	Root         *models.TreeNode // Root node of the tree
	CursorIndex  int              // Current cursor position in the flattened list
	Width        int              // Display width
	Height       int              // Display height
	Theme        theme.Theme      // Color theme
	ScrollOffset int              // Vertical scroll offset for viewport

	// Search/filter state
	SearchMode   bool   // Whether search mode is active
	SearchQuery  string // Current search query
	FilterActive bool   // Whether filter is applied to tree
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
	// Height is already the content area height from the panel
	// Just use it directly, no need to subtract anything
	viewHeight := tv.Height
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

	// Render visible nodes with zone marks for mouse support
	for i := startIdx; i < endIdx; i++ {
		node := visibleNodes[i]
		line := tv.renderNode(node, i == tv.CursorIndex)
		// Wrap each row with zone mark for click detection
		// Use visible row index (i - startIdx) for zone ID
		zoneID := fmt.Sprintf("%s%d", ZoneTreeRowPrefix, i-startIdx)
		lines = append(lines, zone.Mark(zoneID, line))
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

// getNodeIcon returns the appropriate icon for a node with color
func (tv *TreeView) getNodeIcon(node *models.TreeNode) string {
	var icon string
	var iconColor lipgloss.Color

	switch node.Type {
	case models.TreeNodeTypeDatabase:
		// Check if database is active
		isActive := false
		if meta, ok := node.Metadata.(map[string]interface{}); ok {
			if active, ok := meta["active"].(bool); ok && active {
				isActive = true
			}
		}
		if isActive {
			icon = "●"
			iconColor = tv.Theme.DatabaseActive
		} else {
			icon = "○"
			iconColor = tv.Theme.DatabaseInactive
		}

	case models.TreeNodeTypeSchema:
		if node.Expanded {
			icon = "▾"
			iconColor = tv.Theme.SchemaExpanded
		} else {
			icon = "▸"
			iconColor = tv.Theme.SchemaCollapsed
		}

	case models.TreeNodeTypeTableGroup,
		models.TreeNodeTypeViewGroup,
		models.TreeNodeTypeMaterializedViewGroup,
		models.TreeNodeTypeFunctionGroup,
		models.TreeNodeTypeProcedureGroup,
		models.TreeNodeTypeTriggerFunctionGroup,
		models.TreeNodeTypeSequenceGroup,
		models.TreeNodeTypeTypeGroup,
		models.TreeNodeTypeExtensionGroup,
		models.TreeNodeTypeIndexGroup,
		models.TreeNodeTypeTriggerGroup,
		models.TreeNodeTypeCompositeTypeGroup,
		models.TreeNodeTypeEnumTypeGroup,
		models.TreeNodeTypeDomainTypeGroup,
		models.TreeNodeTypeRangeTypeGroup:
		if node.Expanded {
			icon = "▾"
		} else {
			icon = "▸"
		}
		// Color based on group type
		switch node.Type {
		case models.TreeNodeTypeTableGroup:
			iconColor = tv.Theme.TableIcon
		case models.TreeNodeTypeViewGroup:
			iconColor = tv.Theme.ViewIcon
		case models.TreeNodeTypeMaterializedViewGroup:
			iconColor = tv.Theme.MaterializedViewIcon
		case models.TreeNodeTypeFunctionGroup:
			iconColor = tv.Theme.FunctionIcon
		case models.TreeNodeTypeProcedureGroup:
			iconColor = tv.Theme.ProcedureIcon
		case models.TreeNodeTypeTriggerFunctionGroup:
			iconColor = tv.Theme.TriggerFunctionIcon
		case models.TreeNodeTypeSequenceGroup:
			iconColor = tv.Theme.SequenceIcon
		case models.TreeNodeTypeTypeGroup,
			models.TreeNodeTypeCompositeTypeGroup,
			models.TreeNodeTypeEnumTypeGroup,
			models.TreeNodeTypeDomainTypeGroup,
			models.TreeNodeTypeRangeTypeGroup:
			iconColor = tv.Theme.TypeIcon
		case models.TreeNodeTypeExtensionGroup:
			iconColor = tv.Theme.ExtensionIcon
		case models.TreeNodeTypeIndexGroup:
			iconColor = tv.Theme.IndexIcon
		case models.TreeNodeTypeTriggerGroup:
			iconColor = tv.Theme.TriggerIcon
		default:
			iconColor = tv.Theme.Foreground
		}

	case models.TreeNodeTypeTable:
		icon = "▦"
		iconColor = tv.Theme.TableIcon

	case models.TreeNodeTypeView:
		icon = "◎"
		iconColor = tv.Theme.ViewIcon

	case models.TreeNodeTypeMaterializedView:
		icon = "◉"
		iconColor = tv.Theme.MaterializedViewIcon

	case models.TreeNodeTypeFunction:
		icon = "ƒ"
		iconColor = tv.Theme.FunctionIcon

	case models.TreeNodeTypeProcedure:
		icon = "⚙"
		iconColor = tv.Theme.ProcedureIcon

	case models.TreeNodeTypeTriggerFunction:
		icon = "⚡"
		iconColor = tv.Theme.TriggerFunctionIcon

	case models.TreeNodeTypeSequence:
		icon = "#"
		iconColor = tv.Theme.SequenceIcon

	case models.TreeNodeTypeIndex:
		icon = "⊕"
		iconColor = tv.Theme.IndexIcon

	case models.TreeNodeTypeTrigger:
		icon = "↯"
		iconColor = tv.Theme.TriggerIcon

	case models.TreeNodeTypeExtension:
		icon = "◈"
		iconColor = tv.Theme.ExtensionIcon

	case models.TreeNodeTypeCompositeType:
		icon = "◫"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeEnumType:
		icon = "◧"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeDomainType:
		icon = "◨"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeRangeType:
		icon = "◩"
		iconColor = tv.Theme.TypeIcon

	case models.TreeNodeTypeColumn:
		icon = "•"
		iconColor = tv.Theme.ColumnIcon

	default:
		// Generic expandable/collapsible
		if node.Expanded {
			icon = "▾"
			iconColor = tv.Theme.Foreground
		} else {
			icon = "▸"
			iconColor = tv.Theme.Foreground
		}
	}

	// Apply color and return
	return lipgloss.NewStyle().Foreground(iconColor).Render(icon)
}

// buildNodeLabel builds the display label for a node, including metadata
func (tv *TreeView) buildNodeLabel(node *models.TreeNode) string {
	label := node.Label
	metaStyle := lipgloss.NewStyle().Foreground(tv.Theme.Metadata)

	// Add metadata based on node type
	switch node.Type {
	case models.TreeNodeTypeDatabase:
		// Active database already shown with icon color, no need for extra text
		// Just show the database name

	case models.TreeNodeTypeSchema:
		// Label already includes count info from loadTree, show empty marker if no children
		if node.Loaded && len(node.Children) == 0 {
			label += " " + metaStyle.Render("∅")
		}

	case models.TreeNodeTypeTableGroup, models.TreeNodeTypeViewGroup:
		// Label already includes count from loadTree

	case models.TreeNodeTypeTable:
		// Add row count if available with better formatting
		if meta, ok := node.Metadata.(map[string]interface{}); ok {
			if rowCount, ok := meta["row_count"].(int64); ok {
				label += " " + metaStyle.Render(formatNumber(rowCount))
			}
		}

	case models.TreeNodeTypeColumn:
		// Column label already includes type from BuildColumnNodes
		// Add indicators for constraints
		if meta, ok := node.Metadata.(models.ColumnInfo); ok {
			var indicators []string

			if meta.PrimaryKey {
				pkStyle := lipgloss.NewStyle().Foreground(tv.Theme.PrimaryKey)
				indicators = append(indicators, pkStyle.Render("⚿"))
			}

			// Note: ForeignKey and NotNull fields don't exist in ColumnInfo yet
			// They can be added in future enhancement

			if len(indicators) > 0 {
				label += " " + strings.Join(indicators, " ")
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
	lines := strings.Split(content, "\n")

	// Build scroll status indicator (e.g., "↑3 ↓5" meaning 3 above, 5 below)
	var indicators []string
	if startIdx > 0 {
		upIndicator := lipgloss.NewStyle().Foreground(tv.Theme.Info).Render(fmt.Sprintf("↑%d", startIdx))
		indicators = append(indicators, upIndicator)
	}
	if endIdx < total {
		remaining := total - endIdx
		downIndicator := lipgloss.NewStyle().Foreground(tv.Theme.Info).Render(fmt.Sprintf("↓%d", remaining))
		indicators = append(indicators, downIndicator)
	}

	// Append indicator to the last line if there's any scroll info
	if len(indicators) > 0 && len(lines) > 0 {
		lastIdx := len(lines) - 1
		indicatorText := strings.Join(indicators, " ")
		lines[lastIdx] = lines[lastIdx] + "  " + indicatorText
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

// ExpandAndNavigateToNode expands all ancestors of a node and moves cursor to it
// This is useful for programmatic navigation (e.g., from table jump dialog)
func (tv *TreeView) ExpandAndNavigateToNode(nodeID string) bool {
	if tv.Root == nil {
		return false
	}

	// Find the node by ID
	targetNode := tv.Root.FindByID(nodeID)
	if targetNode == nil {
		return false
	}

	// Expand all ancestors from root to parent
	current := targetNode.Parent
	for current != nil && current.Type != models.TreeNodeTypeRoot {
		current.Expanded = true
		current = current.Parent
	}

	// Now the node should be visible, set cursor to it
	visibleNodes := tv.Root.Flatten()
	for i, node := range visibleNodes {
		if node.ID == nodeID {
			tv.CursorIndex = i
			// Adjust scroll offset to make the node visible
			tv.adjustScrollOffset(len(visibleNodes), tv.Height)
			return true
		}
	}

	return false
}

// ScrollUp scrolls the tree view up by n lines (for mouse wheel)
func (tv *TreeView) ScrollUp(n int) {
	if tv.Root == nil {
		return
	}

	visibleNodes := tv.Root.Flatten()
	if len(visibleNodes) == 0 {
		return
	}

	// Scroll viewport up (like lazygit)
	tv.ScrollOffset -= n
	if tv.ScrollOffset < 0 {
		tv.ScrollOffset = 0
	}

	// Keep cursor within visible range
	if tv.CursorIndex >= tv.ScrollOffset+tv.Height {
		tv.CursorIndex = tv.ScrollOffset + tv.Height - 1
	}
	if tv.CursorIndex < tv.ScrollOffset {
		tv.CursorIndex = tv.ScrollOffset
	}
	// Bounds check
	if tv.CursorIndex >= len(visibleNodes) {
		tv.CursorIndex = len(visibleNodes) - 1
	}
	if tv.CursorIndex < 0 {
		tv.CursorIndex = 0
	}
}

// ScrollDown scrolls the tree view down by n lines (for mouse wheel)
func (tv *TreeView) ScrollDown(n int) {
	if tv.Root == nil {
		return
	}

	visibleNodes := tv.Root.Flatten()
	if len(visibleNodes) == 0 {
		return
	}

	// Scroll viewport down (like lazygit)
	maxScrollOffset := len(visibleNodes) - tv.Height
	if maxScrollOffset < 0 {
		maxScrollOffset = 0
	}
	tv.ScrollOffset += n
	if tv.ScrollOffset > maxScrollOffset {
		tv.ScrollOffset = maxScrollOffset
	}

	// Keep cursor within visible range
	if tv.CursorIndex < tv.ScrollOffset {
		tv.CursorIndex = tv.ScrollOffset
	}
	if tv.CursorIndex >= tv.ScrollOffset+tv.Height {
		tv.CursorIndex = tv.ScrollOffset + tv.Height - 1
	}
	// Bounds check
	if tv.CursorIndex >= len(visibleNodes) {
		tv.CursorIndex = len(visibleNodes) - 1
	}
	if tv.CursorIndex < 0 {
		tv.CursorIndex = 0
	}
}

// HandleClick handles mouse click at a specific row offset from the top of the visible area
// Lazygit-style: clicking already selected item triggers action (select for tables, toggle for expandable)
func (tv *TreeView) HandleClick(clickedRow int) (*TreeView, tea.Cmd) {
	if tv.Root == nil {
		return tv, nil
	}

	visibleNodes := tv.Root.Flatten()
	if len(visibleNodes) == 0 {
		return tv, nil
	}

	// Calculate which node was clicked
	targetIndex := tv.ScrollOffset + clickedRow
	if targetIndex < 0 || targetIndex >= len(visibleNodes) {
		return tv, nil
	}

	clickedNode := visibleNodes[targetIndex]
	wasAlreadySelected := tv.CursorIndex == targetIndex

	// Update cursor to clicked node
	tv.CursorIndex = targetIndex

	// If clicking already selected node, trigger action
	if wasAlreadySelected {
		// For expandable nodes, toggle expansion
		if len(clickedNode.Children) > 0 || !clickedNode.Loaded {
			clickedNode.Toggle()
			return tv, func() tea.Msg {
				return TreeNodeExpandedMsg{
					Node:     clickedNode,
					Expanded: clickedNode.Expanded,
				}
			}
		}
		// For selectable leaf nodes (tables), select/activate them
		if clickedNode.Selectable {
			return tv, func() tea.Msg {
				return TreeNodeSelectedMsg{Node: clickedNode}
			}
		}
	}

	// First click just selects the node (no action)
	return tv, nil
}
