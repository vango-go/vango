package vango

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corevango "github.com/vango-go/vango/pkg/vango"
)

func TestAppAPI_DecodesJSONBody_MissingContentTypeAccepted(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPost, "/api/echo", func(ctx Ctx, input Input) (any, error) {
		return input, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"name":"alice"}`))
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got Input
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "alice" {
		t.Fatalf("Name = %q, want %q", got.Name, "alice")
	}
}

func TestAppAPI_DecodesJSONBody_WithParamsAndBody(t *testing.T) {
	type Params struct {
		ID int `param:"id"`
	}
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPut, "/api/users/:id", func(ctx Ctx, p Params, input Input) (any, error) {
		return map[string]any{
			"id":   p.ID,
			"name": input.Name,
		}, nil
	})

	req := httptest.NewRequest(http.MethodPut, "/api/users/123", strings.NewReader(`{"name":"bob"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if gotID, ok := got["id"].(float64); !ok || int(gotID) != 123 {
		t.Fatalf("id = %#v, want %d", got["id"], 123)
	}
	if gotName, ok := got["name"].(string); !ok || gotName != "bob" {
		t.Fatalf("name = %#v, want %q", got["name"], "bob")
	}
}

func TestAppAPI_InvalidJSON_Returns400(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPost, "/api/echo", func(ctx Ctx, input Input) (any, error) {
		return input, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestAppAPI_ExplicitNonJSONContentType_Returns415(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPost, "/api/echo", func(ctx Ctx, input Input) (any, error) {
		return input, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
}

func TestAppAPI_EmptyBodyForNonPointer_Returns400(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPost, "/api/echo", func(ctx Ctx, input Input) (any, error) {
		return input, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/echo", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestAppAPI_EmptyBodyForPointer_AllowsNil(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPost, "/api/echo", func(ctx Ctx, input *Input) (any, error) {
		return map[string]bool{"nil": input == nil}, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/echo", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var got map[string]bool
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !got["nil"] {
		t.Fatalf("nil = %v, want true", got["nil"])
	}
}

func TestAppAPI_MaxBodyBytes_Returns413(t *testing.T) {
	cfg := DefaultConfig()
	cfg.API.MaxBodyBytes = 8

	type Input struct {
		Name string `json:"name"`
	}

	app := New(cfg)
	app.API(http.MethodPost, "/api/echo", func(ctx Ctx, input Input) (any, error) {
		return input, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/echo", strings.NewReader(`{"name":"toolarge"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestAppAPI_ResponseWritePassthrough(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}

	app := New(DefaultConfig())
	app.API(http.MethodPost, "/api/users", func(ctx Ctx, input Input) (any, error) {
		return corevango.Created(map[string]string{"name": input.Name}), nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"name":"ok"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusCreated)
	}

	var got map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := got["data"]; !ok {
		t.Fatalf("expected response to include \"data\" key, got %#v", got)
	}
}
