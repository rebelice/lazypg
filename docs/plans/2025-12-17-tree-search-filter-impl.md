# TreeView Search/Filter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `/` key triggered fuzzy search filtering to TreeView with type prefixes and negation support.

**Architecture:** New `tree_filter.go` file contains search parsing and fuzzy matching. TreeView gains search state machine (Off → Inputting → FilterActive). Filtered view is a separate flattened list, original tree unchanged.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, no external fuzzy library (custom subsequence matcher)

---

## Task 1: Search Query Parser

**Files:**
- Create: `internal/ui/components/tree_filter.go`
- Test: `internal/ui/components/tree_filter_test.go`

**Step 1: Write the failing test for ParseSearchQuery**

```go
// internal/ui/components/tree_filter_test.go
package components

import (
	"testing"
)

func TestParseSearchQuery_Simple(t *testing.T) {
	q := ParseSearchQuery("plan")

	if q.Pattern != "plan" {
		t.Errorf("expected pattern 'plan', got '%s'", q.Pattern)
	}
	if q.Negate {
		t.Error("expected Negate=false")
	}
	if q.TypeFilter != "" {
		t.Errorf("expected empty TypeFilter, got '%s'", q.TypeFilter)
	}
}

func TestParseSearchQuery_Negate(t *testing.T) {
	q := ParseSearchQuery("!test")

	if q.Pattern != "test" {
		t.Errorf("expected pattern 'test', got '%s'", q.Pattern)
	}
	if !q.Negate {
		t.Error("expected Negate=true")
	}
}

func TestParseSearchQuery_TypeShort(t *testing.T) {
	q := ParseSearchQuery("t:plan")

	if q.Pattern != "plan" {
		t.Errorf("expected pattern 'plan', got '%s'", q.Pattern)
	}
	if q.TypeFilter != "table" {
		t.Errorf("expected TypeFilter 'table', got '%s'", q.TypeFilter)
	}
}

func TestParseSearchQuery_TypeLong(t *testing.T) {
	q := ParseSearchQuery("table:plan")

	if q.Pattern != "plan" {
		t.Errorf("expected pattern 'plan', got '%s'", q.Pattern)
	}
	if q.TypeFilter != "table" {
		t.Errorf("expected TypeFilter 'table', got '%s'", q.TypeFilter)
	}
}

func TestParseSearchQuery_NegateWithType(t *testing.T) {
	q := ParseSearchQuery("!f:get")

	if q.Pattern != "get" {
		t.Errorf("expected pattern 'get', got '%s'", q.Pattern)
	}
	if !q.Negate {
		t.Error("expected Negate=true")
	}
	if q.TypeFilter != "function" {
		t.Errorf("expected TypeFilter 'function', got '%s'", q.TypeFilter)
	}
}

func TestParseSearchQuery_TypeOnlyNoPattern(t *testing.T) {
	q := ParseSearchQuery("!t:")

	if q.Pattern != "" {
		t.Errorf("expected empty pattern, got '%s'", q.Pattern)
	}
	if !q.Negate {
		t.Error("expected Negate=true")
	}
	if q.TypeFilter != "table" {
		t.Errorf("expected TypeFilter 'table', got '%s'", q.TypeFilter)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/components/... -run TestParseSearchQuery -v
```

Expected: FAIL with "undefined: ParseSearchQuery"

**Step 3: Write minimal implementation**

