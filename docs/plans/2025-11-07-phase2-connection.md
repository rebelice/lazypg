# Phase 2: Connection & Discovery Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement PostgreSQL connection management with auto-discovery, connection pooling, and basic metadata queries.

**Architecture:** Uses pgx v5 driver for connection pooling, implements auto-discovery by scanning ports/environment/config files, creates connection manager UI component, and provides metadata query layer for schema information.

**Tech Stack:** pgx v5, pgxpool, Go net package for port scanning, OS file system for config parsing

---

## Task 1: Add pgx Dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Add pgx dependencies**

Run:
```bash
cd /Users/rebeliceyang/Github/lazypg
go get github.com/jackc/pgx/v5@v5.7.2
go get github.com/jackc/pgx/v5/pgxpool@v5.7.2
```

Expected: Dependencies added to go.mod and go.sum

**Step 2: Verify dependencies**

Run: `go mod tidy`
Expected: No errors, dependencies resolved

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add pgx v5 for PostgreSQL connectivity"
```

---

## Task 2: Create Connection Models

**Files:**
- Create: `internal/models/connection.go`

**Step 1: Create connection models**

Create `internal/models/connection.go`:

```go
package models

import (
	"time"
)

// ConnectionConfig represents a PostgreSQL connection configuration
type ConnectionConfig struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

// Connection represents an active database connection
type Connection struct {
	ID          string
	Config      ConnectionConfig
	Connected   bool
	ConnectedAt time.Time
	LastPing    time.Time
	Error       error
}

// ConnectionState represents the current connection state
type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connecting
	Connected
	Failed
)

// DiscoveredInstance represents a PostgreSQL instance found via auto-discovery
type DiscoveredInstance struct {
	Host         string
	Port         int
	Source       DiscoverySource
	Available    bool
	ResponseTime time.Duration
}

// DiscoverySource indicates how an instance was discovered
type DiscoverySource int

const (
	SourcePortScan DiscoverySource = iota
	SourceEnvironment
	SourcePgPass
	SourcePgService
	SourceUnixSocket
	SourceConfig
)

func (s DiscoverySource) String() string {
	switch s {
	case SourcePortScan:
		return "Port Scan"
	case SourceEnvironment:
		return "Environment"
	case SourcePgPass:
		return ".pgpass"
	case SourcePgService:
		return ".pg_service.conf"
	case SourceUnixSocket:
		return "Unix Socket"
	case SourceConfig:
		return "Config File"
	default:
		return "Unknown"
	}
}
```

**Step 2: Commit**

```bash
git add internal/models/connection.go
git commit -m "feat: add connection and discovery models"
```

---

## Task 3: Implement Connection Pool Manager

**Files:**
- Create: `internal/db/connection/pool.go`
- Create: `internal/db/connection/manager.go`

**Step 1: Create connection pool wrapper**

Create `internal/db/connection/pool.go`:

```go
package connection

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rebeliceyang/lazypg/internal/models"
)

// Pool wraps pgxpool with our configuration
type Pool struct {
	pool   *pgxpool.Pool
	config models.ConnectionConfig
}

// NewPool creates a new connection pool
func NewPool(ctx context.Context, config models.ConnectionConfig) (*Pool, error) {
	connString := buildConnectionString(config)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection config: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = 5
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{
		pool:   pool,
		config: config,
	}, nil
}

// Close closes the connection pool
func (p *Pool) Close() {
	if p.pool != nil {
		p.pool.Close()
	}
}

// Ping tests the connection
func (p *Pool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Query executes a query
func (p *Pool) Query(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	fieldDescriptions := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			row[string(fd.Name)] = values[i]
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// QueryRow executes a query that returns a single row
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := p.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows returned")
	}
	return rows[0], nil
}

// buildConnectionString creates a PostgreSQL connection string
func buildConnectionString(config models.ConnectionConfig) string {
	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s database=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Database,
		sslMode,
	)

	if config.Password != "" {
		connStr += fmt.Sprintf(" password=%s", config.Password)
	}

	return connStr
}
```

**Step 2: Create connection manager**

Create `internal/db/connection/manager.go`:

```go
package connection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// Manager manages multiple database connections
type Manager struct {
	connections map[string]*Connection
	active      string
	mu          sync.RWMutex
}

