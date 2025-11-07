package discovery

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// PgPassEntry represents a line in .pgpass file
type PgPassEntry struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// ParsePgPass reads and parses .pgpass file
func ParsePgPass() ([]PgPassEntry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	pgpassPath := filepath.Join(home, ".pgpass")
	file, err := os.Open(pgpassPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []PgPassEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []PgPassEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry, err := parsePgPassLine(line)
		if err != nil {
			continue // Skip invalid lines
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// parsePgPassLine parses a single .pgpass line
// Format: hostname:port:database:username:password
func parsePgPassLine(line string) (PgPassEntry, error) {
	parts := strings.Split(line, ":")
	if len(parts) != 5 {
		return PgPassEntry{}, os.ErrInvalid
	}

	port := 5432
	if parts[1] != "*" {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	return PgPassEntry{
		Host:     parts[0],
		Port:     port,
		Database: parts[2],
		User:     parts[3],
		Password: parts[4],
	}, nil
}

// GetDiscoveredInstances converts pgpass entries to discovered instances
func GetDiscoveredInstances() []models.DiscoveredInstance {
	entries, err := ParsePgPass()
	if err != nil {
		return []models.DiscoveredInstance{}
	}

	instances := make([]models.DiscoveredInstance, 0, len(entries))
	seen := make(map[string]bool)

	for _, entry := range entries {
		// Skip wildcards for discovery
		if entry.Host == "*" {
			continue
		}

		key := entry.Host + ":" + strconv.Itoa(entry.Port)
		if seen[key] {
			continue
		}
		seen[key] = true

		instances = append(instances, models.DiscoveredInstance{
			Host:      entry.Host,
			Port:      entry.Port,
			Source:    models.SourcePgPass,
			Available: true,
		})
	}

	return instances
}

// FindPassword looks up password for a connection
func FindPassword(host string, port int, database, user string) string {
	entries, err := ParsePgPass()
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if matches(entry.Host, host) &&
			matches(strconv.Itoa(entry.Port), strconv.Itoa(port)) &&
			matches(entry.Database, database) &&
			matches(entry.User, user) {
			return entry.Password
		}
	}

	return ""
}

// matches checks if pattern matches value (* is wildcard)
func matches(pattern, value string) bool {
	return pattern == "*" || pattern == value
}
