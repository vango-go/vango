package el

import "github.com/vango-go/vango/pkg/vdom"

// Type aliases for the VDOM primitives used by the DSL.
type VNode = vdom.VNode
type VKind = vdom.VKind
type Props = vdom.Props
type Attr = vdom.Attr
type EventHandler = vdom.EventHandler
type Component = vdom.Component
type Case[T comparable] = vdom.Case[T]
type ScriptsOption = vdom.ScriptsOption
