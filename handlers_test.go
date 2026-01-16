package vango

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/server"
	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

type dirtyListener struct {
	id      uint64
	dirtyCt atomic.Uint64
}

func (l *dirtyListener) ID() uint64 { return l.id }
func (l *dirtyListener) MarkDirty() { l.dirtyCt.Add(1) }

func renderWithOwner(owner *corevango.Owner, listener corevango.Listener, comp vdom.Component) *vdom.VNode {
	var out *vdom.VNode
	corevango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()
		corevango.WithListener(listener, func() {
			out = comp.Render()
		})
	})
	return out
}

func TestWrapPageHandler_SignalUpdatesMarkDirty(t *testing.T) {
	var renderCalls atomic.Uint64
	var sig *Signal[int]

	page := func(ctx Ctx) *VNode {
		renderCalls.Add(1)
		sig = NewSignal(0)
		return &vdom.VNode{
			Kind: vdom.KindElement,
			Tag:  "div",
			Children: []*vdom.VNode{
				{Kind: vdom.KindText, Text: fmt.Sprintf("%d", sig.Get())},
			},
		}
	}

	internal := wrapPageHandler(page)
	comp := internal(nil, nil)

	owner := corevango.NewOwner(nil)
	listener := &dirtyListener{id: owner.ID()}

	n1 := renderWithOwner(owner, listener, comp)
	if n1 == nil {
		t.Fatal("expected rendered node")
	}
	if sig == nil {
		t.Fatal("expected signal to be created during render")
	}
	if got := renderCalls.Load(); got != 1 {
		t.Fatalf("renderCalls = %d, want 1", got)
	}
	if got := listener.dirtyCt.Load(); got != 0 {
		t.Fatalf("dirtyCt after initial render = %d, want 0", got)
	}

	sig.Inc()
	if got := listener.dirtyCt.Load(); got == 0 {
		t.Fatal("expected signal update to mark listener dirty")
	}

	n2 := renderWithOwner(owner, listener, comp)
	if got := renderCalls.Load(); got != 2 {
		t.Fatalf("renderCalls after second render = %d, want 2", got)
	}
	if n2 == nil || len(n2.Children) == 0 || n2.Children[0].Kind != vdom.KindText {
		t.Fatal("expected text child in rendered node")
	}
	if got := n2.Children[0].Text; got != "1" {
		t.Fatalf("rendered text = %q, want %q", got, "1")
	}
}

