package vango

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestAppPage_NavigateRedirects(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		app := New(DefaultConfig())
		app.Page("/go", func(ctx Ctx) *VNode {
			ctx.Navigate("/dest")
			return vdom.Text("ignored")
		})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/go", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
		}
		if got := rr.Header().Get("Location"); got != "/dest" {
			t.Fatalf("Location = %q, want %q", got, "/dest")
		}
	})

	t.Run("replace", func(t *testing.T) {
		app := New(DefaultConfig())
		app.Page("/go", func(ctx Ctx) *VNode {
			ctx.Navigate("/dest", WithReplace())
			return vdom.Text("ignored")
		})

		req := httptest.NewRequest(http.MethodGet, "http://example.com/go", nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
		}
		if got := rr.Header().Get("Location"); got != "/dest" {
			t.Fatalf("Location = %q, want %q", got, "/dest")
		}
	})
}

func TestAppPage_Redirect(t *testing.T) {
	app := New(DefaultConfig())
	app.Page("/redir", func(ctx Ctx) *VNode {
		ctx.Redirect("/login", http.StatusTemporaryRedirect)
		return vdom.Text("ignored")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/redir", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTemporaryRedirect)
	}
	if got := rr.Header().Get("Location"); got != "/login" {
		t.Fatalf("Location = %q, want %q", got, "/login")
	}
}

func TestAppPage_ErrorPageUsesLayoutsAndStatus(t *testing.T) {
	app := New(DefaultConfig())
	app.Layout("/", func(ctx Ctx, children Slot) *VNode {
		node := &vdom.VNode{
			Kind:  vdom.KindElement,
			Tag:   "div",
			Props: vdom.Props{"data-layout": "root"},
		}
		if children != nil {
			node.Children = []*vdom.VNode{children}
		}
		return node
	})
	app.Layout("/boom", func(ctx Ctx, children Slot) *VNode {
		node := &vdom.VNode{
			Kind:  vdom.KindElement,
			Tag:   "section",
			Props: vdom.Props{"data-layout": "inner"},
		}
		if children != nil {
			node.Children = []*vdom.VNode{children}
		}
		return node
	})
	app.Middleware("/boom", router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		ctx.Status(http.StatusTeapot)
		return errors.New("boom")
	}))
	app.SetErrorPage(func(ctx Ctx, err error) *VNode {
		return &vdom.VNode{
			Kind:  vdom.KindElement,
			Tag:   "main",
			Props: vdom.Props{"data-error": "true"},
			Children: []*vdom.VNode{
				vdom.Text(err.Error()),
			},
		}
	})
	app.Page("/boom", func(ctx Ctx) *VNode {
		return vdom.Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/boom", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "boom") {
		t.Fatalf("expected error text in response, got %q", body)
	}

	rootIdx := strings.Index(body, "data-layout=\"root\"")
	innerIdx := strings.Index(body, "data-layout=\"inner\"")
	errorIdx := strings.Index(body, "data-error=\"true\"")
	if rootIdx == -1 || innerIdx == -1 || errorIdx == -1 {
		t.Fatalf("missing layout markers in response: root=%d inner=%d error=%d", rootIdx, innerIdx, errorIdx)
	}
	if !(rootIdx < innerIdx && innerIdx < errorIdx) {
		t.Fatalf("layout order unexpected: root=%d inner=%d error=%d", rootIdx, innerIdx, errorIdx)
	}
}
