package auth_test

import (
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/server"
)

func TestRequireAuthMiddleware(t *testing.T) {
	t.Run("unauthenticated returns ErrUnauthorized and does not call next", func(t *testing.T) {
		ctx := server.NewTestContext(server.NewMockSession())

		nextCalled := false
		err := auth.RequireAuth.Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if nextCalled {
			t.Fatal("expected next not to be called")
		}
	})

	t.Run("authenticated calls next", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, &TestUser{ID: "1"})
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAuth.Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nextCalled {
			t.Fatal("expected next to be called")
		}
	})
}

func TestRequireRoleAndPermissionMiddleware(t *testing.T) {
	type roleUser struct{ Role string }

	admin := &roleUser{Role: "admin"}
	user := &roleUser{Role: "user"}

	isAdmin := func(u *roleUser) bool { return u.Role == "admin" }

	tests := []struct {
		name     string
		mw       func() any
		user     *roleUser
		wantErr  error
		wantNext bool
	}{
		{
			name: "require role unauthorized when missing session value",
			mw: func() any {
				return auth.RequireRole[*roleUser](isAdmin)
			},
			user:     nil,
			wantErr:  auth.ErrUnauthorized,
			wantNext: false,
		},
		{
			name: "require role forbidden when check fails",
			mw: func() any {
				return auth.RequireRole[*roleUser](isAdmin)
			},
			user:     user,
			wantErr:  auth.ErrForbidden,
			wantNext: false,
		},
		{
			name: "require role passes when check passes",
			mw: func() any {
				return auth.RequireRole[*roleUser](isAdmin)
			},
			user:     admin,
			wantErr:  nil,
			wantNext: true,
		},
		{
			name: "require permission matches require role semantics",
			mw: func() any {
				return auth.RequirePermission[*roleUser](isAdmin)
			},
			user:     admin,
			wantErr:  nil,
			wantNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := server.NewMockSession()
			if tt.user != nil {
				auth.Set(session, tt.user)
			}
			ctx := server.NewTestContext(session)

			nextCalled := false
			mw := tt.mw().(interface {
				Handle(ctx server.Ctx, next func() error) error
			})
			err := mw.Handle(ctx, func() error {
				nextCalled = true
				return nil
			})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if nextCalled != tt.wantNext {
				t.Fatalf("nextCalled=%v, want %v", nextCalled, tt.wantNext)
			}
		})
	}
}

func TestRequireAnyAndAllMiddleware(t *testing.T) {
	type flagsUser struct {
		IsAdmin  bool
		IsActive bool
	}

	adminActive := &flagsUser{IsAdmin: true, IsActive: true}
	adminInactive := &flagsUser{IsAdmin: true, IsActive: false}

	isAdmin := func(u *flagsUser) bool { return u.IsAdmin }
	isActive := func(u *flagsUser) bool { return u.IsActive }

	t.Run("RequireAny returns ErrForbidden when no checks pass", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, adminInactive)
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAny[*flagsUser](isActive).Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if !errors.Is(err, auth.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
		if nextCalled {
			t.Fatal("expected next not to be called")
		}
	})

	t.Run("RequireAny calls next when at least one check passes", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, adminActive)
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAny[*flagsUser](isActive, isAdmin).Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nextCalled {
			t.Fatal("expected next to be called")
		}
	})

	t.Run("RequireAny with no checks forbids (edge case)", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, adminActive)
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAny[*flagsUser]().Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if !errors.Is(err, auth.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
		if nextCalled {
			t.Fatal("expected next not to be called")
		}
	})

	t.Run("RequireAny returns ErrUnauthorized when user not found", func(t *testing.T) {
		// No user set in session
		session := server.NewMockSession()
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAny[*flagsUser](isAdmin).Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if nextCalled {
			t.Fatal("expected next not to be called")
		}
	})

	t.Run("RequireAll returns ErrForbidden when any check fails", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, adminInactive)
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAll[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if !errors.Is(err, auth.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
		if nextCalled {
			t.Fatal("expected next not to be called")
		}
	})

	t.Run("RequireAll calls next when all checks pass", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, adminActive)
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAll[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nextCalled {
			t.Fatal("expected next to be called")
		}
	})

	t.Run("RequireAll with no checks passes (vacuous truth edge case)", func(t *testing.T) {
		session := server.NewMockSession()
		auth.Set(session, adminActive)
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAll[*flagsUser]().Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !nextCalled {
			t.Fatal("expected next to be called")
		}
	})

	t.Run("RequireAll returns ErrUnauthorized when user not found", func(t *testing.T) {
		// No user set in session
		session := server.NewMockSession()
		ctx := server.NewTestContext(session)

		nextCalled := false
		err := auth.RequireAll[*flagsUser](isAdmin, isActive).Handle(ctx, func() error {
			nextCalled = true
			return nil
		})

		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
		if nextCalled {
			t.Fatal("expected next not to be called")
		}
	})
}

