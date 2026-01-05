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

func TestFormFieldWithError(t *testing.T) {
	form := UseForm(TestContact{})

	// Trigger validation to set errors
	form.Validate()

	input := &vdom.VNode{
		Kind:  vdom.KindElement,
		Tag:   "input",
		Props: vdom.Props{"class": "input"},
	}

	field := form.Field("name", input)

	if field == nil {
		t.Fatal("Field returned nil")
	}

	// Original input should NOT be mutated (we clone it)
	if input.Props["class"] != "input" {
		t.Errorf("Original input was mutated: expected class 'input', got '%v'", input.Props["class"])
	}

	// Field wrapper should be a div with class "field"
	if field.Tag != "div" {
		t.Errorf("Expected wrapper div, got %s", field.Tag)
	}
	if field.Props["class"] != "field" {
		t.Errorf("Expected class 'field', got '%v'", field.Props["class"])
	}

	// The cloned input inside the wrapper should have the error class
	if len(field.Children) == 0 {
		t.Fatal("Field wrapper has no children")
	}
	clonedInput := field.Children[0]
	if clonedInput.Props["class"] != "input field-error" {
		t.Errorf("Cloned input should have error class: expected 'input field-error', got '%v'", clonedInput.Props["class"])
	}
}

func TestFormFieldWithCustomValidators(t *testing.T) {
	form := UseForm(TestContact{Name: "ab"})

	input := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "input",
	}

	// Add custom validator via Field
	field := form.Field("name", input, MinLength(5, "Too short"))

	if field == nil {
		t.Fatal("Field returned nil")
	}

	// Validate the field
	form.ValidateField("name")

	if !form.HasError("name") {
		t.Error("Expected name to have error from custom validator")
	}
}

func TestFormArray(t *testing.T) {
	form := UseForm(TestOrder{
		CustomerName: "Alice",
		Items: []OrderItem{
			{ProductID: 1, Quantity: 2},
			{ProductID: 2, Quantity: 1},
		},
	})

	// Test ArrayLen
	if form.ArrayLen("items") != 2 {
		t.Errorf("Expected ArrayLen 2, got %d", form.ArrayLen("items"))
	}

	// Test Array rendering
	count := 0
	form.Array("items", func(item FormArrayItem, index int) *vdom.VNode {
		count++
		if item.Index() != index {
			t.Errorf("Item index mismatch: %d vs %d", item.Index(), index)
		}
		if item.Path() == "" {
			t.Error("Item path should not be empty")
		}
		return vdom.Div()
	})

	if count != 2 {
		t.Errorf("Expected Array to call render func 2 times, got %d", count)
	}
}

func TestFormArrayAppendTo(t *testing.T) {
	form := UseForm(TestOrder{
		CustomerName: "Bob",
		Items:        []OrderItem{},
	})

	if form.ArrayLen("items") != 0 {
		t.Error("Expected initial array length 0")
	}

	form.AppendTo("items", OrderItem{ProductID: 10, Quantity: 5})

	if form.ArrayLen("items") != 1 {
		t.Errorf("Expected array length 1 after append, got %d", form.ArrayLen("items"))
	}

	form.AppendTo("items", OrderItem{ProductID: 20, Quantity: 3})

	if form.ArrayLen("items") != 2 {
		t.Errorf("Expected array length 2 after second append, got %d", form.ArrayLen("items"))
	}
}

func TestFormArrayRemoveAt(t *testing.T) {
	form := UseForm(TestOrder{
		CustomerName: "Carol",
		Items: []OrderItem{
			{ProductID: 1, Quantity: 1},
			{ProductID: 2, Quantity: 2},
			{ProductID: 3, Quantity: 3},
		},
	})

	if form.ArrayLen("items") != 3 {
		t.Error("Expected initial array length 3")
	}

	form.RemoveAt("items", 1) // Remove middle item

	if form.ArrayLen("items") != 2 {
		t.Errorf("Expected array length 2 after remove, got %d", form.ArrayLen("items"))
	}

	// Check remaining items
	values := form.Values()
	if values.Items[0].ProductID != 1 {
		t.Error("First item should still be ProductID 1")
	}
	if values.Items[1].ProductID != 3 {
		t.Error("Second item should now be ProductID 3")
	}
}

func TestFormArrayInsertAt(t *testing.T) {
	form := UseForm(TestOrder{
		CustomerName: "Dave",
		Items: []OrderItem{
			{ProductID: 1, Quantity: 1},
			{ProductID: 3, Quantity: 3},
		},
	})

	form.InsertAt("items", 1, OrderItem{ProductID: 2, Quantity: 2})

	if form.ArrayLen("items") != 3 {
		t.Errorf("Expected array length 3 after insert, got %d", form.ArrayLen("items"))
	}

	values := form.Values()
	if values.Items[1].ProductID != 2 {
		t.Errorf("Inserted item should be at index 1, got ProductID %d", values.Items[1].ProductID)
	}
}