```go
// internal/ui/components/tree_filter.go
package components

import (
	"strings"
)

// SearchQuery represents a parsed search query
type SearchQuery struct {
	Pattern    string // The search pattern (after removing prefix/type)
	Negate     bool   // True if query starts with !
	TypeFilter string // Normalized type filter (e.g., "table", "function")
}

// Type prefix mappings
var typePrefixes = map[string]string{
	// Short prefixes
	"t:":   "table",
	"v:":   "view",
	"f:":   "function",
	"s:":   "schema",
	"seq:": "sequence",
	"ext:": "extension",
	"col:": "column",
	"idx:": "index",
	// Long prefixes
	"table:":     "table",
	"view:":      "view",
	"func:":      "function",
	"function:":  "function",
	"schema:":    "schema",
	"sequence:":  "sequence",
	"extension:": "extension",
	"column:":    "column",
	"index:":     "index",
}

// ParseSearchQuery parses a search query string into structured form
// Examples:
//   - "plan" → {Pattern: "plan", Negate: false, TypeFilter: ""}
//   - "!test" → {Pattern: "test", Negate: true, TypeFilter: ""}
//   - "t:plan" → {Pattern: "plan", Negate: false, TypeFilter: "table"}
//   - "!f:get" → {Pattern: "get", Negate: true, TypeFilter: "function"}
func ParseSearchQuery(query string) SearchQuery {
	q := SearchQuery{}

	// Check for negation prefix
	if strings.HasPrefix(query, "!") {
		q.Negate = true
		query = query[1:]
	}

	// Check for type prefix
	queryLower := strings.ToLower(query)
	for prefix, typeName := range typePrefixes {
		if strings.HasPrefix(queryLower, prefix) {
			q.TypeFilter = typeName
			query = query[len(prefix):]
			break
		}
	}

	q.Pattern = query
	return q
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ui/components/... -run TestParseSearchQuery -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/components/tree_filter.go internal/ui/components/tree_filter_test.go
git commit -m "feat(tree): add search query parser with type prefixes"
```

---

## Task 2: Fuzzy Match Algorithm

**Files:**
- Modify: `internal/ui/components/tree_filter.go`
- Modify: `internal/ui/components/tree_filter_test.go`

**Step 1: Write the failing test for FuzzyMatch**

```go
// Add to internal/ui/components/tree_filter_test.go

func TestFuzzyMatch_ExactPrefix(t *testing.T) {
	match, positions := FuzzyMatch("plan", "plan_check_run")

	if !match {
		t.Error("expected match")
	}
	if len(positions) != 4 || positions[0] != 0 || positions[1] != 1 || positions[2] != 2 || positions[3] != 3 {
		t.Errorf("expected positions [0,1,2,3], got %v", positions)
	}
}

func TestFuzzyMatch_Subsequence(t *testing.T) {
	match, positions := FuzzyMatch("pcr", "plan_check_run")

	if !match {
		t.Error("expected match")
	}
	// p=0, c=5, r=11
	if len(positions) != 3 {
		t.Errorf("expected 3 positions, got %d", len(positions))
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	match, _ := FuzzyMatch("xyz", "plan_check_run")

	if match {
		t.Error("expected no match")
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	match, _ := FuzzyMatch("PLAN", "plan_check_run")

	if !match {
		t.Error("expected case-insensitive match")
	}
}

func TestFuzzyMatch_EmptyPattern(t *testing.T) {
	match, positions := FuzzyMatch("", "anything")

	if !match {
		t.Error("empty pattern should match everything")
	}
	if len(positions) != 0 {
		t.Error("empty pattern should have no positions")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/components/... -run TestFuzzyMatch -v
```

Expected: FAIL with "undefined: FuzzyMatch"

**Step 3: Write minimal implementation**

```go
// Add to internal/ui/components/tree_filter.go

// FuzzyMatch performs fuzzy subsequence matching
// Returns whether the pattern matches and the positions of matched characters
// Matching is case-insensitive
func FuzzyMatch(pattern, target string) (bool, []int) {
	if pattern == "" {
		return true, []int{}
	}

	patternLower := strings.ToLower(pattern)
	targetLower := strings.ToLower(target)

	positions := make([]int, 0, len(pattern))
	patternIdx := 0

	for i := 0; i < len(targetLower) && patternIdx < len(patternLower); i++ {
		if targetLower[i] == patternLower[patternIdx] {
			positions = append(positions, i)
			patternIdx++
		}
	}

	if patternIdx == len(patternLower) {
		return true, positions
	}
	return false, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ui/components/... -run TestFuzzyMatch -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/components/tree_filter.go internal/ui/components/tree_filter_test.go
git commit -m "feat(tree): add fuzzy subsequence matching algorithm"
```

---

## Task 3: Node Type Matching

**Files:**
- Modify: `internal/ui/components/tree_filter.go`
- Modify: `internal/ui/components/tree_filter_test.go`

**Step 1: Write the failing test for NodeMatchesType**

