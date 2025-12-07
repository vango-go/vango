package router

import (
	"testing"
)

func TestParamParserString(t *testing.T) {
	type Params struct {
		Name string `param:"name"`
	}

	parser := NewParamParser()
	params := map[string]string{"name": "test"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if p.Name != "test" {
		t.Errorf("Name = %q, want %q", p.Name, "test")
	}
}

func TestParamParserInt(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "123"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if p.ID != 123 {
		t.Errorf("ID = %d, want %d", p.ID, 123)
	}
}

func TestParamParserInt64(t *testing.T) {
	type Params struct {
		ID int64 `param:"id"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "9223372036854775807"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if p.ID != 9223372036854775807 {
		t.Errorf("ID = %d, want 9223372036854775807", p.ID)
	}
}

func TestParamParserUint(t *testing.T) {
	type Params struct {
		Count uint `param:"count"`
	}

	parser := NewParamParser()
	params := map[string]string{"count": "42"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if p.Count != 42 {
		t.Errorf("Count = %d, want %d", p.Count, 42)
	}
}

func TestParamParserFloat(t *testing.T) {
	type Params struct {
		Price float64 `param:"price"`
	}

	parser := NewParamParser()
	params := map[string]string{"price": "19.99"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if p.Price != 19.99 {
		t.Errorf("Price = %f, want %f", p.Price, 19.99)
	}
}

func TestParamParserBool(t *testing.T) {
	type Params struct {
		Active bool `param:"active"`
	}

	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"1", true},
		{"0", false},
	}

	parser := NewParamParser()
	for _, tt := range tests {
		params := map[string]string{"active": tt.value}

		var p Params
		if err := parser.Parse(params, &p); err != nil {
			t.Fatalf("Parse(%q) error: %v", tt.value, err)
		}

		if p.Active != tt.want {
			t.Errorf("Parse(%q) Active = %v, want %v", tt.value, p.Active, tt.want)
		}
	}
}

func TestParamParserSlice(t *testing.T) {
	type Params struct {
		Slug []string `param:"slug"`
	}

	parser := NewParamParser()
	params := map[string]string{"slug": "a/b/c"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(p.Slug) != 3 {
		t.Fatalf("len(Slug) = %d, want 3", len(p.Slug))
	}
	if p.Slug[0] != "a" || p.Slug[1] != "b" || p.Slug[2] != "c" {
		t.Errorf("Slug = %v, want [a b c]", p.Slug)
	}
}

func TestParamParserEmptySlice(t *testing.T) {
	type Params struct {
		Slug []string `param:"slug"`
	}

	parser := NewParamParser()
	params := map[string]string{"slug": ""}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Empty string should result in empty slice (or nil)
	if len(p.Slug) != 0 {
		t.Errorf("len(Slug) = %d, want 0", len(p.Slug))
	}
}

func TestParamParserMultipleFields(t *testing.T) {
	type Params struct {
		ID   int    `param:"id"`
		Name string `param:"name"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "123", "name": "test"}

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if p.ID != 123 {
		t.Errorf("ID = %d, want %d", p.ID, 123)
	}
	if p.Name != "test" {
		t.Errorf("Name = %q, want %q", p.Name, "test")
	}
}

func TestParamParserMissingParam(t *testing.T) {
	type Params struct {
		ID   int    `param:"id"`
		Name string `param:"name"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "123"} // name missing

	var p Params
	if err := parser.Parse(params, &p); err != nil {
		t.Fatalf("Parse() should not error on missing param: %v", err)
	}

	if p.ID != 123 {
		t.Errorf("ID = %d, want %d", p.ID, 123)
	}
	if p.Name != "" {
		t.Errorf("Name = %q, want empty", p.Name)
	}
}

func TestParamParserInvalidInt(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "notanumber"}

	var p Params
	err := parser.Parse(params, &p)
	if err == nil {
		t.Error("Parse() should error on invalid int")
	}
}

func TestParamParserNotPointer(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	parser := NewParamParser()
	params := map[string]string{"id": "123"}

	var p Params
	err := parser.Parse(params, p) // Not a pointer
	if err == nil {
		t.Error("Parse() should error when target is not a pointer")
	}
}

func TestParamParserNil(t *testing.T) {
	parser := NewParamParser()
	params := map[string]string{"id": "123"}

	// Should not error on nil target
	if err := parser.Parse(params, nil); err != nil {
		t.Errorf("Parse(nil) error: %v", err)
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", false},
		{"550E8400-E29B-41D4-A716-446655440000", false},
		{"not-a-uuid", true},
		{"550e8400-e29b-41d4-a716", true},
		{"", true},
	}

	for _, tt := range tests {
		err := ValidateUUID(tt.value)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateUUID(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestValidateInt(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"123", false},
		{"-456", false},
		{"0", false},
		{"abc", true},
		{"12.34", true},
		{"", true},
	}

	for _, tt := range tests {
		err := ValidateInt(tt.value)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateInt(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestValidateParam(t *testing.T) {
	tests := []struct {
		value     string
		paramType string
		wantErr   bool
	}{
		{"123", "int", false},
		{"abc", "int", true},
		{"550e8400-e29b-41d4-a716-446655440000", "uuid", false},
		{"not-uuid", "uuid", true},
		{"anything", "string", false},
		{"anything", "", false},
		{"anything", "unknown", false},
	}

	for _, tt := range tests {
		err := ValidateParam(tt.value, tt.paramType)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateParam(%q, %q) error = %v, wantErr %v", tt.value, tt.paramType, err, tt.wantErr)
		}
	}
}
