package auth_test

import (
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
	orig := server.DebugMode
	server.DebugMode = true
	t.Cleanup(func() { server.DebugMode = orig })

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
