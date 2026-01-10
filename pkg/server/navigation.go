package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// Route Navigation Interfaces
// =============================================================================

// RouteMatch contains the result of matching a path against the router.
// This interface is implemented by router.MatchResult.
type RouteMatch interface {
	// GetParams returns the extracted route parameters.
	GetParams() map[string]string

	// GetPageHandler returns the page handler, if any.
	GetPageHandler() PageHandler

	// GetLayoutHandlers returns the layout handlers in order (root to leaf).
	GetLayoutHandlers() []LayoutHandler

	// GetMiddleware returns the middleware chain.
	GetMiddleware() []RouteMiddleware
}

// PageHandler handles a page request, returning a component to render.
type PageHandler func(ctx Ctx, params any) Component

// LayoutHandler wraps child content in a layout.
type LayoutHandler func(ctx Ctx, children *vdom.VNode) *vdom.VNode

// RouteMiddleware processes requests before they reach the handler.
// This is different from HTTP middleware - it operates on the routing level.
type RouteMiddleware interface {
	Handle(ctx Ctx, next func() error) error
}

// Router defines the interface for route matching.
// This interface is implemented by router.Router.
type Router interface {
	// Match finds the handler for a path.
	// Returns the match result and whether a match was found.
	Match(method, path string) (RouteMatch, bool)

	// NotFound returns the 404 handler, if configured.
	NotFound() PageHandler
}

// =============================================================================
// Path Canonicalization
// =============================================================================

// CanonicalizePath normalizes a URL path for navigation.
// This is a simplified version that handles the most common cases.
// For full canonicalization, use router.CanonicalizePath.
func CanonicalizePath(path string) (canonPath, query string, changed bool, err error) {
	if path == "" {
		return "/", "", true, nil
	}

	// Split path and query
	canonPath, query, _ = strings.Cut(path, "?")

	// SECURITY: Reject backslash and null
	if strings.Contains(canonPath, "\\") {
		return "", "", false, errors.New("path contains backslash")
	}
	if strings.Contains(canonPath, "\x00") {
		return "", "", false, errors.New("path contains null byte")
	}

	original := canonPath

	// Ensure starts with /
	if !strings.HasPrefix(canonPath, "/") {
		canonPath = "/" + canonPath
	}

	// Collapse multiple slashes
	for strings.Contains(canonPath, "//") {
		canonPath = strings.ReplaceAll(canonPath, "//", "/")
	}

	// Remove trailing slash (except root)
	if len(canonPath) > 1 && strings.HasSuffix(canonPath, "/") {
		canonPath = strings.TrimSuffix(canonPath, "/")
	}

	return canonPath, query, canonPath != original, nil
}

// =============================================================================
// Route Navigator
// =============================================================================

// RouteNavigator handles route-based navigation for sessions.
// It is responsible for matching routes, invoking page handlers,
// and managing the page component lifecycle during navigation.
type RouteNavigator struct {
	// router is the application router for matching paths to handlers
	router Router

	// session is the session this navigator belongs to
	session *Session

	// currentPath is the current route path (without query string)
	currentPath string

	// currentQuery is the current route query string (without leading "?")
	currentQuery string

	// currentParams are the current route parameters
	currentParams map[string]string
}

type redirectError struct {
	path    string
	replace bool
}

func (e redirectError) Error() string {
	return fmt.Sprintf("redirect to %q", e.path)
}

// NewRouteNavigator creates a new route navigator for a session.
func NewRouteNavigator(session *Session, r Router) *RouteNavigator {
	return &RouteNavigator{
		router:  r,
		session: session,
	}
}

// NavigateResult contains the result of a navigation operation.
type NavigateResult struct {
	// Path is the canonicalized path that was navigated to
	Path string

	// Matched indicates if a route was matched
	Matched bool

	// Patches contains the DOM patches from the navigation
	Patches []vdom.Patch

	// NavPatch is the NAV_PUSH or NAV_REPLACE patch
	NavPatch protocol.Patch

	// Error contains any error that occurred
	Error error
}