```go
// Add to internal/ui/components/tree_filter_test.go

import (
	"github.com/rebelice/lazypg/internal/models"
)

func TestNodeMatchesType_Table(t *testing.T) {
	node := &models.TreeNode{Type: models.TreeNodeTypeTable}

	if !NodeMatchesType(node, "table") {
		t.Error("table node should match 'table' type filter")
	}
	if NodeMatchesType(node, "view") {
		t.Error("table node should not match 'view' type filter")
	}
}

func TestNodeMatchesType_EmptyFilter(t *testing.T) {
	node := &models.TreeNode{Type: models.TreeNodeTypeTable}

	if !NodeMatchesType(node, "") {
		t.Error("empty filter should match any node")
	}
}

func TestNodeMatchesType_Function(t *testing.T) {
	node := &models.TreeNode{Type: models.TreeNodeTypeFunction}

	if !NodeMatchesType(node, "function") {
		t.Error("function node should match 'function' type filter")
	}
}

func TestNodeMatchesType_Schema(t *testing.T) {
	node := &models.TreeNode{Type: models.TreeNodeTypeSchema}

	if !NodeMatchesType(node, "schema") {
		t.Error("schema node should match 'schema' type filter")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/components/... -run TestNodeMatchesType -v
```

Expected: FAIL with "undefined: NodeMatchesType"

**Step 3: Write minimal implementation**

```go
// Add to internal/ui/components/tree_filter.go

import (
	"github.com/rebelice/lazypg/internal/models"
)

// nodeTypeMapping maps type filter strings to TreeNodeTypes
var nodeTypeMapping = map[string][]models.TreeNodeType{
	"table":     {models.TreeNodeTypeTable},
	"view":      {models.TreeNodeTypeView, models.TreeNodeTypeMaterializedView},
	"function":  {models.TreeNodeTypeFunction, models.TreeNodeTypeTriggerFunction},
	"schema":    {models.TreeNodeTypeSchema},
	"sequence":  {models.TreeNodeTypeSequence},
	"extension": {models.TreeNodeTypeExtension},
	"column":    {models.TreeNodeTypeColumn},
	"index":     {models.TreeNodeTypeIndex},
}

// NodeMatchesType checks if a node matches the given type filter
// Empty filter matches all nodes
func NodeMatchesType(node *models.TreeNode, typeFilter string) bool {
	if typeFilter == "" {
		return true
	}

	nodeTypes, ok := nodeTypeMapping[typeFilter]
	if !ok {
		return false
	}

	for _, nt := range nodeTypes {
		if node.Type == nt {
			return true
		}
	}
	return false
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ui/components/... -run TestNodeMatchesType -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/components/tree_filter.go internal/ui/components/tree_filter_test.go
git commit -m "feat(tree): add node type matching for search filters"
```

---

## Task 4: Tree Filtering Logic

**Files:**
- Modify: `internal/ui/components/tree_filter.go`
- Modify: `internal/ui/components/tree_filter_test.go`

**Step 1: Write the failing test for FilterTree**

```go
// Add to internal/ui/components/tree_filter_test.go

func createTestTree() *models.TreeNode {
	root := models.NewTreeNode("root", models.TreeNodeTypeRoot, "Root")
	root.Expanded = true

	db := models.NewTreeNode("db:test", models.TreeNodeTypeDatabase, "test")
	db.Expanded = true
	root.AddChild(db)

	schema := models.NewTreeNode("schema:test.public", models.TreeNodeTypeSchema, "public")
	schema.Expanded = true
	db.AddChild(schema)

	tables := models.NewTreeNode("tables", models.TreeNodeTypeTableGroup, "Tables")
	tables.Expanded = true
	schema.AddChild(tables)

	// Add test tables
	plan := models.NewTreeNode("table:test.public.plan", models.TreeNodeTypeTable, "plan")
	planCheck := models.NewTreeNode("table:test.public.plan_check_run", models.TreeNodeTypeTable, "plan_check_run")
	users := models.NewTreeNode("table:test.public.users", models.TreeNodeTypeTable, "users")
	tables.AddChild(plan)
	tables.AddChild(planCheck)
	tables.AddChild(users)

	funcs := models.NewTreeNode("funcs", models.TreeNodeTypeFunctionGroup, "Functions")
	funcs.Expanded = true
	schema.AddChild(funcs)

	getUser := models.NewTreeNode("func:test.public.get_user", models.TreeNodeTypeFunction, "get_user")
	funcs.AddChild(getUser)

	return root
}

func TestFilterTree_SimpleMatch(t *testing.T) {
	root := createTestTree()
	query := ParseSearchQuery("plan")

	matches := FilterTree(root, query)

	if len(matches) != 2 {
		t.Errorf("expected 2 matches (plan, plan_check_run), got %d", len(matches))
	}
}

func TestFilterTree_TypeFilter(t *testing.T) {
	root := createTestTree()
	query := ParseSearchQuery("t:plan")

	matches := FilterTree(root, query)

	// Should only match tables containing "plan"
	if len(matches) != 2 {
		t.Errorf("expected 2 table matches, got %d", len(matches))
	}
	for _, m := range matches {
		if m.Type != models.TreeNodeTypeTable {
			t.Errorf("expected only table nodes, got %s", m.Type)
		}
	}
}

func TestFilterTree_Negate(t *testing.T) {
	root := createTestTree()
	query := ParseSearchQuery("!plan")

	matches := FilterTree(root, query)

	// Should match users and get_user (not plan or plan_check_run)
	for _, m := range matches {
		if strings.Contains(strings.ToLower(m.Label), "plan") {
			t.Errorf("negated query should not match '%s'", m.Label)
		}
	}
}

func TestFilterTree_EmptyQuery(t *testing.T) {
	root := createTestTree()
	query := ParseSearchQuery("")

	matches := FilterTree(root, query)

	// Empty query should return all leaf nodes
	if len(matches) == 0 {
		t.Error("empty query should return all searchable nodes")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/components/... -run TestFilterTree -v
```

