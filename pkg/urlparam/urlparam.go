// Package urlparam provides enhanced URL parameter synchronization for Vango.
//
// URLParam 2.0 supports:
//   - Push/Replace history modes
//   - Debouncing for search inputs
//   - Multiple encoding options (Flat, JSON, Comma)
//   - Complex type support (structs, arrays)
//
// Example:
//
//	// Search input - replaces history, debounced
//	searchQuery := urlparam.Param("q", "", urlparam.Replace, urlparam.Debounce(300*time.Millisecond))
//
//	// Filter struct - flat encoding
//	filters := urlparam.Param("", Filters{}, urlparam.Encoding(urlparam.EncodingFlat))
//
//	// Tag array - comma encoding
//	tags := urlparam.Param("tags", []string{}, urlparam.Encoding(urlparam.EncodingComma))
package urlparam

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vango-go/vango/pkg/vango"
)

// URLMode determines how URL updates are handled.
type URLMode int

const (
	// ModePush adds a new history entry (default behavior).
	ModePush URLMode = iota

	// ModeReplace replaces the current history entry (no back button spam).
	ModeReplace
)

// Encoding specifies how complex types are serialized to URLs.
type Encoding int

const (
	// EncodingFlat serializes structs as flat params: ?cat=tech&sort=asc
	EncodingFlat Encoding = iota

	// EncodingJSON serializes as base64-encoded JSON: ?filter=eyJjYXQiOiJ0ZWNoIn0
	EncodingJSON

	// EncodingComma serializes arrays as comma-separated: ?tags=go,web,api
	EncodingComma
)

// URLParamOption is a functional option for configuring URL parameters.
type URLParamOption interface {
	applyURLParam(*urlParamConfig)
}

// urlParamConfig holds configuration for a URL parameter.
type urlParamConfig struct {
	mode     URLMode
	debounce time.Duration
	encoding Encoding
}

// Mode options as values (not functions) to avoid collision with navigation methods.
var (
	// Push creates a new history entry (default behavior).
	Push URLParamOption = modeOption{mode: ModePush}

	// Replace updates URL without creating history entry (use for filters, search).
	Replace URLParamOption = modeOption{mode: ModeReplace}
)

type modeOption struct {
	mode URLMode
}

func (o modeOption) applyURLParam(c *urlParamConfig) {
	c.mode = o.mode
}

// debounceOption implements URLParamOption.
type debounceOption struct {
	d time.Duration
}

func (o debounceOption) applyURLParam(c *urlParamConfig) {
	c.debounce = o.d
}

// Debounce delays URL updates by the specified duration.
// Use this for search inputs to avoid spamming the history.
//
// Example:
//
//	searchQuery := urlparam.Param("q", "", urlparam.Replace, urlparam.Debounce(300*time.Millisecond))
func Debounce(d time.Duration) URLParamOption {
	return debounceOption{d: d}
}

// encodingOption implements URLParamOption.
type encodingOption struct {
	e Encoding
}

func (o encodingOption) applyURLParam(c *urlParamConfig) {
	c.encoding = o.e
}

// WithEncoding sets the encoding for complex types.
// Spec shows: vango.Encoding(vango.URLEncodingFlat)
// Due to import cycles, the actual call is: urlparam.WithEncoding(urlparam.EncodingFlat)
//
// Example:
//
//	filters := urlparam.Param("", Filters{}, urlparam.WithEncoding(urlparam.EncodingFlat))
func WithEncoding(e Encoding) URLParamOption {
	return encodingOption{e: e}
}

// URLParam represents a reactive value synchronized with URL parameters.
type URLParam[T any] struct {
	key      string
	value    T
	defaults T
	config   urlParamConfig

	signal *vango.Signal[T]

	// Debounce timer
	timerMu sync.Mutex
	timer   *time.Timer

	// Context for navigation (set during component initialization)
	navigate func(params map[string]string, mode URLMode)
}