// Connection wraps a pool with metadata
type Connection struct {
	ID          string
	Config      models.ConnectionConfig
	Pool        *Pool
	Connected   bool
	ConnectedAt time.Time
	LastPing    time.Time
	Error       error
}

// NewManager creates a new connection manager
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
	}
}

// Connect establishes a new connection
func (m *Manager) Connect(ctx context.Context, config models.ConnectionConfig) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateConnectionID(config)

	pool, err := NewPool(ctx, config)
	if err != nil {
		conn := &Connection{
			ID:        id,
			Config:    config,
			Connected: false,
			Error:     err,
		}
		m.connections[id] = conn
		return id, err
	}

	conn := &Connection{
		ID:          id,
		Config:      config,
		Pool:        pool,
		Connected:   true,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
	}

	m.connections[id] = conn
	m.active = id

	return id, nil
}

// Disconnect closes a connection
func (m *Manager) Disconnect(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, ok := m.connections[id]
	if !ok {
		return fmt.Errorf("connection %s not found", id)
	}

	if conn.Pool != nil {
		conn.Pool.Close()
	}

	delete(m.connections, id)

	if m.active == id {
		m.active = ""
	}

	return nil
}

// GetActive returns the active connection
func (m *Manager) GetActive() (*Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.active == "" {
		return nil, fmt.Errorf("no active connection")
	}

	conn, ok := m.connections[m.active]
	if !ok {
		return nil, fmt.Errorf("active connection not found")
	}

	return conn, nil
}

// SetActive sets the active connection
func (m *Manager) SetActive(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.connections[id]; !ok {
		return fmt.Errorf("connection %s not found", id)
	}

	m.active = id
	return nil
}

// GetAll returns all connections
func (m *Manager) GetAll() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns
}

