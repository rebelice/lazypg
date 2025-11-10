package models

import (
	"fmt"
	"strings"
)

// TreeNodeType represents the type of tree node
type TreeNodeType string

const (
	TreeNodeTypeRoot     TreeNodeType = "root"
	TreeNodeTypeDatabase TreeNodeType = "database"
	TreeNodeTypeSchema   TreeNodeType = "schema"
	TreeNodeTypeTable    TreeNodeType = "table"
	TreeNodeTypeColumn   TreeNodeType = "column"
)

// TreeNode represents a node in the navigation tree
type TreeNode struct {
	ID         string        // Unique identifier (e.g., "db:postgres", "schema:postgres.public")
	Type       TreeNodeType  // Type of node
	Label      string        // Display text
	Parent     *TreeNode     // Parent node (nil for root)
	Children   []*TreeNode   // Child nodes
	Expanded   bool          // Whether node is expanded
	Selectable bool          // Whether node can be selected
	Metadata   interface{}   // Type-specific metadata (table info, column types, etc.)
	Loaded     bool          // Whether children have been loaded (for lazy loading)
}

// NewTreeNode creates a new tree node
func NewTreeNode(id string, nodeType TreeNodeType, label string) *TreeNode {
	return &TreeNode{
		ID:         id,
		Type:       nodeType,
		Label:      label,
		Children:   make([]*TreeNode, 0),
		Expanded:   false,
		Selectable: nodeType != TreeNodeTypeRoot, // Root is not selectable
		Loaded:     false,
	}
}

// AddChild adds a child node to this node
func (n *TreeNode) AddChild(child *TreeNode) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// Toggle toggles the expanded state of the node
// A node can be toggled if it has children OR if it hasn't been loaded yet (lazy loading)
func (n *TreeNode) Toggle() {
	// Can toggle if:
	// 1. It has children, OR
	// 2. It hasn't been loaded yet (might have children when loaded)
	// Exception: Root and Column nodes - columns are leaf nodes with no children
	if n.Type == TreeNodeTypeColumn {
		return // Columns can't be expanded
	}

	// For other nodes, toggle if they have children or haven't been loaded
	if len(n.Children) > 0 || !n.Loaded {
		n.Expanded = !n.Expanded
	}
}

// Flatten returns a flat list of visible nodes for rendering
// This traverses the tree and returns only nodes that should be visible
// based on the expansion state of their parents
func (n *TreeNode) Flatten() []*TreeNode {
	return n.flattenHelper(0, true)
}

// flattenHelper is a recursive helper for Flatten
func (n *TreeNode) flattenHelper(depth int, visible bool) []*TreeNode {
	result := make([]*TreeNode, 0)

	// Skip root node in the flattened list (it's just a container)
	if n.Type != TreeNodeTypeRoot {
		if visible {
			result = append(result, n)
		}
	}

	// If this node is expanded (or it's the root), include its children
	if n.Expanded || n.Type == TreeNodeTypeRoot {
		for _, child := range n.Children {
			childVisible := visible && (n.Type == TreeNodeTypeRoot || n.Expanded)
			result = append(result, child.flattenHelper(depth+1, childVisible)...)
		}
	}

	return result
}

// FindByID finds a node by ID in the tree (depth-first search)
func (n *TreeNode) FindByID(id string) *TreeNode {
	if n.ID == id {
		return n
	}

	for _, child := range n.Children {
		if found := child.FindByID(id); found != nil {
			return found
		}
	}

	return nil
}

// GetPath returns the full path from root to this node
// For example: ["Databases", "postgres", "public", "users"]
func (n *TreeNode) GetPath() []string {
	path := make([]string, 0)
	current := n

	for current != nil {
		if current.Type != TreeNodeTypeRoot {
			path = append([]string{current.Label}, path...)
		}
		current = current.Parent
	}

	return path
}

// GetDepth returns the depth of this node in the tree (root = 0)
func (n *TreeNode) GetDepth() int {
	depth := 0
	current := n.Parent

	for current != nil {
		depth++
		current = current.Parent
	}

	return depth
}

// IsAncestorOf checks if this node is an ancestor of the given node
func (n *TreeNode) IsAncestorOf(other *TreeNode) bool {
	current := other.Parent

	for current != nil {
		if current == n {
			return true
		}
		current = current.Parent
	}

	return false
}