// Navigate handles navigation to a new path.
// This is called when:
//   - ctx.Navigate() is called and pending navigation is processed
//   - EventNavigate is received from the client
//
// The navigation process:
//  1. Canonicalize the path
//  2. Determine if NAV_PUSH or NAV_REPLACE should be used
//  3. Match the route
//  4. Create the new page component
//  5. Mount and render the new page
//  6. Diff against the old tree
//  7. Return patches
//
// Per Section 4.4 (Programmatic Navigation), this is ONE transaction -
// NAV_* patch and DOM patches are returned together.
func (rn *RouteNavigator) Navigate(path string, replace bool) *NavigateResult {
	result := &NavigateResult{}

	const maxRedirects = 10

	for redirects := 0; redirects <= maxRedirects; redirects++ {
		// Canonicalize the path
		canonPath, query, changed, err := CanonicalizePath(path)
		if err != nil {
			result.Error = err
			return result
		}

		// If canonicalization changed the path, force replace to avoid history duplication.
		// Per Section 1.2.4: If canonicalization changed the path, emit NAV_REPLACE.
		localReplace := replace
		if changed && !localReplace {
			localReplace = true
		}
		if redirects > 0 && !localReplace {
			// Redirects should not add a new entry.
			localReplace = true
		}

		// Build full path with query string
		fullPath := canonPath
		if query != "" {
			fullPath = canonPath + "?" + query
		}

		// Match the route
		match, ok := rn.router.Match("GET", canonPath)
		if !ok {
			// No route matched - check for not found handler
			notFoundHandler := rn.router.NotFound()
			if notFoundHandler != nil {
				// Create a minimal match for 404
				match = &simpleRouteMatch{
					pageHandler: notFoundHandler,
					params:      make(map[string]string),
				}
				result.Matched = false
			} else {
				result.Matched = false
				result.Path = fullPath
				if localReplace {
					result.NavPatch = protocol.NewNavReplacePatch(fullPath)
				} else {
					result.NavPatch = protocol.NewNavPushPatch(fullPath)
				}
				return result
			}
		} else {
			result.Matched = true
		}

		// Store the current path/query/params
		rn.currentPath = canonPath
		rn.currentQuery = query
		rn.currentParams = match.GetParams()

		// Update session's current route
		rn.session.CurrentRoute = canonPath

		// Per Section 8.4: Check prefetch cache before rendering.
		var patches []vdom.Patch
		var renderErr error
		if cache := rn.session.PrefetchCache(); cache != nil {
			if entry := cache.Get(canonPath); entry != nil {
				patches, renderErr = rn.useCachedTree(entry.Tree, match)
			}
		}
		if patches == nil && renderErr == nil {
			patches, renderErr = rn.renderRoute(match)
		}

		if renderErr != nil {
			if re, ok := renderErr.(redirectError); ok {
				path = re.path
				replace = true
				continue
			}
			result.Error = renderErr
			return result
		}

		result.Path = fullPath
		if localReplace {
			result.NavPatch = protocol.NewNavReplacePatch(fullPath)
		} else {
			result.NavPatch = protocol.NewNavPushPatch(fullPath)
		}
		result.Patches = patches
		return result
	}

	result.Error = errors.New("too many redirects")
	return result
}

// simpleRouteMatch is a minimal implementation of RouteMatch for 404 pages.
type simpleRouteMatch struct {
	pageHandler PageHandler
	params      map[string]string
}

func (m *simpleRouteMatch) GetParams() map[string]string        { return m.params }
func (m *simpleRouteMatch) GetPageHandler() PageHandler         { return m.pageHandler }
func (m *simpleRouteMatch) GetLayoutHandlers() []LayoutHandler  { return nil }
func (m *simpleRouteMatch) GetMiddleware() []RouteMiddleware    { return nil }

// renderRoute renders a matched route and returns DOM patches.
func (rn *RouteNavigator) renderRoute(match RouteMatch) ([]vdom.Patch, error) {
	pageHandler := match.GetPageHandler()
	if pageHandler == nil {
		return nil, nil
	}

	// Create a render context for the page
	renderCtx := rn.session.createRenderContext()

	// Set route params on context
	if ctxImpl, ok := renderCtx.(*ctx); ok {
		ctxImpl.setParams(match.GetParams())
		if ctxImpl.request == nil {
			ctxImpl.request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path:     rn.currentPath,
					RawQuery: rn.currentQuery,
				},
			}
		}
	}

	// Render within vango.WithCtx for proper reactive context
	var newTree *vdom.VNode
	var ranFinal bool
	var middlewareErr error
	vango.WithCtx(renderCtx, func() {
		ranFinal, middlewareErr = RunRouteMiddleware(renderCtx, match.GetMiddleware(), func() error {
			// Call the page handler to get the component
			comp := pageHandler(renderCtx, match.GetParams())
			if comp == nil {
				return nil
			}

			// Render the component to VNode
			newTree = comp.Render()

			// Apply layouts root to leaf (reverse order so outermost is first)
			layouts := match.GetLayoutHandlers()
			for i := len(layouts) - 1; i >= 0; i-- {
				layout := layouts[i]
				newTree = layout(renderCtx, newTree)
			}

			return nil
		})
	})

	if ctxImpl, ok := renderCtx.(*ctx); ok {
		if path, rep, has := ctxImpl.PendingNavigation(); has {
			ctxImpl.ClearPendingNavigation()
			return nil, redirectError{path: path, replace: rep}
		}
	}

	if middlewareErr != nil {
		return nil, middlewareErr
	}
	if !ranFinal {
		return nil, errors.New("route middleware aborted navigation without redirect")
	}

	if newTree == nil {
		return nil, nil
	}

	// SSR expands nested components inline during HTML generation. For WS navigation,
	// expand KindComponent nodes to match SSR tree shape before HID assignment and diffing.
	newTree = expandComponents(newTree)

	// Get old tree for diffing
	oldTree := rn.session.currentTree

	// Assign HIDs to the new tree
	// Try to copy HIDs from old tree first to preserve stability
	if oldTree != nil {
		vdom.CopyHIDs(oldTree, newTree)
	}
	vdom.AssignHIDs(newTree, rn.session.hidGen)

	// Diff old and new
	patches := vdom.Diff(oldTree, newTree)

	// Update session state
	rn.session.currentTree = newTree

	// Clear old component state and collect new handlers
	if rn.session.root != nil {
		rn.session.clearComponentHandlers(rn.session.root)
		rn.session.unregisterComponent(rn.session.root)
		rn.session.root.Dispose()
		rn.session.root = nil
	}

	// Collect handlers from the new tree
	rn.session.handlers = make(map[string]Handler)
	rn.session.components = make(map[string]*ComponentInstance)
	rn.collectHandlersFromTree(newTree)

	return patches, nil
}

