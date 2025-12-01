package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/rebelice/lazypg/internal/models"
	"github.com/rebelice/lazypg/internal/ui/theme"
)

func init() {
	// Initialize bubblezone for tests that call View() methods
	zone.NewGlobal()
}

func TestNewTreeView(t *testing.T) {
	root := models.NewTreeNode("root", models.TreeNodeTypeRoot, "Databases")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)

	if tv.Root != root {
		t.Error("Root not set correctly")
	}
	if tv.CursorIndex != 0 {
		t.Errorf("Expected initial cursor index 0, got %d", tv.CursorIndex)
	}
	if tv.ScrollOffset != 0 {
		t.Errorf("Expected initial scroll offset 0, got %d", tv.ScrollOffset)
	}
}

func TestTreeView_EmptyState(t *testing.T) {
	testTheme := theme.DefaultTheme()

	// Test with nil root
	tv := NewTreeView(nil, testTheme)
	tv.Width = 40
	tv.Height = 20

	view := tv.View()
	if !strings.Contains(view, "No databases connected") {
		t.Error("Expected empty state message for nil root")
	}

	// Test with empty root
	root := models.NewTreeNode("root", models.TreeNodeTypeRoot, "Databases")
	root.Expanded = true
	tv.Root = root

	view = tv.View()
	if !strings.Contains(view, "No databases connected") {
		t.Error("Expected empty state message for empty root")
	}
}

func TestTreeView_SingleNode(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"postgres"}, "postgres")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20

	view := tv.View()

	// Should contain the database name
	if !strings.Contains(view, "postgres") {
		t.Error("Expected view to contain 'postgres'")
	}

	// Active database is now shown with ● icon (filled circle) instead of "(active)" text
	if !strings.Contains(view, "●") {
		t.Error("Expected view to contain '●' icon for active database")
	}
}

func TestTreeView_NavigationUpDown(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"db1", "db2", "db3"}, "db1")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20

	// Initial cursor should be at 0
	if tv.CursorIndex != 0 {
		t.Errorf("Expected initial cursor at 0, got %d", tv.CursorIndex)
	}

	// Move down
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyDown})
	if tv.CursorIndex != 1 {
		t.Errorf("Expected cursor at 1 after down, got %d", tv.CursorIndex)
	}

	// Move down again
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyDown})
	if tv.CursorIndex != 2 {
		t.Errorf("Expected cursor at 2 after down, got %d", tv.CursorIndex)
	}

	// Move down at bottom (should stay at 2)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyDown})
	if tv.CursorIndex != 2 {
		t.Errorf("Expected cursor to stay at 2 at bottom, got %d", tv.CursorIndex)
	}

	// Move up
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyUp})
	if tv.CursorIndex != 1 {
		t.Errorf("Expected cursor at 1 after up, got %d", tv.CursorIndex)
	}

	// Move up again
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyUp})
	if tv.CursorIndex != 0 {
		t.Errorf("Expected cursor at 0 after up, got %d", tv.CursorIndex)
	}

	// Move up at top (should stay at 0)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyUp})
	if tv.CursorIndex != 0 {
		t.Errorf("Expected cursor to stay at 0 at top, got %d", tv.CursorIndex)
	}
}

func TestTreeView_NavigationJump(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"db1", "db2", "db3", "db4", "db5"}, "db1")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20
	tv.CursorIndex = 2 // Start in middle

	// Jump to top with 'g'
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if tv.CursorIndex != 0 {
		t.Errorf("Expected cursor at 0 after 'g', got %d", tv.CursorIndex)
	}
	if tv.ScrollOffset != 0 {
		t.Errorf("Expected scroll offset 0 after 'g', got %d", tv.ScrollOffset)
	}

	// Jump to bottom with 'G'
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if tv.CursorIndex != 4 {
		t.Errorf("Expected cursor at 4 after 'G', got %d", tv.CursorIndex)
	}
}

func TestTreeView_ExpandCollapse(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"postgres"}, "postgres")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20

	// Get the database node
	dbNode := root.FindByID("db:postgres")
	if dbNode == nil {
		t.Fatal("Could not find postgres node")
	}

	// Initially collapsed
	if dbNode.Expanded {
		t.Error("Expected node to be initially collapsed")
	}

	// Expand with space
	tv, cmd := tv.Update(tea.KeyMsg{Type: tea.KeySpace})

	if !dbNode.Expanded {
		t.Error("Expected node to be expanded after space")
	}

	// Should return expand message
	if cmd == nil {
		t.Error("Expected expand command")
	} else {
		msg := cmd()
		if expandMsg, ok := msg.(TreeNodeExpandedMsg); ok {
			if !expandMsg.Expanded {
				t.Error("Expected Expanded to be true in message")
			}
			if expandMsg.Node != dbNode {
				t.Error("Expected message to contain the correct node")
			}
		} else {
			t.Error("Expected TreeNodeExpandedMsg")
		}
	}

	// Collapse with space again
	tv, cmd = tv.Update(tea.KeyMsg{Type: tea.KeySpace})
	_ = tv // silence unused warning

	if dbNode.Expanded {
		t.Error("Expected node to be collapsed after second space")
	}

	// Should return collapse message
	if cmd == nil {
		t.Error("Expected collapse command")
	} else {
		msg := cmd()
		if expandMsg, ok := msg.(TreeNodeExpandedMsg); ok {
			if expandMsg.Expanded {
				t.Error("Expected Expanded to be false in message")
			}
		}
	}
}

