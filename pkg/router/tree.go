package router

import "strings"

// RouteNode is a node in the radix tree.
type RouteNode struct {
	// segment is the path segment this node matches
	segment string

	// isParam indicates this is a parameter segment (:id)
	isParam bool

	// isCatchAll indicates this is a catch-all segment (*slug)
	isCatchAll bool

	// paramName is the parameter name (without : or *)
	paramName string

	// paramType is the expected parameter type (int, string, uuid)
	paramType string

	// handlers
	pageHandler   PageHandler
	layoutHandler LayoutHandler
	apiHandlers   map[string]APIHandler // method -> handler
	middleware    []Middleware

	// pageLayouts are layouts specified via Page() call - NOT inherited
	// These are used instead of hierarchical layouts when hasPageLayouts is true
	pageLayouts []LayoutHandler

	// hasPageLayouts distinguishes "unset" vs "explicitly empty"
	// This allows supporting "no layouts at all" use case
	hasPageLayouts bool

	// children are static segment children
	children []*RouteNode

	// paramChild is the dynamic parameter child (:id)
	paramChild *RouteNode

	// catchAllChild is the catch-all child (*slug)
	catchAllChild *RouteNode
}

// newRouteNode creates a new route node.
func newRouteNode(segment string) *RouteNode {
	return &RouteNode{
		segment: segment,
	}
}

// findChild finds a child node with an exact segment match.
func (n *RouteNode) findChild(segment string) *RouteNode {
	for _, child := range n.children {
		if child.segment == segment {
			return child
		}
	}
	return nil
}

// addChild adds or retrieves a child node for the given segment.
func (n *RouteNode) addChild(segment string) *RouteNode {
	// Check if child already exists
	if child := n.findChild(segment); child != nil {
		return child
	}

	// Create new child
	child := newRouteNode(segment)
	n.children = append(n.children, child)
	return child
}

// addParamChild sets the parameter child node.
func (n *RouteNode) addParamChild(name, paramType string) *RouteNode {
	if n.paramChild != nil {
		return n.paramChild
	}
	child := newRouteNode("")
	child.isParam = true
	child.paramName = name
	child.paramType = paramType
	n.paramChild = child
	return child
}

// addCatchAllChild sets the catch-all child node.
func (n *RouteNode) addCatchAllChild(name string) *RouteNode {
	if n.catchAllChild != nil {
		return n.catchAllChild
	}
	child := newRouteNode("")
	child.isCatchAll = true
	child.paramName = name
	child.paramType = "[]string"
	n.catchAllChild = child
	return child
}

// insertRoute adds a route to the tree.
func (n *RouteNode) insertRoute(path string) *RouteNode {
	segments := splitPath(path)
	current := n

	for _, seg := range segments {
		if strings.HasPrefix(seg, "*") {
			// Catch-all segment
			name := seg[1:]
			current = current.addCatchAllChild(name)
			break // Catch-all consumes rest of path
		} else if strings.HasPrefix(seg, ":") {
			// Parameter segment
			name, paramType := parseParamSegment(seg)
			current = current.addParamChild(name, paramType)
		} else {
			// Static segment
			current = current.addChild(seg)
		}
	}

	return current
}

// matchContext holds the accumulated layouts and middleware during matching.
type matchContext struct {
	layouts    []LayoutHandler
	middleware []Middleware
}

// match finds a node matching the given path segments.
// Returns the node, collected layouts/middleware, and extracted parameters.
// Layouts and middleware are collected at each level during traversal.
func (n *RouteNode) match(segments []string, params map[string]string, ctx *matchContext) (*RouteNode, *matchContext, bool) {
	// Collect layout and middleware at this node
	if n.layoutHandler != nil {
		ctx.layouts = append(ctx.layouts, n.layoutHandler)
	}
	if len(n.middleware) > 0 {
		ctx.middleware = append(ctx.middleware, n.middleware...)
	}

	// Base case: no more segments
	if len(segments) == 0 {
		// Check if this node has handlers
		if n.pageHandler != nil || n.apiHandlers != nil {
			return n, ctx, true
		}
		// Check for index child (handles trailing slash)
		if child := n.findChild(""); child != nil {
			if child.layoutHandler != nil {
				ctx.layouts = append(ctx.layouts, child.layoutHandler)
			}
			if len(child.middleware) > 0 {
				ctx.middleware = append(ctx.middleware, child.middleware...)
			}
			if child.pageHandler != nil || child.apiHandlers != nil {
				return child, ctx, true
			}
		}
		return nil, nil, false
	}

	segment := segments[0]
	remaining := segments[1:]

	// Try exact match first
	if child := n.findChild(segment); child != nil {
		if node, mctx, ok := child.match(remaining, params, ctx); ok {
			return node, mctx, true
		}
	}

	// Try parameter match
	if n.paramChild != nil {
		params[n.paramChild.paramName] = segment
		if node, mctx, ok := n.paramChild.match(remaining, params, ctx); ok {
			return node, mctx, true
		}
		// Backtrack on failure
		delete(params, n.paramChild.paramName)
	}

	// Try catch-all match
	if n.catchAllChild != nil {
		// Collect all remaining segments
		allSegments := append([]string{segment}, remaining...)
		params[n.catchAllChild.paramName] = strings.Join(allSegments, "/")
		if n.catchAllChild.layoutHandler != nil {
			ctx.layouts = append(ctx.layouts, n.catchAllChild.layoutHandler)
		}
		if len(n.catchAllChild.middleware) > 0 {
			ctx.middleware = append(ctx.middleware, n.catchAllChild.middleware...)
		}
		return n.catchAllChild, ctx, true
	}

	return nil, nil, false
}

// splitPath splits a path into segments.
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// parseParamSegment extracts name and type from a parameter segment.
// Input: ":id" or ":id:int" -> name="id", type="string" or "int"
func parseParamSegment(seg string) (name, paramType string) {
	seg = seg[1:] // Remove leading :
	if idx := strings.Index(seg, ":"); idx != -1 {
		return seg[:idx], seg[idx+1:]
	}
	return seg, "string"
}
