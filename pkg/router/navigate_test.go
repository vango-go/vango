package router

import (
	"testing"
)

func TestNavigateOptions(t *testing.T) {
	// Test default options
	opts := NavigateOptions{}
	if opts.Replace {
		t.Error("Replace should default to false")
	}
	if opts.Scroll {
		t.Error("Scroll should default to false (we set it to true when Navigate is called)")
	}
}

func TestNavigateOptionFunctions(t *testing.T) {
	opts := NavigateOptions{Scroll: true}

	// WithReplace
	WithReplace()(&opts)
	if !opts.Replace {
		t.Error("WithReplace should set Replace to true")
	}

	// WithParams
	params := map[string]any{"page": 1, "sort": "name"}
	WithParams(params)(&opts)
	if opts.Params["page"] != 1 {
		t.Error("WithParams should set params")
	}

	// WithoutScroll
	WithoutScroll()(&opts)
	if opts.Scroll {
		t.Error("WithoutScroll should set Scroll to false")
	}

	// WithPrefetch
	WithPrefetch()(&opts)
	if !opts.Prefetch {
		t.Error("WithPrefetch should set Prefetch to true")
	}
}

func TestNavigationRequestBuildURL(t *testing.T) {
	tests := []struct {
		request NavigationRequest
		want    string
		wantErr bool
	}{
		{
			request: NavigationRequest{Path: "/users"},
			want:    "/users",
		},
		{
			request: NavigationRequest{
				Path: "/search",
				Options: NavigateOptions{
					Params: map[string]any{"q": "test"},
				},
			},
			want: "/search?q=test",
		},
		{
			request: NavigationRequest{
				Path: "/users",
				Options: NavigateOptions{
					Params: map[string]any{"page": 2, "limit": 10},
				},
			},
			want: "/users?limit=10&page=2", // Sorted alphabetically
		},
	}

	for _, tt := range tests {
		got, err := tt.request.BuildURL()
		if (err != nil) != tt.wantErr {
			t.Errorf("BuildURL() error = %v, wantErr %v", err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("BuildURL() = %q, want %q", got, tt.want)
		}
	}
}

func TestNavigator(t *testing.T) {
	nav := NewNavigator(nil)

	// Test Navigate
	nav.Navigate("/users", WithReplace(), WithoutScroll())

	// Cast to access internal state
	n := nav.(*navigator)
	if n.pending == nil {
		t.Fatal("expected pending navigation")
	}
	if n.pending.Path != "/users" {
		t.Errorf("Path = %q, want %q", n.pending.Path, "/users")
	}
	if !n.pending.Options.Replace {
		t.Error("expected Replace to be true")
	}
	if n.pending.Options.Scroll {
		t.Error("expected Scroll to be false")
	}
}

func TestNavigatorBack(t *testing.T) {
	nav := NewNavigator(nil)
	nav.Back()

	n := nav.(*navigator)
	if n.pending == nil || n.pending.Path != "__back__" {
		t.Error("Back() should set path to __back__")
	}
}

func TestNavigatorForward(t *testing.T) {
	nav := NewNavigator(nil)
	nav.Forward()

	n := nav.(*navigator)
	if n.pending == nil || n.pending.Path != "__forward__" {
		t.Error("Forward() should set path to __forward__")
	}
}
