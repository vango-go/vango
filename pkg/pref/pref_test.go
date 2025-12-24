package pref

import (
	"encoding/json"
	"testing"
	"time"
)

// TestPrefNew tests creating new preferences.
func TestPrefNew(t *testing.T) {
	t.Run("WithDefaults", func(t *testing.T) {
		pref := New("theme", "light")

		if pref.Key() != "theme" {
			t.Errorf("Key: got %v, want theme", pref.Key())
		}
		if pref.Get() != "light" {
			t.Errorf("Get: got %v, want light", pref.Get())
		}
	})

	t.Run("WithMergeStrategy", func(t *testing.T) {
		pref := New("settings", "default", MergeWith(DBWins))

		if pref.config.mergeStrategy != DBWins {
			t.Errorf("MergeStrategy: got %v, want DBWins", pref.config.mergeStrategy)
		}
	})

	t.Run("WithLocalOnly", func(t *testing.T) {
		pref := New("volume", 50, LocalOnly())

		if pref.config.syncToServer {
			t.Error("LocalOnly pref should have syncToServer=false")
		}
	})
}

// TestPrefSetGet tests setting and getting values.
func TestPrefSetGet(t *testing.T) {
	pref := New("theme", "light")

	// Get initial value
	if pref.Get() != "light" {
		t.Errorf("Initial Get: got %v, want light", pref.Get())
	}

	// Set new value
	pref.Set("dark")
	if pref.Get() != "dark" {
		t.Errorf("After Set: got %v, want dark", pref.Get())
	}

	// Verify updatedAt changed
	if pref.UpdatedAt().IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

// TestPrefReset tests resetting to default value.
func TestPrefReset(t *testing.T) {
	pref := New("theme", "light")

	pref.Set("dark")
	if pref.Get() != "dark" {
		t.Fatal("Set failed")
	}

	pref.Reset()
	if pref.Get() != "light" {
		t.Errorf("After Reset: got %v, want light", pref.Get())
	}
}

// TestPrefSetFromRemote tests conflict resolution with remote values.
func TestPrefSetFromRemote(t *testing.T) {
	t.Run("DBWins", func(t *testing.T) {
		pref := New("setting", "local", MergeWith(DBWins))
		pref.SetFromRemote("remote", time.Now())

		if pref.Get() != "remote" {
			t.Errorf("DBWins: got %v, want remote", pref.Get())
		}
	})

	t.Run("LocalWins", func(t *testing.T) {
		pref := New("setting", "local", MergeWith(LocalWins))
		pref.Set("local-modified")
		pref.SetFromRemote("remote", time.Now())

		if pref.Get() != "local-modified" {
			t.Errorf("LocalWins: got %v, want local-modified", pref.Get())
		}
	})

	t.Run("LWW_RemoteNewer", func(t *testing.T) {
		pref := New("setting", "local", MergeWith(LWW))
		pref.Set("local-modified")

		// Remote is newer
		remoteTime := time.Now().Add(1 * time.Second)
		pref.SetFromRemote("remote", remoteTime)

		if pref.Get() != "remote" {
			t.Errorf("LWW remote newer: got %v, want remote", pref.Get())
		}
	})

	t.Run("LWW_LocalNewer", func(t *testing.T) {
		pref := New("setting", "local", MergeWith(LWW))

		// Set local with current time
		pref.Set("local-modified")

		// Remote is older
		remoteTime := time.Now().Add(-1 * time.Hour)
		pref.SetFromRemote("remote", remoteTime)

		if pref.Get() != "local-modified" {
			t.Errorf("LWW local newer: got %v, want local-modified", pref.Get())
		}
	})
}

// TestPrefCustomConflictHandler tests custom conflict resolution.
func TestPrefCustomConflictHandler(t *testing.T) {
	handler := func(local, remote any) any {
		// Always concatenate
		return local.(string) + "+" + remote.(string)
	}

	pref := New("setting", "initial", OnConflict(handler))
	pref.Set("local")
	pref.SetFromRemote("remote", time.Now())

	expected := "local+remote"
	if pref.Get() != expected {
		t.Errorf("Custom handler: got %v, want %v", pref.Get(), expected)
	}
}

// TestPrefJSON tests JSON marshaling and unmarshaling.
func TestPrefJSON(t *testing.T) {
	pref := New("theme", "dark")
	pref.Set("light")

	// Marshal
	data, err := json.Marshal(pref)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal into a new pref
	var decoded Pref[string]
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Key() != "theme" {
		t.Errorf("Decoded key: got %v, want theme", decoded.Key())
	}
	if decoded.Get() != "light" {
		t.Errorf("Decoded value: got %v, want light", decoded.Get())
	}
}

// TestPrefConcurrency tests concurrent access.
func TestPrefConcurrency(t *testing.T) {
	pref := New("counter", 0)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				current := pref.Get()
				pref.Set(current + 1)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// We can't assert exact value due to race conditions,
	// but we can verify it doesn't crash and value is non-zero
	if pref.Get() == 0 {
		t.Error("Counter should be non-zero after concurrent updates")
	}
}

// TestMergeStrategyValues tests merge strategy enum values.
func TestMergeStrategyValues(t *testing.T) {
	// Verify enum values are distinct
	strategies := []MergeStrategy{DBWins, LocalWins, Prompt, LWW}
	seen := make(map[MergeStrategy]bool)

	for _, s := range strategies {
		if seen[s] {
			t.Errorf("Duplicate MergeStrategy value: %v", s)
		}
		seen[s] = true
	}
}

// TestPrefTypedValues tests preferences with different types.
func TestPrefTypedValues(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		pref := New("name", "default")
		pref.Set("updated")
		if pref.Get() != "updated" {
			t.Error("String pref failed")
		}
	})

	t.Run("Int", func(t *testing.T) {
		pref := New("count", 0)
		pref.Set(42)
		if pref.Get() != 42 {
			t.Error("Int pref failed")
		}
	})

	t.Run("Bool", func(t *testing.T) {
		pref := New("enabled", false)
		pref.Set(true)
		if !pref.Get() {
			t.Error("Bool pref failed")
		}
	})

	t.Run("Struct", func(t *testing.T) {
		type Settings struct {
			Theme    string
			FontSize int
		}
		defaults := Settings{Theme: "light", FontSize: 14}
		pref := New("settings", defaults)

		updated := Settings{Theme: "dark", FontSize: 16}
		pref.Set(updated)

		got := pref.Get()
		if got.Theme != "dark" || got.FontSize != 16 {
			t.Errorf("Struct pref: got %+v, want %+v", got, updated)
		}
	})
}
