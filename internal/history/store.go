package history

import (
	"database/sql"
	_ "embed"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// HistoryEntry represents a single query history entry
type HistoryEntry struct {
	ID             int
	ConnectionName string
	DatabaseName   string
	Query          string
	ExecutedAt     time.Time
	Duration       time.Duration
	RowsAffected   int64
	Success        bool
	ErrorMessage   string
}

// Store manages query history persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new history store
func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Create schema
	_, err = db.Exec(schemaSQL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Add adds a new query to history
func (s *Store) Add(entry HistoryEntry) error {
	_, err := s.db.Exec(`
		INSERT INTO query_history
		(connection_name, database_name, query, duration_ms, rows_affected, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ConnectionName,
		entry.DatabaseName,
		entry.Query,
		entry.Duration.Milliseconds(),
		entry.RowsAffected,
		entry.Success,
		entry.ErrorMessage,
	)
	return err
}

// GetRecent retrieves the most recent query history entries
func (s *Store) GetRecent(limit int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, connection_name, database_name, query, executed_at,
		       duration_ms, rows_affected, success, error_message
		FROM query_history
		ORDER BY executed_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var durationMs int64
		var executedAt string

		err := rows.Scan(
			&e.ID,
			&e.ConnectionName,
			&e.DatabaseName,
			&e.Query,
			&executedAt,
			&durationMs,
			&e.RowsAffected,
			&e.Success,
			&e.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}

		e.Duration = time.Duration(durationMs) * time.Millisecond
		e.ExecutedAt, _ = time.Parse("2006-01-02 15:04:05", executedAt)

		entries = append(entries, e)
	}

	return entries, nil
}

// Search searches query history by query text
func (s *Store) Search(query string, limit int) ([]HistoryEntry, error) {
	rows, err := s.db.Query(`
		SELECT id, connection_name, database_name, query, executed_at,
		       duration_ms, rows_affected, success, error_message
		FROM query_history
		WHERE query LIKE ?
		ORDER BY executed_at DESC
		LIMIT ?`, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		var durationMs int64
		var executedAt string

		err := rows.Scan(
			&e.ID,
			&e.ConnectionName,
			&e.DatabaseName,
			&e.Query,
			&executedAt,
			&durationMs,
			&e.RowsAffected,
			&e.Success,
			&e.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}

		e.Duration = time.Duration(durationMs) * time.Millisecond
		e.ExecutedAt, _ = time.Parse("2006-01-02 15:04:05", executedAt)

		entries = append(entries, e)
	}

	return entries, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
