package session

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLStore is a SQL-backed session store.
// It works with any database/sql compatible driver (PostgreSQL, MySQL, SQLite).
// Requires a table with schema:
//
//	CREATE TABLE vango_sessions (
//	    id VARCHAR(64) PRIMARY KEY,
//	    data BYTEA NOT NULL,
//	    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
//	    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
//	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
//	);
//	CREATE INDEX idx_vango_sessions_expires ON vango_sessions(expires_at);
type SQLStore struct {
	db              *sql.DB
	tableName       string
	dialect         SQLDialect
	cleanupInterval time.Duration
	closed          bool
	done            chan struct{}
}

// SQLDialect represents the SQL dialect for query generation.
type SQLDialect int

const (
	// DialectPostgreSQL uses PostgreSQL syntax ($1, $2 placeholders).
	DialectPostgreSQL SQLDialect = iota
	// DialectMySQL uses MySQL syntax (? placeholders).
	DialectMySQL
	// DialectSQLite uses SQLite syntax (? placeholders).
	DialectSQLite
)

// SQLStoreOption configures SQLStore behavior.
type SQLStoreOption func(*sqlStoreConfig)

type sqlStoreConfig struct {
	tableName       string
	dialect         SQLDialect
	cleanupInterval time.Duration
}

// WithSQLTableName sets the table name for session storage.
// Default: "vango_sessions".
func WithSQLTableName(name string) SQLStoreOption {
	return func(c *sqlStoreConfig) {
		c.tableName = name
	}
}

// WithSQLDialect sets the SQL dialect for query generation.
// Default: DialectPostgreSQL.
func WithSQLDialect(dialect SQLDialect) SQLStoreOption {
	return func(c *sqlStoreConfig) {
		c.dialect = dialect
	}
}

// WithSQLCleanupInterval sets how often expired sessions are cleaned up.
// Default: 5 minutes.
func WithSQLCleanupInterval(d time.Duration) SQLStoreOption {
	return func(c *sqlStoreConfig) {
		c.cleanupInterval = d
	}
}

