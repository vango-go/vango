package hooks

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHook(t *testing.T) {
	config := map[string]any{
		"foo": "bar",
		"baz": 123,
	}

	attr := Hook("MyHook", config)

	if attr.Key != "v-hook" {
		t.Errorf("Expected attr key 'v-hook', got '%s'", attr.Key)
	}

	// Value should roughly be "MyHook:{\"baz\":123,\"foo\":\"bar\"}"
	// JSON keys order is not guaranteed, so we check parts.
	val := attr.Value.(string)
	if !strings.HasPrefix(val, "MyHook:") {
		t.Errorf("Expected prefix 'MyHook:', got '%s'", val)
	}

	// Check JSON part
	jsonPart := strings.TrimPrefix(val, "MyHook:")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonPart), &decoded); err != nil {
		t.Errorf("Failed to unmarshal JSON part: %v", err)
	}

	if decoded["foo"] != "bar" {
		t.Errorf("Expected foo=bar, got %v", decoded["foo"])
	}
	if f, ok := decoded["baz"].(float64); ok { // JSON unmarshals numbers as float64
		if f != 123 {
			t.Errorf("Expected baz=123, got %v", f)
		}
	} else {
		t.Errorf("Expected baz to be float64, got %T", decoded["baz"])
	}
}

func TestHookEmptyConfig(t *testing.T) {
	attr := Hook("EmptyHook", nil)

	if attr.Key != "v-hook" {
		t.Errorf("Expected attr key 'v-hook', got '%s'", attr.Key)
	}

	val := attr.Value.(string)
	if !strings.HasPrefix(val, "EmptyHook:") {
		t.Errorf("Expected prefix 'EmptyHook:', got '%s'", val)
	}
}

func TestHookWithNestedConfig(t *testing.T) {
	config := map[string]any{
		"nested": map[string]any{
			"inner": "value",
		},
		"array": []int{1, 2, 3},
	}

	attr := Hook("NestedHook", config)
	val := attr.Value.(string)

	if !strings.Contains(val, "NestedHook:") {
		t.Error("Expected NestedHook prefix")
	}

	jsonPart := strings.TrimPrefix(val, "NestedHook:")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonPart), &decoded); err != nil {
		t.Errorf("Failed to unmarshal nested JSON: %v", err)
	}
}

func TestOnEvent(t *testing.T) {
	called := false
	handler := func(e HookEvent) {
		called = true
		if e.Name != "" {
			// Just verify we can access the event
		}
	}

	eventHandler := OnEvent("customEvent", handler)

	if eventHandler.Event != "customEvent" {
		t.Errorf("Expected event 'customEvent', got '%s'", eventHandler.Event)
	}

	if eventHandler.Handler == nil {
		t.Error("Expected handler to be set")
	}

	// Call the handler to verify it works
	if fn, ok := eventHandler.Handler.(func(HookEvent)); ok {
		fn(HookEvent{Name: "test"})
		if !called {
			t.Error("Handler should have been called")
		}
	}
}

func TestHookEventAccessors(t *testing.T) {
	data := map[string]any{
		"s":  "string",
		"i":  42,
		"f":  3.14,
		"b":  true,
		"l":  []any{"a", "b"},
		"ls": []string{"c", "d"},
		"n":  "123", // number as string
	}

	e := HookEvent{Name: "test", Data: data}

	if e.String("s") != "string" {
		t.Errorf("String mismatch")
	}

	if e.Int("i") != 42 {
		t.Errorf("Int mismatch")
	}

	if e.Int("n") != 123 {
		t.Errorf("Int from string mismatch")
	}

	if e.Float("f") != 3.14 {
		t.Errorf("Float mismatch")
	}

	if !e.Bool("b") {
		t.Errorf("Bool mismatch")
	}

	strs := e.Strings("l")
	if len(strs) != 2 || strs[0] != "a" {
		t.Errorf("Strings (any) mismatch")
	}

	strs2 := e.Strings("ls")
	if len(strs2) != 2 || strs2[0] != "c" {
		t.Errorf("Strings (string) mismatch")
	}
}

func TestHookEventStringMissing(t *testing.T) {
	e := HookEvent{Data: map[string]any{}}

	if e.String("missing") != "" {
		t.Error("String of missing key should return empty string")
	}
}

