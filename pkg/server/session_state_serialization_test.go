package server

import (
	"encoding/json"
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/session"
	"github.com/vango-go/vango/pkg/urlparam"
)

func TestSession_DataAccessorsAndURLPatches(t *testing.T) {
	s := NewMockSession()

	s.SetString("k1", "v1")
	s.SetInt("k2", 42)
	s.SetJSON("k3", map[string]any{"a": 1})

	if got := s.GetString("k1"); got != "v1" {
		t.Fatalf("GetString(k1)=%q, want %q", got, "v1")
	}
	if got := s.GetInt("k2"); got != 42 {
		t.Fatalf("GetInt(k2)=%d, want %d", got, 42)
	}

	var decoded map[string]any
	raw := s.Get("k3")
	bytes, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("Marshal(session[k3]) failed: %v", err)
	}
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("Unmarshal(session[k3]) failed: %v", err)
	}
	if decoded["a"] != float64(1) {
		t.Fatalf("decoded[a]=%v, want 1", decoded["a"])
	}

	s.queueURLPatch(protocolURLReplacePatch(map[string]string{"x": "1"}))
	s.queueURLPatch(protocolURLReplacePatch(map[string]string{"y": "2"}))
	got := s.drainURLPatches()
	if len(got) != 2 {
		t.Fatalf("drainURLPatches len=%d, want 2", len(got))
	}
	if len(s.drainURLPatches()) != 0 {
		t.Fatal("drainURLPatches not cleared")
	}

	s.SetInitialURL("/p", map[string]string{"q": "v"})
	stateAny := s.owner.GetValue(urlparam.InitialParamsKey)
	state, ok := stateAny.(*urlparam.InitialURLState)
	if !ok || state == nil {
		t.Fatalf("InitialURLState=%T, want *urlparam.InitialURLState", stateAny)
	}
	if state.Path != "/p" || state.Params["q"] != "v" {
		t.Fatalf("InitialURLState=%+v, want path=/p params[q]=v", state)
	}
}

func protocolURLReplacePatch(params map[string]string) protocol.Patch {
	return protocol.NewURLReplacePatch(params)
}

func TestSession_SerializeDeserialize_RoundTripMergesData(t *testing.T) {
	s := NewMockSession()
	s.ID = "sid"
	s.UserID = "u1"
	s.CurrentRoute = "/r"
	s.SetString("a", "x")
	s.SetInt("n", 7)

	data, err := s.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}
	// Ensure it is parseable as a session.SerializableSession payload.
	if _, err := session.Deserialize(data); err != nil {
		t.Fatalf("session.Deserialize(Serialize()) error: %v", err)
	}

	s2 := NewMockSession()
	s2.SetString("existing", "keep")
	if err := s2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}
	if s2.ID != "sid" || s2.UserID != "u1" || s2.CurrentRoute != "/r" {
		t.Fatalf("identity restore failed: ID=%q UserID=%q Route=%q", s2.ID, s2.UserID, s2.CurrentRoute)
	}
	if s2.GetString("existing") != "keep" {
		t.Fatalf("existing key overwritten: %q", s2.GetString("existing"))
	}
	if s2.GetString("a") != "x" || s2.GetInt("n") != 7 {
		t.Fatalf("restored data mismatch: a=%q n=%d", s2.GetString("a"), s2.GetInt("n"))
	}
}
