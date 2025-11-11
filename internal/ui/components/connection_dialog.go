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
	HistoryMode         bool // true = browsing history, false = browsing discovered

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

	return &ConnectionDialog{
		inputs:      inputs,
		focusIndex:  0,
		cursorMode:  cursor.CursorBlink,
		Theme:       th,
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
		Foreground(lipgloss.Color("#cba6f7")).
		MarginBottom(1)
	sections = append(sections, titleStyle.Render("🔌 Connect to PostgreSQL"))

	// Tabs
	tabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Padding(0, 2)
	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cba6f7")).
		Background(lipgloss.Color("#45475a")).
		Bold(true).
		Padding(0, 2)

	var tabs []string
	if !c.HistoryMode {
		tabs = append(tabs, activeTabStyle.Render("📡 Discovered"))
		tabs = append(tabs, tabStyle.Render("📜 History"))
	} else {
		tabs = append(tabs, tabStyle.Render("📡 Discovered"))
		tabs = append(tabs, activeTabStyle.Render("📜 History"))
	}
	sections = append(sections, strings.Join(tabs, "  "))
	sections = append(sections, "")

	// Show either discovered instances or history based on mode
	if !c.HistoryMode {
		// Discovered instances
		if len(c.DiscoveredInstances) == 0 {
			loadingStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6adc8")).
				Italic(true)
			sections = append(sections, loadingStyle.Render("🔍 Discovering PostgreSQL instances..."))
		} else {
			for i, instance := range c.DiscoveredInstances {
				itemStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#cdd6f4")).
					PaddingLeft(2)

				if i == c.SelectedIndex {
					itemStyle = itemStyle.
						Foreground(lipgloss.Color("#1e1e2e")).
						Background(lipgloss.Color("#cba6f7")).
						Bold(true).
						PaddingLeft(1)
				}

				sourceStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6c7086")).
					Italic(true)

				line := fmt.Sprintf("%s:%d  %s",
					instance.Host,
					instance.Port,
					sourceStyle.Render(fmt.Sprintf("(%s)", instance.Source.String())),
				)
				sections = append(sections, itemStyle.Render(line))
			}
		}
	} else {
		// History entries
		if len(c.HistoryEntries) == 0 {
			emptyStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a6adc8")).
				Italic(true)
			sections = append(sections, emptyStyle.Render("No connection history yet"))
		} else {
			for i, entry := range c.HistoryEntries {
				itemStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#cdd6f4")).
					PaddingLeft(2)

				if i == c.SelectedIndex {
					itemStyle = itemStyle.
						Foreground(lipgloss.Color("#1e1e2e")).
						Background(lipgloss.Color("#cba6f7")).
						Bold(true).
						PaddingLeft(1)
				}

				metaStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6c7086")).
					Italic(true)

				line := fmt.Sprintf("%s  %s",
					entry.Name,
					metaStyle.Render(fmt.Sprintf("(used %dx)", entry.UsageCount)),
				)
				sections = append(sections, itemStyle.Render(line))
			}
		}
	}

	sections = append(sections, "")

	// Instructions (keep under 68 chars to fit MaxWidth(76) with padding)
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086"))
	sections = append(sections, helpStyle.Render("↑↓: Navigate │ Tab: Switch │ Enter: Connect │ m: Manual"))

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
	} else {
		// Get the list size based on current mode
		listSize := 0
		if c.HistoryMode {
			listSize = len(c.HistoryEntries)
		} else {
			listSize = len(c.DiscoveredInstances)
		}

		if listSize == 0 {
			c.SelectedIndex = 0
			return
		}
		c.SelectedIndex += delta
		if c.SelectedIndex < 0 {
			c.SelectedIndex = 0
		}
		if c.SelectedIndex >= listSize {
			c.SelectedIndex = listSize - 1
		}
	}
}

// SwitchTab switches between discovered and history tabs
func (c *ConnectionDialog) SwitchTab() {
	c.HistoryMode = !c.HistoryMode
	c.SelectedIndex = 0 // Reset selection when switching tabs
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
	if c.ManualMode || c.HistoryMode || c.SelectedIndex < 0 || c.SelectedIndex >= len(c.DiscoveredInstances) {
		return nil
	}
	return &c.DiscoveredInstances[c.SelectedIndex]
}

// GetSelectedHistory returns the currently selected history entry
func (c *ConnectionDialog) GetSelectedHistory() *models.ConnectionHistoryEntry {
	if c.ManualMode || !c.HistoryMode || c.SelectedIndex < 0 || c.SelectedIndex >= len(c.HistoryEntries) {
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
	if !c.HistoryMode && c.SelectedIndex >= len(instances) {
		c.SelectedIndex = 0
	}
}

// SetHistoryEntries updates the list of connection history entries
func (c *ConnectionDialog) SetHistoryEntries(entries []models.ConnectionHistoryEntry) {
	c.HistoryEntries = entries
	if c.HistoryMode && c.SelectedIndex >= len(entries) {
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
