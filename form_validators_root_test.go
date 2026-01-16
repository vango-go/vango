package vango

import (
	"errors"
	"testing"
	"time"
)

// =============================================================================
// Form Validator Tests (via root package re-exports)
// =============================================================================

// mockFormContext simulates a form for cross-field validators
type mockFormContext struct {
	fields map[string]any
}

func (m *mockFormContext) Get(field string) any {
	return m.fields[field]
}

func TestFormValidator_EqualTo_ValidatesMatchingFields(t *testing.T) {
	v := EqualTo("password", "Passwords must match")

	// Set up form context
	form := &mockFormContext{
		fields: map[string]any{
			"password":        "secret123",
			"confirmPassword": "secret123",
		},
	}
	v.SetForm(form)

	// Matching values should pass
	if err := v.Validate("secret123"); err != nil {
		t.Errorf("expected no error for matching values, got %v", err)
	}
}

func TestFormValidator_EqualTo_FailsOnMismatch(t *testing.T) {
	v := EqualTo("password", "Passwords must match")

	form := &mockFormContext{
		fields: map[string]any{
			"password": "secret123",
		},
	}
	v.SetForm(form)

	// Non-matching values should fail
	if err := v.Validate("different"); err == nil {
		t.Error("expected error for mismatched values")
	}
}

func TestFormValidator_EqualTo_DefaultMessage(t *testing.T) {
	v := EqualTo("password", "")

	// Should use default message
	if v.Message == "" {
		t.Error("expected default message to be set")
	}
}

func TestFormValidator_EqualTo_NilFormContext(t *testing.T) {
	v := EqualTo("password", "Passwords must match")

	// Without form context, should pass (cannot validate)
	if err := v.Validate("anything"); err != nil {
		t.Errorf("expected no error without form context, got %v", err)
	}
}

func TestFormValidator_NotEqualTo_ValidatesDifferentFields(t *testing.T) {
	v := NotEqualTo("username", "Cannot use username as password")

	form := &mockFormContext{
		fields: map[string]any{
			"username": "alice",
			"password": "secret123",
		},
	}
	v.SetForm(form)

	// Different values should pass
	if err := v.Validate("secret123"); err != nil {
		t.Errorf("expected no error for different values, got %v", err)
	}
}

func TestFormValidator_NotEqualTo_FailsOnMatch(t *testing.T) {
	v := NotEqualTo("username", "Cannot use username as password")

	form := &mockFormContext{
		fields: map[string]any{
			"username": "alice",
		},
	}
	v.SetForm(form)

	// Same values should fail
	if err := v.Validate("alice"); err == nil {
		t.Error("expected error for matching values")
	}
}

func TestFormValidator_NotEqualTo_DefaultMessage(t *testing.T) {
	v := NotEqualTo("username", "")

	if v.Message == "" {
		t.Error("expected default message to be set")
	}
}

func TestFormValidator_NotEqualTo_NilFormContext(t *testing.T) {
	v := NotEqualTo("username", "Cannot match username")

	// Without form context, should pass
	if err := v.Validate("anything"); err != nil {
		t.Errorf("expected no error without form context, got %v", err)
	}
}

func TestFormValidator_Async_ValidatesWithServerCheck(t *testing.T) {
	checkCount := 0
	v := Async(func(value any) (error, bool) {
		checkCount++
		s := value.(string)
		if s == "taken" {
			return errors.New("Username is taken"), true
		}
		return nil, true
	})

	// Valid value should pass
	if err := v.Validate("available"); err != nil {
		t.Errorf("expected no error for available username, got %v", err)
	}
	if checkCount != 1 {
		t.Errorf("check count = %d, want 1", checkCount)
	}

	// Invalid value should fail
	if err := v.Validate("taken"); err == nil {
		t.Error("expected error for taken username")
	}
	if checkCount != 2 {
		t.Errorf("check count = %d, want 2", checkCount)
	}
}

func TestFormValidator_Async_IsLoading(t *testing.T) {
	v := Async(func(value any) (error, bool) {
		return nil, false // Not complete yet
	})

	v.Validate("test")

	if !v.IsLoading() {
		t.Error("IsLoading should be true when validation is not complete")
	}
}

func TestFormValidator_Async_IsComplete(t *testing.T) {
	v := Async(func(value any) (error, bool) {
		return nil, true // Complete
	})

	v.Validate("test")

	if !v.IsComplete() {
		t.Error("IsComplete should be true when validation is done")
	}
	if v.IsLoading() {
		t.Error("IsLoading should be false when validation is complete")
	}
}

// =============================================================================
// Standard Validator Tests (via root package)
// =============================================================================

func TestFormValidator_Required(t *testing.T) {
	v := Required("This field is required")

	// Empty values should fail
	if err := v.Validate(""); err == nil {
		t.Error("expected error for empty string")
	}

	// Non-empty values should pass
	if err := v.Validate("hello"); err != nil {
		t.Errorf("expected no error for non-empty string, got %v", err)
	}
}