// BuildDatabaseTree builds a tree from database metadata
// This creates the root node and adds database nodes as children
func BuildDatabaseTree(databases []string, activeDB string) *TreeNode {
	root := NewTreeNode("root", TreeNodeTypeRoot, "Databases")
	root.Expanded = true // Root is always expanded
	root.Loaded = true   // Root's children (databases) are loaded immediately

	for _, dbName := range databases {
		dbNode := NewTreeNode(
			fmt.Sprintf("db:%s", dbName),
			TreeNodeTypeDatabase,
			dbName,
		)

		// Mark as selectable
		dbNode.Selectable = true

		// Store metadata to indicate if this is the active database
		dbNode.Metadata = map[string]interface{}{
			"active": dbName == activeDB,
		}

		root.AddChild(dbNode)
	}

	return root
}

// RefreshTreeChildren refreshes children of a specific node
// This replaces the node's children with the provided list
// Used for lazy loading when a node is expanded
func RefreshTreeChildren(node *TreeNode, children []*TreeNode) {
	// Clear existing children
	node.Children = make([]*TreeNode, 0, len(children))

	// Add new children
	for _, child := range children {
		node.AddChild(child)
	}

	// Mark as loaded
	node.Loaded = true
}

// BuildSchemaNodes creates schema nodes for a database
// This is a helper function for lazy loading schemas when a database is expanded
func BuildSchemaNodes(dbName string, schemas []string) []*TreeNode {
	nodes := make([]*TreeNode, 0, len(schemas))

	for _, schemaName := range schemas {
		node := NewTreeNode(
			fmt.Sprintf("schema:%s.%s", dbName, schemaName),
			TreeNodeTypeSchema,
			schemaName,
		)
		node.Selectable = true
		nodes = append(nodes, node)
	}

	return nodes
}

// BuildTableNodes creates table nodes for a schema
// This is a helper function for lazy loading tables when a schema is expanded
func BuildTableNodes(dbName, schemaName string, tables []string) []*TreeNode {
	nodes := make([]*TreeNode, 0, len(tables))

	for _, tableName := range tables {
		node := NewTreeNode(
			fmt.Sprintf("table:%s.%s.%s", dbName, schemaName, tableName),
			TreeNodeTypeTable,
			tableName,
		)
		node.Selectable = true
		nodes = append(nodes, node)
	}

	return nodes
}

// ColumnInfo holds metadata about a column
type ColumnInfo struct {
	Name       string
	DataType   string
	Nullable   bool
	PrimaryKey bool
	Default    *string
	IsArray    bool
	IsJsonb    bool
}

// BuildColumnNodes creates column nodes for a table
// This is a helper function for lazy loading columns when a table is expanded
func BuildColumnNodes(dbName, schemaName, tableName string, columns []ColumnInfo) []*TreeNode {
	nodes := make([]*TreeNode, 0, len(columns))

	for _, col := range columns {
		// Build a descriptive label for the column
		label := fmt.Sprintf("%s (%s)", col.Name, col.DataType)

		node := NewTreeNode(
			fmt.Sprintf("column:%s.%s.%s.%s", dbName, schemaName, tableName, col.Name),
			TreeNodeTypeColumn,
			label,
		)
		node.Selectable = false // Columns are typically not selectable
		node.Metadata = col
		nodes = append(nodes, node)
	}

	return nodes
}

// ParseNodeID parses a node ID and returns its components
// For example: "table:postgres.public.users" -> ("table", "postgres", "public", "users")
func ParseNodeID(id string) (nodeType string, components []string) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", nil
	}

	nodeType = parts[0]
	components = strings.Split(parts[1], ".")
	return nodeType, components
}

// GetDatabaseFromNode returns the database name for any node in the tree
func GetDatabaseFromNode(node *TreeNode) string {
	if node == nil {
		return ""
	}

	// Walk up the tree to find the database node
	current := node
	for current != nil {
		if current.Type == TreeNodeTypeDatabase {
			return current.Label
		}
		current = current.Parent
	}

	return ""
}

// GetSchemaFromNode returns the schema name for any node in a schema or below
func GetSchemaFromNode(node *TreeNode) string {
	if node == nil {
		return ""
	}

	// Walk up the tree to find the schema node
	current := node
	for current != nil {
		if current.Type == TreeNodeTypeSchema {
			return current.Label
		}
		current = current.Parent
	}

	return ""
}
