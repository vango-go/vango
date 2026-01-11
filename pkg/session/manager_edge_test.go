package session

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type recordingStore struct {
	mu sync.Mutex

	loadData map[string][]byte

	deletes []string
	saves   []string
	saveAll map[string]SessionData
}

func (s *recordingStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saves = append(s.saves, sessionID)
	if s.loadData == nil {
		s.loadData = make(map[string][]byte)
	}
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	s.loadData[sessionID] = dataCopy
	return nil
}

func (s *recordingStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loadData == nil {
		return nil, nil
	}
	data := s.loadData[sessionID]
	if data == nil {
		return nil, nil
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func (s *recordingStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes = append(s.deletes, sessionID)
	delete(s.loadData, sessionID)
	return nil
}

func (s *recordingStore) Touch(ctx context.Context, sessionID string, expiresAt time.Time) error { return nil }

func (s *recordingStore) SaveAll(ctx context.Context, sessions map[string]SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveAll = make(map[string]SessionData, len(sessions))
	for id, sd := range sessions {
		dataCopy := make([]byte, len(sd.Data))
		copy(dataCopy, sd.Data)
		s.saveAll[id] = SessionData{Data: dataCopy, ExpiresAt: sd.ExpiresAt}
	}
	return nil
}

func (s *recordingStore) Close() error { return nil }

func TestManager_OnReconnect_LoadsFromStoreWhenNotInMemory(t *testing.T) {
	store := &recordingStore{
		loadData: map[string][]byte{
			"s1": []byte("serialized"),
		},
	}
	cfg := DefaultManagerConfig()
	cfg.CleanupInterval = 24 * time.Hour
	manager := NewManager(store, cfg, slog.Default())
	t.Cleanup(func() { _ = manager.Shutdown(context.Background()) })

	sess, data, err := manager.OnReconnect("s1")
	if err != nil {
		t.Fatalf("OnReconnect() error: %v", err)
	}
	if sess != nil {
		t.Fatalf("OnReconnect() expected nil session when only in store, got %+v", sess)
	}
	if string(data) != "serialized" {
		t.Fatalf("OnReconnect() got %q", string(data))
	}
}

func TestManager_OnReconnect_ExpiredSessionIsRemoved(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.ResumeWindow = 10 * time.Millisecond
	cfg.CleanupInterval = 24 * time.Hour

	manager := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = manager.Shutdown(context.Background()) })

	sess := &ManagedSession{
		ID:        "s1",
		IP:        "10.0.0.1",
		CreatedAt: time.Now(),
	}
	if err := manager.Register(sess); err != nil {
		t.Fatalf("Register() error: %v", err)
	}
	manager.OnDisconnect("s1", []byte("x"))

	got := manager.Get("s1")
	if got == nil {
		t.Fatal("expected session to exist after disconnect")
	}
	got.DisconnectedAt = time.Now().Add(-time.Second)

	_, _, err := manager.OnReconnect("s1")
	if err != ErrSessionExpired {
		t.Fatalf("OnReconnect() error got %v want %v", err, ErrSessionExpired)
	}
	if manager.Get("s1") != nil {
		t.Fatal("expected expired session to be removed")
	}
}

func TestManager_TouchUpdatesLRUOrder(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.MaxDetachedSessions = 2
	cfg.CleanupInterval = 24 * time.Hour

	manager := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = manager.Shutdown(context.Background()) })

	for _, id := range []string{"a", "b"} {
		if err := manager.Register(&ManagedSession{ID: id, IP: "10.0.0.1", CreatedAt: time.Now()}); err != nil {
			t.Fatalf("Register(%s) error: %v", id, err)
		}
		manager.OnDisconnect(id, []byte(id))
	}

	// Queue order is: b (front), a (back). Touch a => a becomes front.
	manager.Touch("a")

	if err := manager.Register(&ManagedSession{ID: "c", IP: "10.0.0.1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Register(c) error: %v", err)
	}
	manager.OnDisconnect("c", []byte("c"))

	// Adding c makes queue size 3; eviction removes the back (least recently used), which should be b.
	if manager.Get("b") != nil {
		t.Fatal("expected session b to be evicted after Touch(a) promoted a to MRU")
	}
	if manager.Get("a") == nil || manager.Get("c") == nil {
		t.Fatal("expected sessions a and c to remain")
	}
}

func TestManager_CleanupExpired_RemovesDetachedOnly(t *testing.T) {
	store := &recordingStore{}
	cfg := DefaultManagerConfig()
	cfg.ResumeWindow = 50 * time.Millisecond
	cfg.CleanupInterval = 24 * time.Hour

	manager := NewManager(store, cfg, slog.Default())
	t.Cleanup(func() { _ = manager.Shutdown(context.Background()) })

	if err := manager.Register(&ManagedSession{ID: "connected", IP: "10.0.0.1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Register(connected) error: %v", err)
	}
	if err := manager.Register(&ManagedSession{ID: "detached", IP: "10.0.0.1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Register(detached) error: %v", err)
	}
	manager.OnDisconnect("detached", []byte("x"))

	d := manager.Get("detached")
	if d == nil {
		t.Fatal("expected detached session to exist")
	}
	d.DisconnectedAt = time.Now().Add(-time.Second)

	manager.cleanupExpired()

	if manager.Get("connected") == nil {
		t.Fatal("expected connected session to remain")
	}
	if manager.Get("detached") != nil {
		t.Fatal("expected detached expired session to be removed")
	}
}

func TestManager_ShutdownStopsManager(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.CleanupInterval = 24 * time.Hour

	manager := NewManager(nil, cfg, slog.Default())
	if err := manager.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error: %v", err)
	}

	if err := manager.CheckIPLimit("1.2.3.4"); err != ErrManagerStopped {
		t.Fatalf("CheckIPLimit() after shutdown got %v want %v", err, ErrManagerStopped)
	}
	if err := manager.Register(&ManagedSession{ID: "s1", IP: "1.2.3.4"}); err != ErrManagerStopped {
		t.Fatalf("Register() after shutdown got %v want %v", err, ErrManagerStopped)
	}
	if _, _, err := manager.OnReconnect("missing"); err != ErrManagerStopped {
		t.Fatalf("OnReconnect() after shutdown got %v want %v", err, ErrManagerStopped)
	}
}