Expected: FAIL with "undefined: FilterTree"

**Step 3: Write minimal implementation**

```go
// Add to internal/ui/components/tree_filter.go

// isSearchableNode returns true if this node type should be included in search results
func isSearchableNode(node *models.TreeNode) bool {
	switch node.Type {
	case models.TreeNodeTypeTable,
		models.TreeNodeTypeView,
		models.TreeNodeTypeMaterializedView,
		models.TreeNodeTypeFunction,
		models.TreeNodeTypeProcedure,
		models.TreeNodeTypeTriggerFunction,
		models.TreeNodeTypeSequence,
		models.TreeNodeTypeIndex,
		models.TreeNodeTypeTrigger,
		models.TreeNodeTypeExtension,
		models.TreeNodeTypeCompositeType,
		models.TreeNodeTypeEnumType,
		models.TreeNodeTypeDomainType,
		models.TreeNodeTypeRangeType,
		models.TreeNodeTypeSchema,
		models.TreeNodeTypeColumn:
		return true
	default:
		return false
	}
}

// FilterTree filters the tree based on search query
// Returns a flat list of matching nodes
func FilterTree(root *models.TreeNode, query SearchQuery) []*models.TreeNode {
	var matches []*models.TreeNode

	var traverse func(node *models.TreeNode)
	traverse = func(node *models.TreeNode) {
		if node == nil {
			return
		}

		// Check if this node should be considered for matching
		if isSearchableNode(node) {
			// Check type filter first
			if !NodeMatchesType(node, query.TypeFilter) {
				if !query.Negate {
					// Type doesn't match, skip (unless negating)
					goto children
				}
			}

			// Check pattern match
			patternMatches := true
			if query.Pattern != "" {
				patternMatches, _ = FuzzyMatch(query.Pattern, node.Label)
			}

			// Apply negation logic
			shouldInclude := false
			if query.Negate {
				// Include if it does NOT match (type or pattern)
				typeMatches := NodeMatchesType(node, query.TypeFilter)
				if query.TypeFilter != "" && !typeMatches {
					// Type doesn't match the filter, include it
					shouldInclude = true
				} else if !patternMatches {
					// Pattern doesn't match, include it
					shouldInclude = true
				}
			} else {
				// Normal match: include if type and pattern both match
				shouldInclude = NodeMatchesType(node, query.TypeFilter) && patternMatches
			}

			if shouldInclude {
				matches = append(matches, node)
			}
		}

	children:
		// Always traverse children
		for _, child := range node.Children {
			traverse(child)
		}
	}

	traverse(root)
	return matches
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ui/components/... -run TestFilterTree -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/components/tree_filter.go internal/ui/components/tree_filter_test.go
git commit -m "feat(tree): add tree filtering with fuzzy match and type support"
```

---

## Task 5: Search State in TreeView