func TestTreeView_ExpandAndNavigateToParent(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"postgres"}, "postgres")
	testTheme := theme.DefaultTheme()

	// Add schemas to postgres
	postgres := root.FindByID("db:postgres")
	schemas := models.BuildSchemaNodes("postgres", []string{"public", "information_schema"})
	models.RefreshTreeChildren(postgres, schemas)
	postgres.Expanded = true

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20

	// Cursor should be on postgres (index 0)
	if tv.CursorIndex != 0 {
		t.Errorf("Expected initial cursor at 0, got %d", tv.CursorIndex)
	}

	// Move down to public schema
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyDown})

	currentNode := tv.GetCurrentNode()
	if currentNode.Type != models.TreeNodeTypeSchema {
		t.Error("Expected cursor on schema node")
	}

	// Press left to navigate to parent (postgres)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyLeft})

	currentNode = tv.GetCurrentNode()
	if currentNode.Type != models.TreeNodeTypeDatabase {
		t.Error("Expected cursor to move to database node (parent)")
	}
}

func TestTreeView_SelectNode(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"postgres"}, "postgres")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20

	// Press enter to select
	tv, cmd := tv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = tv // silence unused warning

	if cmd == nil {
		t.Error("Expected select command")
	} else {
		msg := cmd()
		if selectMsg, ok := msg.(TreeNodeSelectedMsg); ok {
			if selectMsg.Node == nil {
				t.Error("Expected node in select message")
			}
			if selectMsg.Node.Type != models.TreeNodeTypeDatabase {
				t.Error("Expected database node to be selected")
			}
		} else {
			t.Error("Expected TreeNodeSelectedMsg")
		}
	}
}

func TestTreeView_GetNodeIcon(t *testing.T) {
	testTheme := theme.DefaultTheme()
	tv := NewTreeView(nil, testTheme)

	// Test database icons (now use ● for active, ○ for inactive)
	t.Run("Inactive database", func(t *testing.T) {
		node := models.NewTreeNode("test", models.TreeNodeTypeDatabase, "Test")
		icon := tv.getNodeIcon(node)
		if !strings.Contains(icon, "○") {
			t.Errorf("Expected icon to contain '○' for inactive database, got '%s'", icon)
		}
	})

	t.Run("Active database", func(t *testing.T) {
		node := models.NewTreeNode("test", models.TreeNodeTypeDatabase, "Test")
		node.Metadata = map[string]interface{}{"active": true}
		icon := tv.getNodeIcon(node)
		if !strings.Contains(icon, "●") {
			t.Errorf("Expected icon to contain '●' for active database, got '%s'", icon)
		}
	})

	// Test schema icons (use ▸/▾)
	t.Run("Collapsed schema", func(t *testing.T) {
		node := models.NewTreeNode("test", models.TreeNodeTypeSchema, "Test")
		node.Loaded = true
		child := models.NewTreeNode("child", models.TreeNodeTypeColumn, "Child")
		node.AddChild(child)
		icon := tv.getNodeIcon(node)
		if !strings.Contains(icon, "▸") {
			t.Errorf("Expected icon to contain '▸' for collapsed schema, got '%s'", icon)
		}
	})

	t.Run("Expanded schema", func(t *testing.T) {
		node := models.NewTreeNode("test", models.TreeNodeTypeSchema, "Test")
		node.Expanded = true
		node.Loaded = true
		child := models.NewTreeNode("child", models.TreeNodeTypeColumn, "Child")
		node.AddChild(child)
		icon := tv.getNodeIcon(node)
		if !strings.Contains(icon, "▾") {
			t.Errorf("Expected icon to contain '▾' for expanded schema, got '%s'", icon)
		}
	})

	// Test column icon (uses •)
	t.Run("Column node", func(t *testing.T) {
		node := models.NewTreeNode("test", models.TreeNodeTypeColumn, "Test")
		node.Loaded = true
		icon := tv.getNodeIcon(node)
		if !strings.Contains(icon, "•") {
			t.Errorf("Expected icon to contain '•' for column, got '%s'", icon)
		}
	})
}

