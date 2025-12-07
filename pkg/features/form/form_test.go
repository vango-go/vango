package form

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// TestContact is a test form struct.
type TestContact struct {
	Name    string `form:"name" validate:"required,min=2,max=100"`
	Email   string `form:"email" validate:"required,email"`
	Phone   string `form:"phone" validate:"phone"`
	Message string `form:"message" validate:"required,max=1000"`
}

// TestNested is a form with nested structs.
type TestNested struct {
	User    TestUser    `form:"user"`
	Address TestAddress `form:"address"`
}

type TestUser struct {
	Name  string `form:"name" validate:"required"`
	Email string `form:"email" validate:"email"`
}

type TestAddress struct {
	Street string `form:"street"`
	City   string `form:"city" validate:"required"`
	Zip    string `form:"zip" validate:"numeric"`
}

// TestOrder is a form with arrays.
type TestOrder struct {
	CustomerName string      `form:"customer_name" validate:"required"`
	Items        []OrderItem `form:"items"`
}

type OrderItem struct {
	ProductID int `form:"product_id" validate:"required"`
	Quantity  int `form:"quantity" validate:"min=1"`
}

func TestUseForm(t *testing.T) {
	initial := TestContact{
		Name: "John",
	}
	form := UseForm(initial)

	if form == nil {
		t.Fatal("UseForm returned nil")
	}

	// Check initial values
	values := form.Values()
	if values.Name != "John" {
		t.Errorf("Expected Name 'John', got '%s'", values.Name)
	}
}

func TestFormGet(t *testing.T) {
	form := UseForm(TestContact{
		Name:  "Alice",
		Email: "alice@example.com",
	})

	name := form.GetString("name")
	if name != "Alice" {
		t.Errorf("Expected 'Alice', got '%s'", name)
	}

	email := form.GetString("email")
	if email != "alice@example.com" {
		t.Errorf("Expected 'alice@example.com', got '%s'", email)
	}
}

func TestFormSet(t *testing.T) {
	form := UseForm(TestContact{})

	form.Set("name", "Bob")
	form.Set("email", "bob@test.com")

	if form.GetString("name") != "Bob" {
		t.Errorf("Expected 'Bob', got '%s'", form.GetString("name"))
	}

	if form.GetString("email") != "bob@test.com" {
		t.Errorf("Expected 'bob@test.com', got '%s'", form.GetString("email"))
	}
}

func TestFormSetValues(t *testing.T) {
	form := UseForm(TestContact{})

	form.SetValues(TestContact{
		Name:    "Charlie",
		Email:   "charlie@example.com",
		Message: "Hello",
	})

	values := form.Values()
	if values.Name != "Charlie" {
		t.Errorf("Expected 'Charlie', got '%s'", values.Name)
	}
	if values.Email != "charlie@example.com" {
		t.Errorf("Expected 'charlie@example.com', got '%s'", values.Email)
	}
	if values.Message != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", values.Message)
	}
}

func TestFormReset(t *testing.T) {
	initial := TestContact{Name: "Initial"}
	form := UseForm(initial)

	form.Set("name", "Modified")
	if form.GetString("name") != "Modified" {
		t.Errorf("Expected 'Modified', got '%s'", form.GetString("name"))
	}

	form.Reset()
	if form.GetString("name") != "Initial" {
		t.Errorf("After reset, expected 'Initial', got '%s'", form.GetString("name"))
	}
}

func TestFormValidate(t *testing.T) {
	form := UseForm(TestContact{})

	// Empty form should fail validation
	valid := form.Validate()
	if valid {
		t.Error("Expected validation to fail for empty form")
	}

	// Check that errors are set for required fields
	if !form.HasError("name") {
		t.Error("Expected error for 'name' field")
	}
	if !form.HasError("email") {
		t.Error("Expected error for 'email' field")
	}
	if !form.HasError("message") {
		t.Error("Expected error for 'message' field")
	}

	// Phone is not required, so no error
	if form.HasError("phone") {
		t.Error("Did not expect error for 'phone' field (not required)")
	}
}

func TestFormValidateWithValidData(t *testing.T) {
	form := UseForm(TestContact{
		Name:    "John Doe",
		Email:   "john@example.com",
		Message: "Hello world",
	})

	valid := form.Validate()
	if !valid {
		t.Errorf("Expected validation to pass, errors: %v", form.Errors())
	}
}

