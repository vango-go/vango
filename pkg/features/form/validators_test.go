package form

import (
	"testing"
	"time"
)

func TestRequiredValidator(t *testing.T) {
	v := Required("")

	// Empty values should fail
	if err := v.Validate(""); err == nil {
		t.Error("Expected error for empty string")
	}
	if err := v.Validate("   "); err == nil {
		t.Error("Expected error for whitespace-only string")
	}
	if err := v.Validate(nil); err == nil {
		t.Error("Expected error for nil")
	}

	// Non-empty values should pass
	if err := v.Validate("hello"); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if err := v.Validate(0); err != nil {
		t.Errorf("Expected no error for 0, got: %v", err)
	}
	if err := v.Validate(false); err != nil {
		t.Errorf("Expected no error for false, got: %v", err)
	}
}

func TestMinLengthValidator(t *testing.T) {
	v := MinLength(3, "")

	// Too short
	if err := v.Validate("ab"); err == nil {
		t.Error("Expected error for 'ab' (len 2)")
	}

	// Exactly minimum
	if err := v.Validate("abc"); err != nil {
		t.Errorf("Expected no error for 'abc', got: %v", err)
	}

	// Longer than minimum
	if err := v.Validate("abcd"); err != nil {
		t.Errorf("Expected no error for 'abcd', got: %v", err)
	}

	// Empty strings should pass (use Required for empty check)
	if err := v.Validate(""); err != nil {
		t.Errorf("Expected no error for empty string (let Required handle it), got: %v", err)
	}
}

func TestMaxLengthValidator(t *testing.T) {
	v := MaxLength(5, "")

	// Within limit
	if err := v.Validate("abc"); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// At limit
	if err := v.Validate("abcde"); err != nil {
		t.Errorf("Expected no error at limit, got: %v", err)
	}

	// Over limit
	if err := v.Validate("abcdef"); err == nil {
		t.Error("Expected error for 'abcdef' (len 6)")
	}
}

func TestEmailValidator(t *testing.T) {
	v := Email("")

	validEmails := []string{
		"test@example.com",
		"user.name@domain.org",
		"user+tag@domain.co.uk",
	}

	invalidEmails := []string{
		"not-an-email",
		"missing@domain",
		"@nodomain.com",
		"spaces in@email.com",
	}

	for _, email := range validEmails {
		if err := v.Validate(email); err != nil {
			t.Errorf("Expected '%s' to be valid, got: %v", email, err)
		}
	}

	for _, email := range invalidEmails {
		if err := v.Validate(email); err == nil {
			t.Errorf("Expected '%s' to be invalid", email)
		}
	}

	// Empty should pass
	if err := v.Validate(""); err != nil {
		t.Errorf("Expected empty to pass (use Required), got: %v", err)
	}
}

func TestURLValidator(t *testing.T) {
	v := URL("")

	validURLs := []string{
		"http://example.com",
		"https://www.example.com/path",
		"https://example.com:8080/path?query=1",
	}

	invalidURLs := []string{
		"not-a-url",
		"example.com", // Missing scheme
		"://example.com",
	}

	for _, u := range validURLs {
		if err := v.Validate(u); err != nil {
			t.Errorf("Expected '%s' to be valid, got: %v", u, err)
		}
	}

	for _, u := range invalidURLs {
		if err := v.Validate(u); err == nil {
			t.Errorf("Expected '%s' to be invalid", u)
		}
	}
}

func TestUUIDValidator(t *testing.T) {
	v := UUID("")

	validUUIDs := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
	}

	invalidUUIDs := []string{
		"not-a-uuid",
		"550e8400-e29b-41d4-a716",
		"550e8400e29b41d4a716446655440000", // Missing dashes
	}

	for _, uuid := range validUUIDs {
		if err := v.Validate(uuid); err != nil {
			t.Errorf("Expected '%s' to be valid, got: %v", uuid, err)
		}
	}

	for _, uuid := range invalidUUIDs {
		if err := v.Validate(uuid); err == nil {
			t.Errorf("Expected '%s' to be invalid", uuid)
		}
	}
}

