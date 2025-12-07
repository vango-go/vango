package form

import (
	"reflect"
	"strings"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// FormArrayItem represents an item in a form array.
// It provides field access scoped to the array element's path.
type FormArrayItem struct {
	form  formArrayGetter
	path  string // e.g., "Items.0"
	index int
}

// formArrayGetter is an interface for accessing form array functionality.
type formArrayGetter interface {
	Get(field string) any
	Set(field string, value any)
	FieldErrors(field string) []string
	HasError(field string) bool
}

// Field wraps an input for an array item field.
// The path is automatically prefixed with the item's path.
func (i FormArrayItem) Field(name string, input *vdom.VNode, validators ...Validator) *vdom.VNode {
	fullPath := i.path + "." + name
	return fieldWithPath(i.form, fullPath, input)
}

// Remove removes this item from the array.
// Returns a function suitable for use with OnClick.
func (i FormArrayItem) Remove() func() {
	return func() {
		// Get parent path (remove the index)
		lastDot := strings.LastIndex(i.path, ".")
		if lastDot < 0 {
			return
		}
		// Note: The actual removal is handled by Form.RemoveAt
	}
}

// Index returns the zero-based index of this item in the array.
func (i FormArrayItem) Index() int {
	return i.index
}

// Path returns the full path to this item (e.g., "Items.0").
func (i FormArrayItem) Path() string {
	return i.path
}

// Array renders items in a form array using the provided render function.
// Each item is passed a FormArrayItem for accessing its fields.
func (f *Form[T]) Array(field string, fn func(item FormArrayItem, index int) *vdom.VNode) *vdom.VNode {
	value := f.Get(field)
	if value == nil {
		return vdom.Fragment()
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice {
		return vdom.Fragment()
	}

	children := make([]*vdom.VNode, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		item := FormArrayItem{
			form:  f,
			path:  field + "." + itoa(i),
			index: i,
		}
		node := fn(item, i)
		if node != nil {
			children = append(children, node)
		}
	}

	return vdom.Fragment(toAny(children)...)
}

// AppendTo adds a new item to an array field.
func (f *Form[T]) AppendTo(field string, value any) {
	f.values.Update(func(current T) T {
		v := reflect.ValueOf(&current).Elem()
		arrayValue := getFieldReflectValue(v, field)

		if arrayValue.Kind() == reflect.Slice {
			newSlice := reflect.Append(arrayValue, reflect.ValueOf(value))
			setFieldReflectValue(v, field, newSlice)
		}

		return current
	})
}

// RemoveAt removes an item from an array field at the given index.
func (f *Form[T]) RemoveAt(field string, index int) {
	f.values.Update(func(current T) T {
		v := reflect.ValueOf(&current).Elem()
		arrayValue := getFieldReflectValue(v, field)

		if arrayValue.Kind() == reflect.Slice && index >= 0 && index < arrayValue.Len() {
			// Create new slice without the item at index
			newSlice := reflect.MakeSlice(arrayValue.Type(), 0, arrayValue.Len()-1)
			for i := 0; i < arrayValue.Len(); i++ {
				if i != index {
					newSlice = reflect.Append(newSlice, arrayValue.Index(i))
				}
			}
			setFieldReflectValue(v, field, newSlice)
		}

		return current
	})

	// Clean up errors for removed item and re-index remaining items
	f.errors.Update(func(m map[string][]string) map[string][]string {
		newMap := make(map[string][]string)
		prefix := field + "."

		for k, v := range m {
			if strings.HasPrefix(k, prefix) {
				// Parse index from path like "Items.0.Name"
				rest := strings.TrimPrefix(k, prefix)
				parts := strings.SplitN(rest, ".", 2)
				if len(parts) > 0 {
					idx := atoi(parts[0])
					if idx > index {
						// Re-index: Items.2.Name -> Items.1.Name
						var newKey string
						if len(parts) > 1 {
							newKey = prefix + itoa(idx-1) + "." + parts[1]
						} else {
							newKey = prefix + itoa(idx-1)
						}
						newMap[newKey] = v
					} else if idx < index {
						newMap[k] = v
					}
					// Skip the removed index
				}
			} else {
				newMap[k] = v
			}
		}

		return newMap
	})
}

// InsertAt inserts an item into an array field at the given index.
func (f *Form[T]) InsertAt(field string, index int, value any) {
	f.values.Update(func(current T) T {
		v := reflect.ValueOf(&current).Elem()
		arrayValue := getFieldReflectValue(v, field)

		if arrayValue.Kind() == reflect.Slice {
			// Clamp index
			if index < 0 {
				index = 0
			}
			if index > arrayValue.Len() {
				index = arrayValue.Len()
			}

			// Create new slice with space for the new item
			newSlice := reflect.MakeSlice(arrayValue.Type(), 0, arrayValue.Len()+1)
			for i := 0; i < arrayValue.Len()+1; i++ {
				if i < index {
					newSlice = reflect.Append(newSlice, arrayValue.Index(i))
				} else if i == index {
					newSlice = reflect.Append(newSlice, reflect.ValueOf(value))
				} else {
					newSlice = reflect.Append(newSlice, arrayValue.Index(i-1))
				}
			}
			setFieldReflectValue(v, field, newSlice)
		}

		return current
	})
}

// ArrayLen returns the length of an array field.
func (f *Form[T]) ArrayLen(field string) int {
	value := f.Get(field)
	if value == nil {
		return 0
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice {
		return rv.Len()
	}
	return 0
}

// fieldWithPath creates a field wrapper for a specific path.
func fieldWithPath(form formArrayGetter, fullPath string, input *vdom.VNode) *vdom.VNode {
	value := form.Get(fullPath)
	errors := form.FieldErrors(fullPath)
	hasError := len(errors) > 0

	if input.Props == nil {
		input.Props = make(vdom.Props)
	}

	input.Props["value"] = value
	input.Props["name"] = fullPath

	if hasError {
		existingClass, _ := input.Props["class"].(string)
		if existingClass != "" {
			input.Props["class"] = existingClass + " field-error"
		} else {
			input.Props["class"] = "field-error"
		}
	}

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

// getFieldReflectValue gets a reflect.Value for a nested field.
func getFieldReflectValue(v reflect.Value, path string) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}

	parts := strings.SplitN(path, ".", 2)
	fieldName := parts[0]

	// Check if fieldName is an array index
	if idx := atoi(fieldName); v.Kind() == reflect.Slice && idx >= 0 && idx < v.Len() {
		if len(parts) == 2 {
			return getFieldReflectValue(v.Index(idx), parts[1])
		}
		return v.Index(idx)
	}

	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}

	// Find field by form tag or name
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
		return reflect.Value{}
	}

	if len(parts) == 2 {
		return getFieldReflectValue(fieldValue, parts[1])
	}

	return fieldValue
}

