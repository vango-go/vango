package urlstate

import (
	"testing"
	"time"
)

func TestUseURLState(t *testing.T) {
	// 1. Basic usage
	key := "search"
	defaultVal := "default"

	state := Use(key, defaultVal)

	if state.Get() != defaultVal {
		t.Errorf("Expected default value '%s', got '%s'", defaultVal, state.Get())
	}

	// 2. Update value
	newVal := "new query"
	state.Set(newVal)

	if state.Get() != newVal {
		t.Errorf("Expected new value '%s', got '%s'", newVal, state.Get())
	}

	// 3. IsSet
	if !state.IsSet() {
		t.Error("Expected IsSet to be true")
	}

	state.Reset()
	if state.IsSet() {
		t.Error("Expected IsSet to be false after Reset")
	}
}

func TestURLStateOptions(t *testing.T) {
	state := Use("filter", 0).
		Debounce(100 * time.Millisecond).
		Serialize(func(i int) string { return "custom" })

	// Just verify method chaining works and no panic
	if state == nil {
		t.Fatal("Use returned nil")
	}
	state.Set(1)
}

func TestDefaultSerializer(t *testing.T) {
	// Test string
	s := DefaultSerializer("")("hello")
	if s != "hello" {
		t.Errorf("Expected 'hello', got '%s'", s)
	}

	// Test int
	i := DefaultSerializer(0)(123)
	if i != "123" {
		t.Errorf("Expected '123', got '%s'", i)
	}

	// Test bool
	b := DefaultSerializer(false)(true)
	if b != "true" {
		t.Errorf("Expected 'true', got '%s'", b)
	}
}

func TestDefaultDeserializer(t *testing.T) {
	// Test string
	s := DefaultDeserializer("")("hello")
	if s != "hello" {
		t.Errorf("Expected 'hello', got '%s'", s)
	}

	// Test int
	i := DefaultDeserializer(0)("123")
	if i != 123 {
		t.Errorf("Expected 123, got %v", i)
	}

	// Test bool
	b := DefaultDeserializer(false)("true")
	if b != true {
		t.Errorf("Expected true, got %v", b)
	}
}

func TestHashState(t *testing.T) {
	h := UseHash("default")

	if h.Get() != "default" {
		t.Errorf("Expected 'default', got '%s'", h.Get())
	}

	h.Set("new-hash")
	if h.Get() != "new-hash" {
		t.Errorf("Expected 'new-hash', got '%s'", h.Get())
	}
}

func TestURLStateReplace(t *testing.T) {
	state := Use("key", "default")

	state.Replace("replaced")

	if state.Get() != "replaced" {
		t.Errorf("Expected 'replaced', got '%s'", state.Get())
	}
}

func TestURLStateDebounce(t *testing.T) {
	state := Use("debounced", "").Debounce(50 * time.Millisecond)

	// Set should not immediately update URL (debounced)
	state.Set("value1")
	state.Set("value2")
	state.Set("value3")

	// Only the last value should be "queued"
	if state.Get() != "value3" {
		t.Errorf("Expected 'value3', got '%s'", state.Get())
	}

	// Wait for debounce
	time.Sleep(100 * time.Millisecond)
}

func TestURLStateCustomSerializer(t *testing.T) {
	state := Use("custom", 0).
		Serialize(func(i int) string {
			return "prefix_" + DefaultSerializer(0)(i)
		})

	state.Set(42)
	// Serializer is used for URL, not internal state
	if state.Get() != 42 {
		t.Errorf("Expected 42, got %d", state.Get())
	}
}

func TestURLStateCustomDeserializer(t *testing.T) {
	state := Use("custom", 0).
		Deserialize(func(s string) int {
			return 999 // Always return 999 for testing
		})

	// Deserializer would be used when reading from URL
	// For now, just verify chaining works
	if state == nil {
		t.Fatal("Deserialize should return state for chaining")
	}
}

func TestDefaultSerializerFloat(t *testing.T) {
	f := DefaultSerializer(0.0)(3.14)
	if f != "3.14" {
		t.Errorf("Expected '3.14', got '%s'", f)
	}
}

func TestDefaultSerializerSlice(t *testing.T) {
	slice := []string{"a", "b", "c"}
	s := DefaultSerializer(slice)(slice)

	// Should be JSON encoded
	if s == "" {
		t.Error("Expected non-empty serialized slice")
	}
}

