package vdom

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var booleanAttrs = map[string]bool{
	"allowfullscreen": true,
	"async":           true,
	"autofocus":       true,
	"autoplay":        true,
	"checked":         true,
	"controls":        true,
	"default":         true,
	"defer":           true,
	"disabled":        true,
	"formnovalidate":  true,
	"hidden":          true,
	"inert":           true,
	"ismap":           true,
	"loop":            true,
	"multiple":        true,
	"muted":           true,
	"novalidate":      true,
	"open":            true,
	"playsinline":     true,
	"readonly":        true,
	"required":        true,
	"reversed":        true,
	"selected":        true,
}

// EffectiveAttrs returns the string attributes that should be present on the DOM
// for the given node.
//
// This includes:
// - regular attributes (excluding internal props, event handlers, etc.)
// - derived attributes for event interception (`data-ve` + modifier attrs)
// - derived attributes for hooks (`data-hook` + `data-hook-config`)
// - derived attributes for optimistic updates (`data-optimistic`)
//
// It intentionally omits `data-hid`, which is managed separately via node.HID.
func EffectiveAttrs(node *VNode) map[string]string {
	if node == nil || node.Props == nil {
		return nil
	}

	attrs := make(map[string]string)

	// 1) Regular attributes from Props (excluding internal and handler keys).
	for key, value := range node.Props {
		if value == nil {
			continue
		}

		if key == "key" || key == "dangerouslySetInnerHTML" {
			continue
		}

		// Skip internal props; they may be expanded below.
		if strings.HasPrefix(key, "_") {
			continue
		}

		// Skip event handlers (onclick, oninput, onhook, etc.)
		if isEventHandler(key) {
			continue
		}

		if s, ok := attrValueToString(key, value); ok {
			attrs[key] = s
		}
	}

	// 2) Expand legacy v-hook="Name:{json}" into data-hook/data-hook-config (deprecated).
	if raw, ok := node.Props["v-hook"].(string); ok && raw != "" {
		if name, cfg, ok := parseVHook(raw); ok {
			attrs["data-hook"] = name
			if cfg != "" && cfg != "{}" && cfg != "null" {
				attrs["data-hook-config"] = cfg
			}
		}
	}

	// 3) Expand _hook (render.HookConfig) into data-hook/data-hook-config.
	if hookName, hookCfg, ok := decodeHookConfig(node.Props["_hook"]); ok {
		attrs["data-hook"] = hookName
		if hookCfg != "" && hookCfg != "{}" && hookCfg != "null" {
			attrs["data-hook-config"] = hookCfg
		}
	}

	// 4) Expand _optimistic into data-optimistic JSON attribute.
	if opt, ok := buildOptimisticJSON(node.Props["_optimistic"]); ok {
		attrs["data-optimistic"] = opt
	}

	// 5) Derived event interception attributes.
	ve, mods := buildEventInterceptionAttrs(node.Props)
	if ve != "" {
		attrs["data-ve"] = ve
		for k, v := range mods {
			attrs[k] = v
		}
	} else {
		// Ensure we don't leave stale markers behind when handlers are removed.
		// Modifiers are only meaningful when data-ve is present.
	}

	return attrs
}

func attrValueToString(key string, value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool:
		if booleanAttrs[strings.ToLower(key)] {
			if v {
				return "", true
			}
			return "", false
		}
		if v {
			return "true", true
		}
		return "false", true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		// Preserve current behavior for simple fmt-printable values, but avoid
		// encoding complex structs/maps as attributes unintentionally.
		rv := reflect.ValueOf(value)
		if rv.IsValid() && rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return string(rv.Bytes()), true
		}
		return "", false
	}
}

func parseVHook(raw string) (name string, cfg string, ok bool) {
	idx := strings.Index(raw, ":")
	if idx == -1 {
		return raw, "", true
	}
	return raw[:idx], raw[idx+1:], true
}

func decodeHookConfig(value any) (name string, cfgJSON string, ok bool) {
	if value == nil {
		return "", "", false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", "", false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", "", false
	}

	nameField := rv.FieldByName("Name")
	configField := rv.FieldByName("Config")
	if !nameField.IsValid() || nameField.Kind() != reflect.String {
		return "", "", false
	}
	if !configField.IsValid() {
		return "", "", false
	}

	name = nameField.String()
	if name == "" {
		return "", "", false
	}

	cfgVal := configField.Interface()
	if cfgVal == nil {
		return name, "", true
	}
	b, err := json.Marshal(cfgVal)
	if err != nil {
		return name, "", true
	}
	cfgJSON = string(b)
	return name, cfgJSON, true
}

