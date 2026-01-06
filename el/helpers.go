// This file re-exports vdom helper functions for the el package.
package el

import "github.com/vango-go/vango/pkg/vdom"

func Text(content string) *VNode {
	return vdom.Text(content)
}
func Textf(format string, args ...any) *VNode {
	return vdom.Textf(format, args...)
}
func Raw(html string) *VNode {
	return vdom.Raw(html)
}
func Fragment(children ...any) *VNode {
	return vdom.Fragment(children...)
}
func If(condition bool, node *VNode) *VNode {
	return vdom.If(condition, node)
}
func IfElse(condition bool, ifTrue, ifFalse *VNode) *VNode {
	return vdom.IfElse(condition, ifTrue, ifFalse)
}
func When(condition bool, fn func() *VNode) *VNode {
	return vdom.When(condition, fn)
}
func IfLazy(condition bool, fn func() *VNode) *VNode {
	return vdom.IfLazy(condition, fn)
}
func ShowWhen(condition bool, fn func() *VNode) *VNode {
	return vdom.ShowWhen(condition, fn)
}
func Unless(condition bool, node *VNode) *VNode {
	return vdom.Unless(condition, node)
}
func Case_[T comparable](value T, node *VNode) Case[T] {
	return vdom.Case_(value, node)
}
func Default[T comparable](node *VNode) Case[T] {
	return vdom.Default(node)
}
func Switch[T comparable](value T, cases ...Case[T]) *VNode {
	return vdom.Switch(value, cases...)
}
func Range[T any](items []T, fn func(item T, index int) *VNode) []*VNode {
	return vdom.Range(items, fn)
}
func RangeMap[K comparable, V any](m map[K]V, fn func(key K, value V) *VNode) []*VNode {
	return vdom.RangeMap(m, fn)
}
func Repeat(n int, fn func(i int) *VNode) []*VNode {
	return vdom.Repeat(n, fn)
}
func Key(key any) Attr {
	return vdom.Key(key)
}
func Nothing() *VNode {
	return vdom.Nothing()
}
func Show(condition bool, node *VNode) *VNode {
	return vdom.Show(condition, node)
}
func Hide(condition bool, node *VNode) *VNode {
	return vdom.Hide(condition, node)
}
func Either(first, second *VNode) *VNode {
	return vdom.Either(first, second)
}
func Maybe(node *VNode) *VNode {
	return vdom.Maybe(node)
}
func Group(children ...any) *VNode {
	return vdom.Group(children...)
}
func NavLink(path string, children ...any) *VNode {
	return vdom.NavLink(path, children...)
}
func WithDebug() ScriptsOption {
	return vdom.WithDebug()
}
func WithScriptPath(path string) ScriptsOption {
	return vdom.WithScriptPath(path)
}
func WithCSRFToken(token string) ScriptsOption {
	return vdom.WithCSRFToken(token)
}
func WithoutDefer() ScriptsOption {
	return vdom.WithoutDefer()
}
func VangoScripts(opts ...ScriptsOption) *VNode {
	return vdom.VangoScripts(opts...)
}