func TestAlphaValidator(t *testing.T) {
	v := Alpha("")

	if err := v.Validate("Hello"); err != nil {
		t.Errorf("Expected 'Hello' to be valid, got: %v", err)
	}

	if err := v.Validate("Hello123"); err == nil {
		t.Error("Expected 'Hello123' to be invalid")
	}

	if err := v.Validate("Hello World"); err == nil {
		t.Error("Expected 'Hello World' to be invalid (spaces)")
	}
}

func TestAlphaNumericValidator(t *testing.T) {
	v := AlphaNumeric("")

	if err := v.Validate("Hello123"); err != nil {
		t.Errorf("Expected 'Hello123' to be valid, got: %v", err)
	}

	if err := v.Validate("Hello-123"); err == nil {
		t.Error("Expected 'Hello-123' to be invalid (dash)")
	}
}

func TestNumericValidator(t *testing.T) {
	v := Numeric("")

	if err := v.Validate("12345"); err != nil {
		t.Errorf("Expected '12345' to be valid, got: %v", err)
	}

	if err := v.Validate("123.45"); err == nil {
		t.Error("Expected '123.45' to be invalid (decimal)")
	}

	if err := v.Validate("123abc"); err == nil {
		t.Error("Expected '123abc' to be invalid")
	}
}

func TestMinValidator(t *testing.T) {
	v := Min(10, "")

	if err := v.Validate(15); err != nil {
		t.Errorf("Expected 15 to pass, got: %v", err)
	}

	if err := v.Validate(10); err != nil {
		t.Errorf("Expected 10 to pass (equal to min), got: %v", err)
	}

	if err := v.Validate(5); err == nil {
		t.Error("Expected 5 to fail")
	}

	// Test with float
	if err := v.Validate(10.5); err != nil {
		t.Errorf("Expected 10.5 to pass, got: %v", err)
	}
}

func TestMaxValidator(t *testing.T) {
	v := Max(100, "")

	if err := v.Validate(50); err != nil {
		t.Errorf("Expected 50 to pass, got: %v", err)
	}

	if err := v.Validate(100); err != nil {
		t.Errorf("Expected 100 to pass (equal to max), got: %v", err)
	}

	if err := v.Validate(150); err == nil {
		t.Error("Expected 150 to fail")
	}
}

func TestBetweenValidator(t *testing.T) {
	v := Between(10, 20, "")

	if err := v.Validate(15); err != nil {
		t.Errorf("Expected 15 to pass, got: %v", err)
	}

	if err := v.Validate(10); err != nil {
		t.Errorf("Expected 10 to pass (equal to min), got: %v", err)
	}

	if err := v.Validate(20); err != nil {
		t.Errorf("Expected 20 to pass (equal to max), got: %v", err)
	}

	if err := v.Validate(5); err == nil {
		t.Error("Expected 5 to fail (below min)")
	}

	if err := v.Validate(25); err == nil {
		t.Error("Expected 25 to fail (above max)")
	}
}

func TestPositiveValidator(t *testing.T) {
	v := Positive("")

	if err := v.Validate(1); err != nil {
		t.Errorf("Expected 1 to pass, got: %v", err)
	}

	if err := v.Validate(0); err == nil {
		t.Error("Expected 0 to fail")
	}

	if err := v.Validate(-1); err == nil {
		t.Error("Expected -1 to fail")
	}
}

func TestNonNegativeValidator(t *testing.T) {
	v := NonNegative("")

	if err := v.Validate(1); err != nil {
		t.Errorf("Expected 1 to pass, got: %v", err)
	}

	if err := v.Validate(0); err != nil {
		t.Errorf("Expected 0 to pass, got: %v", err)
	}

	if err := v.Validate(-1); err == nil {
		t.Error("Expected -1 to fail")
	}
}

func TestPatternValidator(t *testing.T) {
	v := Pattern(`^[A-Z]{2}\d{4}$`, "")

	if err := v.Validate("AB1234"); err != nil {
		t.Errorf("Expected 'AB1234' to pass, got: %v", err)
	}

	if err := v.Validate("ab1234"); err == nil {
		t.Error("Expected 'ab1234' to fail (lowercase)")
	}

	if err := v.Validate("ABC123"); err == nil {
		t.Error("Expected 'ABC123' to fail (wrong pattern)")
	}
}