func TestBuildParamDecoder_SupportsSlicesAndTextUnmarshalers(t *testing.T) {
	type Params struct {
		ID   int       `param:"id"`
		Slug []string  `param:"slug"`
		When time.Time `param:"when"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"id":   "123",
		"slug": "a/b/c",
		"when": "2020-01-02T03:04:05Z",
	})
	p := val.Interface().(Params)

	if got, want := p.ID, 123; got != want {
		t.Fatalf("ID = %d, want %d", got, want)
	}
	if got, want := len(p.Slug), 3; got != want {
		t.Fatalf("len(Slug) = %d, want %d", got, want)
	}
	if got, want := p.Slug[0], "a"; got != want {
		t.Fatalf("Slug[0] = %q, want %q", got, want)
	}
	if got, want := p.Slug[2], "c"; got != want {
		t.Fatalf("Slug[2] = %q, want %q", got, want)
	}
	if p.When.IsZero() {
		t.Fatal("When should not be zero")
	}
}

func TestBuildParamDecoder_UnsupportedFieldTypeDoesNotPanic(t *testing.T) {
	type Params struct {
		Ch chan int `param:"ch"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	val := decoder(map[string]string{"ch": "x"})
	p := val.Interface().(Params)
	if p.Ch != nil {
		t.Fatal("expected Ch to remain nil for unsupported field type")
	}
}

func TestWrapPageHandler_NamedFuncTypeUsesReflection(t *testing.T) {
	type pageFn func(Ctx) *VNode

	var got Ctx
	handler := pageFn(func(ctx Ctx) *VNode {
		got = ctx
		return vdom.Text("ok")
	})

	internal := wrapPageHandler(handler)
	ctx := server.NewTestContext(nil)
	comp := internal(ctx, nil)

	node := comp.Render()
	if got == nil {
		t.Fatal("expected handler to receive ctx")
	}
	if got != ctx {
		t.Fatal("expected ctx passed through to handler")
	}
	if node == nil || node.Kind != vdom.KindText || node.Text != "ok" {
		t.Fatalf("unexpected render result: %#v", node)
	}
}

func TestWrapAPIHandler_DisambiguatesParamsStruct(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	handler := func(ctx Ctx, p *Params) (any, error) {
		if p == nil {
			return nil, errors.New("nil params")
		}
		return map[string]int{"id": p.ID}, nil
	}

	internal := wrapAPIHandler(handler)
	ctx := server.NewTestContext(nil)
	out, err := internal(ctx, map[string]string{"id": "42"}, nil)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	got, ok := out.(map[string]int)
	if !ok {
		t.Fatalf("result type = %T, want map[string]int", out)
	}
	if got["id"] != 42 {
		t.Fatalf("id = %d, want %d", got["id"], 42)
	}
}

func TestWrapAPIHandler_DisambiguatesBodyStruct(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	handler := func(ctx Ctx, input Input) (any, error) {
		return input.Name, nil
	}

	internal := wrapAPIHandler(handler)
	ctx := server.NewTestContext(nil)
	raw := apiRawBody{
		Bytes:       []byte(`{"name":"bob"}`),
		ContentType: "application/json",
	}
	out, err := internal(ctx, nil, raw)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if out != "bob" {
		t.Fatalf("result = %v, want %q", out, "bob")
	}
}

func TestDecodeAPIRequestBody_PassThroughTypes(t *testing.T) {
	raw := apiRawBody{Bytes: []byte("hello")}

	cases := []struct {
		name   string
		target reflect.Type
		want   string
	}{
		{name: "bytes", target: reflect.TypeOf([]byte(nil)), want: "hello"},
		{name: "string", target: reflect.TypeOf(""), want: "hello"},
		{name: "rawmessage", target: reflect.TypeOf(json.RawMessage{}), want: "hello"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := decodeAPIRequestBody(raw, tc.target)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			switch v := val.Interface().(type) {
			case []byte:
				if string(v) != tc.want {
					t.Fatalf("value = %q, want %q", string(v), tc.want)
				}
			case string:
				if v != tc.want {
					t.Fatalf("value = %q, want %q", v, tc.want)
				}
			case json.RawMessage:
				if string(v) != tc.want {
					t.Fatalf("value = %q, want %q", string(v), tc.want)
				}
			default:
				t.Fatalf("unexpected value type: %T", v)
			}
		})
	}
}

func TestDecodeAPIRequestBody_StrictContentTypeRequiresJSON(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:                 []byte(`{"name":"alice"}`),
		StrictJSONContentType: true,
	}
	_, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err == nil {
		t.Fatal("expected error for missing content type")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", httpErr.Code, http.StatusUnsupportedMediaType)
	}
}

func TestDecodeAPIRequestBody_JSONNullPointer(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:       []byte("null"),
		ContentType: "application/json",
	}
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(&Input{}))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if val.Kind() != reflect.Ptr || !val.IsNil() {
		t.Fatalf("expected nil pointer, got %#v", val.Interface())
	}
}

func TestDecodeAPIRequestBody_EmptyBodyReturnsBadRequest(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:       []byte("   "),
		ContentType: "application/json",
	}
	_, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", httpErr.Code, http.StatusBadRequest)
	}
}

