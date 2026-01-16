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