func TestDateAfterValidator(t *testing.T) {
	reference := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	v := DateAfter(reference, "")

	after := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(after); err != nil {
		t.Errorf("Expected date after reference to pass, got: %v", err)
	}

	before := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(before); err == nil {
		t.Error("Expected date before reference to fail")
	}

	// Test with string date
	if err := v.Validate("2024-06-01"); err != nil {
		t.Errorf("Expected string date after reference to pass, got: %v", err)
	}
}

func TestDateBeforeValidator(t *testing.T) {
	reference := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	v := DateBefore(reference, "")

	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(before); err != nil {
		t.Errorf("Expected date before reference to pass, got: %v", err)
	}

	after := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := v.Validate(after); err == nil {
		t.Error("Expected date after reference to fail")
	}
}

func TestFutureValidator(t *testing.T) {
	v := Future("")

	future := time.Now().Add(24 * time.Hour)
	if err := v.Validate(future); err != nil {
		t.Errorf("Expected future date to pass, got: %v", err)
	}

	past := time.Now().Add(-24 * time.Hour)
	if err := v.Validate(past); err == nil {
		t.Error("Expected past date to fail")
	}
}

func TestPastValidator(t *testing.T) {
	v := Past("")

	past := time.Now().Add(-24 * time.Hour)
	if err := v.Validate(past); err != nil {
		t.Errorf("Expected past date to pass, got: %v", err)
	}

	future := time.Now().Add(24 * time.Hour)
	if err := v.Validate(future); err == nil {
		t.Error("Expected future date to fail")
	}
}

func TestCustomValidator(t *testing.T) {
	v := Custom(func(value any) error {
		s, ok := value.(string)
		if !ok {
			return nil
		}
		if s == "forbidden" {
			return ValidationError{Message: "This value is forbidden"}
		}
		return nil
	})

	if err := v.Validate("allowed"); err != nil {
		t.Errorf("Expected 'allowed' to pass, got: %v", err)
	}

	if err := v.Validate("forbidden"); err == nil {
		t.Error("Expected 'forbidden' to fail")
	}
}

func TestPhoneValidator(t *testing.T) {
	v := Phone("")

	validPhones := []string{
		"+1-234-567-8900",
		"(234) 567-8900",
		"234.567.8900",
		"2345678900",
		"+44 20 7946 0958",
	}

	for _, phone := range validPhones {
		if err := v.Validate(phone); err != nil {
			t.Errorf("Expected '%s' to be valid, got: %v", phone, err)
		}
	}

	invalidPhones := []string{
		"not-a-phone",
		"123",
		"abc-def-ghij",
	}

	for _, phone := range invalidPhones {
		if err := v.Validate(phone); err == nil {
			t.Errorf("Expected '%s' to be invalid", phone)
		}
	}
}

func TestValidatorFromTag(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool // true if validator should be created
	}{
		{"required", "", true},
		{"email", "", true},
		{"url", "", true},
		{"uuid", "", true},
		{"min", "5", true},
		{"max", "100", true},
		{"minlen", "3", true},
		{"maxlen", "10", true},
		{"alpha", "", true},
		{"alphanum", "", true},
		{"numeric", "", true},
		{"phone", "", true},
		{"positive", "", true},
		{"nonnegative", "", true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		v := validatorFromTag(tt.name, tt.value, nil)
		if tt.want && v == nil {
			t.Errorf("Expected validator for '%s', got nil", tt.name)
		}
		if !tt.want && v != nil {
			t.Errorf("Expected no validator for '%s', got one", tt.name)
		}
	}
}

func TestParseValidateTag(t *testing.T) {
	validators := parseValidateTag("required,min=2,email", nil)

	if len(validators) != 3 {
		t.Errorf("Expected 3 validators, got %d", len(validators))
	}
}

func TestEmptyParseValidateTag(t *testing.T) {
	validators := parseValidateTag("", nil)

	if len(validators) != 0 {
		t.Errorf("Expected 0 validators for empty tag, got %d", len(validators))
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Message: "Invalid email",
	}

	if err.Error() != "Invalid email" {
		t.Errorf("Expected 'Invalid email', got '%s'", err.Error())
	}
}
