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
//	search := vango.URLParam("q", "", vango.Replace, vango.URLDebounce(300*time.Millisecond))
package vango

import (
	"context"
	"time"

	"github.com/vango-go/vango/pkg/assets"
	"github.com/vango-go/vango/pkg/features/form"
	"github.com/vango-go/vango/pkg/features/hooks"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/urlparam"
	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// Context (server.Ctx exposed as vango.Ctx)
// =============================================================================

// Ctx is the runtime context with full HTTP/navigation/session access.
// This is server.Ctx - the rich context that includes Path(), Param(),
// Query(), QueryParam(), Navigate(), User(), Session(), etc.
type Ctx = server.Ctx

// WithUser stores an authenticated user in a stdlib context for SSR and session bridging.
// Pair with UserFromContext and Config.OnSessionStart.
func WithUser(ctx context.Context, user any) context.Context {
	return server.WithUser(ctx, user)
}

// UserFromContext retrieves an authenticated user stored by WithUser.
func UserFromContext(ctx context.Context) any {
	return server.UserFromContext(ctx)
}

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
// Deprecated: Use Effect instead for spec-aligned naming.
//
// Example:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    fmt.Println("Count changed to:", count.Get())
//	    return nil
//	})
var CreateEffect = corevango.CreateEffect

// Effect registers a side effect that runs when dependencies change.
// This is the spec-aligned name for CreateEffect.
//
// Example:
//
//	vango.Effect(func() vango.Cleanup {
//	    fmt.Println("Count changed to:", count.Get())
//	    return nil
//	})
var Effect = corevango.CreateEffect

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
type EffectHandle = corevango.Effect // Renamed to avoid collision with Effect function
type Cleanup = corevango.Cleanup
type SignalOption = corevango.SignalOption

// Signal options
var Transient = corevango.Transient
var PersistKey = corevango.PersistKey

// =============================================================================
// Events (re-export from pkg/vango)
// =============================================================================

type MouseEvent = corevango.MouseEvent
type KeyboardEvent = corevango.KeyboardEvent
type InputEvent = corevango.InputEvent
type WheelEvent = corevango.WheelEvent
type DragEvent = corevango.DragEvent
type DropEvent = corevango.DropEvent
type Touch = corevango.Touch
type TouchEvent = corevango.TouchEvent
type AnimationEvent = corevango.AnimationEvent
type TransitionEvent = corevango.TransitionEvent
type ScrollEvent = corevango.ScrollEvent
type ResizeEvent = corevango.ResizeEvent
type FormData = corevango.FormData
type HookEvent = corevango.HookEvent
type NavigateEvent = corevango.NavigateEvent

func NewFormData(values map[string][]string) FormData { return corevango.NewFormData(values) }
func NewFormDataFromSingle(values map[string]string) FormData {
	return corevango.NewFormDataFromSingle(values)
}

// =============================================================================
// Key constants (re-export from pkg/vango)
// =============================================================================

const (
	KeyEnter     = corevango.KeyEnter
	KeyEscape    = corevango.KeyEscape
	KeySpace     = corevango.KeySpace
	KeyTab       = corevango.KeyTab
	KeyBackspace = corevango.KeyBackspace
	KeyDelete    = corevango.KeyDelete

	KeyArrowUp    = corevango.KeyArrowUp
	KeyArrowDown  = corevango.KeyArrowDown
	KeyArrowLeft  = corevango.KeyArrowLeft
	KeyArrowRight = corevango.KeyArrowRight

	KeyHome     = corevango.KeyHome
	KeyEnd      = corevango.KeyEnd
	KeyPageUp   = corevango.KeyPageUp
	KeyPageDown = corevango.KeyPageDown

	KeyF1  = corevango.KeyF1
	KeyF2  = corevango.KeyF2
	KeyF3  = corevango.KeyF3
	KeyF4  = corevango.KeyF4
	KeyF5  = corevango.KeyF5
	KeyF6  = corevango.KeyF6
	KeyF7  = corevango.KeyF7
	KeyF8  = corevango.KeyF8
	KeyF9  = corevango.KeyF9
	KeyF10 = corevango.KeyF10
	KeyF11 = corevango.KeyF11
	KeyF12 = corevango.KeyF12

	KeyControl = corevango.KeyControl
	KeyShift   = corevango.KeyShift
	KeyAlt     = corevango.KeyAlt
	KeyMeta    = corevango.KeyMeta

	KeyInsert      = corevango.KeyInsert
	KeyPrintScreen = corevango.KeyPrintScreen
	KeyScrollLock  = corevango.KeyScrollLock
	KeyPause       = corevango.KeyPause
	KeyCapsLock    = corevango.KeyCapsLock
	KeyNumLock     = corevango.KeyNumLock
	KeyContextMenu = corevango.KeyContextMenu
)

// =============================================================================
// Event modifiers (re-export from pkg/vango)
// =============================================================================

type ModifiedHandler = corevango.ModifiedHandler
type KeyMod = corevango.KeyMod

const (
	Ctrl  = corevango.Ctrl
	Shift = corevango.Shift
	Alt   = corevango.Alt
	Meta  = corevango.Meta
)

func PreventDefault(handler any) ModifiedHandler  { return corevango.PreventDefault(handler) }
func StopPropagation(handler any) ModifiedHandler { return corevango.StopPropagation(handler) }
func Self(handler any) ModifiedHandler            { return corevango.Self(handler) }
func Once(handler any) ModifiedHandler            { return corevango.Once(handler) }
func Passive(handler any) ModifiedHandler         { return corevango.Passive(handler) }
func Capture(handler any) ModifiedHandler         { return corevango.Capture(handler) }

func Debounce(duration time.Duration, handler any) ModifiedHandler {
	return corevango.Debounce(duration, handler)
}

func Throttle(duration time.Duration, handler any) ModifiedHandler {
	return corevango.Throttle(duration, handler)
}

func Hotkey(key string, handler any) ModifiedHandler { return corevango.Hotkey(key, handler) }
func Keys(keys []string, handler any) ModifiedHandler { return corevango.Keys(keys, handler) }

func KeyWithModifiers(key string, mods KeyMod, handler any) ModifiedHandler {
	return corevango.KeyWithModifiers(key, mods, handler)
}

// =============================================================================
// Hooks (re-export from pkg/features/hooks, spec-aligned types)
// =============================================================================

func Hook(name string, config any) vdom.Attr { return hooks.Hook(name, config) }

// OnEvent attaches a hook event handler to an element.
// This is spec-aligned: the handler receives `vango.HookEvent`.
func OnEvent(name string, handler func(HookEvent)) vdom.Attr {
	wrapped := func(e HookEvent) {
		if e.Name == name {
			handler(e)
		}
	}
	return vdom.Attr{Key: "onhook", Value: wrapped}
}

// =============================================================================
// Shared & Global Signals (re-export from pkg/vango)
// =============================================================================

// NewSharedSignal creates a session-scoped signal definition.
// Each user session gets its own independent Signal[T] instance.
// The signal is lazily created when first accessed in a session.
//
// Use for state that should be private to each user (e.g., shopping cart, preferences).
//
// Note: Due to Go's type system, SharedSignalDef[T] is not literally Signal[T],
// but it proxies all Signal methods and behaves identically in use.
//
// Example:
//
//	// Define at package level
//	var cartItems = vango.NewSharedSignal([]CartItem{})
//
//	// Use in component (each user sees their own cart)
//	items := cartItems.Get()
//	cartItems.Set(append(items, newItem))
func NewSharedSignal[T any](initial T, opts ...SignalOption) *SharedSignalDef[T] {
	return corevango.NewSharedSignal(initial, opts...)
}

// SharedSignalDef is a session-scoped signal definition.
// It proxies all Signal[T] methods to the session's instance.
type SharedSignalDef[T any] = corevango.SharedSignalDef[T]

// NewGlobalSignal creates an application-wide signal shared across all sessions.
// All users see and modify the same value.
//
// Use for state that should be visible to everyone (e.g., online user count, announcements).
//
// Example:
//
//	// Define at package level
//	var onlineCount = vango.NewGlobalSignal(0)
//
//	// Increment when user connects (all users see the count)
//	onlineCount.Inc()
func NewGlobalSignal[T any](initial T, opts ...SignalOption) *GlobalSignal[T] {
	return corevango.NewGlobalSignal(initial, opts...)
}

// GlobalSignal is an application-wide signal shared across all sessions.
// It embeds Signal[T] and provides direct access to all Signal methods.
type GlobalSignal[T any] = corevango.GlobalSignal[T]

// =============================================================================
// Shared & Global Memos (re-export from pkg/vango)
// =============================================================================

// NewSharedMemo creates a session-scoped memo definition.
// Each user session gets its own independent Memo[T] instance.
func NewSharedMemo[T any](compute func() T) *SharedMemoDef[T] {
	return corevango.NewSharedMemo(compute)
}

// SharedMemoDef is a session-scoped memo definition.
// It proxies Memo[T] reads to the session's instance.
type SharedMemoDef[T any] = corevango.SharedMemoDef[T]

// NewGlobalMemo creates an application-wide memo shared across all sessions.
func NewGlobalMemo[T any](compute func() T) *GlobalMemo[T] {
	return corevango.NewGlobalMemo(compute)
}

// GlobalMemo is an application-wide memo shared across all sessions.
// It embeds Memo[T] and provides direct access to Memo methods.
type GlobalMemo[T any] = corevango.GlobalMemo[T]

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
//	search := vango.URLParam("q", "", vango.Replace, vango.URLDebounce(300*time.Millisecond))
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
//	search := vango.URLParam("q", "", vango.Replace, vango.URLDebounce(300*time.Millisecond))
func URLDebounce(d time.Duration) URLParamOption {
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

// =============================================================================
// Assets (re-export from pkg/assets)
// =============================================================================

// AssetManifest holds the mapping from source asset paths to fingerprinted paths.
// Use LoadAssetManifest to load from a manifest.json file.
type AssetManifest = assets.Manifest

// AssetResolver provides asset path resolution with prefix support.
type AssetResolver = assets.Resolver

// LoadAssetManifest loads a manifest.json file generated by the build process.
// The manifest maps source paths to fingerprinted paths:
//
//	{"vango.js": "vango.a1b2c3d4.min.js", "styles.css": "styles.e5f6g7h8.css"}
//
// Example:
//
//	manifest, err := vango.LoadAssetManifest("dist/manifest.json")
//	if err != nil {
//	    // In dev mode, this is expected - use passthrough resolver
//	    log.Println("No manifest found, using passthrough resolver")
//	}
func LoadAssetManifest(path string) (*AssetManifest, error) {
	return assets.Load(path)
}

// NewAssetResolver creates a resolver from a manifest with a path prefix.
// The prefix is prepended to all resolved paths.
//
// Example:
//
//	manifest, _ := vango.LoadAssetManifest("dist/manifest.json")
//	resolver := vango.NewAssetResolver(manifest, "/public/")
//
//	// Configure server
//	config := server.DefaultServerConfig()
//	config.AssetResolver = resolver
func NewAssetResolver(m *AssetManifest, prefix string) AssetResolver {
	return assets.NewResolver(m, prefix)
}

// NewPassthroughResolver creates a resolver that returns paths unchanged.
// Use this in development mode where fingerprinting is disabled.
//
// Example:
//
//	resolver := vango.NewPassthroughResolver("/public/")
//	resolver.Asset("vango.js") // "/public/vango.js"
func NewPassthroughResolver(prefix string) AssetResolver {
	return assets.NewPassthroughResolver(prefix)
}