// setFieldReflectValue sets a reflect.Value for a nested field.
func setFieldReflectValue(v reflect.Value, path string, newValue reflect.Value) {
	if !v.IsValid() {
		return
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	parts := strings.SplitN(path, ".", 2)
	fieldName := parts[0]

	// Check if fieldName is an array index
	if idx := atoi(fieldName); v.Kind() == reflect.Slice && idx >= 0 && idx < v.Len() {
		if len(parts) == 2 {
			setFieldReflectValue(v.Index(idx), parts[1], newValue)
			return
		}
		if v.Index(idx).CanSet() && newValue.Type().AssignableTo(v.Index(idx).Type()) {
			v.Index(idx).Set(newValue)
		}
		return
	}

	if v.Kind() != reflect.Struct {
		return
	}

	// Find field by form tag or name
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
		return
	}

	if len(parts) == 2 {
		setFieldReflectValue(fieldValue, parts[1], newValue)
		return
	}

	if fieldValue.CanSet() && newValue.Type().AssignableTo(fieldValue.Type()) {
		fieldValue.Set(newValue)
	}
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// atoi converts a string to an int, returns -1 if not a valid integer.
func atoi(s string) int {
	if s == "" {
		return -1
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// toAny converts a slice of VNodes to a slice of any.
func toAny(nodes []*vdom.VNode) []any {
	result := make([]any, len(nodes))
	for i, n := range nodes {
		result[i] = n
	}
	return result
}
