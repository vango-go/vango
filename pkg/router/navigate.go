package router

import (
	"fmt"
	"net/url"

	"github.com/vango-dev/vango/v2/pkg/server"
)

// NavigateOptions configures navigation behavior.
type NavigateOptions struct {
	// Replace replaces the current history entry instead of pushing.
	Replace bool

	// Params are query parameters to add to the URL.
	Params map[string]any

	// Scroll controls whether to scroll to top after navigation.
	// Defaults to true.
	Scroll bool

	// Prefetch indicates this path should be prefetched.
	Prefetch bool
}

// NavigateOption is a functional option for Navigate.
type NavigateOption func(*NavigateOptions)

// WithReplace replaces the current history entry instead of pushing.
func WithReplace() NavigateOption {
	return func(o *NavigateOptions) {
		o.Replace = true
	}
}

// WithParams adds query parameters to the navigation URL.
func WithParams(params map[string]any) NavigateOption {
	return func(o *NavigateOptions) {
		o.Params = params
	}
}

// WithoutScroll disables scrolling to top after navigation.
func WithoutScroll() NavigateOption {
	return func(o *NavigateOptions) {
		o.Scroll = false
	}
}

// WithPrefetch enables prefetching for this navigation.
func WithPrefetch() NavigateOption {
	return func(o *NavigateOptions) {
		o.Prefetch = true
	}
}

// NavigationRequest represents a pending navigation.
type NavigationRequest struct {
	Path    string
	Options NavigateOptions
}

// Navigator handles client-side navigation.
// It is typically obtained from the request context.
type Navigator interface {
	// Navigate performs a client-side navigation to the given path.
	Navigate(path string, opts ...NavigateOption)

	// Back navigates back in browser history.
	Back()

	// Forward navigates forward in browser history.
	Forward()
}

// navigator is the internal implementation of Navigator.
type navigator struct {
	ctx     server.Ctx
	pending *NavigationRequest
}

// NewNavigator creates a new navigator for the given context.
func NewNavigator(ctx server.Ctx) Navigator {
	return &navigator{ctx: ctx}
}

// Navigate queues a navigation to the given path.
func (n *navigator) Navigate(path string, opts ...NavigateOption) {
	options := NavigateOptions{
		Scroll: true, // Default to scrolling
	}
	for _, opt := range opts {
		opt(&options)
	}

	n.pending = &NavigationRequest{
		Path:    path,
		Options: options,
	}
}

// Back navigates back in browser history.
func (n *navigator) Back() {
	n.pending = &NavigationRequest{
		Path: "__back__",
	}
}

// Forward navigates forward in browser history.
func (n *navigator) Forward() {
	n.pending = &NavigationRequest{
		Path: "__forward__",
	}
}

// Pending returns the pending navigation request, if any.
func (n *navigator) Pending() *NavigationRequest {
	return n.pending
}

// BuildURL constructs the full URL for a navigation request.
func (nr *NavigationRequest) BuildURL() (string, error) {
	u, err := url.Parse(nr.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", nr.Path)
	}

	// Add query parameters
	if nr.Options.Params != nil {
		q := u.Query()
		for k, v := range nr.Options.Params {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

// Redirect sends an HTTP redirect response.
// This should only be used for initial page loads, not WebSocket navigations.
func Redirect(ctx server.Ctx, path string, code int) {
	ctx.Redirect(path, code)
}