// Param creates a new URL parameter with the given key and default value.
// If key is empty, the struct fields are used as keys (for flat encoding).
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
//
// On first render, Param hydrates from initial URL params if available.
// On subsequent renders, returns the same URLParam instance (stable identity).
//
// Example:
//
//	// Simple string param
//	query := urlparam.Param("q", "")
//
//	// With options
//	query := urlparam.Param("q", "", urlparam.Replace, urlparam.Debounce(300*time.Millisecond))
//
//	// Struct param with flat encoding
//	type Filters struct {
//	    Category string `url:"cat"`
//	    SortBy   string `url:"sort"`
//	}
//	filters := urlparam.Param("", Filters{}, urlparam.WithEncoding(urlparam.EncodingFlat))
func Param[T any](key string, defaultValue T, opts ...URLParamOption) *URLParam[T] {
	// Track hook call for dev-mode order validation
	vango.TrackHook(vango.HookURLParam)

	// Use hook slot for stable identity across renders
	slot := vango.UseHookSlot()
	var u *URLParam[T]
	first := false
	if slot != nil {
		existing, ok := slot.(*URLParam[T])
		if !ok {
			panic("vango: hook slot type mismatch for URLParam")
		}
		u = existing
	} else {
		first = true
		u = &URLParam[T]{}
		vango.SetHookSlot(u)
	}

	var config urlParamConfig
	if first {
		for _, opt := range opts {
			opt.applyURLParam(&config)
		}
	} else {
		config = u.config
	}

	// Determine initial value from URL or default
	// This is NOT a reactive write - we compute initial value before signal creation
	initial := defaultValue
	if first {
		if initialCtx := vango.GetContext(InitialParamsKey); initialCtx != nil {
			if state, ok := initialCtx.(*InitialURLState); ok {
				if params := state.Consume(); params != nil {
					// Try to parse the initial value from URL params
					temp := &URLParam[T]{key: key, config: config}
					if parsed, err := temp.deserialize(params); err == nil {
						initial = parsed
					}
				}
			}
		}
	}

	if first {
		u.key = key
		u.defaults = defaultValue
		u.config = config
	}

	// Signals are hook-slot stabilized when called during render
	u.signal = vango.NewSignal(initial) // Uses computed initial, no Set() call

	// Wire navigator from context for URL updates (first render only)
	if first {
		if navCtx := vango.GetContext(NavigatorKey); navCtx != nil {
			if nav, ok := navCtx.(*Navigator); ok {
				u.SetNavigator(nav.Navigate)
			}
		}
	}

	return u
}

// Get returns the current value.
// In a tracking context, this will subscribe the listener to changes.
func (u *URLParam[T]) Get() T {
	return u.signal.Get()
}

// Peek returns the current value without subscribing.
func (u *URLParam[T]) Peek() T {
	return u.signal.Peek()
}

// Set updates the value and synchronizes with the URL.
func (u *URLParam[T]) Set(value T) {
	u.signal.Set(value)
	u.scheduleURLUpdate(value)
}

// Update atomically reads and updates the value.
func (u *URLParam[T]) Update(fn func(T) T) {
	u.signal.Update(fn)
	u.scheduleURLUpdate(u.signal.Peek())
}

// Reset resets the value to the default.
func (u *URLParam[T]) Reset() {
	u.Set(u.defaults)
}

// scheduleURLUpdate schedules a URL update, respecting debounce settings.
func (u *URLParam[T]) scheduleURLUpdate(value T) {
	if u.config.debounce > 0 {
		u.timerMu.Lock()
		defer u.timerMu.Unlock()

		if u.timer != nil {
			u.timer.Stop()
		}
		u.timer = time.AfterFunc(u.config.debounce, func() {
			u.performURLUpdate(value)
		})
		return
	}

	u.performURLUpdate(value)
}

// performURLUpdate performs the actual URL update.
func (u *URLParam[T]) performURLUpdate(value T) {
	params := u.serialize(value)
	if u.navigate != nil {
		u.navigate(params, u.config.mode)
	}
}

// serialize converts the value to URL parameters based on encoding.
func (u *URLParam[T]) serialize(value T) map[string]string {
	result := make(map[string]string)

	switch u.config.encoding {
	case EncodingFlat:
		u.serializeFlat(value, result)
	case EncodingJSON:
		u.serializeJSON(value, result)
	case EncodingComma:
		u.serializeComma(value, result)
	default:
		// Default: simple string conversion
		result[u.key] = fmt.Sprintf("%v", value)
	}

	return result
}