func TestHookEventIntEdgeCases(t *testing.T) {
	e := HookEvent{Data: map[string]any{
		"float": 3.7,
		"other": struct{}{},
	}}

	// Float should be converted to int
	if e.Int("float") != 3 {
		t.Errorf("Int from float should be 3, got %d", e.Int("float"))
	}

	// Missing key should return 0
	if e.Int("missing") != 0 {
		t.Error("Int of missing key should return 0")
	}

	// Unhandled type should return 0
	if e.Int("other") != 0 {
		t.Error("Int of unhandled type should return 0")
	}
}

func TestHookEventFloatEdgeCases(t *testing.T) {
	e := HookEvent{Data: map[string]any{
		"int":    42,
		"string": "3.14",
		"other":  struct{}{},
	}}

	// Int should be converted to float
	if e.Float("int") != 42.0 {
		t.Errorf("Float from int should be 42.0, got %f", e.Float("int"))
	}

	// String should be parsed
	if e.Float("string") != 3.14 {
		t.Errorf("Float from string should be 3.14, got %f", e.Float("string"))
	}

	// Missing key should return 0
	if e.Float("missing") != 0.0 {
		t.Error("Float of missing key should return 0")
	}

	// Unhandled type should return 0
	if e.Float("other") != 0.0 {
		t.Error("Float of unhandled type should return 0")
	}
}

func TestHookEventBoolEdgeCases(t *testing.T) {
	e := HookEvent{Data: map[string]any{
		"true_str":  "true",
		"false_str": "false",
		"one":       "1",
		"zero":      "0",
		"other":     struct{}{},
	}}

	// String "true" should parse to true
	if !e.Bool("true_str") {
		t.Error("Bool of 'true' string should be true")
	}

	// String "false" should parse to false
	if e.Bool("false_str") {
		t.Error("Bool of 'false' string should be false")
	}

	// String "1" should parse to true
	if !e.Bool("one") {
		t.Error("Bool of '1' string should be true")
	}

	// Missing key should return false
	if e.Bool("missing") {
		t.Error("Bool of missing key should return false")
	}
}

func TestHookEventStringsEdgeCases(t *testing.T) {
	e := HookEvent{Data: map[string]any{
		"other": "not a slice",
	}}

	// Non-slice should return nil
	if e.Strings("other") != nil {
		t.Error("Strings of non-slice should return nil")
	}

	// Missing key should return nil
	if e.Strings("missing") != nil {
		t.Error("Strings of missing key should return nil")
	}
}

func TestHookEventRaw(t *testing.T) {
	data := map[string]any{
		"complex": map[string]any{"nested": true},
		"simple":  "value",
	}
	e := HookEvent{Data: data}

	// Raw should return the exact value
	raw := e.Raw("complex")
	if raw == nil {
		t.Fatal("Raw should return non-nil for existing key")
	}

	if m, ok := raw.(map[string]any); ok {
		if m["nested"] != true {
			t.Error("Raw should return exact nested value")
		}
	} else {
		t.Error("Raw should return map[string]any")
	}

	// Raw of missing key should return nil
	if e.Raw("missing") != nil {
		t.Error("Raw of missing key should return nil")
	}
}

func TestHookEventRevert(t *testing.T) {
	e := HookEvent{Name: "test", Data: map[string]any{}}

	// Revert should not panic (it's a placeholder)
	e.Revert()
	// If we get here without panic, the test passes
}

func TestHookEventName(t *testing.T) {
	e := HookEvent{Name: "myEvent", Data: nil}

	if e.Name != "myEvent" {
		t.Errorf("Expected Name 'myEvent', got '%s'", e.Name)
	}
}

func TestHookEventNilData(t *testing.T) {
	e := HookEvent{Name: "test", Data: nil}

	// All accessors should handle nil data gracefully
	if e.String("any") != "" {
		t.Error("String should return empty for nil data")
	}
	if e.Int("any") != 0 {
		t.Error("Int should return 0 for nil data")
	}
	if e.Float("any") != 0.0 {
		t.Error("Float should return 0 for nil data")
	}
	if e.Bool("any") {
		t.Error("Bool should return false for nil data")
	}
	if e.Strings("any") != nil {
		t.Error("Strings should return nil for nil data")
	}
	if e.Raw("any") != nil {
		t.Error("Raw should return nil for nil data")
	}
}
