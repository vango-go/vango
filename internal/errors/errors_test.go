package errors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		wantMsg  string
		wantCat  Category
	}{
		{
			name:    "known error code",
			code:    "E001",
			wantMsg: "Signal read outside component context",
			wantCat: CategoryRuntime,
		},
		{
			name:    "hydration error",
			code:    "E040",
			wantMsg: "Hydration mismatch: element type differs",
			wantCat: CategoryHydration,
		},
		{
			name:    "protocol error",
			code:    "E060",
			wantMsg: "WebSocket connection failed",
			wantCat: CategoryProtocol,
		},
		{
			name:    "unknown error code",
			code:    "E999",
			wantMsg: "Unknown error",
			wantCat: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code)
			if err.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
			if err.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", err.Category, tt.wantCat)
			}
			if err.Code != tt.code {
				t.Errorf("Code = %q, want %q", err.Code, tt.code)
			}
		})
	}
}

func TestNewf(t *testing.T) {
	err := Newf(CategoryRuntime, "file %q not found", "test.go")
	if err.Message != `file "test.go" not found` {
		t.Errorf("Message = %q, want %q", err.Message, `file "test.go" not found`)
	}
	if err.Category != CategoryRuntime {
		t.Errorf("Category = %q, want %q", err.Category, CategoryRuntime)
	}
}

func TestVangoError_Error(t *testing.T) {
	err := New("E001")
	got := err.Error()
	want := "E001: Signal read outside component context"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// Without code
	err2 := &VangoError{Message: "test error"}
	if err2.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err2.Error(), "test error")
	}
}

func TestVangoError_WithLocation(t *testing.T) {
	// Create a temp file with some content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
    fmt.Println("Hello")
    count := vango.Signal(0)
    value := count()
    fmt.Println(value)
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := New("E001").WithLocation(tmpFile, 6, 12)

	if err.Location == nil {
		t.Fatal("Location is nil")
	}
	if err.Location.File != tmpFile {
		t.Errorf("Location.File = %q, want %q", err.Location.File, tmpFile)
	}
	if err.Location.Line != 6 {
		t.Errorf("Location.Line = %d, want %d", err.Location.Line, 6)
	}
	if err.Location.Column != 12 {
		t.Errorf("Location.Column = %d, want %d", err.Location.Column, 12)
	}
	if len(err.Context) == 0 {
		t.Error("Context should not be empty")
	}
}

func TestVangoError_WithSuggestion(t *testing.T) {
	err := New("E001").WithSuggestion("Wrap your component in vango.Func()")
	if err.Suggestion != "Wrap your component in vango.Func()" {
		t.Errorf("Suggestion = %q, want %q", err.Suggestion, "Wrap your component in vango.Func()")
	}
}

func TestVangoError_WithExample(t *testing.T) {
	example := `func Counter() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(0)
        return Div(Text(fmt.Sprintf("%d", count())))
    })
}`
	err := New("E001").WithExample(example)
	if err.Example != example {
		t.Errorf("Example = %q, want %q", err.Example, example)
	}
}

func TestVangoError_WithDetail(t *testing.T) {
	err := New("E001").WithDetail("Custom detail")
	if err.Detail != "Custom detail" {
		t.Errorf("Detail = %q, want %q", err.Detail, "Custom detail")
	}
}

func TestVangoError_Wrap(t *testing.T) {
	inner := New("E002")
	outer := New("E001").Wrap(inner)

	if outer.Wrapped != inner {
		t.Error("Wrapped error mismatch")
	}
	if outer.Unwrap() != inner {
		t.Error("Unwrap() should return wrapped error")
	}
}