// useCachedTree uses a prefetched tree for navigation (cache hit path).
// Per Section 8.4: "If hit and not stale → reuse rendered tree and cached data"
// This skips the page handler execution since the tree is already rendered.
func (rn *RouteNavigator) useCachedTree(cachedTree *vdom.VNode, match RouteMatch) ([]vdom.Patch, error) {
	if cachedTree == nil {
		return nil, nil
	}

	// Run route middleware even on cache hits so guards stay correct.
	renderCtx := rn.session.createRenderContext()
	if ctxImpl, ok := renderCtx.(*ctx); ok {
		ctxImpl.setParams(match.GetParams())
		if ctxImpl.request == nil {
			ctxImpl.request = &http.Request{
				Method: http.MethodGet,
				URL: &url.URL{
					Path:     rn.currentPath,
					RawQuery: rn.currentQuery,
				},
			}
		}
	}

	var ranFinal bool
	var middlewareErr error
	vango.WithCtx(renderCtx, func() {
		ranFinal, middlewareErr = RunRouteMiddleware(renderCtx, match.GetMiddleware(), func() error {
			return nil
		})
	})

	if ctxImpl, ok := renderCtx.(*ctx); ok {
		if path, rep, has := ctxImpl.PendingNavigation(); has {
			ctxImpl.ClearPendingNavigation()
			return nil, redirectError{path: path, replace: rep}
		}
	}

	if middlewareErr != nil {
		return nil, middlewareErr
	}
	if !ranFinal {
		return nil, errors.New("route middleware aborted navigation without redirect")
	}

	// Get old tree for diffing
	oldTree := rn.session.currentTree

	// Copy the cached tree (we need our own copy for HID assignment)
	newTree := cachedTree

	// Ensure cached trees are expanded so HID assignment and diffing match SSR.
	newTree = expandComponents(newTree)

	// Assign HIDs to the new tree
	// Try to copy HIDs from old tree first to preserve stability
	if oldTree != nil {
		vdom.CopyHIDs(oldTree, newTree)
	}
	vdom.AssignHIDs(newTree, rn.session.hidGen)

	// Diff old and new
	patches := vdom.Diff(oldTree, newTree)

	// Update session state
	rn.session.currentTree = newTree

	// Clear old component state and collect new handlers
	if rn.session.root != nil {
		rn.session.clearComponentHandlers(rn.session.root)
		rn.session.unregisterComponent(rn.session.root)
		rn.session.root.Dispose()
		rn.session.root = nil
	}

	// Collect handlers from the new tree
	rn.session.handlers = make(map[string]Handler)
	rn.session.components = make(map[string]*ComponentInstance)
	rn.collectHandlersFromTree(newTree)

	return patches, nil
}

// collectHandlersFromTree walks a VNode tree and collects event handlers.
// This is used after rendering a new page to register handlers without
// creating ComponentInstance wrappers (since page handlers return VNodes directly).
func (rn *RouteNavigator) collectHandlersFromTree(node *vdom.VNode) {
	if node == nil {
		return
	}

	// If this node has an HID, check for event handlers
	if node.HID != "" {
		for key, value := range node.Props {
			if value == nil {
				continue
			}

			// Check for on* handlers
			if len(key) > 2 && key[:2] == "on" {
				handler := wrapHandler(value)
				handlerKey := node.HID + "_" + key
				rn.session.handlers[handlerKey] = handler
			}
		}
	}

	// Recurse to children
	for _, child := range node.Children {
		rn.collectHandlersFromTree(child)
	}
}

// expandComponents replaces vdom.KindComponent nodes with their rendered output,
// recursively, returning a tree equivalent to what SSR would render inline.
func expandComponents(node *vdom.VNode) *vdom.VNode {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case vdom.KindComponent:
		if node.Comp == nil {
			return nil
		}
		return expandComponents(node.Comp.Render())
	case vdom.KindElement, vdom.KindFragment:
		if len(node.Children) > 0 {
			expanded := make([]*vdom.VNode, 0, len(node.Children))
			for _, child := range node.Children {
				if c := expandComponents(child); c != nil {
					expanded = append(expanded, c)
				}
			}
			node.Children = expanded
		}
		return node
	default:
		return node
	}
}

// CurrentPath returns the current route path.
func (rn *RouteNavigator) CurrentPath() string {
	return rn.currentPath
}

// CurrentParams returns the current route parameters.
func (rn *RouteNavigator) CurrentParams() map[string]string {
	return rn.currentParams
}