**Files:**
- Modify: `internal/ui/components/tree_view.go`

**Step 1: Update TreeView struct with new search state**

Replace the existing search fields in TreeView struct:

```go
// In internal/ui/components/tree_view.go

// SearchModeState represents the current search state
type SearchModeState int

const (
	SearchOff          SearchModeState = iota // No search active
	SearchInputting                           // User is typing search query
	SearchFilterActive                        // Search done, filter applied, navigating results
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
	SearchState   SearchModeState    // Current search state
	SearchQuery   string             // Current search query text
	FilteredNodes []*models.TreeNode // Flat list of nodes matching filter
	MatchPositions map[*models.TreeNode][]int // Match positions for highlighting
}
```

**Step 2: Run existing tests to ensure no regression**

```bash
go test ./internal/ui/components/... -run TestTreeView -v
```

Expected: PASS (existing tests should still work)

**Step 3: Commit**

```bash
git add internal/ui/components/tree_view.go
git commit -m "refactor(tree): update TreeView search state structure"
```

---

## Task 6: Search Input Handling

**Files:**
- Modify: `internal/ui/components/tree_view.go`
- Modify: `internal/ui/components/tree_view_test.go`

**Step 1: Write the failing test for search activation**

```go
// Add to internal/ui/components/tree_view_test.go

func TestTreeView_SearchActivation(t *testing.T) {
	root := createTestTreeForView()
	testTheme := theme.DefaultTheme()
	tv := NewTreeView(root, testTheme)

	// Press / to start search
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	if tv.SearchState != SearchInputting {
		t.Errorf("expected SearchInputting state, got %d", tv.SearchState)
	}
}

func TestTreeView_SearchTyping(t *testing.T) {
	root := createTestTreeForView()
	testTheme := theme.DefaultTheme()
	tv := NewTreeView(root, testTheme)

	// Start search and type
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if tv.SearchQuery != "plan" {
		t.Errorf("expected query 'plan', got '%s'", tv.SearchQuery)
	}
}

func TestTreeView_SearchEscFirstKeepsFilter(t *testing.T) {
	root := createTestTreeForView()
	testTheme := theme.DefaultTheme()
	tv := NewTreeView(root, testTheme)

	// Start search, type, then Esc
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if tv.SearchState != SearchFilterActive {
		t.Errorf("expected SearchFilterActive after first Esc, got %d", tv.SearchState)
	}
	if tv.SearchQuery != "p" {
		t.Errorf("expected query preserved, got '%s'", tv.SearchQuery)
	}
}

func TestTreeView_SearchEscSecondClears(t *testing.T) {
	root := createTestTreeForView()
	testTheme := theme.DefaultTheme()
	tv := NewTreeView(root, testTheme)

	// Start search, type, Esc twice
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyEsc})
	tv, _ = tv.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if tv.SearchState != SearchOff {
		t.Errorf("expected SearchOff after second Esc, got %d", tv.SearchState)
	}
	if tv.SearchQuery != "" {
		t.Errorf("expected query cleared, got '%s'", tv.SearchQuery)
	}
}

// Helper to create test tree
func createTestTreeForView() *models.TreeNode {
	root := models.NewTreeNode("root", models.TreeNodeTypeRoot, "Root")
	root.Expanded = true
	root.Loaded = true

	db := models.NewTreeNode("db:test", models.TreeNodeTypeDatabase, "test")
	db.Expanded = true
	db.Loaded = true
	root.AddChild(db)

	schema := models.NewTreeNode("schema:test.public", models.TreeNodeTypeSchema, "public")
	schema.Expanded = true
	schema.Loaded = true
	db.AddChild(schema)

	tables := models.NewTreeNode("tables", models.TreeNodeTypeTableGroup, "Tables (3)")
	tables.Expanded = true
	tables.Loaded = true
	schema.AddChild(tables)

	plan := models.NewTreeNode("table:test.public.plan", models.TreeNodeTypeTable, "plan")
	planCheck := models.NewTreeNode("table:test.public.plan_check_run", models.TreeNodeTypeTable, "plan_check_run")
	users := models.NewTreeNode("table:test.public.users", models.TreeNodeTypeTable, "users")
	plan.Loaded = true
	planCheck.Loaded = true
	users.Loaded = true
	tables.AddChild(plan)
	tables.AddChild(planCheck)
	tables.AddChild(users)

	return root
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/ui/components/... -run TestTreeView_Search -v
```

