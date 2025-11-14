package models

// AppState holds the application state
type AppState struct {
	Width          int
	Height         int
	LeftPanelWidth int
	FocusedPanel   PanelType
	ViewMode       ViewMode

	// Connection state (Phase 2)
	ConnectionManager interface{} // Will hold *connection.Manager
	ActiveConnection  *Connection
	Databases         []string
	CurrentDatabase   string
	CurrentSchema     string

	// Navigation tree state (Phase 3)
	TreeRoot         *TreeNode   // Root of the navigation tree
	TreeSelected     *TreeNode   // Currently selected node
	TreeCursorIndex  int         // Cursor position in flat list
	TreeVisibleNodes []*TreeNode // Cached flat list of visible nodes
}

// PanelType identifies which panel is focused
type PanelType int

const (
	LeftPanel PanelType = iota
	RightPanel
)

// ViewMode identifies the current view
type ViewMode int

const (
	NormalMode ViewMode = iota
	HelpMode
)

// NewAppState creates a new AppState with defaults
func NewAppState() AppState {
	return AppState{
		Width:          80,
		Height:         24,
		LeftPanelWidth: 25,
		FocusedPanel:   LeftPanel,
		ViewMode:       NormalMode,
	}
}

// ColumnDetail contains comprehensive column information for structure view
type ColumnDetail struct {
	Name          string
	DataType      string
	IsNullable    bool
	DefaultValue  string
	IsPrimaryKey  bool
	IsForeignKey  bool
	IsUnique      bool
	HasCheck      bool
	Comment       string
}

// Constraint represents a table constraint
type Constraint struct {
	Name         string
	Type         string // 'p'=PK, 'f'=FK, 'u'=Unique, 'c'=Check
	Columns      []string
	Definition   string
	ForeignTable string // For FK: "schema.table"
	ForeignCols  []string
}

// IndexInfo represents an index
type IndexInfo struct {
	Name        string
	Type        string // btree, hash, gin, gist, brin, spgist
	Columns     []string
	Definition  string
	IsUnique    bool
	IsPrimary   bool
	IsPartial   bool
	Size        int64
	Predicate   string // WHERE clause for partial indexes
}
