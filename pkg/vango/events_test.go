package vango

import (
	"reflect"
	"testing"
)

func TestDragEvent_DataTransfer(t *testing.T) {
	var d DragEvent
	if d.HasData("text/plain") {
		t.Fatalf("HasData should be false before SetData")
	}
	if got := d.GetData("text/plain"); got != "" {
		t.Fatalf("GetData before SetData = %q, want empty string", got)
	}

	d.SetData("text/plain", "hello")
	if !d.HasData("text/plain") {
		t.Fatalf("HasData should be true after SetData")
	}
	if got := d.GetData("text/plain"); got != "hello" {
		t.Fatalf("GetData = %q, want %q", got, "hello")
	}
}

func TestFormData_BehaviorAndCopies(t *testing.T) {
	f := NewFormData(nil)
	if f.Has("a") {
		t.Fatalf("expected Has(a)=false for empty FormData")
	}
	if got := f.Get("a"); got != "" {
		t.Fatalf("Get(a) = %q, want empty string", got)
	}
	if got := f.GetAll("a"); got != nil {
		t.Fatalf("GetAll(a) = %v, want nil", got)
	}

	f = NewFormData(map[string][]string{
		"q": {"v1", "v2"},
		"x": {},
	})
	if got := f.Get("q"); got != "v1" {
		t.Fatalf("Get(q) = %q, want %q", got, "v1")
	}
	all := f.GetAll("q")
	if !reflect.DeepEqual(all, []string{"v1", "v2"}) {
		t.Fatalf("GetAll(q) = %v, want %v", all, []string{"v1", "v2"})
	}
	all[0] = "mutated"
	if got := f.GetAll("q")[0]; got != "v1" {
		t.Fatalf("GetAll(q) should return a copy; got[0]=%q want %q", got, "v1")
	}

	flat := f.All()
	if flat["q"] != "v1" {
		t.Fatalf("All()[q] = %q, want %q", flat["q"], "v1")
	}
	if _, ok := flat["x"]; ok {
		t.Fatalf("All() should omit keys with empty value lists")
	}

	keys := f.Keys()
	if len(keys) != 2 {
		t.Fatalf("Keys() length = %d, want %d", len(keys), 2)
	}
	seen := map[string]bool{}
	for _, k := range keys {
		seen[k] = true
	}
	if !seen["q"] || !seen["x"] {
		t.Fatalf("Keys() = %v, expected to include %q and %q", keys, "q", "x")
	}

	backCompat := NewFormDataFromSingle(map[string]string{"a": "1"})
	if got := backCompat.Get("a"); got != "1" {
		t.Fatalf("NewFormDataFromSingle Get(a) = %q, want %q", got, "1")
	}
}

func TestHookEvent_TypedGetters_Revert_SetContext(t *testing.T) {
	var called struct {
		name string
	}
	h := HookEvent{
		Data: map[string]any{
			"s": "str",
			"i": int64(7),
			"f": 3.5,
			"b": true,
			"a": []any{"x", 123, "y"},
		},
	}

	if got := h.Get("s"); got != "str" {
		t.Fatalf("Get(s) = %v, want %q", got, "str")
	}
	if got := h.GetString("s"); got != "str" {
		t.Fatalf("GetString(s) = %q, want %q", got, "str")
	}
	if got := h.GetInt("i"); got != 7 {
		t.Fatalf("GetInt(i) = %d, want %d", got, 7)
	}
	if got := h.GetFloat("i"); got != 7.0 {
		t.Fatalf("GetFloat(i) = %v, want %v", got, 7.0)
	}
	if got := h.GetBool("b"); got != true {
		t.Fatalf("GetBool(b) = %v, want true", got)
	}
	if got := h.GetStrings("a"); !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Fatalf("GetStrings(a) = %v, want %v", got, []string{"x", "y"})
	}

	h.SetContext("hid123", func(name string, payload any) {
		called.name = name
	})
	h.Revert()
	if called.name != "revert" {
		t.Fatalf("Revert() dispatch name = %q, want %q", called.name, "revert")
	}
}