func TestBuildParamDecoder_RespectsTagsDefaultsAndPointers(t *testing.T) {
	type Params struct {
		ID      int  `param:"id"`
		Skip    string `param:"-"`
		Name    string
		Count   *int `param:"count"`
		Enabled bool `param:"enabled"`
		BadInt  int  `param:"badint"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"id":      "123",
		"name":    "alice",
		"count":   "7",
		"enabled": "true",
		"badint":  "nope",
		"skip":    "ignored",
	})
	p := val.Interface().(Params)

	if p.ID != 123 {
		t.Fatalf("ID = %d, want %d", p.ID, 123)
	}
	if p.Skip != "" {
		t.Fatalf("Skip = %q, want empty", p.Skip)
	}
	if p.Name != "alice" {
		t.Fatalf("Name = %q, want %q", p.Name, "alice")
	}
	if p.Count == nil || *p.Count != 7 {
		t.Fatalf("Count = %#v, want pointer to %d", p.Count, 7)
	}
	if !p.Enabled {
		t.Fatalf("Enabled = %v, want true", p.Enabled)
	}
	if p.BadInt != 0 {
		t.Fatalf("BadInt = %d, want 0", p.BadInt)
	}
}

func TestBuildParamDecoder_PointerStructReturnsPointer(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(&Params{}))
	val := decoder(map[string]string{"id": "9"})

	p, ok := val.Interface().(*Params)
	if !ok {
		t.Fatalf("result type = %T, want *Params", val.Interface())
	}
	if p == nil || p.ID != 9 {
		t.Fatalf("params = %#v, want ID=9", p)
	}
}

// =============================================================================
// Additional decodeAPIRequestBody Tests
// =============================================================================

func TestDecodeAPIRequestBody_NilTargetType(t *testing.T) {
	raw := apiRawBody{Bytes: []byte(`{"name":"test"}`)}
	_, err := decodeAPIRequestBody(raw, nil)
	if err == nil {
		t.Fatal("expected error for nil target type")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", httpErr.Code, http.StatusInternalServerError)
	}
}

func TestDecodeAPIRequestBody_PointerToApiRawBody(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := &apiRawBody{
		Bytes:       []byte(`{"name":"pointer"}`),
		ContentType: "application/json",
	}
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	input := val.Interface().(Input)
	if input.Name != "pointer" {
		t.Fatalf("Name = %q, want %q", input.Name, "pointer")
	}
}

func TestDecodeAPIRequestBody_NilPointerToApiRawBody(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	var raw *apiRawBody = nil
	_, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err == nil {
		t.Fatal("expected error for nil pointer body with non-pointer target")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", httpErr.Code, http.StatusBadRequest)
	}
}

func TestDecodeAPIRequestBody_JSONRawMessageInput(t *testing.T) {
	type Input struct {
		Value int `json:"value"`
	}

	raw := json.RawMessage(`{"value":42}`)
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	input := val.Interface().(Input)
	if input.Value != 42 {
		t.Fatalf("Value = %d, want %d", input.Value, 42)
	}
}

func TestDecodeAPIRequestBody_StringInput(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := `{"name":"from-string"}`
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	input := val.Interface().(Input)
	if input.Name != "from-string" {
		t.Fatalf("Name = %q, want %q", input.Name, "from-string")
	}
}

func TestDecodeAPIRequestBody_ByteSliceInput(t *testing.T) {
	type Input struct {
		Count int `json:"count"`
	}

	raw := []byte(`{"count":99}`)
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	input := val.Interface().(Input)
	if input.Count != 99 {
		t.Fatalf("Count = %d, want %d", input.Count, 99)
	}
}

func TestDecodeAPIRequestBody_InvalidJSON(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:       []byte(`{invalid json}`),
		ContentType: "application/json",
	}
	_, err := decodeAPIRequestBody(raw, reflect.TypeOf(Input{}))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", httpErr.Code, http.StatusBadRequest)
	}
}

func TestDecodeAPIRequestBody_InvalidJSONPointerType(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:       []byte(`{not valid}`),
		ContentType: "application/json",
	}
	_, err := decodeAPIRequestBody(raw, reflect.TypeOf(&Input{}))
	if err == nil {
		t.Fatal("expected error for invalid JSON with pointer target")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", httpErr.Code, http.StatusBadRequest)
	}
}

func TestDecodeAPIRequestBody_EmptyBodyPointerType(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:       []byte("   "),
		ContentType: "application/json",
	}
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(&Input{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty body with pointer type returns nil pointer
	if !val.IsNil() {
		t.Fatalf("expected nil pointer for empty body, got %v", val.Interface())
	}
}

func TestDecodeAPIRequestBody_ValidJSONToPointer(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	raw := apiRawBody{
		Bytes:       []byte(`{"name":"alice"}`),
		ContentType: "application/json",
	}
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(&Input{}))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	input := val.Interface().(*Input)
	if input == nil {
		t.Fatal("expected non-nil pointer")
	}
	if input.Name != "alice" {
		t.Fatalf("Name = %q, want %q", input.Name, "alice")
	}
}

func TestDecodeAPIRequestBody_DirectAssignment(t *testing.T) {
	// Test when raw value is already assignable to target type
	type MyString string
	raw := MyString("direct")
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(MyString("")))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if val.Interface().(MyString) != "direct" {
		t.Fatalf("value = %v, want %q", val.Interface(), "direct")
	}
}

func TestDecodeAPIRequestBody_ConvertibleType(t *testing.T) {
	// Test when raw value is convertible to target type
	type MyInt int
	raw := 42
	val, err := decodeAPIRequestBody(raw, reflect.TypeOf(MyInt(0)))
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if val.Interface().(MyInt) != 42 {
		t.Fatalf("value = %v, want %d", val.Interface(), 42)
	}
}

// =============================================================================
// isJSONContentType Tests
// =============================================================================

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"APPLICATION/JSON", true},
		{"application/vnd.api+json", true},
		{"text/plain", false},
		{"text/html", false},
		{"", false},
		{"application/xml", false},
		{"application/javascript", false},
	}

	for _, tc := range tests {
		t.Run(tc.contentType, func(t *testing.T) {
			got := isJSONContentType(tc.contentType)
			if got != tc.want {
				t.Errorf("isJSONContentType(%q) = %v, want %v", tc.contentType, got, tc.want)
			}
		})
	}
}

// =============================================================================
// getTypeConverter Tests for Various Numeric Types
// =============================================================================

func TestGetTypeConverter_IntTypes(t *testing.T) {
	type Params struct {
		I    int    `param:"i"`
		I8   int8   `param:"i8"`
		I16  int16  `param:"i16"`
		I32  int32  `param:"i32"`
		I64  int64  `param:"i64"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"i":   "100",
		"i8":  "127",
		"i16": "32767",
		"i32": "2147483647",
		"i64": "9223372036854775807",
	})
	p := val.Interface().(Params)

	if p.I != 100 {
		t.Errorf("I = %d, want %d", p.I, 100)
	}
	if p.I8 != 127 {
		t.Errorf("I8 = %d, want %d", p.I8, 127)
	}
	if p.I16 != 32767 {
		t.Errorf("I16 = %d, want %d", p.I16, 32767)
	}
	if p.I32 != 2147483647 {
		t.Errorf("I32 = %d, want %d", p.I32, 2147483647)
	}
	if p.I64 != 9223372036854775807 {
		t.Errorf("I64 = %d, want %d", p.I64, int64(9223372036854775807))
	}
}