Expected: FAIL

**Step 3: Update the Update method to handle search**

```go
// Modify Update method in internal/ui/components/tree_view.go

func (tv *TreeView) Update(msg tea.KeyMsg) (*TreeView, tea.Cmd) {
	// Handle search input mode first
	if tv.SearchState == SearchInputting {
		return tv.handleSearchInput(msg)
	}

	// Handle search filter active mode
	if tv.SearchState == SearchFilterActive {
		switch msg.String() {
		case "esc":
			// Second Esc clears filter
			tv.SearchState = SearchOff
			tv.SearchQuery = ""
			tv.FilteredNodes = nil
			tv.MatchPositions = nil
			tv.CursorIndex = 0
			tv.ScrollOffset = 0
			return tv, nil
		case "/":
			// Re-enter search input mode
			tv.SearchState = SearchInputting
			return tv, nil
		}
		// Fall through to normal navigation (but on filtered list)
	}

	// Normal mode: check for search activation
	if msg.String() == "/" {
		tv.SearchState = SearchInputting
		tv.SearchQuery = ""
		return tv, nil
	}

	// ... rest of existing Update logic ...
	if tv.Root == nil {
		return tv, nil
	}

	visibleNodes := tv.getVisibleNodes()
	if len(visibleNodes) == 0 {
		return tv, nil
	}

	var cmd tea.Cmd

	switch msg.String() {
	case "up", "k":
		if tv.CursorIndex > 0 {
			tv.CursorIndex--
		}

	case "down", "j":
		if tv.CursorIndex < len(visibleNodes)-1 {
			tv.CursorIndex++
		}

	case "g":
		tv.CursorIndex = 0
		tv.ScrollOffset = 0

	case "G":
		tv.CursorIndex = len(visibleNodes) - 1

	case "right", "l", " ":
		currentNode := visibleNodes[tv.CursorIndex]
		if currentNode != nil {
			wasExpanded := currentNode.Expanded
			currentNode.Toggle()
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
		currentNode := visibleNodes[tv.CursorIndex]
		if currentNode != nil {
			if currentNode.Expanded {
				currentNode.Toggle()
				cmd = func() tea.Msg {
					return TreeNodeExpandedMsg{
						Node:     currentNode,
						Expanded: false,
					}
				}
			} else if currentNode.Parent != nil && currentNode.Parent.Type != models.TreeNodeTypeRoot {
				parentIndex := tv.findNodeIndex(visibleNodes, currentNode.Parent)
				if parentIndex >= 0 {
					tv.CursorIndex = parentIndex
				}
			}
		}

	case "enter":
		currentNode := visibleNodes[tv.CursorIndex]
		if currentNode != nil && currentNode.Selectable {
			cmd = func() tea.Msg {
				return TreeNodeSelectedMsg{Node: currentNode}
			}
		}
	}

	return tv, cmd
}

// handleSearchInput handles key input during search mode
func (tv *TreeView) handleSearchInput(msg tea.KeyMsg) (*TreeView, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		// First Esc: exit input mode but keep filter
		if tv.SearchQuery != "" {
			tv.SearchState = SearchFilterActive
			tv.applyFilter()
		} else {
			tv.SearchState = SearchOff
		}
		return tv, nil

	case tea.KeyEnter:
		// Confirm search, move to filter active mode
		if tv.SearchQuery != "" {
			tv.SearchState = SearchFilterActive
			tv.applyFilter()
			tv.CursorIndex = 0
			tv.ScrollOffset = 0
		} else {
			tv.SearchState = SearchOff
		}
		return tv, nil

	case tea.KeyBackspace:
		if len(tv.SearchQuery) > 0 {
			tv.SearchQuery = tv.SearchQuery[:len(tv.SearchQuery)-1]
			tv.applyFilter()
		}
		return tv, nil

	case tea.KeyRunes:
		tv.SearchQuery += string(msg.Runes)
		tv.applyFilter()
		return tv, nil
	}

	return tv, nil
}

// applyFilter applies the current search query to filter the tree
func (tv *TreeView) applyFilter() {
	if tv.SearchQuery == "" {
		tv.FilteredNodes = nil
		tv.MatchPositions = nil
		return
	}

	query := ParseSearchQuery(tv.SearchQuery)
	tv.FilteredNodes = FilterTree(tv.Root, query)

	// Build match positions for highlighting
	tv.MatchPositions = make(map[*models.TreeNode][]int)
	for _, node := range tv.FilteredNodes {
		if query.Pattern != "" && !query.Negate {
			_, positions := FuzzyMatch(query.Pattern, node.Label)
			tv.MatchPositions[node] = positions
		}
	}

	// Reset cursor if out of bounds
	if tv.CursorIndex >= len(tv.FilteredNodes) {
		tv.CursorIndex = 0
	}
}

// getVisibleNodes returns the appropriate node list based on search state
func (tv *TreeView) getVisibleNodes() []*models.TreeNode {
	if tv.SearchState == SearchFilterActive && tv.FilteredNodes != nil {
		return tv.FilteredNodes
	}
	if tv.Root == nil {
		return nil
	}
	return tv.Root.Flatten()
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/ui/components/... -run TestTreeView_Search -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/components/tree_view.go internal/ui/components/tree_view_test.go
git commit -m "feat(tree): implement search input handling with two-stage Esc"
```