func TestFormValidateField(t *testing.T) {
	form := UseForm(TestContact{})

	// Validate just the name field
	valid := form.ValidateField("name")
	if valid {
		t.Error("Expected name validation to fail for empty value")
	}

	if !form.HasError("name") {
		t.Error("Expected error for 'name' field")
	}

	// Set name and validate again
	form.Set("name", "Jo") // Min is 2, so this should pass
	valid = form.ValidateField("name")
	if !valid {
		t.Errorf("Expected name validation to pass, errors: %v", form.FieldErrors("name"))
	}
}

func TestFormErrors(t *testing.T) {
	form := UseForm(TestContact{})
	form.Validate()

	errors := form.Errors()
	if len(errors) == 0 {
		t.Error("Expected errors map to be non-empty")
	}

	nameErrors := form.FieldErrors("name")
	if len(nameErrors) == 0 {
		t.Error("Expected name field to have errors")
	}
}

func TestFormSetError(t *testing.T) {
	form := UseForm(TestContact{Name: "Valid"})

	form.SetError("name", "Custom error message")

	if !form.HasError("name") {
		t.Error("Expected name to have error after SetError")
	}

	errors := form.FieldErrors("name")
	if len(errors) == 0 || errors[0] != "Custom error message" {
		t.Errorf("Expected 'Custom error message', got %v", errors)
	}
}

func TestFormClearErrors(t *testing.T) {
	form := UseForm(TestContact{})
	form.Validate()

	if form.IsValid() {
		t.Error("Form should have errors")
	}

	form.ClearErrors()

	if !form.IsValid() {
		t.Error("Form should be valid after clearing errors")
	}
}

func TestFormDirtyState(t *testing.T) {
	form := UseForm(TestContact{})

	if form.IsDirty() {
		t.Error("Form should not be dirty initially")
	}

	form.Set("name", "Changed")

	if !form.IsDirty() {
		t.Error("Form should be dirty after change")
	}

	if !form.FieldDirty("name") {
		t.Error("Name field should be dirty")
	}

	if form.FieldDirty("email") {
		t.Error("Email field should not be dirty")
	}
}

func TestFormSubmittingState(t *testing.T) {
	form := UseForm(TestContact{})

	if form.IsSubmitting() {
		t.Error("Form should not be submitting initially")
	}

	form.SetSubmitting(true)

	if !form.IsSubmitting() {
		t.Error("Form should be submitting after SetSubmitting(true)")
	}

	form.SetSubmitting(false)

	if form.IsSubmitting() {
		t.Error("Form should not be submitting after SetSubmitting(false)")
	}
}

func TestFormNestedFields(t *testing.T) {
	form := UseForm(TestNested{
		User: TestUser{
			Name:  "Alice",
			Email: "alice@example.com",
		},
		Address: TestAddress{
			City: "New York",
		},
	})

	// Access nested fields - the form uses the struct directly
	values := form.Values()
	if values.User.Name != "Alice" {
		t.Errorf("Expected User.Name 'Alice', got '%s'", values.User.Name)
	}
}

func TestFormIsValid(t *testing.T) {
	form := UseForm(TestContact{
		Name:    "John",
		Email:   "john@email.com",
		Message: "Hello",
	})

	// Form should be valid before validation is run (no errors yet)
	if !form.IsValid() {
		t.Error("Form should be valid before validation runs")
	}

	// Run validation
	valid := form.Validate()
	if !valid {
		t.Errorf("Form should be valid, errors: %v", form.Errors())
	}

	if !form.IsValid() {
		t.Error("Form should be valid after successful validation")
	}
}

func TestFormAddValidators(t *testing.T) {
	form := UseForm(TestContact{
		Name: "test",
	})

	// Add a custom validator that requires name to be at least 5 chars
	form.AddValidators("name", MinLength(5, "Name too short"))

	valid := form.Validate()
	if valid {
		t.Error("Expected validation to fail with custom validator")
	}

	if !form.HasError("name") {
		t.Error("Expected name to have error")
	}
}

func TestFormGetInt(t *testing.T) {
	type NumericForm struct {
		Count int `form:"count"`
	}

	form := UseForm(NumericForm{Count: 42})

	count := form.GetInt("count")
	if count != 42 {
		t.Errorf("Expected 42, got %d", count)
	}
}

func TestFormGetBool(t *testing.T) {
	type BoolForm struct {
		Active bool `form:"active"`
	}

	form := UseForm(BoolForm{Active: true})

	active := form.GetBool("active")
	if !active {
		t.Error("Expected true")
	}
}

func TestFormField(t *testing.T) {
	form := UseForm(TestContact{
		Name: "John",
	})

	// Create a mock input node
	input := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "input",
	}

	field := form.Field("name", input)

	if field == nil {
		t.Fatal("Field returned nil")
	}

	if field.Tag != "div" {
		t.Errorf("Expected wrapper div, got %s", field.Tag)
	}
}
