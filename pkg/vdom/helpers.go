package vdom

import "fmt"

// Text creates a text node.
func Text(content string) *VNode {
	return &VNode{
		Kind: KindText,
		Text: content,
	}
}

// Textf creates a formatted text node.
func Textf(format string, args ...any) *VNode {
	return Text(fmt.Sprintf(format, args...))
}

// Raw creates an unescaped HTML node.
// Use with caution - can lead to XSS if content is user-provided.
func Raw(html string) *VNode {
	return &VNode{
		Kind: KindRaw,
		Text: html,
	}
}

// Fragment groups children without a wrapper element.
func Fragment(children ...any) *VNode {
	node := &VNode{
		Kind:     KindFragment,
		Children: make([]*VNode, 0),
	}

	for _, child := range children {
		switch v := child.(type) {
		case nil:
			continue
		case *VNode:
			if v != nil {
				node.Children = append(node.Children, v)
			}
		case []*VNode:
			for _, c := range v {
				if c != nil {
					node.Children = append(node.Children, c)
				}
			}
		case string:
			node.Children = append(node.Children, Text(v))
		case Component:
			node.Children = append(node.Children, &VNode{
				Kind: KindComponent,
				Comp: v,
			})
		}
	}

	return node
}

// If returns the node if condition is true, nil otherwise.
func If(condition bool, node *VNode) *VNode {
	if condition {
		return node
	}
	return nil
}

// IfElse returns the first node if condition is true, the second otherwise.
func IfElse(condition bool, ifTrue, ifFalse *VNode) *VNode {
	if condition {
		return ifTrue
	}
	return ifFalse
}

// When is like If but with lazy evaluation.
// The function is only called if condition is true.
func When(condition bool, fn func() *VNode) *VNode {
	if condition {
		return fn()
	}
	return nil
}

// Unless is the inverse of If.
// Returns the node if condition is false.
func Unless(condition bool, node *VNode) *VNode {
	if !condition {
		return node
	}
	return nil
}

// Case represents a case in a Switch statement.
type Case[T comparable] struct {
	Value     T
	Node      *VNode
	IsDefault bool
}

// Case_ creates a case for Switch.
func Case_[T comparable](value T, node *VNode) Case[T] {
	return Case[T]{Value: value, Node: node}
}

// Default creates a default case for Switch.
func Default[T comparable](node *VNode) Case[T] {
	return Case[T]{Node: node, IsDefault: true}
}

// Switch returns the node for the matching case value.
// If no case matches and there's a default, the default node is returned.
func Switch[T comparable](value T, cases ...Case[T]) *VNode {
	// First pass: look for matching value
	for _, c := range cases {
		if !c.IsDefault && c.Value == value {
			return c.Node
		}
	}
	// Second pass: look for default
	for _, c := range cases {
		if c.IsDefault {
			return c.Node
		}
	}
	return nil
}

// Range maps a slice to VNodes.
func Range[T any](items []T, fn func(item T, index int) *VNode) []*VNode {
	result := make([]*VNode, 0, len(items))
	for i, item := range items {
		node := fn(item, i)
		if node != nil {
			result = append(result, node)
		}
	}
	return result
}

// RangeMap maps a map to VNodes.
// Note: map iteration order is not guaranteed.
func RangeMap[K comparable, V any](m map[K]V, fn func(key K, value V) *VNode) []*VNode {
	result := make([]*VNode, 0, len(m))
	for k, v := range m {
		node := fn(k, v)
		if node != nil {
			result = append(result, node)
		}
	}
	return result
}

// Repeat creates n nodes using the given function.
func Repeat(n int, fn func(i int) *VNode) []*VNode {
	if n <= 0 {
		return nil
	}
	result := make([]*VNode, 0, n)
	for i := 0; i < n; i++ {
		node := fn(i)
		if node != nil {
			result = append(result, node)
		}
	}
	return result
}

// Key creates a key attribute for reconciliation.
// The key is converted to a string using fmt.Sprintf.
func Key(key any) Attr {
	return attr("key", fmt.Sprintf("%v", key))
}

// Nothing returns nil, useful for conditional rendering.
func Nothing() *VNode {
	return nil
}

// Show returns the node if condition is true, otherwise Nothing.
// Alias for If for semantic clarity.
func Show(condition bool, node *VNode) *VNode {
	return If(condition, node)
}

// Hide returns the node if condition is false, otherwise Nothing.
// Alias for Unless for semantic clarity.
func Hide(condition bool, node *VNode) *VNode {
	return Unless(condition, node)
}

// Either returns first if it's not nil, otherwise second.
func Either(first, second *VNode) *VNode {
	if first != nil {
		return first
	}
	return second
}

// Maybe returns the node if it's not nil.
// This is a no-op but can make code more readable.
func Maybe(node *VNode) *VNode {
	return node
}

// Group is an alias for Fragment.
func Group(children ...any) *VNode {
	return Fragment(children...)
}
