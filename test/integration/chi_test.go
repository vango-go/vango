package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/vango-go/vango/pkg/server"
)

// TestUser represents a user for testing.
type TestUser struct {
	ID    string
	Email string
	Role  string
}

// userContextKey is the key for storing user in context.
type userContextKey struct{}

// mockAuthMiddleware simulates authentication middleware.
func mockAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for "Authorization" header to simulate authenticated request
		if r.Header.Get("Authorization") == "Bearer valid-token" {
			user := &TestUser{
				ID:    "user-123",
				Email: "test@example.com",
				Role:  "admin",
			}
			ctx := context.WithValue(r.Context(), userContextKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		// Continue without auth (anonymous)
		next.ServeHTTP(w, r)
	})
}

// TestChiRouterIntegration tests that Vango integrates with Chi router.
func TestChiRouterIntegration(t *testing.T) {
	// Create Vango server with Context Bridge
	app := server.New(&server.ServerConfig{
		Address: ":0", // Random port
		OnSessionStart: func(httpCtx context.Context, session *server.Session) {
			// This is THE CONTEXT BRIDGE
			if val := httpCtx.Value(userContextKey{}); val != nil {
				user := val.(*TestUser)
				session.Set("vango_auth_user", user)
			}
		},
	})

	// Create Chi router with middleware stack
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(mockAuthMiddleware)

	// Traditional API routes
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Mount Vango handler
	r.Handle("/*", app.Handler())

	// Test 1: Health endpoint works
	t.Run("API health endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "OK" {
			t.Errorf("expected OK, got %s", rec.Body.String())
		}
	})

	// Test 2: Chi middleware executes before Vango handler
	t.Run("middleware chain executes", func(t *testing.T) {
		middlewareExecuted := false

		// Create router with tracking middleware
		trackingRouter := chi.NewRouter()
		trackingRouter.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareExecuted = true
				next.ServeHTTP(w, r)
			})
		})
		trackingRouter.Handle("/*", app.Handler())

		req := httptest.NewRequest("GET", "/some-page", nil)
		rec := httptest.NewRecorder()
		trackingRouter.ServeHTTP(rec, req)

		if !middlewareExecuted {
			t.Error("expected middleware to execute before Vango handler")
		}
	})

	// Test 3: Auth middleware sets context, but we can't easily test WebSocket
	// bridge in unit tests. This is more of a smoke test.
	t.Run("auth context available", func(t *testing.T) {
		contextHadUser := false

		trackingRouter := chi.NewRouter()
		trackingRouter.Use(mockAuthMiddleware)
		trackingRouter.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if val := r.Context().Value(userContextKey{}); val != nil {
					contextHadUser = true
				}
				next.ServeHTTP(w, r)
			})
		})
		trackingRouter.Handle("/*", app.Handler())

		req := httptest.NewRequest("GET", "/dashboard", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		trackingRouter.ServeHTTP(rec, req)

		if !contextHadUser {
			t.Error("expected user to be in context from auth middleware")
		}
	})
}

// TestVangoHandlerMethods tests the different handler methods.
func TestVangoHandlerMethods(t *testing.T) {
	app := server.New(nil)

	// Test Handler returns non-nil
	t.Run("Handler returns http.Handler", func(t *testing.T) {
		h := app.Handler()
		if h == nil {
			t.Error("expected non-nil handler")
		}
	})

	// Test PageHandler returns non-nil
	t.Run("PageHandler returns http.Handler", func(t *testing.T) {
		h := app.PageHandler()
		if h == nil {
			t.Error("expected non-nil page handler")
		}
	})

	// Test WebSocketHandler returns non-nil
	t.Run("WebSocketHandler returns http.Handler", func(t *testing.T) {
		h := app.WebSocketHandler()
		if h == nil {
			t.Error("expected non-nil websocket handler")
		}
	})
}

// TestStdlibMuxIntegration tests with stdlib ServeMux.
func TestStdlibMuxIntegration(t *testing.T) {
	app := server.New(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("api"))
	})
	mux.Handle("/", app.Handler())

	t.Run("API route works", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Body.String() != "api" {
			t.Errorf("expected api, got %s", rec.Body.String())
		}
	})

	t.Run("Vango handler mounted", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/page", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		// Vango will return something (not NotFound)
		// The actual response depends on whether a page is registered
	})
}
