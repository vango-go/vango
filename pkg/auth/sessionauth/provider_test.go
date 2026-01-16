package sessionauth

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/auth"
)

type testStore struct {
	getCalls      int
	validateCalls int
	lastID        string
	lastSession   *StoredSession
	getFunc       func(ctx context.Context, sessionID string) (*StoredSession, error)
	validateFunc  func(ctx context.Context, session *StoredSession) error
}

func (s *testStore) Get(ctx context.Context, sessionID string) (*StoredSession, error) {
	s.getCalls++
	s.lastID = sessionID
	if s.getFunc != nil {
		return s.getFunc(ctx, sessionID)
	}
	return nil, nil
}

func (s *testStore) Validate(ctx context.Context, session *StoredSession) error {
	s.validateCalls++
	s.lastSession = session
	if s.validateFunc != nil {
		return s.validateFunc(ctx, session)
	}
	return nil
}

type testCookiePolicy struct{}

func (p testCookiePolicy) ApplyCookiePolicy(r *http.Request, cookie *http.Cookie) (*http.Cookie, error) {
	cookie.Domain = "example.com"
	cookie.SameSite = http.SameSiteStrictMode
	return cookie, nil
}

func TestProviderClearCookieAppliesPolicy(t *testing.T) {
	provider := New(nil, WithCookiePolicy(testCookiePolicy{}))
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	provider.clearCookie(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Domain != "example.com" {
		t.Fatalf("cookie Domain = %q, want %q", cookie.Domain, "example.com")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie SameSite = %v, want %v", cookie.SameSite, http.SameSiteStrictMode)
	}
	if !cookie.Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
}

type errorCookiePolicy struct{}

func (p errorCookiePolicy) ApplyCookiePolicy(r *http.Request, cookie *http.Cookie) (*http.Cookie, error) {
	return nil, errors.New("policy error")
}

func TestProviderClearCookieDefaults(t *testing.T) {
	provider := New(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()

	provider.clearCookie(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "session" {
		t.Fatalf("cookie Name = %q, want %q", cookie.Name, "session")
	}
	if cookie.Value != "" {
		t.Fatalf("cookie Value = %q, want empty", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie Path = %q, want %q", cookie.Path, "/")
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("cookie MaxAge = %d, want -1", cookie.MaxAge)
	}
	if !cookie.HttpOnly {
		t.Fatalf("cookie HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
	if cookie.Secure {
		t.Fatalf("cookie Secure = true, want false")
	}
}

func TestProviderClearCookiePolicyErrorSkipsCookie(t *testing.T) {
	provider := New(nil, WithCookiePolicy(errorCookiePolicy{}))
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	provider.clearCookie(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 0 {
		t.Fatalf("cookies = %d, want 0", len(cookies))
	}
}

func TestProviderMiddlewareNoCookieSkipsStore(t *testing.T) {
	store := &testStore{}
	provider := New(store)
	called := false
	handler := provider.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if store.getCalls != 0 || store.validateCalls != 0 {
		t.Fatalf("store calls = %d/%d, want 0/0", store.getCalls, store.validateCalls)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %d, want 0", len(rec.Result().Cookies()))
	}
}

func TestProviderMiddlewareEmptyCookieSkipsStore(t *testing.T) {
	store := &testStore{}
	provider := New(store)
	called := false
	handler := provider.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: ""})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if store.getCalls != 0 || store.validateCalls != 0 {
		t.Fatalf("store calls = %d/%d, want 0/0", store.getCalls, store.validateCalls)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %d, want 0", len(rec.Result().Cookies()))
	}
}

func TestProviderMiddlewareGetErrorClearsCookie(t *testing.T) {
	sentinel := errors.New("get failed")
	store := &testStore{
		getFunc: func(ctx context.Context, sessionID string) (*StoredSession, error) {
			return nil, sentinel
		},
	}
	provider := New(store, WithCookieName("custom"))
	called := false
	handler := provider.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := SessionFromContext(r.Context()); ok {
			t.Fatal("session should not be in context")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	req.AddCookie(&http.Cookie{Name: "custom", Value: "sess"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if store.getCalls != 1 || store.validateCalls != 0 {
		t.Fatalf("store calls = %d/%d, want 1/0", store.getCalls, store.validateCalls)
	}
	if store.lastID != "sess" {
		t.Fatalf("sessionID = %q, want %q", store.lastID, "sess")
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "custom" {
		t.Fatalf("cookie Name = %q, want %q", cookie.Name, "custom")
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("cookie MaxAge = %d, want -1", cookie.MaxAge)
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie Path = %q, want %q", cookie.Path, "/")
	}
	if !cookie.HttpOnly {
		t.Fatalf("cookie HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
	if !cookie.Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
}

func TestProviderMiddlewareValidateErrorClearsCookie(t *testing.T) {
	sentinel := errors.New("validate failed")
	stored := &StoredSession{ID: "sess"}
	store := &testStore{
		getFunc: func(ctx context.Context, sessionID string) (*StoredSession, error) {
			return stored, nil
		},
		validateFunc: func(ctx context.Context, session *StoredSession) error {
			return sentinel
		},
	}
	provider := New(store)
	called := false
	handler := provider.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := SessionFromContext(r.Context()); ok {
			t.Fatal("session should not be in context")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if store.getCalls != 1 || store.validateCalls != 1 {
		t.Fatalf("store calls = %d/%d, want 1/1", store.getCalls, store.validateCalls)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
}

func TestProviderMiddlewareSuccessInjectsSession(t *testing.T) {
	stored := &StoredSession{ID: "sess"}
	store := &testStore{
		getFunc: func(ctx context.Context, sessionID string) (*StoredSession, error) {
			return stored, nil
		},
	}
	provider := New(store)
	called := false
	handler := provider.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		got, ok := SessionFromContext(r.Context())
		if !ok {
			t.Fatal("expected session in context")
		}
		if got != stored {
			t.Fatalf("stored session = %p, want %p", got, stored)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "sess"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if store.getCalls != 1 || store.validateCalls != 1 {
		t.Fatalf("store calls = %d/%d, want 1/1", store.getCalls, store.validateCalls)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %d, want 0", len(rec.Result().Cookies()))
	}
}

func TestProviderPrincipal(t *testing.T) {
	provider := New(nil)
	stored := &StoredSession{
		ID:          "sess_123",
		UserID:      "user_123",
		Email:       "user@example.com",
		Name:        "Test User",
		Roles:       []string{"admin", "editor"},
		TenantID:    "tenant_1",
		ExpiresAt:   time.Unix(1700000000, 500000000),
		AuthVersion: 7,
	}
	ctx := context.WithValue(context.Background(), sessionContextKey{}, stored)

	principal, ok := provider.Principal(ctx)
	if !ok {
		t.Fatal("expected principal")
	}
	if principal.ID != stored.UserID {
		t.Fatalf("ID = %q, want %q", principal.ID, stored.UserID)
	}
	if principal.Email != stored.Email {
		t.Fatalf("Email = %q, want %q", principal.Email, stored.Email)
	}
	if principal.Name != stored.Name {
		t.Fatalf("Name = %q, want %q", principal.Name, stored.Name)
	}
	if principal.TenantID != stored.TenantID {
		t.Fatalf("TenantID = %q, want %q", principal.TenantID, stored.TenantID)
	}
	if principal.SessionID != stored.ID {
		t.Fatalf("SessionID = %q, want %q", principal.SessionID, stored.ID)
	}
	if principal.ExpiresAtUnixMs != stored.ExpiresAt.UnixMilli() {
		t.Fatalf("ExpiresAtUnixMs = %d, want %d", principal.ExpiresAtUnixMs, stored.ExpiresAt.UnixMilli())
	}
	if principal.AuthVersion != stored.AuthVersion {
		t.Fatalf("AuthVersion = %d, want %d", principal.AuthVersion, stored.AuthVersion)
	}
}

func TestProviderPrincipalMissingSession(t *testing.T) {
	provider := New(nil)
	if _, ok := provider.Principal(context.Background()); ok {
		t.Fatal("expected no principal")
	}
}

func TestProviderVerify(t *testing.T) {
	t.Run("empty session id skips store", func(t *testing.T) {
		store := &testStore{}
		provider := New(store)
		if err := provider.Verify(context.Background(), auth.Principal{}); err != nil {
			t.Fatalf("Verify error: %v", err)
		}
		if store.getCalls != 0 || store.validateCalls != 0 {
			t.Fatalf("store calls = %d/%d, want 0/0", store.getCalls, store.validateCalls)
		}
	})

	t.Run("get error returns error", func(t *testing.T) {
		sentinel := errors.New("get failed")
		store := &testStore{
			getFunc: func(ctx context.Context, sessionID string) (*StoredSession, error) {
				return nil, sentinel
			},
		}
		provider := New(store)
		err := provider.Verify(context.Background(), auth.Principal{SessionID: "sess"})
		if !errors.Is(err, sentinel) {
			t.Fatalf("Verify error = %v, want %v", err, sentinel)
		}
	})

	t.Run("validate error returns error", func(t *testing.T) {
		sentinel := errors.New("validate failed")
		stored := &StoredSession{ID: "sess"}
		store := &testStore{
			getFunc: func(ctx context.Context, sessionID string) (*StoredSession, error) {
				return stored, nil
			},
			validateFunc: func(ctx context.Context, session *StoredSession) error {
				return sentinel
			},
		}
		provider := New(store)
		err := provider.Verify(context.Background(), auth.Principal{SessionID: "sess"})
		if !errors.Is(err, sentinel) {
			t.Fatalf("Verify error = %v, want %v", err, sentinel)
		}
	})

	t.Run("success validates session", func(t *testing.T) {
		stored := &StoredSession{ID: "sess"}
		store := &testStore{
			getFunc: func(ctx context.Context, sessionID string) (*StoredSession, error) {
				return stored, nil
			},
		}
		provider := New(store)
		if err := provider.Verify(context.Background(), auth.Principal{SessionID: "sess"}); err != nil {
			t.Fatalf("Verify error: %v", err)
		}
		if store.getCalls != 1 || store.validateCalls != 1 {
			t.Fatalf("store calls = %d/%d, want 1/1", store.getCalls, store.validateCalls)
		}
	})
}
