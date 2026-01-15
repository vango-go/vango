package router

import (
	"testing"

	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestRouterMatchParamDecodesPercentEscapes(t *testing.T) {
	r := NewRouter()
	r.AddPage("/hello/:name", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/hello/hello%20world")
	if !ok {
		t.Fatal("expected match for /hello/hello%20world")
	}
	if got := result.Params["name"]; got != "hello world" {
		t.Fatalf("params[name] = %q, want %q", got, "hello world")
	}
}

func TestRouterRejectsEncodedSlashInParam(t *testing.T) {
	r := NewRouter()
	r.AddPage("/users/:id", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	if _, ok := r.Match("GET", "/users/123%2Fprofile"); ok {
		t.Fatal("expected no match for encoded slash in param segment")
	}
}

func TestRouterAllowsEncodedSlashInCatchAll(t *testing.T) {
	r := NewRouter()
	r.AddPage("/files/*path", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/files/a%2Fb/c")
	if !ok {
		t.Fatal("expected match for /files/a%2Fb/c")
	}
	if got := result.Params["path"]; got != "a/b/c" {
		t.Fatalf("params[path] = %q, want %q", got, "a/b/c")
	}
}

func TestRouterMatchStaticDecodesPercentEscapes(t *testing.T) {
	r := NewRouter()
	r.AddPage("/docs/hello world", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	if _, ok := r.Match("GET", "/docs/hello%20world"); !ok {
		t.Fatal("expected match for /docs/hello%20world")
	}
}

func TestRouterRejectsInvalidPercentEscape(t *testing.T) {
	r := NewRouter()
	r.AddPage("/docs/:slug", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	if _, ok := r.Match("GET", "/docs/%ZZ"); ok {
		t.Fatal("expected no match for invalid percent escape")
	}
}

func TestRouterTypedParamRejectsDecodedValue(t *testing.T) {
	r := NewRouter()
	r.AddPage("/users/:id:int", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	if _, ok := r.Match("GET", "/users/12%203"); ok {
		t.Fatal("expected no match for non-integer decoded param")
	}
}
