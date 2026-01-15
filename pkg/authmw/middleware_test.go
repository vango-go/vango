package authmw_test

import (
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/authmw"
	"github.com/vango-go/vango/pkg/server"
)

type TestUser struct {
	ID string
}

func TestRequireAuthMiddleware(t *testing.T) {
	t.Run("unauthenticated returns ErrUnauthorized and does not call next", func(t *testing.T) {
		ctx := server.NewTestContext(server.NewMockSession())
		err := authmw.RequireAuth.Handle(ctx, func() error {
			t.Fatal("next should not be called")
			return nil
		})
		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("authenticated calls next", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, &TestUser{ID: "1"})
		ctx := server.NewTestContext(session)
		called := false
		err := authmw.RequireAuth.Handle(ctx, func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !called {
			t.Fatal("expected next to be called")
		}
	})
}

func TestRequireRole(t *testing.T) {
	type roleUser struct {
		Role string
	}

	isAdmin := func(u *roleUser) bool {
		return u.Role == "admin"
	}

	tests := []struct {
		name    string
		user    *roleUser
		wantErr error
	}{
		{name: "unauthenticated returns ErrUnauthorized", user: nil, wantErr: auth.ErrUnauthorized},
		{name: "authenticated but not authorized returns ErrForbidden", user: &roleUser{Role: "user"}, wantErr: auth.ErrForbidden},
		{name: "authorized calls next", user: &roleUser{Role: "admin"}, wantErr: nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			session := server.NewMockSession()
			if tt.user != nil {
				auth.Set(session, tt.user)
			}
			ctx := server.NewTestContext(session)
			called := false
			err := authmw.RequireRole[*roleUser](isAdmin).Handle(ctx, func() error {
				called = true
				return nil
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if tt.wantErr == nil && !called {
				t.Fatal("expected next to be called")
			}
		})
	}
}

func TestRequirePermission(t *testing.T) {
	type permUser struct {
		Allowed bool
	}

	can := func(u *permUser) bool { return u.Allowed }

	session := server.NewMockSession()
	auth.Set(session, &permUser{Allowed: true})
	ctx := server.NewTestContext(session)

	err := authmw.RequirePermission[*permUser](can).Handle(ctx, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRequireAny(t *testing.T) {
	type flagsUser struct {
		IsAdmin bool
		Active  bool
	}

	isAdmin := func(u *flagsUser) bool { return u.IsAdmin }
	isActive := func(u *flagsUser) bool { return u.Active }

	t.Run("unauthenticated returns ErrUnauthorized", func(t *testing.T) {
		ctx := server.NewTestContext(server.NewMockSession())
		err := authmw.RequireAny[*flagsUser](isAdmin).Handle(ctx, func() error {
			return nil
		})
		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("no checks pass returns ErrForbidden", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, &flagsUser{IsAdmin: false, Active: false})
		ctx := server.NewTestContext(session)
		err := authmw.RequireAny[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			return nil
		})
		if !errors.Is(err, auth.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("any check passes calls next", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, &flagsUser{IsAdmin: true, Active: false})
		ctx := server.NewTestContext(session)
		called := false
		err := authmw.RequireAny[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !called {
			t.Fatal("expected next to be called")
		}
	})
}

func TestRequireAll(t *testing.T) {
	type flagsUser struct {
		IsAdmin bool
		Active  bool
	}

	isAdmin := func(u *flagsUser) bool { return u.IsAdmin }
	isActive := func(u *flagsUser) bool { return u.Active }

	t.Run("unauthenticated returns ErrUnauthorized", func(t *testing.T) {
		ctx := server.NewTestContext(server.NewMockSession())
		err := authmw.RequireAll[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			return nil
		})
		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("any check fails returns ErrForbidden", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, &flagsUser{IsAdmin: true, Active: false})
		ctx := server.NewTestContext(session)
		err := authmw.RequireAll[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			return nil
		})
		if !errors.Is(err, auth.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("all checks pass calls next", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, &flagsUser{IsAdmin: true, Active: true})
		ctx := server.NewTestContext(session)
		called := false
		err := authmw.RequireAll[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if !called {
			t.Fatal("expected next to be called")
		}
	})
}