func TestDefaultDeserializerFloat(t *testing.T) {
	f := DefaultDeserializer(0.0)("3.14")
	if f != 3.14 {
		t.Errorf("Expected 3.14, got %v", f)
	}
}

func TestDefaultDeserializerInvalidInt(t *testing.T) {
	// Invalid int should return zero value
	i := DefaultDeserializer(0)("not-a-number")
	if i != 0 {
		t.Errorf("Expected 0 for invalid int, got %d", i)
	}
}

func TestDefaultDeserializerInvalidFloat(t *testing.T) {
	// Invalid float should return zero value
	f := DefaultDeserializer(0.0)("not-a-float")
	if f != 0.0 {
		t.Errorf("Expected 0.0 for invalid float, got %f", f)
	}
}

func TestDefaultDeserializerInvalidBool(t *testing.T) {
	// Invalid bool should return false
	b := DefaultDeserializer(false)("not-a-bool")
	if b != false {
		t.Errorf("Expected false for invalid bool, got %v", b)
	}
}

func TestDefaultDeserializerSlice(t *testing.T) {
	defaultSlice := []string{}
	result := DefaultDeserializer(defaultSlice)(`["a","b","c"]`)

	if len(result) != 3 {
		t.Errorf("Expected slice of length 3, got %d", len(result))
	}
}

func TestDefaultDeserializerStringSlice(t *testing.T) {
	defaultSlice := []string{}
	// []string uses comma-separated parsing, not JSON
	result := DefaultDeserializer(defaultSlice)("a,b,c")

	if len(result) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Errorf("Expected [a,b,c], got %v", result)
	}

	// Single value becomes single-element slice
	single := DefaultDeserializer(defaultSlice)("single")
	if len(single) != 1 || single[0] != "single" {
		t.Errorf("Expected [single], got %v", single)
	}
}

func TestURLStateIsSetAfterSet(t *testing.T) {
	state := Use("test", "default")

	if state.IsSet() {
		t.Error("Should not be set initially")
	}

	state.Set("something")
	if !state.IsSet() {
		t.Error("Should be set after Set")
	}

	state.Set("default")
	if state.IsSet() {
		t.Error("Should not be set when equal to default")
	}
}

func TestURLStateMethodChaining(t *testing.T) {
	state := Use("chain", "").
		Debounce(100 * time.Millisecond).
		Serialize(func(s string) string { return s }).
		Deserialize(func(s string) string { return s })

	if state == nil {
		t.Fatal("Method chaining should work")
	}
}

func TestUseWithDifferentTypes(t *testing.T) {
	// Test with int
	intState := Use("intKey", 0)
	intState.Set(42)
	if intState.Get() != 42 {
		t.Errorf("Int state: expected 42, got %d", intState.Get())
	}

	// Test with bool
	boolState := Use("boolKey", false)
	boolState.Set(true)
	if !boolState.Get() {
		t.Error("Bool state: expected true")
	}

	// Test with struct
	type Config struct {
		Name string
		Val  int
	}
	structState := Use("structKey", Config{})
	structState.Set(Config{Name: "test", Val: 123})
	if structState.Get().Name != "test" {
		t.Errorf("Struct state: expected 'test', got '%s'", structState.Get().Name)
	}
}

func TestURLStateReset(t *testing.T) {
	state := Use("reset", "initial")

	state.Set("modified")
	if state.Get() != "modified" {
		t.Error("Expected modified value")
	}

	state.Reset()
	if state.Get() != "initial" {
		t.Errorf("Expected 'initial' after reset, got '%s'", state.Get())
	}
}

func TestHashStateMultipleUpdates(t *testing.T) {
	h := UseHash("")

	h.Set("first")
	h.Set("second")
	h.Set("third")

	if h.Get() != "third" {
		t.Errorf("Expected 'third', got '%s'", h.Get())
	}
}

func TestDefaultSerializerMap(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	s := DefaultSerializer(m)(m)

	// Should be JSON encoded
	if s == "" {
		t.Error("Expected non-empty serialized map")
	}
}

func TestDefaultDeserializerMap(t *testing.T) {
	defaultMap := map[string]int{}
	result := DefaultDeserializer(defaultMap)(`{"a":1,"b":2}`)

	if len(result) != 2 {
		t.Errorf("Expected map of length 2, got %d", len(result))
	}
}
