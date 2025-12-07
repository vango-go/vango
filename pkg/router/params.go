package router

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ParamParser parses string parameters into typed struct fields.
type ParamParser struct{}

// NewParamParser creates a new parameter parser.
func NewParamParser() *ParamParser {
	return &ParamParser{}
}

// Parse populates a struct with values from the params map.
// The target must be a pointer to a struct with `param` tags.
func (p *ParamParser) Parse(params map[string]string, target any) error {
	if target == nil {
		return nil
	}

	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer, got %s", v.Kind())
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct, got pointer to %s", v.Kind())
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		paramName := field.Tag.Get("param")
		if paramName == "" {
			continue
		}

		value, ok := params[paramName]
		if !ok {
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		if err := p.setField(fieldValue, value); err != nil {
			return fmt.Errorf("parsing param %q: %w", paramName, err)
		}
	}

	return nil
}

// setField sets a field value from a string.
func (p *ParamParser) setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		field.SetInt(n)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer: %s", value)
		}
		field.SetUint(n)

	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %s", value)
		}
		field.SetFloat(n)

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %s", value)
		}
		field.SetBool(b)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			// For catch-all routes: "a/b/c" → ["a", "b", "c"]
			var parts []string
			if value != "" {
				parts = strings.Split(value, "/")
			}
			field.Set(reflect.ValueOf(parts))
		} else {
			return fmt.Errorf("unsupported slice element type: %s", field.Type().Elem().Kind())
		}

	default:
		return fmt.Errorf("unsupported type: %s", field.Kind())
	}

	return nil
}

// uuidRegex matches valid UUIDs.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateUUID validates that a string is a valid UUID.
func ValidateUUID(value string) error {
	if !uuidRegex.MatchString(value) {
		return fmt.Errorf("invalid UUID: %s", value)
	}
	return nil
}

// ValidateInt validates that a string is a valid integer.
func ValidateInt(value string) error {
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return fmt.Errorf("invalid integer: %s", value)
	}
	return nil
}

// ValidateParam validates a parameter value against its expected type.
func ValidateParam(value, paramType string) error {
	switch paramType {
	case "int", "int64", "int32", "int16", "int8":
		return ValidateInt(value)
	case "uint", "uint64", "uint32", "uint16", "uint8":
		if _, err := strconv.ParseUint(value, 10, 64); err != nil {
			return fmt.Errorf("invalid unsigned integer: %s", value)
		}
	case "uuid":
		return ValidateUUID(value)
	case "string", "":
		// All strings are valid
	default:
		// Unknown type, allow any value
	}
	return nil
}
