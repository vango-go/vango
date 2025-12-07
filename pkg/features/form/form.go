package form

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/vango-dev/vango/v2/pkg/vango"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// Form is a type-safe form handler with validation support.
// It provides reactive form state management, field-level validation,
// and automatic binding to Go structs via struct tags.
type Form[T any] struct {
	initial    T
	values     *vango.Signal[T]
	errors     *vango.Signal[map[string][]string]
	touched    *vango.Signal[map[string]bool]
	dirty      *vango.Signal[map[string]bool]
	submitting *vango.Signal[bool]
	validators map[string][]Validator
	fieldMeta  map[string]fieldMeta

	mu sync.RWMutex
}

// fieldMeta stores metadata extracted from struct tags.
type fieldMeta struct {
	formTag     string
	validateTag string
	fieldType   reflect.Type
	fieldIndex  int
}

// UseForm creates a new Form bound to the given struct type.
// The initial value is used as the default state and for Reset().
func UseForm[T any](initial T) *Form[T] {
	f := &Form[T]{
		initial:    initial,
		values:     vango.NewSignal(initial),
		errors:     vango.NewSignal(make(map[string][]string)),
		touched:    vango.NewSignal(make(map[string]bool)),
		dirty:      vango.NewSignal(make(map[string]bool)),
		submitting: vango.NewSignal(false),
		validators: make(map[string][]Validator),
		fieldMeta:  make(map[string]fieldMeta),
	}

	// Parse struct tags to extract field metadata and validators
	f.parseStructTags(reflect.TypeOf(initial), "")

	return f
}

// parseStructTags extracts form and validate tags from struct fields.
func (f *Form[T]) parseStructTags(t reflect.Type, prefix string) {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get form tag or use lowercase field name
		formTag := field.Tag.Get("form")
		if formTag == "" {
			formTag = strings.ToLower(field.Name)
		}
		if formTag == "-" {
			continue
		}

		// Build full path with prefix
		fullPath := formTag
		if prefix != "" {
			fullPath = prefix + "." + formTag
		}

		validateTag := field.Tag.Get("validate")

		f.fieldMeta[fullPath] = fieldMeta{
			formTag:     formTag,
			validateTag: validateTag,
			fieldType:   field.Type,
			fieldIndex:  i,
		}

		// Parse validate tag into validators
		if validateTag != "" {
			f.validators[fullPath] = parseValidateTag(validateTag, field.Type)
		}

		// Recurse for nested structs (but not slices of structs)
		if field.Type.Kind() == reflect.Struct {
			f.parseStructTags(field.Type, fullPath)
		}
	}
}

// Values returns a copy of the current form values as the typed struct.
func (f *Form[T]) Values() T {
	return f.values.Get()
}

// Get returns the value of a single field by name.
// Supports dot notation for nested fields (e.g., "address.city").
func (f *Form[T]) Get(field string) any {
	values := f.values.Get()
	return getFieldValue(reflect.ValueOf(values), field)
}

