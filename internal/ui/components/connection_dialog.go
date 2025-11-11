package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// ConnectionDialog represents a connection dialog
type ConnectionDialog struct {
	Width               int
	Height              int
	Theme               theme.Theme
	DiscoveredInstances []models.DiscoveredInstance
	HistoryEntries      []models.ConnectionHistoryEntry
	ManualMode          bool
	SelectedIndex       int
	InHistorySection    bool // true = selecting in history, false = selecting in discovered

	// Search
	searchInput textinput.Model
	searchQuery string

	// Text input fields for manual mode
	inputs      []textinput.Model
	focusIndex  int
	cursorMode  cursor.Mode
}

const (
	hostField = iota
	portField
	databaseField
	userField
	passwordField
)

// NewConnectionDialog creates a new connection dialog
func NewConnectionDialog(th theme.Theme) *ConnectionDialog {
	// Create text inputs for each field
	inputs := make([]textinput.Model, 5)

	// Host input
	inputs[hostField] = textinput.New()
	inputs[hostField].Placeholder = "localhost"
	inputs[hostField].Focus()
	inputs[hostField].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	inputs[hostField].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	inputs[hostField].Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	inputs[hostField].CharLimit = 100
	inputs[hostField].Width = 40

	// Port input
	inputs[portField] = textinput.New()
	inputs[portField].Placeholder = "5432"
	inputs[portField].SetValue("5432")
	inputs[portField].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	inputs[portField].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	inputs[portField].Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	inputs[portField].CharLimit = 5
	inputs[portField].Width = 40

	// Database input
	inputs[databaseField] = textinput.New()
	inputs[databaseField].Placeholder = "postgres"
	inputs[databaseField].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	inputs[databaseField].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	inputs[databaseField].Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	inputs[databaseField].CharLimit = 100
	inputs[databaseField].Width = 40

	// User input
	inputs[userField] = textinput.New()
	inputs[userField].Placeholder = "postgres"
	inputs[userField].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	inputs[userField].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	inputs[userField].Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	inputs[userField].CharLimit = 100
	inputs[userField].Width = 40

	// Password input
	inputs[passwordField] = textinput.New()
	inputs[passwordField].Placeholder = ""
	inputs[passwordField].EchoMode = textinput.EchoPassword
	inputs[passwordField].EchoCharacter = '•'
	inputs[passwordField].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
	inputs[passwordField].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	inputs[passwordField].Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	inputs[passwordField].CharLimit = 100
	inputs[passwordField].Width = 40

	// Create search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search for connection..."
	searchInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	searchInput.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	searchInput.CharLimit = 100
	searchInput.Width = 60

	return &ConnectionDialog{
		inputs:           inputs,
		focusIndex:       0,
		cursorMode:       cursor.CursorBlink,
		Theme:            th,
		searchInput:      searchInput,
		InHistorySection: true, // Start in history section
	}
}

// Init initializes the connection dialog (required for tea.Model)
func (c *ConnectionDialog) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the connection dialog
func (c *ConnectionDialog) Update(msg tea.Msg) (*ConnectionDialog, tea.Cmd) {
	if !c.ManualMode {
		return c, nil
	}

	// Update the focused text input
	var cmd tea.Cmd
	c.inputs[c.focusIndex], cmd = c.inputs[c.focusIndex].Update(msg)
	return c, cmd
}

// View renders the connection dialog
func (c *ConnectionDialog) View() string {
	if c.Width <= 0 || c.Height <= 0 {
		return ""
	}

	var content string
	if c.ManualMode {
		content = c.renderManualMode()
	} else {
		content = c.renderDiscoveryMode()
	}

	// Create compact container - use MaxWidth instead of Width to avoid border overflow
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#cba6f7")).
		Padding(1, 2).
		MaxWidth(76) // MaxWidth constrains content, border adds 2 more chars (total ~78)

	return lipgloss.Place(
		c.Width,
		c.Height,
		lipgloss.Center,
		lipgloss.Center,
		containerStyle.Render(content),
	)
}

