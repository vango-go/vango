package standard

import (
	"strings"
	"testing"
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

	if attr.Key != "v-hook" {
		t.Errorf("Expected key 'v-hook', got '%s'", attr.Key)
	}

	val := attr.Value.(string)
	if !strings.HasPrefix(val, "Sortable:") {
		t.Errorf("Expected prefix 'Sortable:', got '%s'", val)
	}

	// Verify it contains expected config values
	if !strings.Contains(val, "items") {
		t.Error("Expected value to contain group name 'items'")
	}
	if !strings.Contains(val, "150") {
		t.Error("Expected value to contain animation value '150'")
	}
	if !strings.Contains(val, "ghost") {
		t.Error("Expected value to contain ghostClass 'ghost'")
	}
}

func TestSortableDefaults(t *testing.T) {
	// Empty config should still work
	attr := Sortable(SortableConfig{})

	if attr.Key != "v-hook" {
		t.Errorf("Expected key 'v-hook', got '%s'", attr.Key)
	}

	val := attr.Value.(string)
	if !strings.HasPrefix(val, "Sortable:") {
		t.Error("Expected Sortable prefix")
	}
}

func TestDraggable(t *testing.T) {
	config := DraggableConfig{
		Axis:   "x",
		Handle: ".drag-handle",
		Revert: true,
	}

	attr := Draggable(config)

	if attr.Key != "v-hook" {
		t.Errorf("Expected key 'v-hook', got '%s'", attr.Key)
	}

	val := attr.Value.(string)
	if !strings.HasPrefix(val, "Draggable:") {
		t.Errorf("Expected prefix 'Draggable:', got '%s'", val)
	}

	// Verify config values
	if !strings.Contains(val, `"x"`) {
		t.Error("Expected value to contain axis 'x'")
	}
	if !strings.Contains(val, ".drag-handle") {
		t.Error("Expected value to contain handle")
	}
	if !strings.Contains(val, "true") {
		t.Error("Expected value to contain revert true")
	}
}

func TestDraggableAxisY(t *testing.T) {
	config := DraggableConfig{
		Axis: "y",
	}

	attr := Draggable(config)
	val := attr.Value.(string)

	if !strings.Contains(val, `"y"`) {
		t.Error("Expected value to contain axis 'y'")
	}
}

func TestDraggableBoth(t *testing.T) {
	config := DraggableConfig{
		Axis: "both",
	}

	attr := Draggable(config)
	val := attr.Value.(string)

	if !strings.Contains(val, `"both"`) {
		t.Error("Expected value to contain axis 'both'")
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

	if attr.Key != "v-hook" {
		t.Errorf("Expected key 'v-hook', got '%s'", attr.Key)
	}

	val := attr.Value.(string)
	if !strings.HasPrefix(val, "Tooltip:") {
		t.Errorf("Expected prefix 'Tooltip:', got '%s'", val)
	}

	// Verify config values
	if !strings.Contains(val, "Hello World") {
		t.Error("Expected value to contain content")
	}
	if !strings.Contains(val, "top") {
		t.Error("Expected value to contain placement 'top'")
	}
	if !strings.Contains(val, "300") {
		t.Error("Expected value to contain delay")
	}
	if !strings.Contains(val, "hover") {
		t.Error("Expected value to contain trigger")
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
			val := attr.Value.(string)

			if !strings.Contains(val, placement) {
				t.Errorf("Expected value to contain placement '%s'", placement)
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
			val := attr.Value.(string)

			if !strings.Contains(val, trigger) {
				t.Errorf("Expected value to contain trigger '%s'", trigger)
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

	if attr.Key != "v-hook" {
		t.Errorf("Expected key 'v-hook', got '%s'", attr.Key)
	}

	val := attr.Value.(string)
	if !strings.HasPrefix(val, "Dropdown:") {
		t.Errorf("Expected prefix 'Dropdown:', got '%s'", val)
	}

	// Both should be true
	if !strings.Contains(val, "true") {
		t.Error("Expected value to contain true for close options")
	}
}

func TestDropdownFalseOptions(t *testing.T) {
	config := DropdownConfig{
		CloseOnEscape: false,
		CloseOnClick:  false,
	}

	attr := Dropdown(config)
	val := attr.Value.(string)

	// Both should be false
	if !strings.Contains(val, "false") {
		t.Error("Expected value to contain false for close options")
	}
}

func TestDropdownMixedOptions(t *testing.T) {
	config := DropdownConfig{
		CloseOnEscape: true,
		CloseOnClick:  false,
	}

	attr := Dropdown(config)
	val := attr.Value.(string)

	// Should contain both true and false
	if !strings.Contains(val, "true") || !strings.Contains(val, "false") {
		t.Error("Expected value to contain both true and false")
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
