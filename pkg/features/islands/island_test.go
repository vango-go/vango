package islands

import (
	"encoding/json"
	"testing"
)

func TestJSIsland(t *testing.T) {
	props := JSProps{"foo": "bar", "baz": 123}
	node := JSIsland("test-island", "/js/test.js", props)

	if node.Tag != "div" {
		t.Errorf("Expected tag div, got %s", node.Tag)
	}

	if node.Props["data-island"] != "test-island" {
		t.Errorf("Expected data-island test-island, got %v", node.Props["data-island"])
	}

	if node.Props["data-module"] != "/js/test.js" {
		t.Errorf("Expected data-module /js/test.js, got %v", node.Props["data-module"])
	}

	propsStr := node.Props["data-props"].(string)
	var decoded map[string]any
	json.Unmarshal([]byte(propsStr), &decoded)

	if decoded["foo"] != "bar" {
		t.Errorf("Expected foo=bar")
	}
}

func TestPlaceholders(t *testing.T) {
	// calling placeholders to ensure no panic
	SendToIsland("id", nil)
	OnIslandMessage("id", func(m map[string]any) {})
}