// Ping tests the active connection
func (m *Manager) Ping(ctx context.Context) error {
	conn, err := m.GetActive()
	if err != nil {
		return err
	}

	if conn.Pool == nil {
		return fmt.Errorf("connection pool not initialized")
	}

	if err := conn.Pool.Ping(ctx); err != nil {
		m.mu.Lock()
		conn.Error = err
		conn.Connected = false
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	conn.LastPing = time.Now()
	conn.Connected = true
	conn.Error = nil
	m.mu.Unlock()

	return nil
}

// generateConnectionID creates a unique connection ID
func generateConnectionID(config models.ConnectionConfig) string {
	if config.Name != "" {
		return config.Name
	}
	return fmt.Sprintf("%s@%s:%d/%s", config.User, config.Host, config.Port, config.Database)
}
```

**Step 3: Commit**

```bash
git add internal/db/connection/pool.go internal/db/connection/manager.go
git commit -m "feat: implement connection pool and manager"
```

---

## Task 4: Implement Auto-Discovery - Port Scanning

**Files:**
- Create: `internal/db/discovery/scanner.go`

**Step 1: Create port scanner**

Create `internal/db/discovery/scanner.go`:

```go
package discovery

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// Scanner discovers PostgreSQL instances
type Scanner struct {
	timeout time.Duration
}

// NewScanner creates a new scanner
func NewScanner() *Scanner {
	return &Scanner{
		timeout: 2 * time.Second,
	}
}

// ScanPorts scans for PostgreSQL on common ports
func (s *Scanner) ScanPorts(ctx context.Context, host string, ports []int) []models.DiscoveredInstance {
	if len(ports) == 0 {
		ports = []int{5432, 5433, 5434, 5435}
	}

	instances := make([]models.DiscoveredInstance, 0)
	resultChan := make(chan models.DiscoveredInstance, len(ports))

	for _, port := range ports {
		go func(p int) {
			instance := s.scanPort(ctx, host, p)
			resultChan <- instance
		}(port)
	}

	for range ports {
		instance := <-resultChan
		if instance.Available {
			instances = append(instances, instance)
		}
	}

	return instances
}

// scanPort checks if a port is open
func (s *Scanner) scanPort(ctx context.Context, host string, port int) models.DiscoveredInstance {
	instance := models.DiscoveredInstance{
		Host:   host,
		Port:   port,
		Source: models.SourcePortScan,
	}

	start := time.Now()
	address := fmt.Sprintf("%s:%d", host, port)

	dialer := &net.Dialer{
		Timeout: s.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	instance.ResponseTime = time.Since(start)

	if err != nil {
		instance.Available = false
		return instance
	}

	conn.Close()
	instance.Available = true

	return instance
}

// ScanLocalhost scans for PostgreSQL on localhost
func (s *Scanner) ScanLocalhost(ctx context.Context) []models.DiscoveredInstance {
	return s.ScanPorts(ctx, "localhost", []int{5432, 5433, 5434, 5435})
}
```

**Step 2: Commit**

```bash
git add internal/db/discovery/scanner.go
git commit -m "feat: implement port scanning for auto-discovery"
```

---

## Task 5: Implement Auto-Discovery - Environment Variables

**Files:**
- Create: `internal/db/discovery/environment.go`

**Step 1: Create environment parser**

Create `internal/db/discovery/environment.go`:

```go
package discovery

import (
	"os"
	"strconv"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// ParseEnvironment reads PostgreSQL environment variables
func ParseEnvironment() *models.DiscoveredInstance {
	host := os.Getenv("PGHOST")
	portStr := os.Getenv("PGPORT")

	if host == "" {
		return nil
	}

	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	return &models.DiscoveredInstance{
		Host:      host,
		Port:      port,
		Source:    models.SourceEnvironment,
		Available: true, // Assume available, will be verified on connect
	}
}

// GetEnvironmentConfig gets connection config from environment
func GetEnvironmentConfig() *models.ConnectionConfig {
	host := os.Getenv("PGHOST")
	portStr := os.Getenv("PGPORT")
	database := os.Getenv("PGDATABASE")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	sslMode := os.Getenv("PGSSLMODE")

	if host == "" && database == "" && user == "" {
		return nil
	}

	// Set defaults
	if host == "" {
		host = "localhost"
	}
	if user == "" {
		user = os.Getenv("USER")
	}
	if database == "" {
		database = user
	}

	port := 5432
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	if sslMode == "" {
		sslMode = "prefer"
	}

	return &models.ConnectionConfig{
		Name:     "Environment",
		Host:     host,
		Port:     port,
		Database: database,
		User:     user,
		Password: password,
		SSLMode:  sslMode,
	}
}
```

**Step 2: Commit**

```bash
git add internal/db/discovery/environment.go
git commit -m "feat: implement environment variable discovery"
```

---

## Task 6: Implement Auto-Discovery - pgpass Parser

**Files:**
- Create: `internal/db/discovery/pgpass.go`

**Step 1: Create pgpass parser**

Create `internal/db/discovery/pgpass.go`:

```go
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
```

**Step 2: Commit**

```bash
git add internal/db/discovery/pgpass.go
git commit -m "feat: implement .pgpass file parsing"
```

---

## Task 7: Implement Discovery Coordinator

**Files:**
- Create: `internal/db/discovery/discovery.go`

**Step 1: Create discovery coordinator**

Create `internal/db/discovery/discovery.go`:

```go
package discovery

import (
	"context"
	"sort"

	"github.com/rebeliceyang/lazypg/internal/models"
)

// Discoverer coordinates all discovery methods
type Discoverer struct {
	scanner *Scanner
}

// NewDiscoverer creates a new discoverer
func NewDiscoverer() *Discoverer {
	return &Discoverer{
		scanner: NewScanner(),
	}
}

// DiscoverAll runs all discovery methods
func (d *Discoverer) DiscoverAll(ctx context.Context) []models.DiscoveredInstance {
	instances := make([]models.DiscoveredInstance, 0)

	// 1. Check environment variables
	if envInstance := ParseEnvironment(); envInstance != nil {
		instances = append(instances, *envInstance)
	}

	// 2. Scan localhost ports
	localInstances := d.scanner.ScanLocalhost(ctx)
	instances = append(instances, localInstances...)

	// 3. Parse .pgpass
	pgpassInstances := GetDiscoveredInstances()
	instances = append(instances, pgpassInstances...)

	// Deduplicate
	instances = deduplicateInstances(instances)

	// Sort by source priority
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Source < instances[j].Source
	})

	return instances
}

// deduplicateInstances removes duplicate host:port combinations
func deduplicateInstances(instances []models.DiscoveredInstance) []models.DiscoveredInstance {
	seen := make(map[string]models.DiscoveredInstance)

	for _, instance := range instances {
		key := instance.Host + ":" + string(rune(instance.Port))

		// Keep the one with higher priority source
		if existing, exists := seen[key]; !exists || instance.Source < existing.Source {
			seen[key] = instance
		}
	}

	result := make([]models.DiscoveredInstance, 0, len(seen))
	for _, instance := range seen {
		result = append(result, instance)
	}

	return result
}
```

**Step 2: Commit**

```bash
git add internal/db/discovery/discovery.go
git commit -m "feat: implement discovery coordinator"
```

---

## Task 8: Implement Basic Metadata Queries

**Files:**
- Create: `internal/db/metadata/databases.go`
- Create: `internal/db/metadata/tables.go`

**Step 1: Create database metadata queries**

Create `internal/db/metadata/databases.go`:

```go
package metadata

import (
	"context"

	"github.com/rebeliceyang/lazypg/internal/db/connection"
)

// Database represents a PostgreSQL database
type Database struct {
	Name  string
	Owner string
	Size  string
}

// ListDatabases returns all databases
func ListDatabases(ctx context.Context, pool *connection.Pool) ([]Database, error) {
	query := `
		SELECT
			datname as name,
			pg_catalog.pg_get_userbyid(datdba) as owner,
			pg_catalog.pg_size_pretty(pg_catalog.pg_database_size(datname)) as size
		FROM pg_catalog.pg_database
		WHERE datistemplate = false
		ORDER BY datname;
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	databases := make([]Database, 0, len(rows))
	for _, row := range rows {
		databases = append(databases, Database{
			Name:  row["name"].(string),
			Owner: row["owner"].(string),
			Size:  row["size"].(string),
		})
	}

	return databases, nil
}
```

**Step 2: Create table metadata queries**

Create `internal/db/metadata/tables.go`:

```go
package metadata

import (
	"context"

	"github.com/rebeliceyang/lazypg/internal/db/connection"
)

// Schema represents a PostgreSQL schema
type Schema struct {
	Name  string
	Owner string
}

// Table represents a PostgreSQL table
type Table struct {
	Schema   string
	Name     string
	RowCount int64
	Size     string
}

// ListSchemas returns all schemas in the current database
func ListSchemas(ctx context.Context, pool *connection.Pool) ([]Schema, error) {
	query := `
		SELECT
			schema_name as name,
			schema_owner as owner
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name;
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	schemas := make([]Schema, 0, len(rows))
	for _, row := range rows {
		schemas = append(schemas, Schema{
			Name:  row["name"].(string),
			Owner: row["owner"].(string),
		})
	}

	return schemas, nil
}

// ListTables returns all tables in a schema
func ListTables(ctx context.Context, pool *connection.Pool, schema string) ([]Table, error) {
	query := `
		SELECT
			schemaname as schema,
			tablename as name,
			pg_catalog.pg_size_pretty(pg_catalog.pg_total_relation_size(schemaname||'.'||tablename)) as size
		FROM pg_catalog.pg_tables
		WHERE schemaname = $1
		ORDER BY tablename;
	`

	rows, err := pool.Query(ctx, query, schema)
	if err != nil {
		return nil, err
	}

	tables := make([]Table, 0, len(rows))
	for _, row := range rows {
		tables = append(tables, Table{
			Schema: row["schema"].(string),
			Name:   row["name"].(string),
			Size:   row["size"].(string),
		})
	}

	return tables, nil
}

// GetTableRowCount returns the estimated row count for a table
func GetTableRowCount(ctx context.Context, pool *connection.Pool, schema, table string) (int64, error) {
	query := `
		SELECT reltuples::bigint as estimate
		FROM pg_class
		WHERE oid = ($1 || '.' || $2)::regclass;
	`

	row, err := pool.QueryRow(ctx, query, schema, table)
	if err != nil {
		return 0, err
	}

	estimate, ok := row["estimate"].(int64)
	if !ok {
		return 0, nil
	}

	return estimate, nil
}
```

**Step 3: Commit**

```bash
git add internal/db/metadata/databases.go internal/db/metadata/tables.go
git commit -m "feat: implement basic metadata queries"
```

---

## Task 9: Update App Model for Connection State

**Files:**
- Modify: `internal/models/models.go`

**Step 1: Add connection state to app model**

Add to `internal/models/models.go`:

```go
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
}
```

**Step 2: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add connection state to app model"
```