// NewSQLStore creates a new SQL-backed session store.
func NewSQLStore(db *sql.DB, opts ...SQLStoreOption) *SQLStore {
	cfg := &sqlStoreConfig{
		tableName:       "vango_sessions",
		dialect:         DialectPostgreSQL,
		cleanupInterval: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	store := &SQLStore{
		db:              db,
		tableName:       cfg.tableName,
		dialect:         cfg.dialect,
		cleanupInterval: cfg.cleanupInterval,
		done:            make(chan struct{}),
	}

	go store.cleanupLoop()
	return store
}

// placeholder returns the placeholder syntax for the dialect.
func (s *SQLStore) placeholder(n int) string {
	switch s.dialect {
	case DialectPostgreSQL:
		return fmt.Sprintf("$%d", n)
	default:
		return "?"
	}
}

// Save stores session data with an expiration time.
func (s *SQLStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
	if s.closed {
		return ErrStoreClosed{}
	}

	var query string
	switch s.dialect {
	case DialectPostgreSQL:
		query = fmt.Sprintf(`
			INSERT INTO %s (id, data, expires_at, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (id) DO UPDATE SET
				data = EXCLUDED.data,
				expires_at = EXCLUDED.expires_at,
				updated_at = NOW()
		`, s.tableName)
	case DialectMySQL:
		query = fmt.Sprintf(`
			INSERT INTO %s (id, data, expires_at, updated_at)
			VALUES (?, ?, ?, NOW())
			ON DUPLICATE KEY UPDATE
				data = VALUES(data),
				expires_at = VALUES(expires_at),
				updated_at = NOW()
		`, s.tableName)
	case DialectSQLite:
		query = fmt.Sprintf(`
			INSERT OR REPLACE INTO %s (id, data, expires_at, updated_at)
			VALUES (?, ?, ?, datetime('now'))
		`, s.tableName)
	}

	_, err := s.db.ExecContext(ctx, query, sessionID, data, expiresAt)
	return err
}

// Load retrieves session data if it exists and hasn't expired.
func (s *SQLStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
	if s.closed {
		return nil, ErrStoreClosed{}
	}

	var query string
	switch s.dialect {
	case DialectPostgreSQL:
		query = fmt.Sprintf(`
			SELECT data FROM %s
			WHERE id = $1 AND expires_at > NOW()
		`, s.tableName)
	case DialectMySQL:
		query = fmt.Sprintf(`
			SELECT data FROM %s
			WHERE id = ? AND expires_at > NOW()
		`, s.tableName)
	case DialectSQLite:
		query = fmt.Sprintf(`
			SELECT data FROM %s
			WHERE id = ? AND expires_at > datetime('now')
		`, s.tableName)
	}

	var data []byte
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return data, nil
}

// Delete removes a session from the database.
func (s *SQLStore) Delete(ctx context.Context, sessionID string) error {
	if s.closed {
		return ErrStoreClosed{}
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE id = %s`, s.tableName, s.placeholder(1))
	_, err := s.db.ExecContext(ctx, query, sessionID)
	return err
}

// Touch updates the expiration time for a session.
func (s *SQLStore) Touch(ctx context.Context, sessionID string, expiresAt time.Time) error {
	if s.closed {
		return ErrStoreClosed{}
	}

	var query string
	switch s.dialect {
	case DialectPostgreSQL:
		query = fmt.Sprintf(`
			UPDATE %s SET expires_at = $1, updated_at = NOW()
			WHERE id = $2
		`, s.tableName)
	case DialectMySQL:
		query = fmt.Sprintf(`
			UPDATE %s SET expires_at = ?, updated_at = NOW()
			WHERE id = ?
		`, s.tableName)
	case DialectSQLite:
		query = fmt.Sprintf(`
			UPDATE %s SET expires_at = ?, updated_at = datetime('now')
			WHERE id = ?
		`, s.tableName)
	}

	_, err := s.db.ExecContext(ctx, query, expiresAt, sessionID)
	return err
}

// SaveAll saves multiple sessions using a transaction.
func (s *SQLStore) SaveAll(ctx context.Context, sessions map[string]SessionData) error {
	if s.closed {
		return ErrStoreClosed{}
	}

	if len(sessions) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var query string
	switch s.dialect {
	case DialectPostgreSQL:
		query = fmt.Sprintf(`
			INSERT INTO %s (id, data, expires_at, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (id) DO UPDATE SET
				data = EXCLUDED.data,
				expires_at = EXCLUDED.expires_at,
				updated_at = NOW()
		`, s.tableName)
	case DialectMySQL:
		query = fmt.Sprintf(`
			INSERT INTO %s (id, data, expires_at, updated_at)
			VALUES (?, ?, ?, NOW())
			ON DUPLICATE KEY UPDATE
				data = VALUES(data),
				expires_at = VALUES(expires_at),
				updated_at = NOW()
		`, s.tableName)
	case DialectSQLite:
		query = fmt.Sprintf(`
			INSERT OR REPLACE INTO %s (id, data, expires_at, updated_at)
			VALUES (?, ?, ?, datetime('now'))
		`, s.tableName)
	}

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for id, sd := range sessions {
		if _, err := stmt.ExecContext(ctx, id, sd.Data, sd.ExpiresAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Close shuts down the store and releases resources.
// Note: This does not close the underlying database connection,
// as it may be shared with other components.
func (s *SQLStore) Close() error {
	if s.closed {
		return nil
	}

	s.closed = true
	close(s.done)
	return nil
}

// cleanupLoop periodically removes expired sessions.
func (s *SQLStore) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.done:
			return
		}
	}
}

// cleanup removes expired sessions from the database.
func (s *SQLStore) cleanup() {
	if s.closed {
		return
	}

	var query string
	switch s.dialect {
	case DialectPostgreSQL:
		query = fmt.Sprintf(`DELETE FROM %s WHERE expires_at < NOW()`, s.tableName)
	case DialectMySQL:
		query = fmt.Sprintf(`DELETE FROM %s WHERE expires_at < NOW()`, s.tableName)
	case DialectSQLite:
		query = fmt.Sprintf(`DELETE FROM %s WHERE expires_at < datetime('now')`, s.tableName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.db.ExecContext(ctx, query)
}

// CreateTable creates the session table if it doesn't exist.
// This is a convenience method for development/testing.
func (s *SQLStore) CreateTable(ctx context.Context) error {
	var query string
	switch s.dialect {
	case DialectPostgreSQL:
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id VARCHAR(64) PRIMARY KEY,
				data BYTEA NOT NULL,
				expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)
		`, s.tableName)
	case DialectMySQL:
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id VARCHAR(64) PRIMARY KEY,
				data BLOB NOT NULL,
				expires_at DATETIME NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
			)
		`, s.tableName)
	case DialectSQLite:
		query = fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				id TEXT PRIMARY KEY,
				data BLOB NOT NULL,
				expires_at TEXT NOT NULL,
				created_at TEXT DEFAULT (datetime('now')),
				updated_at TEXT DEFAULT (datetime('now'))
			)
		`, s.tableName)
	}

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	// Create index on expires_at for efficient cleanup
	var indexQuery string
	switch s.dialect {
	case DialectPostgreSQL, DialectSQLite:
		indexQuery = fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS idx_%s_expires ON %s(expires_at)
		`, s.tableName, s.tableName)
	case DialectMySQL:
		// MySQL doesn't support IF NOT EXISTS for indexes directly
		// We'll just try to create it and ignore the error
		indexQuery = fmt.Sprintf(`
			CREATE INDEX idx_%s_expires ON %s(expires_at)
		`, s.tableName, s.tableName)
	}

	// Try to create the index, ignore errors (may already exist)
	s.db.ExecContext(ctx, indexQuery)

	return nil
}
