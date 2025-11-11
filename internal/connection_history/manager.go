package connection_history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rebeliceyang/lazypg/internal/models"
	"gopkg.in/yaml.v3"
)

// Manager manages connection history
type Manager struct {
	path          string
	history       []models.ConnectionHistoryEntry
	passwordStore *PasswordStore
}

// NewManager creates a new connection history manager
func NewManager(configDir string) (*Manager, error) {
	path := filepath.Join(configDir, "connection_history.yaml")

	m := &Manager{
		path:          path,
		history:       []models.ConnectionHistoryEntry{},
		passwordStore: NewPasswordStore(),
	}

	// Load existing history if file exists
	if _, err := os.Stat(path); err == nil {
		if err := m.Load(); err != nil {
			return nil, fmt.Errorf("failed to load connection history: %w", err)
		}
	}

	return m, nil
}

// Load loads connection history from YAML file
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("failed to read connection history file: %w", err)
	}

	if err := yaml.Unmarshal(data, &m.history); err != nil {
		return fmt.Errorf("failed to parse connection history: %w", err)
	}

	return nil
}

// Save saves connection history to YAML file
func (m *Manager) Save() error {
	data, err := yaml.Marshal(m.history)
	if err != nil {
		return fmt.Errorf("failed to marshal connection history: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(m.path, data, 0600); err != nil { // 0600 for security (connection info)
		return fmt.Errorf("failed to write connection history file: %w", err)
	}

	return nil
}

// Add adds or updates a connection in history
func (m *Manager) Add(config models.ConnectionConfig) error {
	// Save password to secure keyring (if provided)
	if config.Password != "" && m.passwordStore != nil {
		if err := m.passwordStore.Save(config.Host, config.Port, config.Database, config.User, config.Password); err != nil {
			// Log error but don't fail - password storage is optional
			fmt.Printf("Warning: Failed to save password to keyring: %v\n", err)
		}
	}

	// Check if this connection already exists (match by host, port, database, user)
	for i, entry := range m.history {
		if entry.Host == config.Host &&
			entry.Port == config.Port &&
			entry.Database == config.Database &&
			entry.User == config.User {
			// Update existing entry
			m.history[i].LastUsed = time.Now()
			m.history[i].UsageCount++
			m.history[i].SSLMode = config.SSLMode
			// Update name if config has one
			if config.Name != "" {
				m.history[i].Name = config.Name
			}
			return m.Save()
		}
	}

	// Create new entry
	name := config.Name
	if name == "" {
		name = fmt.Sprintf("%s@%s:%d/%s", config.User, config.Host, config.Port, config.Database)
	}

	entry := models.ConnectionHistoryEntry{
		ID:         uuid.New().String(),
		Name:       name,
		Host:       config.Host,
		Port:       config.Port,
		Database:   config.Database,
		User:       config.User,
		SSLMode:    config.SSLMode,
		LastUsed:   time.Now(),
		UsageCount: 1,
		CreatedAt:  time.Now(),
	}

	m.history = append(m.history, entry)

	return m.Save()
}

// GetAll returns all connection history entries
func (m *Manager) GetAll() []models.ConnectionHistoryEntry {
	return m.history
}

// GetRecent returns the most recently used connections
func (m *Manager) GetRecent(limit int) []models.ConnectionHistoryEntry {
	sorted := make([]models.ConnectionHistoryEntry, len(m.history))
	copy(sorted, m.history)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LastUsed.After(sorted[j].LastUsed)
	})

	if limit > 0 && limit < len(sorted) {
		sorted = sorted[:limit]
	}

	return sorted
}

// GetMostUsed returns the most frequently used connections
func (m *Manager) GetMostUsed(limit int) []models.ConnectionHistoryEntry {
	sorted := make([]models.ConnectionHistoryEntry, len(m.history))
	copy(sorted, m.history)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UsageCount > sorted[j].UsageCount
	})

	if limit > 0 && limit < len(sorted) {
		sorted = sorted[:limit]
	}

	return sorted
}

// Delete removes a connection from history by ID
func (m *Manager) Delete(id string) error {
	for i, entry := range m.history {
		if entry.ID == id {
			// Also delete password from keyring
			if m.passwordStore != nil {
				_ = m.passwordStore.Delete(entry.Host, entry.Port, entry.Database, entry.User)
			}
			m.history = append(m.history[:i], m.history[i+1:]...)
			return m.Save()
		}
	}
	return fmt.Errorf("connection history entry with ID '%s' not found", id)
}

// GetConnectionConfigWithPassword returns a ConnectionConfig with password retrieved from keyring
func (m *Manager) GetConnectionConfigWithPassword(entry *models.ConnectionHistoryEntry) models.ConnectionConfig {
	config := entry.ToConnectionConfig()

	// Try to get password from keyring
	if m.passwordStore != nil {
		password, err := m.passwordStore.Get(entry.Host, entry.Port, entry.Database, entry.User)
		if err == nil {
			config.Password = password
		}
	}

	return config
}