func TestFormArrayEmpty(t *testing.T) {
	form := UseForm(TestOrder{
		CustomerName: "Eve",
		Items:        nil,
	})

	// Array on nil should return empty fragment
	node := form.Array("items", func(item FormArrayItem, index int) *vdom.VNode {
		t.Error("Should not be called for empty array")
		return nil
	})

	if node == nil {
		t.Error("Array should return non-nil node even for empty array")
	}
}

func TestFormArrayNonSliceField(t *testing.T) {
	form := UseForm(TestContact{Name: "Test"})

	// Array on non-slice field should return empty fragment
	node := form.Array("name", func(item FormArrayItem, index int) *vdom.VNode {
		t.Error("Should not be called for non-slice field")
		return nil
	})

	if node == nil {
		t.Error("Array should return non-nil node")
	}
}

func TestFormGetIntVariousTypes(t *testing.T) {
	type NumericForm struct {
		Int8Val    int8    `form:"int8"`
		Int16Val   int16   `form:"int16"`
		Int32Val   int32   `form:"int32"`
		Int64Val   int64   `form:"int64"`
		UintVal    uint    `form:"uint"`
		Uint8Val   uint8   `form:"uint8"`
		Uint16Val  uint16  `form:"uint16"`
		Uint32Val  uint32  `form:"uint32"`
		Uint64Val  uint64  `form:"uint64"`
		Float32Val float32 `form:"float32"`
		Float64Val float64 `form:"float64"`
	}

	form := UseForm(NumericForm{
		Int8Val:    8,
		Int16Val:   16,
		Int32Val:   32,
		Int64Val:   64,
		UintVal:    100,
		Uint8Val:   200,
		Uint16Val:  300,
		Uint32Val:  400,
		Uint64Val:  500,
		Float32Val: 1.5,
		Float64Val: 2.5,
	})

	tests := []struct {
		field string
		want  int
	}{
		{"int8", 8},
		{"int16", 16},
		{"int32", 32},
		{"int64", 64},
		{"uint", 100},
		{"uint8", 200},
		{"uint16", 300},
		{"uint32", 400},
		{"uint64", 500},
		{"float32", 1},
		{"float64", 2},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := form.GetInt(tt.field)
			if got != tt.want {
				t.Errorf("GetInt(%s) = %d, want %d", tt.field, got, tt.want)
			}
		})
	}

	// Test non-numeric field returns 0
	strForm := UseForm(TestContact{Name: "Test"})
	if strForm.GetInt("name") != 0 {
		t.Error("GetInt on string field should return 0")
	}
}

func TestFormGetBoolVariousValues(t *testing.T) {
	type MixedForm struct {
		BoolVal   bool   `form:"bool"`
		StringVal string `form:"string"`
		IntVal    int    `form:"int"`
	}

	form := UseForm(MixedForm{
		BoolVal:   true,
		StringVal: "test",
		IntVal:    42,
	})

	if !form.GetBool("bool") {
		t.Error("GetBool(bool) should return true")
	}

	// Non-bool fields should return false
	if form.GetBool("string") {
		t.Error("GetBool(string) should return false")
	}

	if form.GetBool("int") {
		t.Error("GetBool(int) should return false")
	}
}

func TestFormNestedFieldAccess(t *testing.T) {
	form := UseForm(TestNested{
		User: TestUser{
			Name:  "Alice",
			Email: "alice@test.com",
		},
		Address: TestAddress{
			Street: "123 Main St",
			City:   "Boston",
			Zip:    "02101",
		},
	})

	// Test nested field Get
	name := form.Get("user.name")
	if name != "Alice" {
		t.Errorf("Get(user.name) = %v, want 'Alice'", name)
	}

	city := form.GetString("address.city")
	if city != "Boston" {
		t.Errorf("GetString(address.city) = %s, want 'Boston'", city)
	}
}

func TestFormNestedFieldSet(t *testing.T) {
	form := UseForm(TestNested{})

	form.Set("user.name", "Bob")
	form.Set("address.city", "Seattle")

	values := form.Values()
	if values.User.Name != "Bob" {
		t.Errorf("User.Name = %s, want 'Bob'", values.User.Name)
	}
	if values.Address.City != "Seattle" {
		t.Errorf("Address.City = %s, want 'Seattle'", values.Address.City)
	}
}

func TestFormIsTouched(t *testing.T) {
	form := UseForm(TestContact{})

	if form.IsTouched("name") {
		t.Error("Field should not be touched initially")
	}

	// Validate field marks it as touched
	form.ValidateField("name")

	if !form.IsTouched("name") {
		t.Error("Field should be touched after validation")
	}
}

