package session

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func normalizeQuery(q string) string {
	return strings.Join(strings.Fields(q), " ")
}

type recordedExec struct {
	query string
	args  []driver.NamedValue
}

type recordedQuery struct {
	query string
	args  []driver.NamedValue
}

type fakeSQLRecorder struct {
	mu sync.Mutex

	execs   []recordedExec
	queries []recordedQuery

	// Queue of query responses returned by QueryContext, in order.
	queryResponses []fakeRowsResult
}

type fakeRowsResult struct {
	columns []string
	rows    [][]driver.Value
}

func (r *fakeSQLRecorder) recordExec(query string, args []driver.NamedValue) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.execs = append(r.execs, recordedExec{query: normalizeQuery(query), args: append([]driver.NamedValue(nil), args...)})
}

func (r *fakeSQLRecorder) recordQuery(query string, args []driver.NamedValue) fakeRowsResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queries = append(r.queries, recordedQuery{query: normalizeQuery(query), args: append([]driver.NamedValue(nil), args...)})
	if len(r.queryResponses) == 0 {
		return fakeRowsResult{columns: []string{"data"}, rows: nil}
	}
	resp := r.queryResponses[0]
	r.queryResponses = r.queryResponses[1:]
	return resp
}

type fakeSQLDriver struct{}

var (
	fakeSQLRegisterOnce sync.Once
	fakeSQLMu        sync.Mutex
	fakeSQLRecorders = map[string]*fakeSQLRecorder{}
)

func (d fakeSQLDriver) Open(name string) (driver.Conn, error) {
	fakeSQLMu.Lock()
	rec := fakeSQLRecorders[name]
	fakeSQLMu.Unlock()
	if rec == nil {
		return nil, fmt.Errorf("unknown fake db name: %s", name)
	}
	return &fakeSQLConn{rec: rec}, nil
}

type fakeSQLConn struct {
	rec *fakeSQLRecorder
}

func (c *fakeSQLConn) Prepare(query string) (driver.Stmt, error) { return c.PrepareContext(context.Background(), query) }
func (c *fakeSQLConn) Close() error                              { return nil }
func (c *fakeSQLConn) Begin() (driver.Tx, error)                 { return c.BeginTx(context.Background(), driver.TxOptions{}) }

func (c *fakeSQLConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return &fakeSQLTx{conn: c}, nil
}

func (c *fakeSQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.rec.recordExec(query, args)
	return driver.RowsAffected(1), nil
}

func (c *fakeSQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	resp := c.rec.recordQuery(query, args)
	return &fakeSQLRows{
		columns: resp.columns,
		rows:    resp.rows,
	}, nil
}

func (c *fakeSQLConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return &fakeSQLStmt{rec: c.rec, query: query}, nil
}

type fakeSQLTx struct {
	conn *fakeSQLConn
}

func (t *fakeSQLTx) Commit() error   { return nil }
func (t *fakeSQLTx) Rollback() error { return nil }

type fakeSQLStmt struct {
	rec   *fakeSQLRecorder
	query string
}

func (s *fakeSQLStmt) Close() error { return nil }
func (s *fakeSQLStmt) NumInput() int {
	return -1
}
func (s *fakeSQLStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), namedFromValues(args))
}
func (s *fakeSQLStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), namedFromValues(args))
}
func (s *fakeSQLStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	s.rec.recordExec(s.query, args)
	return driver.RowsAffected(1), nil
}
func (s *fakeSQLStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	resp := s.rec.recordQuery(s.query, args)
	return &fakeSQLRows{
		columns: resp.columns,
		rows:    resp.rows,
	}, nil
}

func namedFromValues(values []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, 0, len(values))
	for i, v := range values {
		out = append(out, driver.NamedValue{Ordinal: i + 1, Value: v})
	}
	return out
}

type fakeSQLRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *fakeSQLRows) Columns() []string { return r.columns }
func (r *fakeSQLRows) Close() error      { return nil }
func (r *fakeSQLRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

func openFakeDB(t *testing.T) (*sql.DB, *fakeSQLRecorder) {
	t.Helper()

	// Register driver once per test binary.
	fakeSQLRegisterOnce.Do(func() {
		sql.Register("vango_fake_sql", fakeSQLDriver{})
	})

	rec := &fakeSQLRecorder{}
	name := t.Name()

	fakeSQLMu.Lock()
	fakeSQLRecorders[name] = rec
	fakeSQLMu.Unlock()

	t.Cleanup(func() {
		fakeSQLMu.Lock()
		delete(fakeSQLRecorders, name)
		fakeSQLMu.Unlock()
	})

	db, err := sql.Open("vango_fake_sql", name)
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db, rec
}

func TestSQLStore_Placeholders(t *testing.T) {
	db, _ := openFakeDB(t)
	store := NewSQLStore(db, WithSQLDialect(DialectPostgreSQL), WithSQLCleanupInterval(24*time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	if got := store.placeholder(3); got != "$3" {
		t.Fatalf("placeholder() got %q want %q", got, "$3")
	}

	storeMy := NewSQLStore(db, WithSQLDialect(DialectMySQL), WithSQLCleanupInterval(24*time.Hour))
	t.Cleanup(func() { _ = storeMy.Close() })
	if got := storeMy.placeholder(3); got != "?" {
		t.Fatalf("placeholder() got %q want %q", got, "?")
	}
}

func TestSQLStore_SaveLoadDeleteTouch_PostgresQueries(t *testing.T) {
	db, rec := openFakeDB(t)
	store := NewSQLStore(db, WithSQLDialect(DialectPostgreSQL), WithSQLCleanupInterval(24*time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	expiresAt := time.Now().Add(time.Minute)

	if err := store.Save(ctx, "s1", []byte("data"), expiresAt); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	rec.mu.Lock()
	if len(rec.execs) != 1 {
		rec.mu.Unlock()
		t.Fatalf("execs got %d want 1", len(rec.execs))
	}
	saveQuery := rec.execs[0].query
	rec.mu.Unlock()
	if !strings.Contains(saveQuery, "INSERT INTO vango_sessions") || !strings.Contains(saveQuery, "ON CONFLICT (id) DO UPDATE") {
		t.Fatalf("unexpected Save query: %q", saveQuery)
	}

	rec.mu.Lock()
	rec.queryResponses = append(rec.queryResponses, fakeRowsResult{
		columns: []string{"data"},
		rows:    [][]driver.Value{{[]byte("blob")}},
	})
	rec.mu.Unlock()

	loaded, err := store.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if string(loaded) != "blob" {
		t.Fatalf("Load() got %q want %q", string(loaded), "blob")
	}

	if err := store.Touch(ctx, "s1", expiresAt.Add(time.Minute)); err != nil {
		t.Fatalf("Touch() error: %v", err)
	}
	if err := store.Delete(ctx, "s1"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.queries) != 1 {
		t.Fatalf("queries got %d want 1", len(rec.queries))
	}
	if !strings.Contains(rec.queries[0].query, "WHERE id = $1") {
		t.Fatalf("unexpected Load query: %q", rec.queries[0].query)
	}

	if len(rec.execs) < 3 {
		t.Fatalf("exec count got %d want >= 3", len(rec.execs))
	}
	if !strings.Contains(rec.execs[1].query, "UPDATE vango_sessions SET expires_at = $1") {
		t.Fatalf("unexpected Touch query: %q", rec.execs[1].query)
	}
	if got := rec.execs[len(rec.execs)-1].query; !strings.Contains(got, "DELETE FROM vango_sessions WHERE id = $1") {
		t.Fatalf("unexpected Delete query: %q", got)
	}
}

func TestSQLStore_Load_NoRowsReturnsNil(t *testing.T) {
	db, rec := openFakeDB(t)
	store := NewSQLStore(db, WithSQLDialect(DialectSQLite), WithSQLCleanupInterval(24*time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	rec.mu.Lock()
	rec.queryResponses = append(rec.queryResponses, fakeRowsResult{
		columns: []string{"data"},
		rows:    nil,
	})
	rec.mu.Unlock()

	data, err := store.Load(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if data != nil {
		t.Fatalf("Load() got %v want nil", data)
	}
}

func TestSQLStore_SaveAll_UsesTransaction(t *testing.T) {
	db, rec := openFakeDB(t)
	store := NewSQLStore(db, WithSQLDialect(DialectSQLite), WithSQLCleanupInterval(24*time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	expiresAt := time.Now().Add(time.Minute)
	if err := store.SaveAll(ctx, map[string]SessionData{
		"a": {Data: []byte("1"), ExpiresAt: expiresAt},
		"b": {Data: []byte("2"), ExpiresAt: expiresAt},
	}); err != nil {
		t.Fatalf("SaveAll() error: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.execs) != 2 {
		t.Fatalf("exec count got %d want 2", len(rec.execs))
	}
	if !strings.Contains(rec.execs[0].query, "INSERT OR REPLACE INTO vango_sessions") {
		t.Fatalf("unexpected SaveAll query: %q", rec.execs[0].query)
	}
}

func TestSQLStore_CleanupAndCreateTable(t *testing.T) {
	db, rec := openFakeDB(t)
	store := NewSQLStore(db, WithSQLDialect(DialectMySQL), WithSQLCleanupInterval(24*time.Hour))
	t.Cleanup(func() { _ = store.Close() })

	store.cleanup()

	ctx := context.Background()
	if err := store.CreateTable(ctx); err != nil {
		t.Fatalf("CreateTable() error: %v", err)
	}

	rec.mu.Lock()
	defer rec.mu.Unlock()

	if len(rec.execs) < 3 {
		t.Fatalf("exec count got %d want >= 3", len(rec.execs))
	}
	if got := rec.execs[0].query; !strings.Contains(got, "DELETE FROM vango_sessions WHERE expires_at < NOW()") {
		t.Fatalf("cleanup query got %q", got)
	}
	if got := rec.execs[1].query; !strings.Contains(got, "CREATE TABLE IF NOT EXISTS vango_sessions") {
		t.Fatalf("CreateTable query got %q", got)
	}
	if got := rec.execs[2].query; !strings.Contains(got, "CREATE INDEX idx_vango_sessions_expires") {
		t.Fatalf("Create index query got %q", got)
	}
}

func TestSQLStore_Close_MakesOperationsFail(t *testing.T) {
	db, _ := openFakeDB(t)
	store := NewSQLStore(db, WithSQLDialect(DialectPostgreSQL), WithSQLCleanupInterval(24*time.Hour))

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() second call error: %v", err)
	}

	ctx := context.Background()
	expiresAt := time.Now().Add(time.Minute)
	if err := store.Save(ctx, "s", []byte("x"), expiresAt); err == nil {
		t.Fatal("Save() expected error after Close, got nil")
	}
	if _, err := store.Load(ctx, "s"); err == nil {
		t.Fatal("Load() expected error after Close, got nil")
	}
	if err := store.Delete(ctx, "s"); err == nil {
		t.Fatal("Delete() expected error after Close, got nil")
	}
	if err := store.Touch(ctx, "s", expiresAt); err == nil {
		t.Fatal("Touch() expected error after Close, got nil")
	}
	if err := store.SaveAll(ctx, map[string]SessionData{}); err == nil {
		t.Fatal("SaveAll() expected error after Close, got nil")
	}
}