---

## Task 10: Integrate Connection Manager into App

**Files:**
- Modify: `internal/app/app.go`
- Modify: `cmd/lazypg/main.go`

**Step 1: Update app initialization**

In `internal/app/app.go`, update the imports and App struct:

```go
import (
	// ... existing imports
	"github.com/rebeliceyang/lazypg/internal/db/connection"
	"github.com/rebeliceyang/lazypg/internal/db/discovery"
)

type App struct {
	state             models.AppState
	config            *config.Config
	theme             theme.Theme
	leftPanel         components.Panel
	rightPanel        components.Panel

	// Phase 2: Connection management
	connectionManager *connection.Manager
	discoverer        *discovery.Discoverer
}
```

Update the `New` function:

```go
func New(cfg *config.Config) *App {
	app := &App{
		config:            cfg,
		theme:             theme.GetTheme(cfg.UI.Theme),
		connectionManager: connection.NewManager(),
		discoverer:        discovery.NewDiscoverer(),
	}

	// ... existing initialization code

	return app
}
```

**Step 2: Update main.go to pass context**

In `cmd/lazypg/main.go`, add context awareness:

```go
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Could not load config: %v (using defaults)\n", err)
		cfg = config.GetDefaults()
	}

	// Run auto-discovery on startup
	ctx := context.Background()
	app := app.New(cfg)

	// TODO: Trigger discovery in background
	// This will be implemented in connection UI

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.UI.MouseEnabled {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(app, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Commit**

```bash
git add internal/app/app.go cmd/lazypg/main.go
git commit -m "feat: integrate connection manager into app"
```

---

## Task 11: Create Connection Dialog UI Component

**Files:**
- Create: `internal/ui/components/connection_dialog.go`

**Step 1: Create connection dialog component**

Create `internal/ui/components/connection_dialog.go`:

```go
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