func buildOptimisticJSON(value any) (string, bool) {
	if value == nil {
		return "", false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", false
	}

	data := make(map[string]string)
	if s := fieldString(rv, "Class"); s != "" {
		data["class"] = s
	}
	if s := fieldString(rv, "Text"); s != "" {
		data["text"] = s
	}
	if s := fieldString(rv, "Attr"); s != "" {
		data["attr"] = s
		if v := fieldString(rv, "Value"); v != "" {
			data["value"] = v
		}
	}

	if len(data) == 0 {
		return "", false
	}

	b, err := json.Marshal(data)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func fieldString(rv reflect.Value, name string) string {
	f := rv.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.String {
		return ""
	}
	return f.String()
}

func buildEventInterceptionAttrs(props Props) (dataVE string, modifierAttrs map[string]string) {
	if props == nil {
		return "", nil
	}

	events := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	modifierAttrs = make(map[string]string)

	for key, value := range props {
		if value == nil {
			continue
		}
		if !isEventHandler(key) {
			continue
		}
		// Hook events are not DOM events; they are sent via HOOK frame.
		if strings.EqualFold(key, "onhook") {
			continue
		}

		eventName := strings.ToLower(key[2:])
		if eventName == "" {
			continue
		}
		if _, ok := seen[eventName]; !ok {
			seen[eventName] = struct{}{}
			events = append(events, eventName)
		}

		// Extract modifier flags from wrappers (best-effort reflection).
		if mods, ok := extractModifierFlags(value); ok {
			if mods.PreventDefault {
				modifierAttrs["data-pd-"+eventName] = "true"
			}
			if mods.StopPropagation {
				modifierAttrs["data-sp-"+eventName] = "true"
			}
			if mods.Self {
				modifierAttrs["data-self-"+eventName] = "true"
			}
			if mods.Once {
				modifierAttrs["data-once-"+eventName] = "true"
			}
			if mods.Passive {
				modifierAttrs["data-passive-"+eventName] = "true"
			}
			if mods.Capture {
				modifierAttrs["data-capture-"+eventName] = "true"
			}
			if mods.DebounceMs > 0 {
				modifierAttrs["data-debounce-"+eventName] = strconv.FormatInt(mods.DebounceMs, 10)
			}
			if mods.ThrottleMs > 0 {
				modifierAttrs["data-throttle-"+eventName] = strconv.FormatInt(mods.ThrottleMs, 10)
			}
		}
	}

	if len(events) == 0 {
		return "", nil
	}

	sort.Strings(events)
	dataVE = strings.Join(events, ",")

	if len(modifierAttrs) == 0 {
		return dataVE, nil
	}

	return dataVE, modifierAttrs
}

type modifierFlags struct {
	PreventDefault  bool
	StopPropagation bool
	Self            bool
	Once            bool
	Passive         bool
	Capture         bool
	DebounceMs      int64
	ThrottleMs      int64
}

func extractModifierFlags(value any) (modifierFlags, bool) {
	var out modifierFlags
	if value == nil {
		return out, false
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return out, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return out, false
	}

	// Recognize vango.ModifiedHandler by shape; best-effort.
	if !rv.FieldByName("Handler").IsValid() {
		return out, false
	}

	out.PreventDefault = fieldBool(rv, "PreventDefault")
	out.StopPropagation = fieldBool(rv, "StopPropagation")
	out.Self = fieldBool(rv, "Self")
	out.Once = fieldBool(rv, "Once")
	out.Passive = fieldBool(rv, "Passive")
	out.Capture = fieldBool(rv, "Capture")

	out.DebounceMs = durationFieldMillis(rv, "Debounce")
	out.ThrottleMs = durationFieldMillis(rv, "Throttle")

	return out, true
}

func fieldBool(rv reflect.Value, name string) bool {
	f := rv.FieldByName(name)
	if !f.IsValid() || f.Kind() != reflect.Bool {
		return false
	}
	return f.Bool()
}

func durationFieldMillis(rv reflect.Value, name string) int64 {
	f := rv.FieldByName(name)
	if !f.IsValid() {
		return 0
	}

	// time.Duration is int64 underneath; handle either.
	switch f.Kind() {
	case reflect.Int64:
		d := time.Duration(f.Int())
		if d <= 0 {
			return 0
		}
		return d.Milliseconds()
	case reflect.Int:
		d := time.Duration(f.Int())
		if d <= 0 {
			return 0
		}
		return d.Milliseconds()
	default:
		// Fall back to fmt string parsing if needed.
		s := fmt.Sprintf("%v", f.Interface())
		if s == "" {
			return 0
		}
		if parsed, err := time.ParseDuration(s); err == nil {
			if parsed <= 0 {
				return 0
			}
			return parsed.Milliseconds()
		}
		return 0
	}
}