func TestFormValidator_MinLength(t *testing.T) {
	v := MinLength(5, "Must be at least 5 characters")

	if err := v.Validate("hi"); err == nil {
		t.Error("expected error for short string")
	}

	if err := v.Validate("hello"); err != nil {
		t.Errorf("expected no error for valid string, got %v", err)
	}

	if err := v.Validate("hello world"); err != nil {
		t.Errorf("expected no error for long string, got %v", err)
	}
}

func TestFormValidator_MaxLength(t *testing.T) {
	v := MaxLength(5, "Must be at most 5 characters")

	if err := v.Validate("hello world"); err == nil {
		t.Error("expected error for long string")
	}

	if err := v.Validate("hello"); err != nil {
		t.Errorf("expected no error for max length string, got %v", err)
	}

	if err := v.Validate("hi"); err != nil {
		t.Errorf("expected no error for short string, got %v", err)
	}
}

func TestFormValidator_Pattern(t *testing.T) {
	v := Pattern(`^\d{3}-\d{4}$`, "Invalid format")

	if err := v.Validate("123-4567"); err != nil {
		t.Errorf("expected no error for valid pattern, got %v", err)
	}

	if err := v.Validate("abc-defg"); err == nil {
		t.Error("expected error for invalid pattern")
	}

	// Empty string should pass (let Required handle emptiness)
	if err := v.Validate(""); err != nil {
		t.Errorf("expected empty to pass, got %v", err)
	}
}

func TestFormValidator_Email(t *testing.T) {
	v := Email("Invalid email address")

	validEmails := []string{
		"test@example.com",
		"user.name@domain.org",
		"user+tag@example.co.uk",
	}
	for _, email := range validEmails {
		if err := v.Validate(email); err != nil {
			t.Errorf("expected valid email %q, got error: %v", email, err)
		}
	}

	invalidEmails := []string{
		"not-an-email",
		"@missing-local.com",
		"missing-domain@",
	}
	for _, email := range invalidEmails {
		if err := v.Validate(email); err == nil {
			t.Errorf("expected error for invalid email %q", email)
		}
	}
}

