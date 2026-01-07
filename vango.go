// Package vango provides the public API for the Vango web framework.
//
// This is the recommended import for most applications:
//
//	import "github.com/vango-go/vango"
//
// Usage:
//
//	ctx := vango.UseCtx()
//	count := vango.NewSignal(0)
//	form := vango.UseForm(MyFormData{})
//	search := vango.URLParam("q", "", vango.Replace, vango.Debounce(300*time.Millisecond))
package vango

import (
	"context"
	"time"

	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/urlparam"
	"github.com/vango-go/vango/pkg/features/form"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// Context (server.Ctx exposed as vango.Ctx)
// =============================================================================

// Ctx is the runtime context with full HTTP/navigation/session access.
// This is server.Ctx - the rich context that includes Path(), Param(),
// Query(), QueryParam(), Navigate(), User(), Session(), etc.
type Ctx = server.Ctx

// UseCtx returns the current runtime context.
// Returns nil if called outside of a render/effect/handler context.
//
// Example:
//
//	func MyComponent() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        ctx := vango.UseCtx()
//	        path := ctx.Path()
//	        userId := ctx.Param("id")
//	        search := ctx.QueryParam("q")
//	        return Div(Text(path))
//	    })
//	}
func UseCtx() Ctx {
	raw := corevango.UseCtx()
	if raw == nil {
		return nil
	}
	// Type-assert from core vango.Ctx to server.Ctx
	if ctx, ok := raw.(server.Ctx); ok {
		return ctx
	}
	return nil
}

// =============================================================================
// Navigation options (re-export from server)
// =============================================================================

// NavigateOption configures programmatic navigation.
type NavigateOption = server.NavigateOption

// WithReplace replaces the current history entry instead of pushing.
var WithReplace = server.WithReplace

// WithNavigateParams adds query parameters to the navigation URL.
var WithNavigateParams = server.WithNavigateParams

// WithoutScroll disables scrolling to top after navigation.
var WithoutScroll = server.WithoutScroll

// =============================================================================
// Reactive primitives (re-export from pkg/vango)
// =============================================================================

// NewSignal creates a new reactive signal with the given initial value.
//
// Example:
//
//	count := vango.NewSignal(0)
//	count.Set(1)
//	value := count.Get() // 1
func NewSignal[T any](initial T, opts ...SignalOption) *Signal[T] {
	return corevango.NewSignal(initial, opts...)
}

// NewMemo creates a new computed value that automatically tracks dependencies.
//
// Example:
//
//	doubled := vango.NewMemo(func() int {
//	    return count.Get() * 2
//	})
func NewMemo[T any](compute func() T) *Memo[T] {
	return corevango.NewMemo(compute)
}

// CreateEffect registers a side effect that runs when dependencies change.
//
// Example:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    fmt.Println("Count changed to:", count.Get())
//	    return nil
//	})
var CreateEffect = corevango.CreateEffect

// NewAction creates a structured async mutation with state tracking.
func NewAction[A any, R any](do func(ctx context.Context, arg A) (R, error), opts ...ActionOption) *Action[A, R] {
	return corevango.NewAction(do, opts...)
}

// NewRef creates a mutable reference (primarily for DOM elements).
func NewRef[T any](initial T) *Ref[T] {
	return corevango.NewRef(initial)
}

// Batch groups multiple signal updates into a single notification.
var Batch = corevango.Batch

// Tx is an alias for Batch.
var Tx = corevango.Tx

// TxNamed is a named transaction for observability.
var TxNamed = corevango.TxNamed

// Untracked reads signals without creating subscriptions.
var Untracked = corevango.Untracked

// UntrackedGet reads a signal's value without subscribing.
func UntrackedGet[T any](s *Signal[T]) T {
	return corevango.UntrackedGet(s)
}

// Signal type aliases
type Signal[T any] = corevango.Signal[T]
type Memo[T any] = corevango.Memo[T]
type Action[A any, R any] = corevango.Action[A, R]
type Ref[T any] = corevango.Ref[T]
type Effect = corevango.Effect
type Cleanup = corevango.Cleanup
type SignalOption = corevango.SignalOption

// Signal options
var Transient = corevango.Transient
var PersistKey = corevango.PersistKey

// =============================================================================
// Effect helpers (re-export from pkg/vango)
// =============================================================================

// Interval runs a function at regular intervals.
var Interval = corevango.Interval

// Subscribe subscribes to a stream of values.
func Subscribe[T any](stream Stream[T], fn func(T), opts ...SubscribeOption) Cleanup {
	return corevango.Subscribe(stream, fn, opts...)
}

// GoLatest runs async work with key coalescing and cancellation.
func GoLatest[K comparable, R any](
	key K,
	work func(ctx context.Context, key K) (R, error),
	apply func(result R, err error),
	opts ...GoLatestOption,
) Cleanup {
	return corevango.GoLatest(key, work, apply, opts...)
}

// Timeout runs a function after a delay.
var Timeout = corevango.Timeout

// Effect options
type EffectOption = corevango.EffectOption
type IntervalOption = corevango.IntervalOption
type SubscribeOption = corevango.SubscribeOption
type GoLatestOption = corevango.GoLatestOption
type TimeoutOption = corevango.TimeoutOption
type ActionOption = corevango.ActionOption

var AllowWrites = corevango.AllowWrites
var EffectTxName = corevango.EffectTxName
var IntervalTxName = corevango.IntervalTxName
var IntervalImmediate = corevango.IntervalImmediate
var SubscribeTxName = corevango.SubscribeTxName
var GoLatestTxName = corevango.GoLatestTxName
var GoLatestForceRestart = corevango.GoLatestForceRestart
var TimeoutTxName = corevango.TimeoutTxName
var ActionTxName = corevango.ActionTxName

// Stream is an interface for event streams (used with Subscribe).
type Stream[T any] = corevango.Stream[T]

// ActionState represents the current state of an action.
type ActionState = corevango.ActionState

// ActionState constants
const (
	ActionIdle    = corevango.ActionIdle
	ActionRunning = corevango.ActionRunning
	ActionSuccess = corevango.ActionSuccess
	ActionError   = corevango.ActionError
)

// =============================================================================
// Errors (re-export from pkg/vango)
// =============================================================================

var ErrBudgetExceeded = corevango.ErrBudgetExceeded
var ErrQueueFull = corevango.ErrQueueFull
var ErrActionRunning = corevango.ErrActionRunning
var ErrEffectContext = corevango.ErrEffectContext
var ErrGoLatestContext = corevango.ErrGoLatestContext

type HTTPError = corevango.HTTPError

var BadRequest = corevango.BadRequest
var BadRequestf = corevango.BadRequestf
var Unauthorized = corevango.Unauthorized
var Forbidden = corevango.Forbidden
var NotFound = corevango.NotFound
var Conflict = corevango.Conflict
var UnprocessableEntity = corevango.UnprocessableEntity
var InternalError = corevango.InternalError
var ServiceUnavailable = corevango.ServiceUnavailable

// =============================================================================
// Context API (re-export from pkg/vango)
// =============================================================================

// CreateContext creates a new context type for dependency injection.
func CreateContext[T any](defaultValue T) *Context[T] {
	return corevango.CreateContext(defaultValue)
}

// Context is a reactive context for dependency injection.
type Context[T any] = corevango.Context[T]

// SetContext sets a value in the component context.
var SetContext = corevango.SetContext

// GetContext retrieves a value from the component context.
var GetContext = corevango.GetContext

// =============================================================================
// URLParam (re-export from pkg/urlparam)
// =============================================================================

// URLParam creates a URL parameter synced with query string.
// This is a hook-like API and MUST be called unconditionally during render.
//
// Example:
//
//	// Simple string param
//	query := vango.URLParam("q", "")
//
//	// With options
//	search := vango.URLParam("q", "", vango.Replace, vango.Debounce(300*time.Millisecond))
//
//	// Struct param with flat encoding
//	type Filters struct {
//	    Category string `url:"cat"`
//	    SortBy   string `url:"sort"`
//	}
//	filters := vango.URLParam("", Filters{}, vango.Encoding(vango.URLEncodingFlat))
func URLParam[T any](key string, def T, opts ...URLParamOption) *urlparam.URLParam[T] {
	return urlparam.Param(key, def, opts...)
}

// URLParamOption configures URL parameter behavior.
type URLParamOption = urlparam.URLParamOption

// URL parameter mode options
var (
	// Push creates a new history entry (default behavior).
	Push URLParamOption = urlparam.Push

	// Replace updates URL without creating history entry (use for filters, search).
	Replace URLParamOption = urlparam.Replace
)

// Debounce delays URL updates by the specified duration.
// Use this for search inputs to avoid spamming the history.
//
// Example:
//
//	search := vango.URLParam("q", "", vango.Replace, vango.Debounce(300*time.Millisecond))
func Debounce(d time.Duration) URLParamOption {
	return urlparam.Debounce(d)
}

// Encoding sets the URL encoding mode for complex types.
//
// Example:
//
//	filters := vango.URLParam("", Filters{}, vango.Encoding(vango.URLEncodingFlat))
func Encoding(e URLEncoding) URLParamOption {
	return urlparam.WithEncoding(e)
}

// URLEncoding specifies how complex types are serialized to URLs.
type URLEncoding = urlparam.Encoding

const (
	// URLEncodingFlat serializes structs as flat params: ?cat=tech&sort=asc
	URLEncodingFlat URLEncoding = urlparam.EncodingFlat

	// URLEncodingJSON serializes as base64-encoded JSON: ?filter=eyJjYXQiOiJ0ZWNoIn0
	URLEncodingJSON URLEncoding = urlparam.EncodingJSON

	// URLEncodingComma serializes arrays as comma-separated: ?tags=go,web,api
	URLEncodingComma URLEncoding = urlparam.EncodingComma
)

// =============================================================================
// Form (re-export from pkg/features/form)
// =============================================================================

// UseForm creates a reactive form handler bound to the given struct type.
// This is a hook-like API and MUST be called unconditionally during render.
//
// Example:
//
//	type ContactForm struct {
//	    Name    string `form:"name" validate:"required,min=2"`
//	    Email   string `form:"email" validate:"required,email"`
//	    Message string `form:"message" validate:"required"`
//	}
//
//	form := vango.UseForm(ContactForm{})
//	if form.Validate() {
//	    data := form.Values()
//	}
func UseForm[T any](initial T) *form.Form[T] {
	return form.UseForm(initial)
}

// Form is a type-safe form handler with validation support.
type Form[T any] = form.Form[T]

// Validator is an interface for form field validation.
type Validator = form.Validator

// ValidatorFunc is a function that implements Validator.
type ValidatorFunc = form.ValidatorFunc

// ValidationError represents a validation failure.
type ValidationError = form.ValidationError

// Validators (functions that return Validator)

// Required validates that the value is non-empty.
func Required(msg string) Validator { return form.Required(msg) }

// MinLength validates that a string has at least n characters.
func MinLength(n int, msg string) Validator { return form.MinLength(n, msg) }

// MaxLength validates that a string has at most n characters.
func MaxLength(n int, msg string) Validator { return form.MaxLength(n, msg) }

// Pattern validates that a string matches the given regular expression.
func Pattern(pattern, msg string) Validator { return form.Pattern(pattern, msg) }

// Email validates that the value is a valid email address.
func Email(msg string) Validator { return form.Email(msg) }

// URL validates that the value is a valid URL.
func URL(msg string) Validator { return form.URL(msg) }

// UUID validates that the value is a valid UUID.
func UUID(msg string) Validator { return form.UUID(msg) }

// Alpha validates that the value contains only ASCII letters.
func Alpha(msg string) Validator { return form.Alpha(msg) }

// AlphaNumeric validates that the value contains only letters and digits.
func AlphaNumeric(msg string) Validator { return form.AlphaNumeric(msg) }

// Numeric validates that the value contains only digits.
func Numeric(msg string) Validator { return form.Numeric(msg) }

// Phone validates that the value looks like a phone number.
func Phone(msg string) Validator { return form.Phone(msg) }

// Min validates that a numeric value is >= n.
func Min(n any, msg string) Validator { return form.Min(n, msg) }

// Max validates that a numeric value is <= n.
func Max(n any, msg string) Validator { return form.Max(n, msg) }

// Between validates that a numeric value is between min and max (inclusive).
func Between(min, max any, msg string) Validator { return form.Between(min, max, msg) }

// Positive validates that a numeric value is > 0.
func Positive(msg string) Validator { return form.Positive(msg) }

// NonNegative validates that a numeric value is >= 0.
func NonNegative(msg string) Validator { return form.NonNegative(msg) }

// DateAfter validates that a date/time is after the given time.
func DateAfter(t time.Time, msg string) Validator { return form.DateAfter(t, msg) }

// DateBefore validates that a date/time is before the given time.
func DateBefore(t time.Time, msg string) Validator { return form.DateBefore(t, msg) }

// Future validates that a date/time is in the future.
func Future(msg string) Validator { return form.Future(msg) }

// Past validates that a date/time is in the past.
func Past(msg string) Validator { return form.Past(msg) }

// Custom creates a validator from a custom function.
func Custom(fn func(value any) error) Validator { return form.Custom(fn) }

// EqualTo returns a validator that checks if the value equals another field.
func EqualTo(field string, msg string) *form.EqualToField { return form.EqualTo(field, msg) }

// NotEqualTo returns a validator that ensures the value differs from another field.
func NotEqualTo(field string, msg string) *form.NotEqualToField { return form.NotEqualTo(field, msg) }

// Async creates an async validator for server-side checks.
func Async(fn func(value any) (error, bool)) *form.AsyncValidator { return form.Async(fn) }

// =============================================================================
// Component/VNode (re-export from pkg/vdom)
// =============================================================================

// Component is anything that can render to a VNode.
type Component = vdom.Component

// VNode represents a virtual DOM node.
type VNode = vdom.VNode

// Props holds attributes and event handlers.
type Props = vdom.Props

// VKind is the node type discriminator.
type VKind = vdom.VKind

// VKind constants
const (
	KindElement   = vdom.KindElement
	KindText      = vdom.KindText
	KindFragment  = vdom.KindFragment
	KindComponent = vdom.KindComponent
	KindRaw       = vdom.KindRaw
)

// Func wraps a render function as a Component.
// This is the primary way to create stateful components.
//
// Example:
//
//	func Counter(initial int) vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        count := vango.NewSignal(initial)
//	        return Div(
//	            H1(Textf("Count: %d", count.Get())),
//	            Button(OnClick(count.Inc), Text("+")),
//	        )
//	    })
//	}
func Func(render func() *vdom.VNode) vdom.Component {
	return vdom.Func(render)
}

// =============================================================================
// Configuration (re-export from pkg/vango)
// =============================================================================

// DevMode enables development-time validation.
var DevMode = &corevango.DevMode
