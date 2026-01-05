package standard

import (
	"encoding/json"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/render"
)

func TestSortable(t *testing.T) {
	config := SortableConfig{
		Group:      "items",
		Animation:  150,
		GhostClass: "ghost",
		Handle:     ".handle",
		Disabled:   false,
	}

	attr := Sortable(config)

	if attr.Key != "_hook" {
		t.Errorf("Expected key '_hook', got '%s'", attr.Key)
	}

	hookConfig, ok := attr.Value.(render.HookConfig)
	if !ok {
		t.Fatalf("Expected value to be render.HookConfig, got %T", attr.Value)
	}

	if hookConfig.Name != "Sortable" {
		t.Errorf("Expected Name 'Sortable', got '%s'", hookConfig.Name)
	}

	// Marshal and check the config contains expected values
	configJSON, err := json.Marshal(hookConfig.Config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configStr := string(configJSON)
	if !contains(configStr, "items") {
		t.Error("Expected config to contain group name 'items'")
	}
	if !contains(configStr, "150") {
		t.Error("Expected config to contain animation value '150'")
	}
	if !contains(configStr, "ghost") {
		t.Error("Expected config to contain ghostClass 'ghost'")
	}
}

func TestSortableDefaults(t *testing.T) {
	// Empty config should still work
	attr := Sortable(SortableConfig{})

	if attr.Key != "_hook" {
		t.Errorf("Expected key '_hook', got '%s'", attr.Key)
	}

	hookConfig, ok := attr.Value.(render.HookConfig)
	if !ok {
		t.Fatalf("Expected value to be render.HookConfig, got %T", attr.Value)
	}

	if hookConfig.Name != "Sortable" {
		t.Errorf("Expected Name 'Sortable', got '%s'", hookConfig.Name)
	}
}

func TestDraggable(t *testing.T) {
	config := DraggableConfig{
		Axis:   "x",
		Handle: ".drag-handle",
		Revert: true,
	}

	attr := Draggable(config)

	if attr.Key != "_hook" {
		t.Errorf("Expected key '_hook', got '%s'", attr.Key)
	}

	hookConfig, ok := attr.Value.(render.HookConfig)
	if !ok {
		t.Fatalf("Expected value to be render.HookConfig, got %T", attr.Value)
	}

	if hookConfig.Name != "Draggable" {
		t.Errorf("Expected Name 'Draggable', got '%s'", hookConfig.Name)
	}

	// Marshal and check config values
	configJSON, err := json.Marshal(hookConfig.Config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configStr := string(configJSON)
	if !contains(configStr, `"x"`) {
		t.Error("Expected config to contain axis 'x'")
	}
	if !contains(configStr, ".drag-handle") {
		t.Error("Expected config to contain handle")
	}
	if !contains(configStr, "true") {
		t.Error("Expected config to contain revert true")
	}
}

func TestDraggableAxisY(t *testing.T) {
	config := DraggableConfig{
		Axis: "y",
	}

	attr := Draggable(config)
	hookConfig := attr.Value.(render.HookConfig)

	configJSON, _ := json.Marshal(hookConfig.Config)
	if !contains(string(configJSON), `"y"`) {
		t.Error("Expected config to contain axis 'y'")
	}
}

func TestDraggableBoth(t *testing.T) {
	config := DraggableConfig{
		Axis: "both",
	}

	attr := Draggable(config)
	hookConfig := attr.Value.(render.HookConfig)

	configJSON, _ := json.Marshal(hookConfig.Config)
	if !contains(string(configJSON), `"both"`) {
		t.Error("Expected config to contain axis 'both'")
	}
}

func TestTooltip(t *testing.T) {
	config := TooltipConfig{
		Content:   "Hello World",
		Placement: "top",
		Delay:     300,
		Trigger:   "hover",
	}

	attr := Tooltip(config)

	if attr.Key != "_hook" {
		t.Errorf("Expected key '_hook', got '%s'", attr.Key)
	}

	hookConfig, ok := attr.Value.(render.HookConfig)
	if !ok {
		t.Fatalf("Expected value to be render.HookConfig, got %T", attr.Value)
	}

	if hookConfig.Name != "Tooltip" {
		t.Errorf("Expected Name 'Tooltip', got '%s'", hookConfig.Name)
	}

	// Marshal and check config values
	configJSON, err := json.Marshal(hookConfig.Config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	configStr := string(configJSON)
	if !contains(configStr, "Hello World") {
		t.Error("Expected config to contain content")
	}
	if !contains(configStr, "top") {
		t.Error("Expected config to contain placement 'top'")
	}
	if !contains(configStr, "300") {
		t.Error("Expected config to contain delay")
	}
	if !contains(configStr, "hover") {
		t.Error("Expected config to contain trigger")
	}
}

func TestTooltipPlacements(t *testing.T) {
	placements := []string{"top", "bottom", "left", "right"}

	for _, placement := range placements {
		t.Run(placement, func(t *testing.T) {
			config := TooltipConfig{
				Content:   "Test",
				Placement: placement,
			}

			attr := Tooltip(config)
			hookConfig := attr.Value.(render.HookConfig)

			configJSON, _ := json.Marshal(hookConfig.Config)
			if !contains(string(configJSON), placement) {
				t.Errorf("Expected config to contain placement '%s'", placement)
			}
		})
	}
}

func TestTooltipTriggers(t *testing.T) {
	triggers := []string{"hover", "click", "focus"}

	for _, trigger := range triggers {
		t.Run(trigger, func(t *testing.T) {
			config := TooltipConfig{
				Content: "Test",
				Trigger: trigger,
			}

			attr := Tooltip(config)
			hookConfig := attr.Value.(render.HookConfig)

			configJSON, _ := json.Marshal(hookConfig.Config)
			if !contains(string(configJSON), trigger) {
				t.Errorf("Expected config to contain trigger '%s'", trigger)
			}
		})
	}
}

func TestDropdown(t *testing.T) {
	config := DropdownConfig{
		CloseOnEscape: true,
		CloseOnClick:  true,
	}

	attr := Dropdown(config)

	if attr.Key != "_hook" {
		t.Errorf("Expected key '_hook', got '%s'", attr.Key)
	}

	hookConfig, ok := attr.Value.(render.HookConfig)
	if !ok {
		t.Fatalf("Expected value to be render.HookConfig, got %T", attr.Value)
	}

	if hookConfig.Name != "Dropdown" {
		t.Errorf("Expected Name 'Dropdown', got '%s'", hookConfig.Name)
	}

	configJSON, _ := json.Marshal(hookConfig.Config)
	// Both should be true
	if !contains(string(configJSON), "true") {
		t.Error("Expected config to contain true for close options")
	}
}

func TestDropdownFalseOptions(t *testing.T) {
	config := DropdownConfig{
		CloseOnEscape: false,
		CloseOnClick:  false,
	}

	attr := Dropdown(config)
	hookConfig := attr.Value.(render.HookConfig)

	configJSON, _ := json.Marshal(hookConfig.Config)
	// Both should be false
	if !contains(string(configJSON), "false") {
		t.Error("Expected config to contain false for close options")
	}
}

func TestDropdownMixedOptions(t *testing.T) {
	config := DropdownConfig{
		CloseOnEscape: true,
		CloseOnClick:  false,
	}

	attr := Dropdown(config)
	hookConfig := attr.Value.(render.HookConfig)

	configJSON, _ := json.Marshal(hookConfig.Config)
	configStr := string(configJSON)
	// Should contain both true and false
	if !contains(configStr, "true") || !contains(configStr, "false") {
		t.Error("Expected config to contain both true and false")
	}
}

func TestAllHooksReturnValidAttr(t *testing.T) {
	attrs := []struct {
		name string
		attr interface {
			IsEmpty() bool
		}
	}{
		{"Sortable", Sortable(SortableConfig{})},
		{"Draggable", Draggable(DraggableConfig{})},
		{"Tooltip", Tooltip(TooltipConfig{Content: "test"})},
		{"Dropdown", Dropdown(DropdownConfig{})},
	}

	for _, tt := range attrs {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.IsEmpty() {
				t.Errorf("%s() returned empty attr", tt.name)
			}
		})
	}
}

// contains is a helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsRune(s, substr)))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