---

## Task 7: Search UI Rendering

**Files:**
- Modify: `internal/ui/components/tree_view.go`

**Step 1: Update View method to show search bar and filtered results**

```go
// Modify View method in internal/ui/components/tree_view.go

func (tv *TreeView) View() string {
	var lines []string

	// Render search bar if in search mode
	searchBarHeight := 0
	if tv.SearchState != SearchOff {
		searchBar := tv.renderSearchBar()
		lines = append(lines, searchBar)
		searchBarHeight = 1
	}

	// Get nodes to display
	visibleNodes := tv.getVisibleNodes()

	// Handle empty state
	if tv.Root == nil || len(visibleNodes) == 0 {
		if tv.SearchState != SearchOff {
			lines = append(lines, tv.noMatchesState())
		} else {
			lines = append(lines, tv.emptyState())
		}
		for len(lines) < tv.Height {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	// Ensure cursor is within bounds
	if tv.CursorIndex < 0 {
		tv.CursorIndex = 0
	}
	if tv.CursorIndex >= len(visibleNodes) {
		tv.CursorIndex = len(visibleNodes) - 1
	}

	// Calculate viewport dimensions (account for search bar)
	viewHeight := tv.Height - searchBarHeight
	if viewHeight < 1 {
		viewHeight = 1
	}

	// Check if we need scroll indicators
	needsScrollIndicator := len(visibleNodes) > viewHeight
	nodeViewHeight := viewHeight
	if needsScrollIndicator && viewHeight > 1 {
		nodeViewHeight = viewHeight - 1
	}

	// Auto-scroll
	tv.adjustScrollOffset(len(visibleNodes), nodeViewHeight)

	// Calculate visible range
	startIdx := tv.ScrollOffset
	endIdx := tv.ScrollOffset + nodeViewHeight
	if endIdx > len(visibleNodes) {
		endIdx = len(visibleNodes)
	}

	// Render visible nodes
	for i := startIdx; i < endIdx; i++ {
		node := visibleNodes[i]
		line := tv.renderNodeWithHighlight(node, i == tv.CursorIndex)
		zoneID := fmt.Sprintf("%s%d", ZoneTreeRowPrefix, i-startIdx)
		lines = append(lines, zone.Mark(zoneID, line))
	}

	// Add scroll indicator if needed
	if needsScrollIndicator {
		indicatorLine := tv.renderScrollIndicator(startIdx, endIdx, len(visibleNodes))
		lines = append(lines, indicatorLine)
	}

	// Fill remaining space
	for len(lines) < tv.Height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// renderSearchBar renders the search input bar
func (tv *TreeView) renderSearchBar() string {
	maxWidth := tv.Width - 2

	var content string
	if tv.SearchState == SearchInputting {
		// Show input with cursor
		content = fmt.Sprintf("/ %s_", tv.SearchQuery)
	} else {
		// Show match count
		count := len(tv.FilteredNodes)
		content = fmt.Sprintf("[%d] / %s", count, tv.SearchQuery)
	}

	// Truncate if too long
	if len(content) > maxWidth {
		content = content[:maxWidth-1] + "…"
	}

	style := lipgloss.NewStyle().
		Foreground(tv.Theme.Info).
		Width(maxWidth)

	return style.Render(content)
}

// renderNodeWithHighlight renders a node with fuzzy match highlighting
func (tv *TreeView) renderNodeWithHighlight(node *models.TreeNode, selected bool) string {
	if node == nil {
		return ""
	}

	depth := node.GetDepth() - 1
	if depth < 0 {
		depth = 0
	}
	indent := strings.Repeat("  ", depth)
	icon := tv.getNodeIcon(node)

	// Build label with highlighting
	label := tv.buildNodeLabelWithHighlight(node)

	content := fmt.Sprintf("%s%s %s", indent, icon, label)

	maxWidth := tv.Width - 2
	if len(content) > maxWidth {
		content = content[:maxWidth-1] + "…"
	}

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

// buildNodeLabelWithHighlight builds label with match positions highlighted
func (tv *TreeView) buildNodeLabelWithHighlight(node *models.TreeNode) string {
	label := node.Label

	// Get match positions if available
	positions, hasPositions := tv.MatchPositions[node]
	if !hasPositions || len(positions) == 0 {
		return tv.buildNodeLabel(node)
	}

	// Build highlighted label
	highlightStyle := lipgloss.NewStyle().
		Foreground(tv.Theme.Warning).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		Foreground(tv.Theme.Foreground)

	var result strings.Builder
	posSet := make(map[int]bool)
	for _, p := range positions {
		posSet[p] = true
	}

	for i, ch := range label {
		if posSet[i] {
			result.WriteString(highlightStyle.Render(string(ch)))
		} else {
			result.WriteString(normalStyle.Render(string(ch)))
		}
	}

	// Add metadata suffix (from original buildNodeLabel)
	metaStyle := lipgloss.NewStyle().Foreground(tv.Theme.Metadata)
	switch node.Type {
	case models.TreeNodeTypeTable:
		if meta, ok := node.Metadata.(map[string]interface{}); ok {
			if rowCount, ok := meta["row_count"].(int64); ok {
				result.WriteString(" ")
				result.WriteString(metaStyle.Render(formatNumber(rowCount)))
			}
		}
	}

	return result.String()
}

// noMatchesState returns the no matches view
func (tv *TreeView) noMatchesState() string {
	style := lipgloss.NewStyle().
		Foreground(tv.Theme.Comment).
		Italic(true).
		Width(tv.Width - 2).
		Align(lipgloss.Center)

	return style.Render("No matches found")
}
```