func TestGetTypeConverter_UintTypes(t *testing.T) {
	type Params struct {
		U    uint    `param:"u"`
		U8   uint8   `param:"u8"`
		U16  uint16  `param:"u16"`
		U32  uint32  `param:"u32"`
		U64  uint64  `param:"u64"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"u":   "100",
		"u8":  "255",
		"u16": "65535",
		"u32": "4294967295",
		"u64": "18446744073709551615",
	})
	p := val.Interface().(Params)

	if p.U != 100 {
		t.Errorf("U = %d, want %d", p.U, 100)
	}
	if p.U8 != 255 {
		t.Errorf("U8 = %d, want %d", p.U8, 255)
	}
	if p.U16 != 65535 {
		t.Errorf("U16 = %d, want %d", p.U16, 65535)
	}
	if p.U32 != 4294967295 {
		t.Errorf("U32 = %d, want %d", p.U32, uint32(4294967295))
	}
	if p.U64 != 18446744073709551615 {
		t.Errorf("U64 = %d, want %d", p.U64, uint64(18446744073709551615))
	}
}

func TestGetTypeConverter_FloatTypes(t *testing.T) {
	type Params struct {
		F32 float32 `param:"f32"`
		F64 float64 `param:"f64"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"f32": "3.14",
		"f64": "2.718281828",
	})
	p := val.Interface().(Params)

	if p.F32 < 3.13 || p.F32 > 3.15 {
		t.Errorf("F32 = %f, want ~3.14", p.F32)
	}
	if p.F64 < 2.71 || p.F64 > 2.72 {
		t.Errorf("F64 = %f, want ~2.718", p.F64)
	}
}