func (c *ConnectionDialog) renderDiscoveryMode() string {
	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#cba6f7"))
	sections = append(sections, titleStyle.Render("🔌 Open Connection"))
	sections = append(sections, "")

	// Search box
	searchBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#89b4fa")).
		Padding(0, 1).
		MaxWidth(66) // Safe width for 76-char container
	sections = append(sections, searchBoxStyle.Render("🔍 "+c.searchInput.View()))
	sections = append(sections, "")

	// History section header
	historyHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Bold(true)
	sections = append(sections, historyHeaderStyle.Render("Recent Connections"))

	// History entries
	if len(c.HistoryEntries) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Italic(true).
			PaddingLeft(2)
		sections = append(sections, emptyStyle.Render("No history yet"))
	} else {
		historyCount := 0
		for i, entry := range c.HistoryEntries {
			if historyCount >= 5 {
				break // Limit to 5 history items
			}

			itemStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cdd6f4")).
				PaddingLeft(2)

			// Check if this item is selected and we're in history section
			if c.InHistorySection && i == c.SelectedIndex {
				itemStyle = itemStyle.
					Foreground(lipgloss.Color("#1e1e2e")).
					Background(lipgloss.Color("#a6e3a1")).
					Bold(true).
					PaddingLeft(1)
			}

			// Format: name (local)
			metaStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6c7086"))
			line := fmt.Sprintf("%s  %s",
				entry.Name,
				metaStyle.Render("(local)"),
			)
			sections = append(sections, itemStyle.Render(line))
			historyCount++
		}
	}

	sections = append(sections, "")

	// Discovered section header
	discoveredHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Bold(true)
	sections = append(sections, discoveredHeaderStyle.Render("Discovered"))

	// Discovered instances
	if len(c.DiscoveredInstances) == 0 {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086")).
			Italic(true).
			PaddingLeft(2)
		sections = append(sections, loadingStyle.Render("Searching..."))
	} else {
		discoveredCount := 0
		for i, instance := range c.DiscoveredInstances {
			if discoveredCount >= 3 {
				break // Limit to 3 discovered items
			}

			itemStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cdd6f4")).
				PaddingLeft(2)

			// Check if this item is selected and we're in discovered section
			if !c.InHistorySection && i == c.SelectedIndex {
				itemStyle = itemStyle.
					Foreground(lipgloss.Color("#1e1e2e")).
					Background(lipgloss.Color("#a6e3a1")).
					Bold(true).
					PaddingLeft(1)
			}

			sourceStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6c7086"))
			line := fmt.Sprintf("%s:%d  %s",
				instance.Host,
				instance.Port,
				sourceStyle.Render(fmt.Sprintf("(%s)", instance.Source.String())),
			)
			sections = append(sections, itemStyle.Render(line))
			discoveredCount++
		}
	}

	sections = append(sections, "")

	// Instructions (keep under 68 chars)
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086"))
	sections = append(sections, helpStyle.Render("↑↓: Navigate │ Enter: Connect │ m: Manual │ Esc: Close"))

	return strings.Join(sections, "\n")
}

func (c *ConnectionDialog) renderManualMode() string {
	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#cba6f7")).
		MarginBottom(1)
	sections = append(sections, titleStyle.Render("🔧 Manual Connection"))

	// Form fields
	fieldLabels := []string{"Host:", "Port:", "Database:", "User:", "Password:"}

	for i, label := range fieldLabels {
		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6adc8")).
			Width(10).
			Align(lipgloss.Right)

		// Add focus indicator
		focusIndicator := "  "
		if i == c.focusIndex {
			focusIndicator = "▸ "
		}

		fieldLine := fmt.Sprintf("%s%s %s",
			focusIndicator,
			labelStyle.Render(label),
			c.inputs[i].View(),
		)
		sections = append(sections, fieldLine)
	}

	sections = append(sections, "")

	// Instructions - shorter to fit within MaxWidth
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086"))
	sections = append(sections, helpStyle.Render("Tab: Next  │  Enter: Connect  │  Ctrl+D: Back  │  Esc: Cancel"))

	return strings.Join(sections, "\n")
}

// NextInput focuses the next input field
func (c *ConnectionDialog) NextInput() {
	c.inputs[c.focusIndex].Blur()
	c.focusIndex = (c.focusIndex + 1) % len(c.inputs)
	c.inputs[c.focusIndex].Focus()
}

// PrevInput focuses the previous input field
func (c *ConnectionDialog) PrevInput() {
	c.inputs[c.focusIndex].Blur()
	c.focusIndex--
	if c.focusIndex < 0 {
		c.focusIndex = len(c.inputs) - 1
	}
	c.inputs[c.focusIndex].Focus()
}