**Step 2: Run all tests**

```bash
go test ./internal/ui/components/... -v
```

Expected: PASS

**Step 3: Commit**

```bash
git add internal/ui/components/tree_view.go
git commit -m "feat(tree): add search bar UI and match highlighting"
```

---

## Task 8: Route / Key from App

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Find where TreeView key events are handled and add / routing**

Search for where TreeView.Update is called and ensure `/` key is passed through when TreeView has focus (FocusArea == FocusTreeView).

The key routing should already work if TreeView is receiving key events when focused. Verify by testing.

**Step 2: Test manually**

```bash
go build -o lazypg ./cmd/lazypg && ./lazypg
```

1. Connect to a database
2. Focus on TreeView (left panel)
3. Press `/`
4. Type a search query
5. Press Esc once (filter stays)
6. Press Esc again (filter clears)

**Step 3: Commit if changes needed**

```bash
git add internal/app/app.go
git commit -m "feat(tree): ensure / key routes to TreeView search"
```

---

## Task 9: Final Integration Test

**Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: All PASS

**Step 2: Manual testing checklist**

- [ ] `/` activates search in TreeView
- [ ] Typing filters results in real-time
- [ ] Fuzzy matching works (e.g., "pcr" matches "plan_check_run")
- [ ] Type prefix works (e.g., "t:plan" only shows tables)
- [ ] Negation works (e.g., "!plan" excludes plan)
- [ ] First Esc keeps filter, exits input mode
- [ ] Second Esc clears filter
- [ ] Match characters are highlighted
- [ ] Navigation (j/k) works on filtered results
- [ ] Enter selects node from filtered results

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat(tree): complete TreeView search/filter implementation"
```

---

Plan complete and saved to `docs/plans/2025-12-17-tree-search-filter-impl.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