func TestFromError(t *testing.T) {
	// nil error
	if FromError(nil, "E001") != nil {
		t.Error("FromError(nil, ...) should return nil")
	}

	// Already VangoError
	ve := New("E001")
	if FromError(ve, "E002") != ve {
		t.Error("FromError should return VangoError as-is")
	}

	// Standard error
	stdErr := &testError{msg: "test error"}
	result := FromError(stdErr, "E001")
	if result.Wrapped != stdErr {
		t.Error("Standard error should be wrapped")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestLocation_String(t *testing.T) {
	tests := []struct {
		name string
		loc  *Location
		want string
	}{
		{
			name: "nil location",
			loc:  nil,
			want: "",
		},
		{
			name: "with column",
			loc:  &Location{File: "test.go", Line: 10, Column: 5},
			want: "test.go:10:5",
		},
		{
			name: "without column",
			loc:  &Location{File: "test.go", Line: 10, Column: 0},
			want: "test.go:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.loc.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	DisableColors()
	defer EnableColors()

	// Create a temp file with some content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
    count := vango.Signal(0)
    value := count()
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := New("E001").
		WithLocation(tmpFile, 5, 12).
		WithSuggestion("Wrap your component in vango.Func()").
		WithExample("return vango.Func(func() *vango.VNode { ... })")

	formatted := err.Format()

	// Check that key components are present
	if !strings.Contains(formatted, "E001") {
		t.Error("Format should contain error code")
	}
	if !strings.Contains(formatted, "Signal read outside component context") {
		t.Error("Format should contain error message")
	}
	if !strings.Contains(formatted, tmpFile) {
		t.Error("Format should contain file path")
	}
	if !strings.Contains(formatted, "Hint:") {
		t.Error("Format should contain hint")
	}
	if !strings.Contains(formatted, "Example:") {
		t.Error("Format should contain example")
	}
	if !strings.Contains(formatted, "Learn more:") {
		t.Error("Format should contain doc URL")
	}
}

func TestFormatCompact(t *testing.T) {
	err := New("E001").WithLocation("test.go", 10, 5)
	compact := err.FormatCompact()

	want := "test.go:10:5: E001: Signal read outside component context"
	if compact != want {
		t.Errorf("FormatCompact() = %q, want %q", compact, want)
	}
}

func TestFormatJSON(t *testing.T) {
	err := New("E001").WithLocation("test.go", 10, 5)
	json := err.FormatJSON()

	if !strings.Contains(json, `"code":"E001"`) {
		t.Error("JSON should contain code")
	}
	if !strings.Contains(json, `"category":"runtime"`) {
		t.Error("JSON should contain category")
	}
	if !strings.Contains(json, `"message":"Signal read outside component context"`) {
		t.Error("JSON should contain message")
	}
	if !strings.Contains(json, `"location":`) {
		t.Error("JSON should contain location")
	}
}

func TestGetAllCodes(t *testing.T) {
	codes := GetAllCodes()
	if len(codes) == 0 {
		t.Error("GetAllCodes() should return codes")
	}

	// Check that E001 is in the list
	found := false
	for _, code := range codes {
		if code == "E001" {
			found = true
			break
		}
	}
	if !found {
		t.Error("E001 should be in the codes list")
	}
}

func TestGetTemplate(t *testing.T) {
	template, ok := GetTemplate("E001")
	if !ok {
		t.Error("E001 should exist")
	}
	if template.Message != "Signal read outside component context" {
		t.Error("Template message mismatch")
	}

	_, ok = GetTemplate("E999")
	if ok {
		t.Error("E999 should not exist")
	}
}

func TestRegister(t *testing.T) {
	Register("E999", ErrorTemplate{
		Category: CategoryRuntime,
		Message:  "Custom test error",
		Detail:   "This is a test error",
		DocURL:   "https://test.dev/E999",
	})

	err := New("E999")
	if err.Message != "Custom test error" {
		t.Errorf("Message = %q, want %q", err.Message, "Custom test error")
	}

	// Cleanup
	delete(registry, "E999")
}

func TestWrapText(t *testing.T) {
	// Test short text that doesn't need wrapping
	got := wrapText("short text", 100)
	if len(got) != 1 || got[0] != "short text" {
		t.Errorf("wrapText short text: got %v", got)
	}

	// Test text that needs wrapping
	got = wrapText("this is a longer text that should be wrapped", 20)
	if len(got) != 3 {
		t.Errorf("wrapText long text: expected 3 lines, got %d: %v", len(got), got)
	}

	// Test empty string returns empty/nil
	got = wrapText("", 10)
	if len(got) != 0 {
		t.Errorf("wrapText empty: expected empty, got %v", got)
	}
}

func TestColorFunctions(t *testing.T) {
	// With colors enabled
	EnableColors()
	if !strings.Contains(red("test"), "\033[31m") {
		t.Error("red should contain ANSI code when colors enabled")
	}

	// With colors disabled
	DisableColors()
	if strings.Contains(red("test"), "\033[") {
		t.Error("red should not contain ANSI code when colors disabled")
	}
	EnableColors()
}