func TestGetTypeConverter_InvalidNumericValues(t *testing.T) {
	type Params struct {
		I   int     `param:"i"`
		U   uint    `param:"u"`
		F   float64 `param:"f"`
		B   bool    `param:"b"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"i": "not-an-int",
		"u": "not-a-uint",
		"f": "not-a-float",
		"b": "not-a-bool",
	})
	p := val.Interface().(Params)

	// Invalid values should result in zero values
	if p.I != 0 {
		t.Errorf("I = %d, want 0 for invalid input", p.I)
	}
	if p.U != 0 {
		t.Errorf("U = %d, want 0 for invalid input", p.U)
	}
	if p.F != 0 {
		t.Errorf("F = %f, want 0 for invalid input", p.F)
	}
	if p.B != false {
		t.Errorf("B = %v, want false for invalid input", p.B)
	}
}

func TestGetTypeConverter_EmptySlice(t *testing.T) {
	type Params struct {
		Tags []string `param:"tags"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{
		"tags": "",
	})
	p := val.Interface().(Params)

	// Empty string should result in empty slice (not nil)
	if p.Tags != nil && len(p.Tags) != 0 {
		t.Errorf("Tags = %v, want nil or empty for empty input", p.Tags)
	}
}

// =============================================================================
// wrapPageHandler Panic Tests
// =============================================================================

func TestWrapPageHandler_PanicsOnNonFunction(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-function handler")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "must be a function") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	wrapPageHandler("not a function")
}

func TestWrapPageHandler_PanicsOnWrongOutputCount(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for wrong output count")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "must return exactly 1 value") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	// Handler that returns nothing
	handler := func(ctx Ctx) {}
	wrapPageHandler(handler)
}

func TestWrapPageHandler_PanicsOnInvalidArgCount(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for invalid arg count")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "invalid signature") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	// Handler with 3 args (invalid)
	handler := func(ctx Ctx, a, b int) *VNode { return nil }
	wrapPageHandler(handler)
}

func TestWrapPageHandler_TwoArgWithParams(t *testing.T) {
	type Params struct {
		ID   int    `param:"id"`
		Slug string `param:"slug"`
	}

	var receivedParams Params
	handler := func(ctx Ctx, p Params) *VNode {
		receivedParams = p
		return vdom.Text(fmt.Sprintf("id=%d,slug=%s", p.ID, p.Slug))
	}

	internal := wrapPageHandler(handler)
	ctx := server.NewTestContext(nil)
	comp := internal(ctx, map[string]string{"id": "42", "slug": "hello"})
	node := comp.Render()

	if receivedParams.ID != 42 {
		t.Errorf("ID = %d, want %d", receivedParams.ID, 42)
	}
	if receivedParams.Slug != "hello" {
		t.Errorf("Slug = %q, want %q", receivedParams.Slug, "hello")
	}
	if node == nil || node.Text != "id=42,slug=hello" {
		t.Errorf("unexpected render result: %v", node)
	}
}

func TestWrapPageHandler_TwoArgWithNilParams(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	handler := func(ctx Ctx, p Params) *VNode {
		return vdom.Text(fmt.Sprintf("id=%d", p.ID))
	}

	internal := wrapPageHandler(handler)
	ctx := server.NewTestContext(nil)
	// Pass nil instead of map
	comp := internal(ctx, nil)
	node := comp.Render()

	// Should handle nil gracefully with zero values
	if node == nil || node.Text != "id=0" {
		t.Errorf("unexpected render result: %v", node)
	}
}

// =============================================================================
// wrapAPIHandler Tests
// =============================================================================

func TestWrapAPIHandler_PanicsOnNonFunction(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-function handler")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "must be a function") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	wrapAPIHandler(123)
}

func TestWrapAPIHandler_PanicsOnWrongOutputCount(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for wrong output count")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "must return exactly 2 values") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	handler := func(ctx Ctx) string { return "oops" }
	wrapAPIHandler(handler)
}

func TestWrapAPIHandler_PanicsOnInvalidArgCount(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for invalid arg count")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "invalid signature") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	handler := func(ctx Ctx, a, b, c, d int) (any, error) { return nil, nil }
	wrapAPIHandler(handler)
}