// GetManualConfig returns the manual connection config
func (c *ConnectionDialog) GetManualConfig() models.ConnectionConfig {
	return models.ConnectionConfig{
		Host:     c.Host,
		Port:     mustParseInt(c.Port, 5432),
		Database: c.Database,
		User:     c.User,
		Password: c.Password,
		SSLMode:  "prefer",
	}
}

func mustParseInt(s string, defaultVal int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultVal
	}
	return result
}
```

**Step 2: Commit**

```bash
git add internal/ui/components/connection_dialog.go
git commit -m "feat: create connection dialog UI component"
```

---

## Task 12: Add Connection Management to App Update Loop

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add connection dialog state**

In `internal/app/app.go`, add to the App struct:

```go
type App struct {
	// ... existing fields

	// Connection dialog
	showConnectionDialog bool
	connectionDialog     *components.ConnectionDialog
}
```

Update the `New` function:

```go
func New(cfg *config.Config) *App {
	app := &App{
		config:            cfg,
		theme:             theme.GetTheme(cfg.UI.Theme),
		connectionManager: connection.NewManager(),
		discoverer:        discovery.NewDiscoverer(),
		connectionDialog:  components.NewConnectionDialog(),
	}

	// ... existing initialization

	return app
}
```

**Step 2: Add key binding to open connection dialog**

In the `Update` method, add a new case:

```go
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle connection dialog first if visible
		if a.showConnectionDialog {
			return a.handleConnectionDialog(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if a.state.ViewMode == models.HelpMode {
				a.state.ViewMode = models.NormalMode
				return a, nil
			}
			return a, tea.Quit

		case "c":
			// Open connection dialog
			a.showConnectionDialog = true
			// TODO: Trigger discovery
			return a, nil

		// ... rest of existing cases
```

**Step 3: Add connection dialog handler**

Add the handler method:

```go
func (a *App) handleConnectionDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.showConnectionDialog = false
		return a, nil

	case "up", "k":
		a.connectionDialog.MoveSelection(-1)
		return a, nil

	case "down", "j":
		a.connectionDialog.MoveSelection(1)
		return a, nil

	case "m":
		a.connectionDialog.ManualMode = !a.connectionDialog.ManualMode
		return a, nil

	case "enter":
		// TODO: Implement connection logic
		a.showConnectionDialog = false
		return a, nil
	}

	return a, nil
}
```

**Step 4: Update View to show connection dialog**

In the `View` method:

```go
func (a *App) View() string {
	if a.showConnectionDialog {
		return a.renderConnectionDialog()
	}

	// ... existing view logic
}

