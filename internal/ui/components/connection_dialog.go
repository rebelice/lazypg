package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/models"
)

// ConnectionDialog represents a connection dialog
type ConnectionDialog struct {
	Width              int
	Height             int
	Style              lipgloss.Style
	DiscoveredInstances []models.DiscoveredInstance
	ManualMode         bool
	SelectedIndex      int

	// Manual connection fields
	Host     string
	Port     string
	Database string
	User     string
	Password string
	ActiveField int
}

// NewConnectionDialog creates a new connection dialog
func NewConnectionDialog() *ConnectionDialog {
	return &ConnectionDialog{
		Port:        "5432",
		ActiveField: 0,
	}
}

// View renders the connection dialog
func (c *ConnectionDialog) View() string {
	if c.Width <= 0 || c.Height <= 0 {
		return ""
	}

	var content strings.Builder

	if c.ManualMode {
		content.WriteString(c.renderManualMode())
	} else {
		content.WriteString(c.renderDiscoveryMode())
	}

	style := c.Style.
		Width(c.Width).
		Height(c.Height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return style.Render(content.String())
}

func (c *ConnectionDialog) renderDiscoveryMode() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	b.WriteString(titleStyle.Render("Connect to PostgreSQL"))
	b.WriteString("\n\n")

	if len(c.DiscoveredInstances) == 0 {
		b.WriteString("Discovering PostgreSQL instances...\n")
		b.WriteString("\n")
		b.WriteString("Press 'm' for manual connection\n")
		return b.String()
	}

	b.WriteString("Discovered instances:\n\n")

	for i, instance := range c.DiscoveredInstances {
		prefix := "  "
		if i == c.SelectedIndex {
			prefix = "> "
		}

		b.WriteString(fmt.Sprintf("%s%s:%d (%s)\n",
			prefix,
			instance.Host,
			instance.Port,
			instance.Source.String(),
		))
	}

	b.WriteString("\n")
	b.WriteString("↑/↓: Select | Enter: Connect | m: Manual | Esc: Cancel\n")

	return b.String()
}

func (c *ConnectionDialog) renderManualMode() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	b.WriteString(titleStyle.Render("Manual Connection"))
	b.WriteString("\n\n")

	fields := []struct {
		label string
		value string
		index int
	}{
		{"Host:", c.Host, 0},
		{"Port:", c.Port, 1},
		{"Database:", c.Database, 2},
		{"User:", c.User, 3},
		{"Password:", strings.Repeat("*", len(c.Password)), 4},
	}

	for _, field := range fields {
		prefix := "  "
		if field.index == c.ActiveField {
			prefix = "> "
		}
		b.WriteString(fmt.Sprintf("%s%-10s %s\n", prefix, field.label, field.value))
	}

	b.WriteString("\n")
	b.WriteString("↑/↓: Navigate | Type to edit | Enter: Connect | Esc: Cancel\n")

	return b.String()
}

// HandleInput processes text input for the active field in manual mode
func (c *ConnectionDialog) HandleInput(char rune) {
	if !c.ManualMode {
		return
	}

	switch c.ActiveField {
	case 0:
		c.Host += string(char)
	case 1:
		c.Port += string(char)
	case 2:
		c.Database += string(char)
	case 3:
		c.User += string(char)
	case 4:
		c.Password += string(char)
	}
}

// HandleBackspace removes the last character from the active field
func (c *ConnectionDialog) HandleBackspace() {
	if !c.ManualMode {
		return
	}

	var field *string
	switch c.ActiveField {
	case 0:
		field = &c.Host
	case 1:
		field = &c.Port
	case 2:
		field = &c.Database
	case 3:
		field = &c.User
	case 4:
		field = &c.Password
	default:
		return
	}

	if len(*field) > 0 {
		*field = (*field)[:len(*field)-1]
	}
}

// MoveSelection moves the selection up or down
func (c *ConnectionDialog) MoveSelection(delta int) {
	if c.ManualMode {
		c.ActiveField += delta
		if c.ActiveField < 0 {
			c.ActiveField = 4
		}
		if c.ActiveField > 4 {
			c.ActiveField = 0
		}
	} else {
		if len(c.DiscoveredInstances) == 0 {
			c.SelectedIndex = 0
			return
		}
		c.SelectedIndex += delta
		if c.SelectedIndex < 0 {
			c.SelectedIndex = 0
		}
		if c.SelectedIndex >= len(c.DiscoveredInstances) {
			c.SelectedIndex = len(c.DiscoveredInstances) - 1
		}
	}
}

// GetSelectedInstance returns the currently selected instance
func (c *ConnectionDialog) GetSelectedInstance() *models.DiscoveredInstance {
	if c.ManualMode || c.SelectedIndex < 0 || c.SelectedIndex >= len(c.DiscoveredInstances) {
		return nil
	}
	return &c.DiscoveredInstances[c.SelectedIndex]
}

// GetManualConfig returns the manual connection config if valid, or error
func (c *ConnectionDialog) GetManualConfig() (models.ConnectionConfig, error) {
	if c.Host == "" {
		return models.ConnectionConfig{}, fmt.Errorf("host is required")
	}
	if c.User == "" {
		return models.ConnectionConfig{}, fmt.Errorf("user is required")
	}
	if c.Database == "" {
		return models.ConnectionConfig{}, fmt.Errorf("database is required")
	}

	return models.ConnectionConfig{
		Host:     c.Host,
		Port:     mustParseInt(c.Port, 5432),
		Database: c.Database,
		User:     c.User,
		Password: c.Password,
		SSLMode:  "prefer",
	}, nil
}

func mustParseInt(s string, defaultVal int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultVal
	}
	return result
}