func TestWrapAPIHandler_ThreeArgs_ParamsAndBody(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}
	type Body struct {
		Name string `json:"name"`
	}

	handler := func(ctx Ctx, p Params, b Body) (any, error) {
		return map[string]any{"id": p.ID, "name": b.Name}, nil
	}

	internal := wrapAPIHandler(handler)
	ctx := server.NewTestContext(nil)
	raw := apiRawBody{
		Bytes:       []byte(`{"name":"alice"}`),
		ContentType: "application/json",
	}
	out, err := internal(ctx, map[string]string{"id": "99"}, raw)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	result := out.(map[string]any)
	if result["id"] != 99 {
		t.Errorf("id = %v, want %d", result["id"], 99)
	}
	if result["name"] != "alice" {
		t.Errorf("name = %v, want %q", result["name"], "alice")
	}
}

func TestWrapAPIHandler_OneArg_NoParamsOrBody(t *testing.T) {
	handler := func(ctx Ctx) (any, error) {
		return map[string]string{"status": "ok"}, nil
	}

	internal := wrapAPIHandler(handler)
	ctx := server.NewTestContext(nil)
	out, err := internal(ctx, nil, nil)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	result := out.(map[string]string)
	if result["status"] != "ok" {
		t.Errorf("status = %q, want %q", result["status"], "ok")
	}
}

func TestWrapAPIHandler_ReturnsError(t *testing.T) {
	expectedErr := errors.New("test error")
	handler := func(ctx Ctx) (any, error) {
		return nil, expectedErr
	}

	internal := wrapAPIHandler(handler)
	ctx := server.NewTestContext(nil)
	_, err := internal(ctx, nil, nil)
	if err != expectedErr {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}
}

func TestWrapAPIHandler_BodyDecodingError(t *testing.T) {
	type Body struct {
		Value int `json:"value"`
	}

	handler := func(ctx Ctx, b Body) (any, error) {
		return b.Value, nil
	}

	internal := wrapAPIHandler(handler)
	ctx := server.NewTestContext(nil)
	raw := apiRawBody{
		Bytes:       []byte(`{invalid}`),
		ContentType: "application/json",
	}
	_, err := internal(ctx, nil, raw)
	if err == nil {
		t.Fatal("expected error for invalid body")
	}
}

// =============================================================================
// hasParamStructTags Tests
// =============================================================================

func TestHasParamStructTags_NonStruct(t *testing.T) {
	if hasParamStructTags(reflect.TypeOf(42)) {
		t.Error("hasParamStructTags should return false for non-struct")
	}
}

func TestHasParamStructTags_StructWithoutTags(t *testing.T) {
	type NoTags struct {
		Name string
		Age  int
	}
	if hasParamStructTags(reflect.TypeOf(NoTags{})) {
		t.Error("hasParamStructTags should return false for struct without param tags")
	}
}

func TestHasParamStructTags_StructWithTags(t *testing.T) {
	type WithTags struct {
		ID int `param:"id"`
	}
	if !hasParamStructTags(reflect.TypeOf(WithTags{})) {
		t.Error("hasParamStructTags should return true for struct with param tags")
	}
}

func TestHasParamStructTags_PointerToStruct(t *testing.T) {
	type WithTags struct {
		ID int `param:"id"`
	}
	if !hasParamStructTags(reflect.TypeOf(&WithTags{})) {
		t.Error("hasParamStructTags should return true for pointer to struct with param tags")
	}
}

// =============================================================================
// buildParamDecoder Edge Cases
// =============================================================================

func TestBuildParamDecoder_PanicsOnNonStruct(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for non-struct type")
		}
		msg := fmt.Sprint(r)
		if !containsString(msg, "must be a struct") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()

	buildParamDecoder(reflect.TypeOf(42))
}

func TestBuildParamDecoder_EmptyParams(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}

	decoder := buildParamDecoder(reflect.TypeOf(Params{}))
	val := decoder(map[string]string{})
	p := val.Interface().(Params)

	if p.ID != 0 {
		t.Errorf("ID = %d, want 0 for empty params", p.ID)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || containsString(s[1:], substr)))
}