func TestFormPointerStruct(t *testing.T) {
	type PtrForm struct {
		Name *string `form:"name"`
	}

	name := "Test"
	form := UseForm(PtrForm{Name: &name})

	values := form.Values()
	if values.Name == nil || *values.Name != "Test" {
		t.Error("Pointer field should be accessible")
	}
}

func TestFormUnexportedFields(t *testing.T) {
	type FormWithPrivate struct {
		Public  string `form:"public"`
		private string `form:"private"` //nolint:unused
	}

	form := UseForm(FormWithPrivate{Public: "visible"})

	// Should only process exported fields
	if form.Get("public") != "visible" {
		t.Error("Should be able to get exported field")
	}
}

func TestFormTagDash(t *testing.T) {
	type FormWithDash struct {
		Included string `form:"included"`
		Excluded string `form:"-"`
	}

	form := UseForm(FormWithDash{
		Included: "yes",
		Excluded: "no",
	})

	// Field with form:"-" should be excluded
	if form.Get("included") != "yes" {
		t.Error("Included field should be accessible")
	}
}

func TestFormArrayItemField(t *testing.T) {
	form := UseForm(TestOrder{
		Items: []OrderItem{{ProductID: 1, Quantity: 2}},
	})

	form.Array("items", func(item FormArrayItem, index int) *vdom.VNode {
		input := &vdom.VNode{
			Kind: vdom.KindElement,
			Tag:  "input",
		}
		field := item.Field("product_id", input)

		if field == nil {
			t.Error("FormArrayItem.Field returned nil")
		}

		// Original input should NOT be mutated (we clone it)
		if input.Props != nil && input.Props["name"] != nil {
			t.Errorf("Original input was mutated: name = %v, want nil", input.Props["name"])
		}

		// The cloned input inside the wrapper should have the name set
		if len(field.Children) == 0 {
			t.Error("Field wrapper has no children")
		}
		clonedInput := field.Children[0]
		expectedName := "items.0.product_id"
		if clonedInput.Props["name"] != expectedName {
			t.Errorf("Cloned input name = %v, want %s", clonedInput.Props["name"], expectedName)
		}

		return field
	})
}

func TestFormArrayItemRemove(t *testing.T) {
	form := UseForm(TestOrder{
		Items: []OrderItem{{ProductID: 1}, {ProductID: 2}},
	})

	form.Array("items", func(item FormArrayItem, index int) *vdom.VNode {
		// Get the remove function
		removeFn := item.Remove()
		if removeFn == nil {
			t.Error("Remove() should return a function")
		}
		return nil
	})
}

func TestFormArrayRemoveAtReindexesErrors(t *testing.T) {
	form := UseForm(TestOrder{
		Items: []OrderItem{
			{ProductID: 0, Quantity: 0},
			{ProductID: 0, Quantity: 0},
			{ProductID: 0, Quantity: 0},
		},
	})

	// Set some errors
	form.SetError("items.0.product_id", "Error 0")
	form.SetError("items.1.product_id", "Error 1")
	form.SetError("items.2.product_id", "Error 2")

	// Remove item 1
	form.RemoveAt("items", 1)

	// Error for items.2 should now be at items.1
	if !form.HasError("items.1.product_id") {
		t.Error("Error from items.2 should be reindexed to items.1")
	}

	// Original items.1 error should be gone
	// items.0 error should remain
	if !form.HasError("items.0.product_id") {
		t.Error("Error for items.0 should remain")
	}
}

func TestFormArrayInsertAtEdgeCases(t *testing.T) {
	form := UseForm(TestOrder{
		Items: []OrderItem{{ProductID: 1}},
	})

	// Insert at negative index (should clamp to 0)
	form.InsertAt("items", -5, OrderItem{ProductID: 0})
	values := form.Values()
	if values.Items[0].ProductID != 0 {
		t.Error("Insert at negative should insert at beginning")
	}

	// Insert at index beyond length (should append)
	form.InsertAt("items", 100, OrderItem{ProductID: 99})
	values = form.Values()
	if values.Items[len(values.Items)-1].ProductID != 99 {
		t.Error("Insert beyond length should append")
	}
}

func TestFormGetNilField(t *testing.T) {
	form := UseForm(TestContact{})

	// Get non-existent field
	val := form.Get("nonexistent")
	if val != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", val)
	}
}

func TestFormGetStringNonString(t *testing.T) {
	type NumForm struct {
		Count int `form:"count"`
	}

	form := UseForm(NumForm{Count: 42})

	// GetString on int should return string representation
	str := form.GetString("count")
	if str != "42" {
		t.Errorf("GetString(count) = %s, want '42'", str)
	}
}
