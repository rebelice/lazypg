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