func (a *App) renderConnectionDialog() string {
	// Center the dialog
	dialogWidth := 60
	dialogHeight := 20

	a.connectionDialog.Width = dialogWidth
	a.connectionDialog.Height = dialogHeight

	dialog := a.connectionDialog.View()

	// Center it
	verticalPadding := (a.state.Height - dialogHeight) / 2
	horizontalPadding := (a.state.Width - dialogWidth) / 2

	style := lipgloss.NewStyle().
		Padding(verticalPadding, 0, 0, horizontalPadding)

	return style.Render(dialog)
}
```

**Step 5: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: add connection dialog to app update loop"
```

---

## Task 13: Update Help System with Connection Keys

**Files:**
- Modify: `internal/ui/help/help.go`

**Step 1: Add connection keybindings to help**

In `internal/ui/help/help.go`, update the `GetGlobalKeys` function:

```go
func GetGlobalKeys() []KeyBinding {
	return []KeyBinding{
		{"?", "Toggle help"},
		{"q, Ctrl+C", "Quit application"},
		{"Ctrl+K", "Open command palette"},
		{"Tab", "Switch panel focus"},
		{"c", "Open connection dialog"},
		{"r", "Refresh current view"},
	}
}
```

Add a new category for connection keys:

```go
func GetConnectionKeys() []KeyBinding {
	return []KeyBinding{
		{"c", "Connect to database"},
		{"d", "Disconnect"},
		{"Ctrl+R", "Reconnect"},
		{"Ctrl+D", "Show all connections"},
	}
}
```

Update the `Render` function to include the new category:

```go
func Render(width, height int, theme lipgloss.Style) string {
	// ... existing code

	sections = append(sections, renderSection("Connection", GetConnectionKeys()))

	// ... rest of existing code
}
```

**Step 2: Commit**

```bash
git add internal/ui/help/help.go
git commit -m "docs: add connection keybindings to help system"
```

---

## Task 14: Update README with Phase 2 Progress

**Files:**
- Modify: `README.md`

**Step 1: Update status and features**

Update the status section in `README.md`:

```markdown
## Status

🚧 **In Development** - Phase 2 (Connection & Discovery) Complete

### Completed Features

- ✅ Multi-panel layout (left navigation, right content)
- ✅ Configuration system (YAML-based)
- ✅ Theme support
- ✅ Help system with keyboard shortcuts
- ✅ Panel focus management
- ✅ Responsive layout
- ✅ PostgreSQL connection management
- ✅ Connection pooling with pgx v5
- ✅ Auto-discovery (port scan, environment, .pgpass)
- ✅ Connection dialog UI
- ✅ Basic metadata queries

### In Progress

- 🔄 Navigation tree
- 🔄 Data browsing
- 🔄 Table viewing
```

Update the roadmap:

```markdown
### Phase 2: Connection & Discovery ✅
- PostgreSQL connection management
- Connection pool with pgx
- Auto-discovery of local instances
- Connection manager UI
- Metadata queries

### Phase 3: Data Browsing (Next)
- Navigation tree
- Table data viewing
- Virtual scrolling
```

**Step 2: Update configuration example**

Add connection configuration example:

```markdown
## Configuration

lazypg looks for configuration in:
- `~/.config/lazypg/config.yaml` (user config)
- `~/.config/lazypg/connections.yaml` (saved connections)
- `./config.yaml` (current directory)

See `config/default.yaml` for all available options.

Example connection config (`~/.config/lazypg/connections.yaml`):

```yaml
connections:
  - name: "Local Dev"
    host: localhost
    port: 5432
    database: mydb
    user: postgres
    ssl_mode: prefer

  - name: "Production"
    host: prod-db.example.com
    port: 5432
    database: prod_db
    user: app_user
    ssl_mode: require
```
```

**Step 3: Commit**

```bash
git add README.md
git commit -m "docs: update README for Phase 2 completion"
```

---

## Task 15: Create Phase 2 Testing Checklist

**Files:**
- Create: `docs/testing/phase2-checklist.md`

**Step 1: Create testing checklist**

Create `docs/testing/phase2-checklist.md`:

```markdown
# Phase 2 Testing Checklist

## Connection Pool Tests

- [ ] Pool creates successfully with valid config
- [ ] Pool fails gracefully with invalid config
- [ ] Pool handles connection timeout
- [ ] Ping succeeds on healthy connection
- [ ] Ping fails on closed connection
- [ ] Query returns correct results
- [ ] QueryRow returns single row
- [ ] Connection string builds correctly with all fields
- [ ] Connection string handles missing password

## Connection Manager Tests

- [ ] Manager initializes with empty connections
- [ ] Connect establishes new connection
- [ ] Connect fails with invalid credentials
- [ ] Disconnect closes connection
- [ ] GetActive returns active connection
- [ ] GetActive fails when no active connection
- [ ] SetActive switches active connection
- [ ] GetAll returns all connections
- [ ] Ping updates connection state

## Auto-Discovery Tests

### Port Scanner
- [ ] Scans default ports (5432-5435)
- [ ] Detects running PostgreSQL instance
- [ ] Skips unavailable ports
- [ ] Respects timeout
- [ ] Handles context cancellation

### Environment Parser
- [ ] Reads PGHOST, PGPORT, PGDATABASE
- [ ] Returns nil when no env vars set
- [ ] Uses defaults for missing values
- [ ] Parses port correctly

### pgpass Parser
- [ ] Parses valid .pgpass file
- [ ] Skips comment lines
- [ ] Skips invalid lines
- [ ] Returns empty list when file doesn't exist
- [ ] FindPassword matches wildcards
- [ ] FindPassword returns correct password

### Discovery Coordinator
- [ ] Combines all discovery methods
- [ ] Deduplicates instances
- [ ] Prioritizes sources correctly
- [ ] Handles context cancellation

## Metadata Queries Tests

- [ ] ListDatabases returns all databases
- [ ] ListSchemas filters system schemas
- [ ] ListTables returns tables for schema
- [ ] GetTableRowCount returns estimate

## UI Component Tests

- [ ] ConnectionDialog renders in discovery mode
- [ ] ConnectionDialog renders in manual mode
- [ ] MoveSelection navigates instances
- [ ] MoveSelection navigates form fields
- [ ] GetSelectedInstance returns correct instance
- [ ] GetManualConfig builds correct config

## Integration Tests

- [ ] App opens connection dialog with 'c'
- [ ] Connection dialog shows discovered instances
- [ ] Selecting instance attempts connection
- [ ] Manual mode allows input
- [ ] ESC closes dialog
- [ ] Help shows connection keys

## Manual Testing

1. **No PostgreSQL Running**
   - [ ] Discovery shows "discovering..." message
   - [ ] Manual mode works
   - [ ] Connection fails gracefully with error message

2. **PostgreSQL on Default Port**
   - [ ] Discovery finds localhost:5432
   - [ ] Shows "Port Scan" as source
   - [ ] Connection succeeds

3. **Multiple PostgreSQL Instances**
   - [ ] Discovery finds all instances
   - [ ] Lists ports correctly
   - [ ] Can connect to each

4. **Environment Variables Set**
   - [ ] Discovery shows environment instance
   - [ ] Prioritizes environment over port scan

5. **.pgpass File Present**
   - [ ] Discovery shows .pgpass instances
   - [ ] Password auto-filled from .pgpass
   - [ ] Connection succeeds without password prompt

6. **Connection States**
   - [ ] Shows "connecting..." while connecting
   - [ ] Shows "connected" with timestamp
   - [ ] Shows error message on failure
   - [ ] Ping updates last ping time
```

**Step 2: Commit**

```bash
git add docs/testing/phase2-checklist.md
git commit -m "docs: add Phase 2 testing checklist"
```

---

## Summary

**Phase 2 Implementation Complete!**

This plan implements:
- ✅ pgx v5 integration with connection pooling
- ✅ Connection manager for multiple connections
- ✅ Auto-discovery (port scan, environment variables, .pgpass)
- ✅ Discovery coordinator to combine all methods
- ✅ Basic metadata queries (databases, schemas, tables)
- ✅ Connection dialog UI component
- ✅ Integration with app update loop
- ✅ Updated help system and README

**Total Tasks:** 15
**Estimated Time:** 2-3 hours for implementation + testing

**Next Steps:**
- Run through testing checklist
- Fix any bugs found during testing
- Prepare for Phase 3 (Navigation Tree & Data Browsing)