func TestTreeView_GetCurrentNode(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"db1", "db2"}, "db1")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)

	// Test at index 0
	node := tv.GetCurrentNode()
	if node == nil {
		t.Fatal("Expected node at index 0")
	}
	if node.Label != "db1" {
		t.Errorf("Expected 'db1', got '%s'", node.Label)
	}

	// Test at index 1
	tv.CursorIndex = 1
	node = tv.GetCurrentNode()
	if node == nil {
		t.Fatal("Expected node at index 1")
	}
	if node.Label != "db2" {
		t.Errorf("Expected 'db2', got '%s'", node.Label)
	}

	// Test out of bounds
	tv.CursorIndex = 999
	node = tv.GetCurrentNode()
	if node != nil {
		t.Error("Expected nil for out of bounds index")
	}
}

func TestTreeView_SetCursorToNode(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"db1", "db2", "db3"}, "db1")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)

	// Find db2
	found := tv.SetCursorToNode("db:db2")
	if !found {
		t.Error("Expected to find db2")
	}
	if tv.CursorIndex != 1 {
		t.Errorf("Expected cursor at 1, got %d", tv.CursorIndex)
	}

	// Try to find non-existent node
	found = tv.SetCursorToNode("db:nonexistent")
	if found {
		t.Error("Expected not to find nonexistent node")
	}
}

func TestTreeView_ViewportScrolling(t *testing.T) {
	// Create a tree with many nodes
	databases := make([]string, 20)
	for i := 0; i < 20; i++ {
		databases[i] = "db" + string(rune('A'+i))
	}
	root := models.BuildDatabaseTree(databases, "dbA")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 10 // Small height to trigger scrolling

	// Move cursor to bottom
	tv.CursorIndex = 19

	// Render to trigger scroll adjustment
	_ = tv.View()

	// Scroll offset should be adjusted to keep cursor visible
	// The cursor should be within the visible range
	if tv.CursorIndex < tv.ScrollOffset || tv.CursorIndex >= tv.ScrollOffset+tv.Height {
		t.Errorf("Cursor %d should be visible with scroll offset %d and height %d",
			tv.CursorIndex, tv.ScrollOffset, tv.Height)
	}

	// Move cursor to top
	tv.CursorIndex = 0
	_ = tv.View()

	// Scroll offset should be 0
	if tv.ScrollOffset != 0 {
		t.Errorf("Expected scroll offset 0 when cursor at top, got %d", tv.ScrollOffset)
	}
}

func TestTreeView_FormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1k"},      // Round numbers show no decimal
		{1500, "1.5k"},    // Non-round numbers show decimal
		{9999, "10.0k"},   // Just under 10k
		{10000, "10k"},    // 10k and above lose decimals
		{99999, "100k"},
		{999999, "1000k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
	}

	for _, tt := range tests {
		result := formatNumber(tt.input)
		if result != tt.expected {
			t.Errorf("formatNumber(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestTreeView_ActiveDatabaseHighlight(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"postgres", "mydb"}, "postgres")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)
	tv.Width = 40
	tv.Height = 20

	view := tv.View()

	// Active database is shown with ● icon (filled circle)
	if !strings.Contains(view, "●") {
		t.Error("Expected ● icon for active database in view")
	}

	// Inactive database is shown with ○ icon (empty circle)
	if !strings.Contains(view, "○") {
		t.Error("Expected ○ icon for inactive database in view")
	}

	// Verify active database has correct metadata
	postgres := root.FindByID("db:postgres")
	if meta, ok := postgres.Metadata.(map[string]interface{}); ok {
		if active, ok := meta["active"].(bool); !ok || !active {
			t.Error("Expected postgres to have active=true metadata")
		}
	} else {
		t.Error("Expected postgres to have metadata map")
	}

	// Verify inactive database has correct metadata
	mydb := root.FindByID("db:mydb")
	if meta, ok := mydb.Metadata.(map[string]interface{}); ok {
		if active, ok := meta["active"].(bool); ok && active {
			t.Error("Expected mydb to have active=false metadata")
		}
	}
}

func TestTreeView_ViKeybindings(t *testing.T) {
	root := models.BuildDatabaseTree([]string{"db1", "db2", "db3"}, "db1")
	testTheme := theme.DefaultTheme()

	tv := NewTreeView(root, testTheme)

	// Test j (down)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if tv.CursorIndex != 1 {
		t.Errorf("Expected cursor at 1 after 'j', got %d", tv.CursorIndex)
	}

	// Test k (up)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if tv.CursorIndex != 0 {
		t.Errorf("Expected cursor at 0 after 'k', got %d", tv.CursorIndex)
	}

	// Expand node first
	dbNode := root.FindByID("db:db1")
	schemas := models.BuildSchemaNodes("db1", []string{"public"})
	models.RefreshTreeChildren(dbNode, schemas)

	// Test l (right/expand)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !dbNode.Expanded {
		t.Error("Expected node to be expanded after 'l'")
	}

	// Test h (left/collapse)
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	_ = tv // silence unused warning
	if dbNode.Expanded {
		t.Error("Expected node to be collapsed after 'h'")
	}
}