// GetString returns a field value as a string.
func (f *Form[T]) GetString(field string) string {
	v := f.Get(field)
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// GetInt returns a field value as an int.
func (f *Form[T]) GetInt(field string) int {
	v := f.Get(field)
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// GetBool returns a field value as a bool.
func (f *Form[T]) GetBool(field string) bool {
	v := f.Get(field)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// Set updates a single field value.
// Supports dot notation for nested fields.
func (f *Form[T]) Set(field string, value any) {
	f.values.Update(func(current T) T {
		v := reflect.ValueOf(&current).Elem()
		setFieldValue(v, field, value)
		return current
	})

	// Mark field as dirty
	f.dirty.Update(func(m map[string]bool) map[string]bool {
		newMap := make(map[string]bool, len(m)+1)
		for k, v := range m {
			newMap[k] = v
		}
		newMap[field] = true
		return newMap
	})
}

// SetValues replaces all form values with the given struct.
func (f *Form[T]) SetValues(values T) {
	f.values.Set(values)

	// Mark all fields as dirty
	f.dirty.Update(func(_ map[string]bool) map[string]bool {
		newMap := make(map[string]bool, len(f.fieldMeta))
		for field := range f.fieldMeta {
			newMap[field] = true
		}
		return newMap
	})
}

// Reset restores the form to its initial values and clears errors.
func (f *Form[T]) Reset() {
	f.values.Set(f.initial)
	f.errors.Set(make(map[string][]string))
	f.touched.Set(make(map[string]bool))
	f.dirty.Set(make(map[string]bool))
	f.submitting.Set(false)
}

// Validate runs all validators and returns true if the form is valid.
// Validation errors are stored and can be accessed via Errors() or FieldErrors().
func (f *Form[T]) Validate() bool {
	allErrors := make(map[string][]string)

	for field, validators := range f.validators {
		value := f.Get(field)
		var fieldErrors []string

		for _, v := range validators {
			if err := v.Validate(value); err != nil {
				fieldErrors = append(fieldErrors, err.Error())
			}
		}

		if len(fieldErrors) > 0 {
			allErrors[field] = fieldErrors
		}
	}

	f.errors.Set(allErrors)
	return len(allErrors) == 0
}

// ValidateField validates a single field and returns true if valid.
func (f *Form[T]) ValidateField(field string) bool {
	validators, ok := f.validators[field]
	if !ok {
		return true
	}

	value := f.Get(field)
	var fieldErrors []string

	for _, v := range validators {
		if err := v.Validate(value); err != nil {
			fieldErrors = append(fieldErrors, err.Error())
		}
	}

	// Update errors map
	f.errors.Update(func(m map[string][]string) map[string][]string {
		newMap := make(map[string][]string, len(m))
		for k, v := range m {
			newMap[k] = v
		}
		if len(fieldErrors) > 0 {
			newMap[field] = fieldErrors
		} else {
			delete(newMap, field)
		}
		return newMap
	})

	// Mark as touched
	f.touched.Update(func(m map[string]bool) map[string]bool {
		newMap := make(map[string]bool, len(m)+1)
		for k, v := range m {
			newMap[k] = v
		}
		newMap[field] = true
		return newMap
	})

	return len(fieldErrors) == 0
}

// Errors returns all validation errors keyed by field name.
func (f *Form[T]) Errors() map[string][]string {
	return f.errors.Get()
}

// FieldErrors returns validation errors for a specific field.
func (f *Form[T]) FieldErrors(field string) []string {
	errors := f.errors.Get()
	return errors[field]
}

// HasError returns true if the field has any validation errors.
func (f *Form[T]) HasError(field string) bool {
	errors := f.errors.Get()
	return len(errors[field]) > 0
}

// ClearErrors removes all validation errors.
func (f *Form[T]) ClearErrors() {
	f.errors.Set(make(map[string][]string))
}

// SetError manually sets an error message for a field.
func (f *Form[T]) SetError(field string, msg string) {
	f.errors.Update(func(m map[string][]string) map[string][]string {
		newMap := make(map[string][]string, len(m)+1)
		for k, v := range m {
			newMap[k] = v
		}
		newMap[field] = append(newMap[field], msg)
		return newMap
	})
}

// IsDirty returns true if any field has been modified.
func (f *Form[T]) IsDirty() bool {
	dirty := f.dirty.Get()
	return len(dirty) > 0
}

// FieldDirty returns true if the specific field has been modified.
func (f *Form[T]) FieldDirty(field string) bool {
	dirty := f.dirty.Get()
	return dirty[field]
}

// IsSubmitting returns true if the form is currently being submitted.
func (f *Form[T]) IsSubmitting() bool {
	return f.submitting.Get()
}

// SetSubmitting sets the submitting state.
func (f *Form[T]) SetSubmitting(submitting bool) {
	f.submitting.Set(submitting)
}

// IsValid returns true if there are no validation errors.
func (f *Form[T]) IsValid() bool {
	errors := f.errors.Get()
	return len(errors) == 0
}

// IsTouched returns true if the field has been interacted with.
func (f *Form[T]) IsTouched(field string) bool {
	touched := f.touched.Get()
	return touched[field]
}

// Field wraps an input element with value binding and error display.
// It automatically binds the input value to the form field and shows errors.
func (f *Form[T]) Field(name string, input *vdom.VNode, validators ...Validator) *vdom.VNode {
	// Register additional validators if provided
	if len(validators) > 0 {
		f.mu.Lock()
		f.validators[name] = append(f.validators[name], validators...)
		f.mu.Unlock()
	}

	// Get current value and errors
	value := f.Get(name)
	errors := f.FieldErrors(name)
	hasError := len(errors) > 0

	// Clone input with value binding
	if input.Props == nil {
		input.Props = make(vdom.Props)
	}

	// Set value and name
	input.Props["value"] = value
	input.Props["name"] = name

	// Add error class if needed
	if hasError {
		existingClass, _ := input.Props["class"].(string)
		if existingClass != "" {
			input.Props["class"] = existingClass + " field-error"
		} else {
			input.Props["class"] = "field-error"
		}
	}

	// Build the wrapper with error display
	children := make([]any, 0, 2)
	children = append(children, input)

	if hasError {
		errorNodes := make([]*vdom.VNode, 0, len(errors))
		for _, err := range errors {
			errorNodes = append(errorNodes, vdom.Span(
				vdom.Class("field-error-message"),
				vdom.Text(err),
			))
		}
		children = append(children, vdom.Div(
			vdom.Class("field-errors"),
			errorNodes,
		))
	}

	return vdom.Div(
		vdom.Class("field"),
		children,
	)
}

// AddValidators adds validators for a field.
// This allows adding validators programmatically instead of via struct tags.
func (f *Form[T]) AddValidators(field string, validators ...Validator) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.validators[field] = append(f.validators[field], validators...)
}

// getFieldValue gets a nested field value using dot notation.
func getFieldValue(v reflect.Value, path string) any {
	if !v.IsValid() {
		return nil
	}

	// Handle pointer
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	parts := strings.SplitN(path, ".", 2)
	fieldName := parts[0]

	// Find field by form tag first, then by name
	t := v.Type()
	var fieldValue reflect.Value
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("form")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		if tag == fieldName || strings.EqualFold(field.Name, fieldName) {
			fieldValue = v.Field(i)
			break
		}
	}

	if !fieldValue.IsValid() {
		return nil
	}

	if len(parts) == 2 {
		return getFieldValue(fieldValue, parts[1])
	}

	if fieldValue.CanInterface() {
		return fieldValue.Interface()
	}
	return nil
}

// setFieldValue sets a nested field value using dot notation.
func setFieldValue(v reflect.Value, path string, value any) {
	if !v.IsValid() || !v.CanSet() {
		// Try to get addressable value
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
	}

	if v.Kind() != reflect.Struct {
		return
	}

	parts := strings.SplitN(path, ".", 2)
	fieldName := parts[0]

	// Find field by form tag first, then by name
	t := v.Type()
	var fieldValue reflect.Value
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("form")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		if tag == fieldName || strings.EqualFold(field.Name, fieldName) {
			fieldValue = v.Field(i)
			break
		}
	}

	if !fieldValue.IsValid() || !fieldValue.CanSet() {
		return
	}

	if len(parts) == 2 {
		setFieldValue(fieldValue, parts[1], value)
		return
	}

	// Convert and set the value
	newValue := reflect.ValueOf(value)
	if newValue.Type().ConvertibleTo(fieldValue.Type()) {
		fieldValue.Set(newValue.Convert(fieldValue.Type()))
	} else if fieldValue.Kind() == reflect.String && newValue.Kind() == reflect.String {
		fieldValue.SetString(newValue.String())
	}
}
