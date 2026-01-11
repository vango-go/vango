package server

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/session"
)

func TestSessionManager_restoreSessionFromPersistence_RestoresValues(t *testing.T) {
	sm := NewSessionManager(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default())
	t.Cleanup(func() { sm.Shutdown() })

	values := map[string]json.RawMessage{
		"s": json.RawMessage(`"str"`),
		"n": json.RawMessage(`123`),
	}
	ss := &session.SerializableSession{
		ID:         "id",
		UserID:     "u",
		CreatedAt:  time.Now().Add(-time.Hour),
		LastActive: time.Now().Add(-time.Minute),
		Route:      "/r",
		Values:     values,
	}
	data, err := session.Serialize(ss)
	if err != nil {
		t.Fatalf("session.Serialize error: %v", err)
	}

	restored := sm.restoreSessionFromPersistence(ss.ID, data)
	if restored == nil {
		t.Fatal("restoreSessionFromPersistence returned nil")
	}
	if restored.GetString("s") != "str" {
		t.Fatalf("GetString(s)=%q, want %q", restored.GetString("s"), "str")
	}
	if restored.GetInt("n") != 123 {
		t.Fatalf("GetInt(n)=%d, want %d", restored.GetInt("n"), 123)
	}
}

