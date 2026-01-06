package server

import (
	"testing"
)

func TestCanonicalizePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPath    string
		wantQuery   string
		wantChanged bool
		wantErr     bool
	}{
		{
			name:        "root path",
			input:       "/",
			wantPath:    "/",
			wantQuery:   "",
			wantChanged: false,
		},
		{
			name:        "simple path",
			input:       "/about",
			wantPath:    "/about",
			wantQuery:   "",
			wantChanged: false,
		},
		{
			name:        "trailing slash removed",
			input:       "/about/",
			wantPath:    "/about",
			wantQuery:   "",
			wantChanged: true,
		},
		{
			name:        "multiple slashes collapsed",
			input:       "/users//123",
			wantPath:    "/users/123",
			wantQuery:   "",
			wantChanged: true,
		},
		{
			name:        "trailing slash and multiple slashes",
			input:       "/users//123/",
			wantPath:    "/users/123",
			wantQuery:   "",
			wantChanged: true,
		},
		{
			name:        "with query string",
			input:       "/users?filter=active",
			wantPath:    "/users",
			wantQuery:   "filter=active",
			wantChanged: false,
		},
		{
			name:        "trailing slash with query",
			input:       "/users/?filter=active",
			wantPath:    "/users",
			wantQuery:   "filter=active",
			wantChanged: true,
		},
		{
			name:        "empty string becomes root",
			input:       "",
			wantPath:    "/",
			wantQuery:   "",
			wantChanged: true,
		},
		{
			name:        "no leading slash",
			input:       "about",
			wantPath:    "/about",
			wantQuery:   "",
			wantChanged: true,
		},
		{
			name:    "backslash rejected",
			input:   "/users\\admin",
			wantErr: true,
		},
		{
			name:    "null byte rejected",
			input:   "/users\x00admin",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotQuery, gotChanged, err := CanonicalizePath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query = %q, want %q", gotQuery, tt.wantQuery)
			}
			if gotChanged != tt.wantChanged {
				t.Errorf("changed = %v, want %v", gotChanged, tt.wantChanged)
			}
		})
	}
}

func TestRouteMatch_Interface(t *testing.T) {
	// Test that simpleRouteMatch implements RouteMatch
	match := &simpleRouteMatch{
		pageHandler: func(ctx Ctx, params any) Component { return nil },
		params:      map[string]string{"id": "123"},
	}

	// Check GetParams
	params := match.GetParams()
	if params["id"] != "123" {
		t.Errorf("GetParams()[\"id\"] = %q, want %q", params["id"], "123")
	}

	// Check GetPageHandler returns non-nil
	handler := match.GetPageHandler()
	if handler == nil {
		t.Error("GetPageHandler() should return non-nil")
	}

	// Check GetLayoutHandlers returns nil
	layouts := match.GetLayoutHandlers()
	if layouts != nil {
		t.Errorf("GetLayoutHandlers() = %v, want nil", layouts)
	}

	// Check GetMiddleware returns nil
	mw := match.GetMiddleware()
	if mw != nil {
		t.Errorf("GetMiddleware() = %v, want nil", mw)
	}
}

func TestPendingNavigation(t *testing.T) {
	// Create a context
	ctx := &ctx{}

	// Initially no pending navigation
	path, replace, has := ctx.PendingNavigation()
	if has {
		t.Error("should not have pending navigation initially")
	}
	if path != "" || replace {
		t.Errorf("unexpected values: path=%q, replace=%v", path, replace)
	}

	// Set pending navigation
	ctx.pendingNavigation = &pendingNav{
		Path:    "/users/123",
		Replace: true,
	}

	// Check pending navigation
	path, replace, has = ctx.PendingNavigation()
	if !has {
		t.Error("should have pending navigation")
	}
	if path != "/users/123" {
		t.Errorf("path = %q, want %q", path, "/users/123")
	}
	if !replace {
		t.Error("replace should be true")
	}

	// Clear pending navigation
	ctx.ClearPendingNavigation()

	// Check cleared
	_, _, has = ctx.PendingNavigation()
	if has {
		t.Error("should not have pending navigation after clear")
	}
}

func TestNavigateResult(t *testing.T) {
	result := &NavigateResult{
		Path:    "/users/123",
		Matched: true,
	}

	if result.Path != "/users/123" {
		t.Errorf("Path = %q, want %q", result.Path, "/users/123")
	}
	if !result.Matched {
		t.Error("Matched should be true")
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

func TestRouteNavigator_CurrentPath(t *testing.T) {
	session := NewMockSession()
	navigator := &RouteNavigator{
		session:     session,
		currentPath: "/projects/456",
		currentParams: map[string]string{
			"id": "456",
		},
	}

	if navigator.CurrentPath() != "/projects/456" {
		t.Errorf("CurrentPath() = %q, want %q", navigator.CurrentPath(), "/projects/456")
	}

	params := navigator.CurrentParams()
	if params["id"] != "456" {
		t.Errorf("CurrentParams()[\"id\"] = %q, want %q", params["id"], "456")
	}
}
