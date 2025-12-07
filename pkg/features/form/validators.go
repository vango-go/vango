package form

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Validator is an interface for form field validation.
type Validator interface {
	// Validate checks if the value is valid.
	// Returns nil if valid, or an error with a message if invalid.
	Validate(value any) error
}

// ValidatorFunc is a function that implements Validator.
type ValidatorFunc func(value any) error

func (f ValidatorFunc) Validate(value any) error {
	return f(value)
}

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// ----------------------------------------------------------------------------
// String Validators
// ----------------------------------------------------------------------------

// Required validates that the value is non-empty.
func Required(msg string) Validator {
	if msg == "" {
		msg = "This field is required"
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// MinLength validates that a string has at least n characters.
func MinLength(n int, msg string) Validator {
	if msg == "" {
		msg = fmt.Sprintf("Must be at least %d characters", n)
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil // Let Required handle empty values
		}
		if len([]rune(s)) < n {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// MaxLength validates that a string has at most n characters.
func MaxLength(n int, msg string) Validator {
	if msg == "" {
		msg = fmt.Sprintf("Must be at most %d characters", n)
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if len([]rune(s)) > n {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Pattern validates that a string matches the given regular expression.
func Pattern(pattern string, msg string) Validator {
	re := regexp.MustCompile(pattern)
	if msg == "" {
		msg = "Invalid format"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		if !re.MatchString(s) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Email validates that the value is a valid email address.
func Email(msg string) Validator {
	if msg == "" {
		msg = "Invalid email address"
	}
	// Simple regex for common email validation (basic sanity check)
	// Requires @ and . in domain part
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		if !re.MatchString(s) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// URL validates that the value is a valid URL.
func URL(msg string) Validator {
	if msg == "" {
		msg = "Invalid URL"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// uuidPattern matches UUID format.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID validates that the value is a valid UUID.
func UUID(msg string) Validator {
	if msg == "" {
		msg = "Invalid UUID"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		if !uuidPattern.MatchString(s) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Alpha validates that the value contains only ASCII letters.
func Alpha(msg string) Validator {
	if msg == "" {
		msg = "Must contain only letters"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return ValidationError{Message: msg}
			}
		}
		return nil
	})
}

// AlphaNumeric validates that the value contains only letters and digits.
func AlphaNumeric(msg string) Validator {
	if msg == "" {
		msg = "Must contain only letters and numbers"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return ValidationError{Message: msg}
			}
		}
		return nil
	})
}

// Numeric validates that the value contains only digits.
func Numeric(msg string) Validator {
	if msg == "" {
		msg = "Must contain only numbers"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		for _, r := range s {
			if !unicode.IsDigit(r) {
				return ValidationError{Message: msg}
			}
		}
		return nil
	})
}

// Phone validates that the value looks like a phone number.
func Phone(msg string) Validator {
	// Matches common phone formats: +1-234-567-8900, (234) 567-8900, 234.567.8900, etc.
	pattern := regexp.MustCompile(`^[\+]?[(]?[0-9]{1,4}[)]?[-\s\.]?[(]?[0-9]{1,3}[)]?[-\s\.]?[0-9]{1,4}[-\s\.]?[0-9]{1,4}[-\s\.]?[0-9]{1,9}$`)
	if msg == "" {
		msg = "Invalid phone number"
	}
	return ValidatorFunc(func(value any) error {
		s := toString(value)
		if s == "" {
			return nil
		}
		if !pattern.MatchString(s) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// ----------------------------------------------------------------------------
// Numeric Validators
// ----------------------------------------------------------------------------

// Min validates that a numeric value is >= n.
func Min(n any, msg string) Validator {
	minVal := toFloat64(n)
	if msg == "" {
		msg = fmt.Sprintf("Must be at least %v", n)
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		v := toFloat64(value)
		if v < minVal {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Max validates that a numeric value is <= n.
func Max(n any, msg string) Validator {
	maxVal := toFloat64(n)
	if msg == "" {
		msg = fmt.Sprintf("Must be at most %v", n)
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		v := toFloat64(value)
		if v > maxVal {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Between validates that a numeric value is between min and max (inclusive).
func Between(min, max any, msg string) Validator {
	minVal := toFloat64(min)
	maxVal := toFloat64(max)
	if msg == "" {
		msg = fmt.Sprintf("Must be between %v and %v", min, max)
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		v := toFloat64(value)
		if v < minVal || v > maxVal {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Positive validates that a numeric value is > 0.
func Positive(msg string) Validator {
	if msg == "" {
		msg = "Must be positive"
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		if toFloat64(value) <= 0 {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// NonNegative validates that a numeric value is >= 0.
func NonNegative(msg string) Validator {
	if msg == "" {
		msg = "Must not be negative"
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		if toFloat64(value) < 0 {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// ----------------------------------------------------------------------------
// Comparison Validators
// ----------------------------------------------------------------------------

// EqualToField creates a validator that checks if the value equals another field.
// Note: This validator needs form context, so it's typically added via Form.AddValidators.
type EqualToField struct {
	Field   string
	Message string
	form    interface{ Get(string) any }
}

// EqualTo returns a validator factory function.
// The actual comparison happens when the form is validated.
func EqualTo(field string, msg string) *EqualToField {
	if msg == "" {
		msg = fmt.Sprintf("Must match %s", field)
	}
	return &EqualToField{Field: field, Message: msg}
}

func (e *EqualToField) Validate(value any) error {
	if e.form == nil {
		return nil // Cannot validate without form context
	}
	otherValue := e.form.Get(e.Field)
	if !equals(value, otherValue) {
		return ValidationError{Message: e.Message}
	}
	return nil
}

// SetForm sets the form context for comparison validators.
func (e *EqualToField) SetForm(form interface{ Get(string) any }) {
	e.form = form
}

// NotEqualToField creates a validator that checks if the value differs from another field.
type NotEqualToField struct {
	Field   string
	Message string
	form    interface{ Get(string) any }
}

// NotEqualTo returns a validator that ensures the value differs from another field.
func NotEqualTo(field string, msg string) *NotEqualToField {
	if msg == "" {
		msg = fmt.Sprintf("Must not match %s", field)
	}
	return &NotEqualToField{Field: field, Message: msg}
}

func (e *NotEqualToField) Validate(value any) error {
	if e.form == nil {
		return nil
	}
	otherValue := e.form.Get(e.Field)
	if equals(value, otherValue) {
		return ValidationError{Message: e.Message}
	}
	return nil
}

func (e *NotEqualToField) SetForm(form interface{ Get(string) any }) {
	e.form = form
}

// ----------------------------------------------------------------------------
// Date Validators
// ----------------------------------------------------------------------------

// DateAfter validates that a date/time is after the given time.
func DateAfter(t time.Time, msg string) Validator {
	if msg == "" {
		msg = fmt.Sprintf("Must be after %s", t.Format(time.RFC3339))
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		dt := toTime(value)
		if dt.IsZero() {
			return nil
		}
		if !dt.After(t) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// DateBefore validates that a date/time is before the given time.
func DateBefore(t time.Time, msg string) Validator {
	if msg == "" {
		msg = fmt.Sprintf("Must be before %s", t.Format(time.RFC3339))
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		dt := toTime(value)
		if dt.IsZero() {
			return nil
		}
		if !dt.Before(t) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Future validates that a date/time is in the future.
func Future(msg string) Validator {
	if msg == "" {
		msg = "Must be in the future"
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		dt := toTime(value)
		if dt.IsZero() {
			return nil
		}
		if !dt.After(time.Now()) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// Past validates that a date/time is in the past.
func Past(msg string) Validator {
	if msg == "" {
		msg = "Must be in the past"
	}
	return ValidatorFunc(func(value any) error {
		if isEmpty(value) {
			return nil
		}
		dt := toTime(value)
		if dt.IsZero() {
			return nil
		}
		if !dt.Before(time.Now()) {
			return ValidationError{Message: msg}
		}
		return nil
	})
}

// ----------------------------------------------------------------------------
// Custom Validators
// ----------------------------------------------------------------------------

// Custom creates a validator from a custom function.
func Custom(fn func(value any) error) Validator {
	return ValidatorFunc(fn)
}

// AsyncValidator wraps an async validation function.
// The function returns (error, isComplete).
type AsyncValidator struct {
	fn       func(value any) (error, bool)
	loading  bool
	lastErr  error
	complete bool
}

// Async creates an async validator for server-side checks.
func Async(fn func(value any) (error, bool)) *AsyncValidator {
	return &AsyncValidator{fn: fn}
}

func (a *AsyncValidator) Validate(value any) error {
	err, complete := a.fn(value)
	a.complete = complete
	a.loading = !complete
	a.lastErr = err
	return err
}

// IsLoading returns true if the async validation is in progress.
func (a *AsyncValidator) IsLoading() bool {
	return a.loading
}

// IsComplete returns true if the async validation has completed.
func (a *AsyncValidator) IsComplete() bool {
	return a.complete
}

// ----------------------------------------------------------------------------
// Helper Functions
// ----------------------------------------------------------------------------

// isEmpty checks if a value is considered empty.
func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []byte:
		return len(v) == 0
	case int:
		return false // 0 is not empty for integers
	case int64:
		return false
	case float64:
		return false
	case bool:
		return false
	default:
		return false
	}
}

// toString converts a value to a string.
func toString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toFloat64 converts a value to float64.
func toFloat64(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// toTime converts a value to time.Time.
func toTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case *time.Time:
		if v != nil {
			return *v
		}
		return time.Time{}
	case string:
		// Try common formats
		for _, layout := range []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
			"01/02/2006",
			"02-01-2006",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

// equals checks if two values are equal.
func equals(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// validatorFromTag creates a validator from a tag name and value.
func validatorFromTag(name, value string, t reflect.Type) Validator {
	switch name {
	case "required":
		return Required("")
	case "min":
		if n, err := strconv.Atoi(value); err == nil {
			// If it's a string or slice, use MinLength
			if t != nil && (t.Kind() == reflect.String || t.Kind() == reflect.Slice || t.Kind() == reflect.Map) {
				return MinLength(n, "")
			}
			// Otherwise assume numeric
			return Min(n, "")
		}
		return MinLength(0, "") // Fallback if Atoi fails
	case "max":
		if n, err := strconv.Atoi(value); err == nil {
			if t != nil && (t.Kind() == reflect.String || t.Kind() == reflect.Slice || t.Kind() == reflect.Map) {
				return MaxLength(n, "")
			}
			return Max(n, "")
		}
		return MaxLength(0, "") // Fallback if Atoi fails
	case "minlen", "minlength":
		n, _ := strconv.Atoi(value)
		return MinLength(n, "")
	case "maxlen", "maxlength":
		n, _ := strconv.Atoi(value)
		return MaxLength(n, "")
	case "email":
		return Email("")
	case "url":
		return URL("")
	case "uuid":
		return UUID("")
	case "alpha":
		return Alpha("")
	case "alphanum", "alphanumeric":
		return AlphaNumeric("")
	case "numeric":
		return Numeric("")
	case "phone":
		return Phone("")
	case "pattern", "regex":
		return Pattern(value, "")
	case "positive":
		return Positive("")
	case "nonnegative":
		return NonNegative("")
	default:
		return nil
	}
}

// parseValidateTag parses a validate tag string into validators.
func parseValidateTag(tag string, t reflect.Type) []Validator {
	if tag == "" {
		return nil
	}

	rules := strings.Split(tag, ",")
	validators := make([]Validator, 0, len(rules))

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		// Parse rule=value format
		parts := strings.SplitN(rule, "=", 2)
		ruleName := parts[0]
		var ruleValue string
		if len(parts) > 1 {
			ruleValue = parts[1]
		}

		if v := validatorFromTag(ruleName, ruleValue, t); v != nil {
			validators = append(validators, v)
		}
	}

	return validators
}
