package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

const MaxResultTabs = 10

// Pre-compiled regex patterns for performance
var (
	dashCommentRe  = regexp.MustCompile(`^\s*--\s*(.+)$`)
	blockCommentRe = regexp.MustCompile(`^\s*/\*\s*(.+?)\s*\*/`)
	fromRe         = regexp.MustCompile(`(?i)\bFROM\s+([a-zA-Z_][a-zA-Z0-9_.]*)(?:\s+(?:AS\s+)?[a-zA-Z_][a-zA-Z0-9_]*)?`)
	updateRe       = regexp.MustCompile(`(?i)\bUPDATE\s+([a-zA-Z_][a-zA-Z0-9_.]*)`)
	deleteRe       = regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+([a-zA-Z_][a-zA-Z0-9_.]*)`)
	insertRe       = regexp.MustCompile(`(?i)\bINSERT\s+INTO\s+([a-zA-Z_][a-zA-Z0-9_.]*)`)
)

// ResultTab represents a single query result tab
type ResultTab struct {
	ID        int
	Title     string
	SQL       string
	Result    models.QueryResult
	CreatedAt time.Time
	TableView *TableView
}

// ResultTabs manages multiple query result tabs
type ResultTabs struct {
	tabs      []*ResultTab
	activeIdx int
	nextID    int
	Theme     theme.Theme
}

// NewResultTabs creates a new result tabs manager
func NewResultTabs(th theme.Theme) *ResultTabs {
	return &ResultTabs{
		tabs:      []*ResultTab{},
		activeIdx: 0,
		nextID:    1,
		Theme:     th,
	}
}

// AddResult adds a new query result as a tab (newest appears on the left)
func (rt *ResultTabs) AddResult(sql string, result models.QueryResult) {
	// Create TableView for this result
	tableView := NewTableView(rt.Theme)
	tableView.SetData(result.Columns, result.Rows, len(result.Rows))

	tab := &ResultTab{
		ID:        rt.nextID,
		Title:     rt.generateTitle(sql, result),
		SQL:       sql,
		Result:    result,
		CreatedAt: time.Now(),
		TableView: tableView,
	}
	rt.nextID++

	// Insert new tab at the beginning (leftmost position)
	rt.tabs = append([]*ResultTab{tab}, rt.tabs...)

	// Remove oldest (rightmost) if exceeding max
	if len(rt.tabs) > MaxResultTabs {
		rt.tabs = rt.tabs[:MaxResultTabs]
	}

	// Set new tab as active (index 0 = leftmost)
	rt.activeIdx = 0
}

// generateTitle generates a smart title for the tab
func (rt *ResultTabs) generateTitle(sql string, result models.QueryResult) string {
	// Check for custom comment title
	if title := rt.extractCommentTitle(sql); title != "" {
		return title
	}

	// Extract table name from SQL
	if tableName := rt.extractTableName(sql); tableName != "" {
		return tableName
	}

	// Fallback to truncated SQL
	cleaned := strings.TrimSpace(sql)
	cleaned = strings.ReplaceAll(cleaned, "\n", " ")
	if len(cleaned) > 20 {
		cleaned = cleaned[:17] + "..."
	}
	return cleaned
}

// extractCommentTitle extracts title from SQL comment (-- title or /* title */)
func (rt *ResultTabs) extractCommentTitle(sql string) string {
	// Match -- comment at start
	lines := strings.Split(sql, "\n")
	if len(lines) > 0 {
		if matches := dashCommentRe.FindStringSubmatch(lines[0]); len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Match /* comment */ at start
	if matches := blockCommentRe.FindStringSubmatch(sql); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// extractTableName extracts the main table name from SQL
func (rt *ResultTabs) extractTableName(sql string) string {
	upperSQL := strings.ToUpper(sql)

	// SELECT ... FROM table
	if matches := fromRe.FindStringSubmatch(sql); len(matches) > 1 {
		tableName := matches[1]
		// Check for JOIN
		if strings.Contains(upperSQL, "JOIN") {
			return tableName + "(+)"
		}
		return tableName
	}

	// UPDATE table
	if matches := updateRe.FindStringSubmatch(sql); len(matches) > 1 {
		return "UPDATE " + matches[1]
	}

	// DELETE FROM table
	if matches := deleteRe.FindStringSubmatch(sql); len(matches) > 1 {
		return "DELETE " + matches[1]
	}

	// INSERT INTO table
	if matches := insertRe.FindStringSubmatch(sql); len(matches) > 1 {
		return "INSERT " + matches[1]
	}

	return ""
}

// GetActiveTab returns the currently active tab
func (rt *ResultTabs) GetActiveTab() *ResultTab {
	if len(rt.tabs) == 0 || rt.activeIdx < 0 || rt.activeIdx >= len(rt.tabs) {
		return nil
	}
	return rt.tabs[rt.activeIdx]
}

// GetActiveTableView returns the TableView of the active tab
func (rt *ResultTabs) GetActiveTableView() *TableView {
	tab := rt.GetActiveTab()
	if tab == nil {
		return nil
	}
	return tab.TableView
}

// GetActiveSQL returns the SQL of the active tab
func (rt *ResultTabs) GetActiveSQL() string {
	tab := rt.GetActiveTab()
	if tab == nil {
		return ""
	}
	return tab.SQL
}

// NextTab switches to the next tab
func (rt *ResultTabs) NextTab() {
	if len(rt.tabs) > 0 {
		rt.activeIdx = (rt.activeIdx + 1) % len(rt.tabs)
	}
}

// PrevTab switches to the previous tab
func (rt *ResultTabs) PrevTab() {
	if len(rt.tabs) > 0 {
		rt.activeIdx = (rt.activeIdx - 1 + len(rt.tabs)) % len(rt.tabs)
	}
}

// TabCount returns the number of tabs
func (rt *ResultTabs) TabCount() int {
	return len(rt.tabs)
}

// HasTabs returns whether there are any tabs
func (rt *ResultTabs) HasTabs() bool {
	return len(rt.tabs) > 0
}

// RenderTabBar renders the tab bar
func (rt *ResultTabs) RenderTabBar(width int) string {
	if len(rt.tabs) == 0 {
		return ""
	}

	var tabViews []string

	for i, tab := range rt.tabs {
		// Format: [index] title (rows)
		rowCount := len(tab.Result.Rows)
		rowStr := fmt.Sprintf("%d rows", rowCount)
		if rowCount == 1 {
			rowStr = "1 row"
		}

		label := fmt.Sprintf("[%d] %s (%s)", i+1, tab.Title, rowStr)

		// Truncate if too long
		maxLabelLen := width / MaxResultTabs
		if maxLabelLen < 15 {
			maxLabelLen = 15
		}
		if len(label) > maxLabelLen {
			// Try without row count
			label = fmt.Sprintf("[%d] %s", i+1, tab.Title)
			if len(label) > maxLabelLen {
				label = label[:maxLabelLen-3] + "..."
			}
		}

		var style lipgloss.Style
		if i == rt.activeIdx {
			style = lipgloss.NewStyle().
				Foreground(rt.Theme.Background).
				Background(rt.Theme.Info).
				Bold(true).
				Padding(0, 1)
		} else {
			style = lipgloss.NewStyle().
				Foreground(rt.Theme.Foreground).
				Background(rt.Theme.Selection).
				Padding(0, 1)
		}

		tabViews = append(tabViews, style.Render(label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)
}
