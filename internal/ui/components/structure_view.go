package components

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/metadata"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// StructureView is a tabbed container for viewing table structure
type StructureView struct {
	Width  int
	Height int
	Theme  theme.Theme

	// Current active tab (0=Data, 1=Columns, 2=Constraints, 3=Indexes)
	activeTab int

	// Tab views
	tableView       *TableView      // For Data tab
	columnsView     *ColumnsView
	constraintsView *ConstraintsView
	indexesView     *IndexesView

	// Table info
	schema string
	table  string
	pool   *connection.Pool

	// Status
	loading      bool
	errorMessage string
}

// NewStructureView creates a new structure view
func NewStructureView(th theme.Theme, tableView *TableView) *StructureView {
	return &StructureView{
		Theme:           th,
		activeTab:       0, // Start with Data tab
		tableView:       tableView,
		columnsView:     NewColumnsView(th),
		constraintsView: NewConstraintsView(th),
		indexesView:     NewIndexesView(th),
	}
}

// HasTableLoaded checks if structure data has been loaded for the given table
func (sv *StructureView) HasTableLoaded(schema, table string) bool {
	return sv.schema == schema && sv.table == table
}

// SetTable sets the current table and loads structure data
func (sv *StructureView) SetTable(ctx context.Context, pool *connection.Pool, schema, table string) error {
	sv.schema = schema
	sv.table = table
	sv.pool = pool
	sv.loading = true
	sv.errorMessage = ""

	// Load columns
	columns, err := metadata.GetColumnDetails(ctx, pool, schema, table)
	if err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to load columns: %v", err)
		sv.loading = false
		return err
	}
	sv.columnsView.SetColumns(columns)

	// Load constraints
	constraints, err := metadata.GetConstraints(ctx, pool, schema, table)
	if err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to load constraints: %v", err)
		sv.loading = false
		return err
	}
	sv.constraintsView.SetConstraints(constraints)

	// Load indexes
	indexes, err := metadata.GetIndexes(ctx, pool, schema, table)
	if err != nil {
		sv.errorMessage = fmt.Sprintf("Failed to load indexes: %v", err)
		sv.loading = false
		return err
	}
	sv.indexesView.SetIndexes(indexes)

	sv.loading = false
	return nil
}

// SwitchTab switches to a specific tab
func (sv *StructureView) SwitchTab(tabIndex int) {
	if tabIndex >= 0 && tabIndex <= 3 {
		sv.activeTab = tabIndex
	}
}

// Update handles keyboard input
func (sv *StructureView) Update(msg tea.KeyMsg) {
	if sv.activeTab == 0 {
		// Data tab - handled by app.go with existing table view
		return
	}

	// Handle navigation keys for structure tabs
	switch msg.String() {
	case "up", "k":
		sv.getCurrentView().MoveSelection(-1)
	case "down", "j":
		sv.getCurrentView().MoveSelection(1)
	case "left", "h":
		sv.SwitchTab(sv.activeTab - 1)
	case "right", "l":
		sv.SwitchTab(sv.activeTab + 1)
	}
}

type structureViewNavigator interface {
	MoveSelection(delta int)
}

func (sv *StructureView) getCurrentView() structureViewNavigator {
	switch sv.activeTab {
	case 1:
		return sv.columnsView
	case 2:
		return sv.constraintsView
	case 3:
		return sv.indexesView
	default:
		return sv.columnsView
	}
}

// View renders the structure view
func (sv *StructureView) View() string {
	if sv.loading {
		return lipgloss.NewStyle().
			Foreground(sv.Theme.Metadata).
			Render("Loading structure...")
	}

	if sv.errorMessage != "" {
		return lipgloss.NewStyle().
			Foreground(sv.Theme.Error).
			Render(sv.errorMessage)
	}

	var b strings.Builder

	// Render tab bar
	b.WriteString(sv.renderTabBar())
	b.WriteString("\n")

	// Calculate content height (subtract tab bar)
	contentHeight := sv.Height - 1

	// Update view dimensions
	sv.tableView.Width = sv.Width
	sv.tableView.Height = contentHeight
	sv.columnsView.Width = sv.Width
	sv.columnsView.Height = contentHeight
	sv.constraintsView.Width = sv.Width
	sv.constraintsView.Height = contentHeight
	sv.indexesView.Width = sv.Width
	sv.indexesView.Height = contentHeight

	// Render active tab content
	switch sv.activeTab {
	case 0:
		b.WriteString(sv.tableView.View())
	case 1:
		b.WriteString(sv.columnsView.View())
	case 2:
		b.WriteString(sv.constraintsView.View())
	case 3:
		b.WriteString(sv.indexesView.View())
	default:
		b.WriteString("Unknown tab")
	}

	return b.String()
}

func (sv *StructureView) renderTabBar() string {
	tabs := []struct {
		index int
		label string
	}{
		{0, "Data"},
		{1, "Columns"},
		{2, "Constraints"},
		{3, "Indexes"},
	}

	tabParts := make([]string, len(tabs))
	for i, tab := range tabs {
		var style lipgloss.Style
		if tab.index == sv.activeTab {
			// Active tab
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(sv.Theme.BorderFocused).
				Background(sv.Theme.Selection).
				Padding(0, 2)
		} else {
			// Inactive tab
			style = lipgloss.NewStyle().
				Foreground(sv.Theme.Metadata).
				Padding(0, 2)
		}
		tabParts[i] = style.Render(tab.label)
	}

	separator := lipgloss.NewStyle().
		Foreground(sv.Theme.Border).
		Render(" │ ")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		tabParts[0], separator,
		tabParts[1], separator,
		tabParts[2], separator,
		tabParts[3],
	)
}

// CopyCurrentName copies the name of the selected item
func (sv *StructureView) CopyCurrentName() string {
	var name string
	switch sv.activeTab {
	case 1:
		if col := sv.columnsView.GetSelectedColumn(); col != nil {
			name = col.Name
		}
	case 2:
		if con := sv.constraintsView.GetSelectedConstraint(); con != nil {
			name = con.Name
		}
	case 3:
		if idx := sv.indexesView.GetSelectedIndex(); idx != nil {
			name = idx.Name
		}
	}

	if name != "" {
		clipboard.WriteAll(name)
		return fmt.Sprintf("✓ Copied: %s", name)
	}
	return ""
}

// CopyCurrentDefinition copies the full definition of the selected item
func (sv *StructureView) CopyCurrentDefinition() string {
	var definition string
	switch sv.activeTab {
	case 1:
		if col := sv.columnsView.GetSelectedColumn(); col != nil {
			definition = fmt.Sprintf("%s %s %s DEFAULT %s",
				col.Name, col.DataType,
				map[bool]string{true: "NULL", false: "NOT NULL"}[col.IsNullable],
				col.DefaultValue)
		}
	case 2:
		if con := sv.constraintsView.GetSelectedConstraint(); con != nil {
			definition = con.Definition
		}
	case 3:
		if idx := sv.indexesView.GetSelectedIndex(); idx != nil {
			definition = idx.Definition
		}
	}

	if definition != "" {
		clipboard.WriteAll(definition)
		preview := definition
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		return fmt.Sprintf("✓ Copied: %s", preview)
	}
	return ""
}