func TestFormValidator_URL(t *testing.T) {
	v := URL("Invalid URL")

	if err := v.Validate("https://example.com"); err != nil {
		t.Errorf("expected valid URL, got error: %v", err)
	}

	if err := v.Validate("not-a-url"); err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestFormValidator_UUID(t *testing.T) {
	v := UUID("Invalid UUID")

	validUUIDs := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"00000000-0000-0000-0000-000000000000",
	}
	for _, uuid := range validUUIDs {
		if err := v.Validate(uuid); err != nil {
			t.Errorf("expected valid UUID %q, got error: %v", uuid, err)
		}
	}

	if err := v.Validate("not-a-uuid"); err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestFormValidator_Alpha(t *testing.T) {
	v := Alpha("Must contain only letters")

	if err := v.Validate("Hello"); err != nil {
		t.Errorf("expected no error for letters, got %v", err)
	}

	if err := v.Validate("Hello123"); err == nil {
		t.Error("expected error for letters with numbers")
	}
}

func TestFormValidator_AlphaNumeric(t *testing.T) {
	v := AlphaNumeric("Must be alphanumeric")

	if err := v.Validate("Hello123"); err != nil {
		t.Errorf("expected no error for alphanumeric, got %v", err)
	}

	if err := v.Validate("Hello-World"); err == nil {
		t.Error("expected error for string with special chars")
	}
}

func TestFormValidator_Numeric(t *testing.T) {
	v := Numeric("Must be numeric")

	if err := v.Validate("12345"); err != nil {
		t.Errorf("expected no error for digits, got %v", err)
	}

	if err := v.Validate("123.45"); err == nil {
		t.Error("expected error for decimal number")
	}
}

func TestFormValidator_Phone(t *testing.T) {
	v := Phone("Invalid phone number")

	validPhones := []string{
		"+1-555-123-4567",
		"555-123-4567",
		"5551234567",
	}
	for _, phone := range validPhones {
		if err := v.Validate(phone); err != nil {
			t.Errorf("expected valid phone %q, got error: %v", phone, err)
		}
	}
}

func TestFormValidator_Min(t *testing.T) {
	v := Min(10, "Must be at least 10")

	if err := v.Validate(15); err != nil {
		t.Errorf("expected no error for 15, got %v", err)
	}

	if err := v.Validate(10); err != nil {
		t.Errorf("expected no error for 10, got %v", err)
	}

	if err := v.Validate(5); err == nil {
		t.Error("expected error for 5")
	}
}

func TestFormValidator_Max(t *testing.T) {
	v := Max(100, "Must be at most 100")

	if err := v.Validate(50); err != nil {
		t.Errorf("expected no error for 50, got %v", err)
	}

	if err := v.Validate(100); err != nil {
		t.Errorf("expected no error for 100, got %v", err)
	}

	if err := v.Validate(150); err == nil {
		t.Error("expected error for 150")
	}
}

func TestFormValidator_Between(t *testing.T) {
	v := Between(10, 100, "Must be between 10 and 100")

	if err := v.Validate(50); err != nil {
		t.Errorf("expected no error for 50, got %v", err)
	}

	if err := v.Validate(10); err != nil {
		t.Errorf("expected no error for 10, got %v", err)
	}

	if err := v.Validate(100); err != nil {
		t.Errorf("expected no error for 100, got %v", err)
	}

	if err := v.Validate(5); err == nil {
		t.Error("expected error for 5")
	}

	if err := v.Validate(150); err == nil {
		t.Error("expected error for 150")
	}
}

func TestFormValidator_Positive(t *testing.T) {
	v := Positive("Must be positive")

	if err := v.Validate(1); err != nil {
		t.Errorf("expected no error for 1, got %v", err)
	}

	if err := v.Validate(0); err == nil {
		t.Error("expected error for 0")
	}

	if err := v.Validate(-1); err == nil {
		t.Error("expected error for -1")
	}
}

func TestFormValidator_NonNegative(t *testing.T) {
	v := NonNegative("Must be non-negative")

	if err := v.Validate(1); err != nil {
		t.Errorf("expected no error for 1, got %v", err)
	}

	if err := v.Validate(0); err != nil {
		t.Errorf("expected no error for 0, got %v", err)
	}

	if err := v.Validate(-1); err == nil {
		t.Error("expected error for -1")
	}
}

func TestFormValidator_DateAfter(t *testing.T) {
	threshold := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	v := DateAfter(threshold, "Must be after 2024-01-01")

	after := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(after); err != nil {
		t.Errorf("expected no error for date after threshold, got %v", err)
	}

	before := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(before); err == nil {
		t.Error("expected error for date before threshold")
	}
}

func TestFormValidator_DateBefore(t *testing.T) {
	threshold := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	v := DateBefore(threshold, "Must be before 2024-12-31")

	before := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(before); err != nil {
		t.Errorf("expected no error for date before threshold, got %v", err)
	}

	after := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(after); err == nil {
		t.Error("expected error for date after threshold")
	}
}

func TestFormValidator_Future(t *testing.T) {
	v := Future("Must be in the future")

	future := time.Now().Add(24 * time.Hour)
	if err := v.Validate(future); err != nil {
		t.Errorf("expected no error for future date, got %v", err)
	}

	past := time.Now().Add(-24 * time.Hour)
	if err := v.Validate(past); err == nil {
		t.Error("expected error for past date")
	}
}

func TestFormValidator_Past(t *testing.T) {
	v := Past("Must be in the past")

	past := time.Now().Add(-24 * time.Hour)
	if err := v.Validate(past); err != nil {
		t.Errorf("expected no error for past date, got %v", err)
	}

	future := time.Now().Add(24 * time.Hour)
	if err := v.Validate(future); err == nil {
		t.Error("expected error for future date")
	}
}

func TestFormValidator_Custom(t *testing.T) {
	v := Custom(func(value any) error {
		s := value.(string)
		if len(s) < 3 {
			return errors.New("too short")
		}
		if s[0] != 'A' {
			return errors.New("must start with A")
		}
		return nil
	})

	if err := v.Validate("Alice"); err != nil {
		t.Errorf("expected no error for 'Alice', got %v", err)
	}

	if err := v.Validate("Bob"); err == nil {
		t.Error("expected error for 'Bob' (doesn't start with A)")
	}

	if err := v.Validate("Ab"); err == nil {
		t.Error("expected error for 'Ab' (too short)")
	}
}

// =============================================================================
// Validator Type Tests
// =============================================================================

func TestFormValidator_ValidatorInterface(t *testing.T) {
	// Verify all validators implement the Validator interface
	validators := []Validator{
		Required("required"),
		MinLength(5, "min length"),
		MaxLength(100, "max length"),
		Email("email"),
		Pattern(`\d+`, "pattern"),
		URL("url"),
		UUID("uuid"),
		Alpha("alpha"),
		AlphaNumeric("alphanum"),
		Numeric("numeric"),
		Phone("phone"),
		Min(0, "min"),
		Max(100, "max"),
		Between(0, 100, "between"),
		Positive("positive"),
		NonNegative("non-negative"),
		DateAfter(time.Now(), "date after"),
		DateBefore(time.Now(), "date before"),
		Future("future"),
		Past("past"),
		Custom(func(any) error { return nil }),
	}

	for i, v := range validators {
		if v == nil {
			t.Errorf("validator %d is nil", i)
		}
	}
}

func TestFormValidator_ValidatorFunc(t *testing.T) {
	// Test ValidatorFunc type
	var v Validator = ValidatorFunc(func(value any) error {
		if value == nil {
			return errors.New("value is nil")
		}
		return nil
	})

	if err := v.Validate("hello"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if err := v.Validate(nil); err == nil {
		t.Error("expected error for nil value")
	}
}

func TestFormValidator_ValidationError(t *testing.T) {
	err := ValidationError{Message: "Test error"}

	if err.Error() != "Test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "Test error")
	}
}
