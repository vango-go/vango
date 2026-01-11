package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSerialize_SetsVersionAndRoundTrips(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	ss := &SerializableSession{
		ID:         "sess-1",
		UserID:     "user-1",
		CreatedAt:  now.Add(-time.Minute),
		LastActive: now,
		Route:      "/dashboard",
		RouteParams: map[string]string{
			"org": "acme",
		},
		Values: map[string]json.RawMessage{
			"theme": json.RawMessage(`"dark"`),
		},
		Signals: map[string]json.RawMessage{
			"user_id": json.RawMessage(`123`),
		},
		Version: 999, // should be overwritten
	}

	data, err := Serialize(ss)
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}
	if ss.Version != CurrentSerializationVersion {
		t.Fatalf("Serialize() did not set Version: got %d want %d", ss.Version, CurrentSerializationVersion)
	}

	roundTripped, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}
	if roundTripped.ID != ss.ID || roundTripped.UserID != ss.UserID {
		t.Fatalf("round-trip mismatch: got %+v want %+v", roundTripped, ss)
	}
	if roundTripped.Route != ss.Route {
		t.Fatalf("Route mismatch: got %q want %q", roundTripped.Route, ss.Route)
	}
	if roundTripped.RouteParams["org"] != "acme" {
		t.Fatalf("RouteParams mismatch: got %v", roundTripped.RouteParams)
	}
	if string(roundTripped.Values["theme"]) != `"dark"` {
		t.Fatalf("Values mismatch: got %s", roundTripped.Values["theme"])
	}
	if string(roundTripped.Signals["user_id"]) != `123` {
		t.Fatalf("Signals mismatch: got %s", roundTripped.Signals["user_id"])
	}
	if roundTripped.Version != CurrentSerializationVersion {
		t.Fatalf("Version mismatch: got %d want %d", roundTripped.Version, CurrentSerializationVersion)
	}
}

func TestDeserialize_InvalidJSONErrors(t *testing.T) {
	_, err := Deserialize([]byte("{not-json"))
	if err == nil {
		t.Fatal("Deserialize() expected error, got nil")
	}
}

func TestNewSignalConfig_Defaults(t *testing.T) {
	cfg := NewSignalConfig()
	if cfg == nil {
		t.Fatal("NewSignalConfig() returned nil")
	}
	if cfg.Transient {
		t.Fatalf("default Transient=true, want false")
	}
	if cfg.PersistKey != "" {
		t.Fatalf("default PersistKey=%q, want empty", cfg.PersistKey)
	}
}