// serializeFlat serializes a struct as flat URL params.
func (u *URLParam[T]) serializeFlat(value T, result map[string]string) {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		// Not a struct, use simple serialization
		result[u.key] = fmt.Sprintf("%v", value)
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Get URL tag or use lowercase field name
		key := field.Tag.Get("url")
		if key == "" {
			key = strings.ToLower(field.Name)
		}
		if key == "-" {
			continue
		}

		// Skip zero values
		if isZero(fieldValue) {
			continue
		}

		result[key] = formatValue(fieldValue)
	}
}

// serializeJSON serializes as base64-encoded JSON.
func (u *URLParam[T]) serializeJSON(value T, result map[string]string) {
	data, err := json.Marshal(value)
	if err != nil {
		result[u.key] = ""
		return
	}
	result[u.key] = base64.RawURLEncoding.EncodeToString(data)
}

// serializeComma serializes arrays as comma-separated values.
func (u *URLParam[T]) serializeComma(value T, result map[string]string) {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		result[u.key] = fmt.Sprintf("%v", value)
		return
	}

	var parts []string
	for i := 0; i < v.Len(); i++ {
		parts = append(parts, formatValue(v.Index(i)))
	}
	result[u.key] = strings.Join(parts, ",")
}

// SetFromURL updates the value from URL parameters.
// This is called during initialization to sync with the current URL.
func (u *URLParam[T]) SetFromURL(params map[string]string) error {
	value, err := u.deserialize(params)
	if err != nil {
		return err
	}
	u.signal.Set(value)
	return nil
}

// deserialize converts URL parameters back to the value type.
func (u *URLParam[T]) deserialize(params map[string]string) (T, error) {
	var result T

	switch u.config.encoding {
	case EncodingFlat:
		return u.deserializeFlat(params)
	case EncodingJSON:
		return u.deserializeJSON(params)
	case EncodingComma:
		return u.deserializeComma(params)
	default:
		// Simple string conversion
		if val, ok := params[u.key]; ok {
			return u.parseValue(val)
		}
		return result, nil
	}
}

// deserializeFlat deserializes flat URL params to a struct.
func (u *URLParam[T]) deserializeFlat(params map[string]string) (T, error) {
	var result T
	v := reflect.ValueOf(&result).Elem()

	if v.Kind() != reflect.Struct {
		// Not a struct, use simple deserialization
		if val, ok := params[u.key]; ok {
			return u.parseValue(val)
		}
		return result, nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get URL tag or use lowercase field name
		key := field.Tag.Get("url")
		if key == "" {
			key = strings.ToLower(field.Name)
		}
		if key == "-" {
			continue
		}

		if val, ok := params[key]; ok {
			if err := setFieldValue(fieldValue, val); err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

// deserializeJSON deserializes base64-encoded JSON.
func (u *URLParam[T]) deserializeJSON(params map[string]string) (T, error) {
	var result T

	val, ok := params[u.key]
	if !ok || val == "" {
		return result, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}

// deserializeComma deserializes comma-separated values to a slice.
func (u *URLParam[T]) deserializeComma(params map[string]string) (T, error) {
	var result T

	val, ok := params[u.key]
	if !ok || val == "" {
		return result, nil
	}

	parts := strings.Split(val, ",")

	// Check if T is a slice type
	v := reflect.ValueOf(&result).Elem()
	if v.Kind() != reflect.Slice {
		return result, fmt.Errorf("comma encoding requires slice type")
	}

	elemType := v.Type().Elem()
	slice := reflect.MakeSlice(v.Type(), len(parts), len(parts))

	for i, part := range parts {
		elem := slice.Index(i)
		if err := setFieldValue(elem, part); err != nil {
			return result, err
		}
		// Handle if elemType is different from default element handling
		_ = elemType
	}

	v.Set(slice)
	return result, nil
}

// parseValue parses a string to the value type T.
func (u *URLParam[T]) parseValue(s string) (T, error) {
	var result T
	v := reflect.ValueOf(&result).Elem()
	if err := setFieldValue(v, s); err != nil {
		return result, err
	}
	return result, nil
}

// SetNavigator sets the navigation function for URL updates.
// This is called during component initialization.
func (u *URLParam[T]) SetNavigator(fn func(params map[string]string, mode URLMode)) {
	u.navigate = fn
}

// Helper functions

func isZero(v reflect.Value) bool {
	return v.IsZero()
}

func formatValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func setFieldValue(v reflect.Value, s string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
	default:
		return fmt.Errorf("unsupported type: %v", v.Kind())
	}
	return nil
}
