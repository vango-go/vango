package server

import (
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/routepath"
)

// TestServerSetRouter verifies that SetRouter stores the router in the server.
func TestServerSetRouter(t *testing.T) {
	// Create a mock router
	mockRouter := &mockRouter{
		notFound: func(ctx Ctx, params any) Component { return nil },
	}

	// Create a server with default config
	config := DefaultServerConfig()
	server := New(config)

	// Initially, router should be nil
	if server.Router() != nil {
		t.Error("expected Router() to return nil initially")
	}

	// Set the router
	server.SetRouter(mockRouter)

	// Now router should be set
	if server.Router() != mockRouter {
		t.Error("expected Router() to return the set router")
	}
}

// TestSessionSetRouter verifies that SetRouter creates a navigator.
func TestSessionSetRouter(t *testing.T) {
	// Create a mock session
	session := NewMockSession()

	// Initially, navigator should be nil
	if session.Navigator() != nil {
		t.Error("expected Navigator() to return nil initially")
	}

	// Create a mock router
	mockRouter := &mockRouter{}

	// Set the router
	session.SetRouter(mockRouter)

	// Now navigator should be created
	if session.Navigator() == nil {
		t.Error("expected Navigator() to return non-nil after SetRouter")
	}

	// Navigator should have the router
	if session.Navigator().router != mockRouter {
		t.Error("expected Navigator().router to be the set router")
	}
}

// TestNavigateWithoutRouter verifies HandleNavigate works without a router.
func TestNavigateWithoutRouter(t *testing.T) {
	// Create a mock session without a router
	session := NewMockSession()

	// HandleNavigate should not panic
	err := session.HandleNavigate("/test", false)

	// Should succeed (sends NAV patch only, no route matching)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestNavigateWithRouter verifies HandleNavigate uses the router.
func TestNavigateWithRouter(t *testing.T) {
	// Create a mock session
	session := NewMockSession()

	// Track if page handler was called
	handlerCalled := false

	// Create a mock router with a matching route
	mockRouter := &mockRouter{
		matchResult: &mockRouteMatch{
			params: map[string]string{"id": "123"},
			pageHandler: func(ctx Ctx, params any) Component {
				handlerCalled = true
				return nil
			},
		},
	}

	// Set the router
	session.SetRouter(mockRouter)

	// Navigate to a path
	err := session.HandleNavigate("/projects/123", false)

	// Should succeed
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Page handler should have been called
	if !handlerCalled {
		t.Error("expected page handler to be called during navigation")
	}
}

func TestNavigateInvalidPathRejected(t *testing.T) {
	session := NewMockSession()
	session.SetRouter(&mockRouter{})

	err := session.HandleNavigate("about", false)
	if err == nil {
		t.Fatal("expected error for invalid navigation path")
	}
	if !errors.Is(err, routepath.ErrInvalidPath) {
		t.Fatalf("error=%v, want %v", err, routepath.ErrInvalidPath)
	}
}

// TestNavigateWithNotFound verifies HandleNavigate uses NotFound handler.
func TestNavigateWithNotFound(t *testing.T) {
	// Create a mock session
	session := NewMockSession()

	// Track if not found handler was called
	notFoundCalled := false

	// Create a mock router with no matching route but a NotFound handler
	mockRouter := &mockRouter{
		matchResult: nil, // No match
		notFound: func(ctx Ctx, params any) Component {
			notFoundCalled = true
			return nil
		},
	}

	// Set the router
	session.SetRouter(mockRouter)

	// Navigate to a non-existent path
	err := session.HandleNavigate("/nonexistent", false)

	// Should succeed (NotFound handler is used)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Not found handler should have been called
	if !notFoundCalled {
		t.Error("expected NotFound handler to be called for unmatched route")
	}
}

// TestNavigateNoRouteNoNotFound verifies error when no route and no NotFound.
func TestNavigateNoRouteNoNotFound(t *testing.T) {
	// Create a mock session
	session := NewMockSession()

	// Create a mock router with no matching route and no NotFound handler
	mockRouter := &mockRouter{
		matchResult: nil,
		notFound:    nil,
	}

	// Set the router
	session.SetRouter(mockRouter)

	// Navigate to a non-existent path
	err := session.HandleNavigate("/nonexistent", false)

	// Should return error
	if err == nil {
		t.Error("expected error when no route matches and no NotFound handler")
	}
}

// TestNavigationPreservesReplace verifies replace flag is honored.
func TestNavigationPreservesReplace(t *testing.T) {
	session := NewMockSession()

	mockRouter := &mockRouter{
		matchResult: &mockRouteMatch{
			params:      map[string]string{},
			pageHandler: func(ctx Ctx, params any) Component { return nil },
		},
	}

	session.SetRouter(mockRouter)

	// Test with replace=false
	err := session.HandleNavigate("/test", false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test with replace=true
	err = session.HandleNavigate("/test", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// mockRouter implements the Router interface for testing.
type mockRouter struct {
	matchResult *mockRouteMatch
	notFound    PageHandler
}

func (m *mockRouter) Match(method, path string) (RouteMatch, bool) {
	if m.matchResult != nil {
		return m.matchResult, true
	}
	return nil, false
}

func (m *mockRouter) NotFound() PageHandler {
	return m.notFound
}

// mockRouteMatch implements the RouteMatch interface for testing.
type mockRouteMatch struct {
	params         map[string]string
	pageHandler    PageHandler
	layoutHandlers []LayoutHandler
	middleware     []RouteMiddleware
}

func (m *mockRouteMatch) GetParams() map[string]string {
	return m.params
}

func (m *mockRouteMatch) GetPageHandler() PageHandler {
	return m.pageHandler
}

func (m *mockRouteMatch) GetLayoutHandlers() []LayoutHandler {
	return m.layoutHandlers
}

func (m *mockRouteMatch) GetMiddleware() []RouteMiddleware {
	return m.middleware
}