// MoveSelection moves the selection up or down in discovery mode
func (c *ConnectionDialog) MoveSelection(delta int) {
	if c.ManualMode {
		if delta > 0 {
			c.NextInput()
		} else {
			c.PrevInput()
		}
		return
	}

	// Get the list size based on current section
	listSize := 0
	if c.InHistorySection {
		listSize = len(c.HistoryEntries)
		if listSize > 5 {
			listSize = 5 // Limit to 5 displayed history items
		}
	} else {
		listSize = len(c.DiscoveredInstances)
		if listSize > 3 {
			listSize = 3 // Limit to 3 displayed discovered items
		}
	}

	if listSize == 0 {
		c.SelectedIndex = 0
		return
	}

	c.SelectedIndex += delta
	if c.SelectedIndex < 0 {
		// Move to discovered section if at top of history
		if c.InHistorySection {
			c.InHistorySection = false
			c.SelectedIndex = 0
		} else {
			c.SelectedIndex = 0
		}
	} else if c.SelectedIndex >= listSize {
		// Move to next section or wrap
		if c.InHistorySection {
			// Move to discovered section
			c.InHistorySection = false
			c.SelectedIndex = 0
		} else {
			c.SelectedIndex = listSize - 1
		}
	}
}

// SwitchSection switches between history and discovered sections
func (c *ConnectionDialog) SwitchSection() {
	c.InHistorySection = !c.InHistorySection
	c.SelectedIndex = 0 // Reset selection when switching sections
}

// ToggleMode switches between discovery and manual mode
func (c *ConnectionDialog) ToggleMode() {
	c.ManualMode = !c.ManualMode
	if c.ManualMode {
		// Focus first input when entering manual mode
		c.focusIndex = 0
		c.inputs[c.focusIndex].Focus()
	} else {
		// Blur all inputs when leaving manual mode
		for i := range c.inputs {
			c.inputs[i].Blur()
		}
	}
}

// GetSelectedInstance returns the currently selected instance
func (c *ConnectionDialog) GetSelectedInstance() *models.DiscoveredInstance {
	if c.ManualMode || c.InHistorySection || c.SelectedIndex < 0 || c.SelectedIndex >= len(c.DiscoveredInstances) {
		return nil
	}
	return &c.DiscoveredInstances[c.SelectedIndex]
}

// GetSelectedHistory returns the currently selected history entry
func (c *ConnectionDialog) GetSelectedHistory() *models.ConnectionHistoryEntry {
	if c.ManualMode || !c.InHistorySection || c.SelectedIndex < 0 || c.SelectedIndex >= len(c.HistoryEntries) {
		return nil
	}
	return &c.HistoryEntries[c.SelectedIndex]
}

// GetManualConfig returns the manual connection config if valid, or error
func (c *ConnectionDialog) GetManualConfig() (models.ConnectionConfig, error) {
	host := strings.TrimSpace(c.inputs[hostField].Value())
	port := strings.TrimSpace(c.inputs[portField].Value())
	database := strings.TrimSpace(c.inputs[databaseField].Value())
	user := strings.TrimSpace(c.inputs[userField].Value())
	password := c.inputs[passwordField].Value()

	// Use placeholder values as defaults when fields are empty
	if host == "" {
		host = c.inputs[hostField].Placeholder
	}
	if port == "" {
		port = c.inputs[portField].Placeholder
	}
	if database == "" {
		database = c.inputs[databaseField].Placeholder
	}
	if user == "" {
		user = c.inputs[userField].Placeholder
	}

	// Validate required fields after applying defaults
	if host == "" {
		return models.ConnectionConfig{}, fmt.Errorf("host is required")
	}
	if user == "" {
		return models.ConnectionConfig{}, fmt.Errorf("user is required")
	}
	if database == "" {
		return models.ConnectionConfig{}, fmt.Errorf("database is required")
	}

	return models.ConnectionConfig{
		Host:     host,
		Port:     mustParseInt(port, 5432),
		Database: database,
		User:     user,
		Password: password,
		SSLMode:  "prefer",
	}, nil
}

// SetDiscoveredInstances updates the list of discovered instances
func (c *ConnectionDialog) SetDiscoveredInstances(instances []models.DiscoveredInstance) {
	c.DiscoveredInstances = instances
	if !c.InHistorySection && c.SelectedIndex >= len(instances) {
		c.SelectedIndex = 0
	}
}

// SetHistoryEntries updates the list of connection history entries
func (c *ConnectionDialog) SetHistoryEntries(entries []models.ConnectionHistoryEntry) {
	c.HistoryEntries = entries
	if c.InHistorySection && c.SelectedIndex >= len(entries) {
		c.SelectedIndex = 0
	}
}

func mustParseInt(s string, defaultVal int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultVal
	}
	return result
}
