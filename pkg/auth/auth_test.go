package auth_test

import (
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/server"
)

// TestUser is a mock user type for testing.
type TestUser struct {
	ID    string
	Email string
	Role  string
}

func TestGet_Authenticated(t *testing.T) {
	session := server.NewMockSession()
	auth.Set(session, &TestUser{ID: "123", Email: "test@example.com", Role: "admin"})

	ctx := server.NewTestContext(session)

	user, ok := auth.Get[*TestUser](ctx)
	if !ok {
		t.Fatal("expected authenticated user")
	}
	if user.ID != "123" {
		t.Errorf("expected ID 123, got %s", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
}

func TestGet_Unauthenticated(t *testing.T) {
	session := server.NewMockSession()
	ctx := server.NewTestContext(session)

	user, ok := auth.Get[*TestUser](ctx)
	if ok {
		t.Fatal("expected unauthenticated")
	}
	if user != nil {
		t.Error("expected nil user")
	}
}

func TestGet_WrongType(t *testing.T) {
	session := server.NewMockSession()
	// Store value type
	session.Set(auth.SessionKey, TestUser{ID: "123"})

	ctx := server.NewTestContext(session)

	// Request pointer type
	user, ok := auth.Get[*TestUser](ctx)
	if ok {
		t.Fatal("expected type mismatch to return false")
	}
	if user != nil {
		t.Error("expected nil user on type mismatch")
	}
}

func TestGet_DebugMode_TypeMismatch_Interface(t *testing.T) {
	orig := auth.DebugMode
	auth.DebugMode = true
	t.Cleanup(func() { auth.DebugMode = orig })

	session := server.NewMockSession()
	session.Set(auth.SessionKey, 123) // does not implement io.Reader
	ctx := server.NewTestContext(session)

	_, ok := auth.Get[interface{ Read([]byte) (int, error) }](ctx)
	if ok {
		t.Fatal("expected type mismatch to return false")
	}
}

func TestGet_NilSession(t *testing.T) {
	ctx := server.NewTestContext(nil)

	_, ok := auth.Get[*TestUser](ctx)
	if ok {
		t.Fatal("expected unauthenticated with nil session")
	}
}

func TestRequire_Authenticated(t *testing.T) {
	session := server.NewMockSession()
	auth.Set(session, &TestUser{ID: "456"})

	ctx := server.NewTestContext(session)

	user, err := auth.Require[*TestUser](ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "456" {
		t.Errorf("expected ID 456, got %s", user.ID)
	}
}

func TestRequire_Unauthenticated(t *testing.T) {
	session := server.NewMockSession()
	ctx := server.NewTestContext(session)

	_, err := auth.Require[*TestUser](ctx)
	if err != auth.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestIsAuthenticated(t *testing.T) {
	session := server.NewMockSession()
	ctx := server.NewTestContext(session)

	if auth.IsAuthenticated(ctx) {
		t.Error("expected unauthenticated")
	}

	auth.Set(session, &TestUser{ID: "789"})

	if !auth.IsAuthenticated(ctx) {
		t.Error("expected authenticated")
	}
}

func TestClear(t *testing.T) {
	session := server.NewMockSession()
	auth.Set(session, &TestUser{ID: "abc"})

	ctx := server.NewTestContext(session)
	if !auth.IsAuthenticated(ctx) {
		t.Fatal("expected authenticated before clear")
	}

	auth.Clear(session)

	if auth.IsAuthenticated(ctx) {
		t.Error("expected unauthenticated after clear")
	}
}

func TestClear_NilSession(t *testing.T) {
	// Should not panic when called with nil session
	auth.Clear(nil)
}

func TestMustGet_Panics(t *testing.T) {
	session := server.NewMockSession()
	ctx := server.NewTestContext(session)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustGet")
		}
	}()

	_ = auth.MustGet[*TestUser](ctx)
}

func TestMustGet_Authenticated(t *testing.T) {
	session := server.NewMockSession()
	auth.Set(session, &TestUser{ID: "must"})

	ctx := server.NewTestContext(session)

	user := auth.MustGet[*TestUser](ctx)
	if user.ID != "must" {
		t.Errorf("expected ID must, got %s", user.ID)
	}
}

func TestValueTypeVsPointerType(t *testing.T) {
	session := server.NewMockSession()

	// Test storing value type and retrieving value type
	session.Set(auth.SessionKey, TestUser{ID: "value"})
	ctx := server.NewTestContext(session)

	user, ok := auth.Get[TestUser](ctx)
	if !ok {
		t.Error("expected value type to work with value retrieval")
	}
	if user.ID != "value" {
		t.Errorf("expected ID value, got %s", user.ID)
	}
}

// =============================================================================
// SSR Auth Tests - auth.Get should use ctx.User() which works for SSR
// =============================================================================

func TestGet_SSR_ViaCtxUser(t *testing.T) {
	// SSR has no session, but user is set via ctx.SetUser()
	ctx := server.NewTestContext(nil)
	testUser := &TestUser{ID: "ssr-user", Email: "ssr@example.com"}

	// Set user on context (simulates SSR middleware setting user)
	ctx.SetUser(testUser)

	// auth.Get should find it via ctx.User()
	user, ok := auth.Get[*TestUser](ctx)
	if !ok {
		t.Fatal("expected auth.Get to find user set via ctx.SetUser()")
	}
	if user.ID != "ssr-user" {
		t.Errorf("expected ID ssr-user, got %s", user.ID)
	}
}

func TestIsAuthenticated_SSR_ViaCtxUser(t *testing.T) {
	ctx := server.NewTestContext(nil)

	// Not authenticated initially
	if auth.IsAuthenticated(ctx) {
		t.Error("expected unauthenticated with no user set")
	}

	// Set user
	ctx.SetUser(&TestUser{ID: "ssr-auth"})

	// Now authenticated
	if !auth.IsAuthenticated(ctx) {
		t.Error("expected authenticated after SetUser")
	}
}

// =============================================================================
// Login/Logout Tests
// =============================================================================

func TestLogin_WithSession(t *testing.T) {
	session := server.NewMockSession()
	ctx := server.NewTestContext(session)

	testUser := &TestUser{ID: "login-test", Email: "login@example.com"}
	auth.Login(ctx, testUser)

	// Should be available via auth.Get
	user, ok := auth.Get[*TestUser](ctx)
	if !ok {
		t.Fatal("expected user after Login")
	}
	if user.ID != "login-test" {
		t.Errorf("expected ID login-test, got %s", user.ID)
	}

	// Should also set presence flag
	if !auth.WasAuthenticated(session) {
		t.Error("expected WasAuthenticated to return true after Login")
	}
}

func TestLogin_WithoutSession(t *testing.T) {
	// SSR mode - no session
	ctx := server.NewTestContext(nil)

	testUser := &TestUser{ID: "ssr-login", Email: "ssr-login@example.com"}
	auth.Login(ctx, testUser)

	// Should still be available via ctx.User()
	user, ok := auth.Get[*TestUser](ctx)
	if !ok {
		t.Fatal("expected user after Login in SSR mode")
	}
	if user.ID != "ssr-login" {
		t.Errorf("expected ID ssr-login, got %s", user.ID)
	}
}

func TestLogout(t *testing.T) {
	session := server.NewMockSession()
	ctx := server.NewTestContext(session)

	// Login first
	auth.Login(ctx, &TestUser{ID: "logout-test"})
	if !auth.IsAuthenticated(ctx) {
		t.Fatal("expected authenticated after Login")
	}

	// Logout
	auth.Logout(ctx)

	if auth.IsAuthenticated(ctx) {
		t.Error("expected unauthenticated after Logout")
	}

	if auth.WasAuthenticated(session) {
		t.Error("expected WasAuthenticated to return false after Logout")
	}
}

// =============================================================================
// Presence Flag Tests
// =============================================================================

func TestWasAuthenticated_AfterSet(t *testing.T) {
	session := server.NewMockSession()

	// Not authenticated initially
	if auth.WasAuthenticated(session) {
		t.Error("expected WasAuthenticated false initially")
	}

	// Set user
	auth.Set(session, &TestUser{ID: "presence"})

	// Now has presence flag
	if !auth.WasAuthenticated(session) {
		t.Error("expected WasAuthenticated true after Set")
	}
}

func TestWasAuthenticated_AfterClear(t *testing.T) {
	session := server.NewMockSession()
	auth.Set(session, &TestUser{ID: "clear-test"})

	if !auth.WasAuthenticated(session) {
		t.Fatal("expected WasAuthenticated true after Set")
	}

	auth.Clear(session)

	if auth.WasAuthenticated(session) {
		t.Error("expected WasAuthenticated false after Clear")
	}
}

func TestWasAuthenticated_NilSession(t *testing.T) {
	if auth.WasAuthenticated(nil) {
		t.Error("expected WasAuthenticated false for nil session")
	}
}

func TestSetPrincipal(t *testing.T) {
	session := server.NewMockSession()
	principal := auth.Principal{
		ID:              "user-1",
		Email:           "user@example.com",
		ExpiresAtUnixMs: 1234567890,
	}

	auth.SetPrincipal(session, principal)

	gotPrincipal, ok := session.Get(auth.SessionKeyPrincipal).(auth.Principal)
	if !ok {
		t.Fatal("expected principal to be stored")
	}
	if gotPrincipal.ID != principal.ID || gotPrincipal.Email != principal.Email {
		t.Errorf("unexpected principal: got %+v want %+v", gotPrincipal, principal)
	}

	expiry, ok := session.Get(auth.SessionKeyExpiryUnixMs).(int64)
	if !ok || expiry != principal.ExpiresAtUnixMs {
		t.Errorf("expected expiry %d, got %v", principal.ExpiresAtUnixMs, session.Get(auth.SessionKeyExpiryUnixMs))
	}

	if session.Get(auth.SessionKeyHadAuth) != true {
		t.Error("expected SessionKeyHadAuth to be true")
	}

	if session.Get(auth.SessionPresenceKey()) != true {
		t.Error("expected presence flag to be true")
	}
}

// =============================================================================
// SessionPresenceKey Tests
// =============================================================================

func TestSessionPresenceKey(t *testing.T) {
	// SessionPresenceKey should return the internal key used for presence tracking
	key := auth.SessionPresenceKey()
	if key == "" {
		t.Error("expected non-empty presence key")
	}
	// Key should be based on SessionKey
	if key != auth.SessionKey+":present" {
		t.Errorf("expected presence key to be %q, got %q", auth.SessionKey+":present", key)
	}
}

// =============================================================================
// StatusCode Tests
// =============================================================================

func TestStatusCode_Unauthorized(t *testing.T) {
	code, ok := auth.StatusCode(auth.ErrUnauthorized)
	if !ok {
		t.Fatal("expected StatusCode to recognize ErrUnauthorized")
	}
	if code != 401 {
		t.Errorf("expected 401, got %d", code)
	}
}

func TestStatusCode_Forbidden(t *testing.T) {
	code, ok := auth.StatusCode(auth.ErrForbidden)
	if !ok {
		t.Fatal("expected StatusCode to recognize ErrForbidden")
	}
	if code != 403 {
		t.Errorf("expected 403, got %d", code)
	}
}

func TestStatusCode_OtherError(t *testing.T) {
	_, ok := auth.StatusCode(errors.New("some other error"))
	if ok {
		t.Error("expected StatusCode to not recognize other errors")
	}
}

func TestStatusCode_Nil(t *testing.T) {
	_, ok := auth.StatusCode(nil)
	if ok {
		t.Error("expected StatusCode to return false for nil")
	}
}

// =============================================================================
// IsAuthError Tests
// =============================================================================

func TestIsAuthError_Unauthorized(t *testing.T) {
	if !auth.IsAuthError(auth.ErrUnauthorized) {
		t.Error("expected ErrUnauthorized to be an auth error")
	}
}

func TestIsAuthError_Forbidden(t *testing.T) {
	if !auth.IsAuthError(auth.ErrForbidden) {
		t.Error("expected ErrForbidden to be an auth error")
	}
}

func TestIsAuthError_Other(t *testing.T) {
	if auth.IsAuthError(errors.New("other")) {
		t.Error("expected other error to not be an auth error")
	}
}

func TestIsAuthError_Nil(t *testing.T) {
	if auth.IsAuthError(nil) {
		t.Error("expected nil to not be an auth error")
	}
}
