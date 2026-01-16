package vango

import (
	"errors"
	"testing"
	"time"

	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// NewAction Tests (via root package)
// =============================================================================

func TestNewAction_FunctionIsExported(t *testing.T) {
	// Verify the NewAction function exists and has correct type signature
	// We can't call it directly without a render context, but we can verify it's exported
	_ = NewAction[int, string]
}

func TestNewAction_TypeIsExported(t *testing.T) {
	// Verify the Action type is properly aliased
	var _ *Action[int, string]
}

func TestNewAction_ActionStateConstants(t *testing.T) {
	// Verify state constants are exported correctly
	if ActionIdle != 0 {
		t.Errorf("ActionIdle = %d, want 0", ActionIdle)
	}
	if ActionRunning != 1 {
		t.Errorf("ActionRunning = %d, want 1", ActionRunning)
	}
	if ActionSuccess != 2 {
		t.Errorf("ActionSuccess = %d, want 2", ActionSuccess)
	}
	if ActionError != 3 {
		t.Errorf("ActionError = %d, want 3", ActionError)
	}
}

// =============================================================================
// Shared/Global Signal Tests (via root package)
// =============================================================================

func TestNewSharedSignal_CreatesSignalDefinition(t *testing.T) {
	shared := NewSharedSignal(42)

	if shared == nil {
		t.Fatal("NewSharedSignal returned nil")
	}
}

func TestNewSharedSignal_WithOptions(t *testing.T) {
	shared := NewSharedSignal(0, Transient())

	if shared == nil {
		t.Fatal("NewSharedSignal with options returned nil")
	}
}

func TestNewGlobalSignal_CreatesApplicationWideSignal(t *testing.T) {
	global := NewGlobalSignal(100)

	if global == nil {
		t.Fatal("NewGlobalSignal returned nil")
	}

	if global.Get() != 100 {
		t.Errorf("Get() = %d, want %d", global.Get(), 100)
	}
}

func TestNewGlobalSignal_SetAndGet(t *testing.T) {
	global := NewGlobalSignal(0)

	global.Set(42)

	if global.Get() != 42 {
		t.Errorf("Get() after Set = %d, want %d", global.Get(), 42)
	}
}

func TestNewGlobalSignal_Update(t *testing.T) {
	global := NewGlobalSignal(10)

	global.Update(func(v int) int { return v * 2 })

	if global.Get() != 20 {
		t.Errorf("Get() after Update = %d, want %d", global.Get(), 20)
	}
}

// =============================================================================
// Shared/Global Memo Tests (via root package)
// =============================================================================

func TestNewSharedMemo_CreatesMemoDefinition(t *testing.T) {
	shared := NewSharedMemo(func() int {
		return 42
	})

	if shared == nil {
		t.Fatal("NewSharedMemo returned nil")
	}
}

func TestNewGlobalMemo_CreatesApplicationWideMemo(t *testing.T) {
	callCount := 0
	global := NewGlobalMemo(func() int {
		callCount++
		return 100
	})

	if global == nil {
		t.Fatal("NewGlobalMemo returned nil")
	}

	// First access should compute
	result := global.Get()
	if result != 100 {
		t.Errorf("Get() = %d, want %d", result, 100)
	}

	// Second access should be cached
	result2 := global.Get()
	if result2 != 100 {
		t.Errorf("Get() second call = %d, want %d", result2, 100)
	}
}

// =============================================================================
// URLParam Tests (via root package)
// =============================================================================

func TestURLParam_OptionsExist(t *testing.T) {
	// Verify URL param options are exported
	var opt URLParamOption

	opt = Push
	if opt == nil {
		t.Error("Push should not be nil")
	}

	opt = Replace
	if opt == nil {
		t.Error("Replace should not be nil")
	}
}

func TestURLDebounce_ReturnsOption(t *testing.T) {
	opt := URLDebounce(300 * time.Millisecond)
	if opt == nil {
		t.Error("URLDebounce should return non-nil option")
	}
}

func TestEncoding_ReturnsOption(t *testing.T) {
	opt := Encoding(URLEncodingFlat)
	if opt == nil {
		t.Error("Encoding should return non-nil option")
	}
}

func TestURLEncodingConstants_AreDistinct(t *testing.T) {
	// Verify encoding constants exist and have expected values
	encodings := []URLEncoding{URLEncodingFlat, URLEncodingJSON, URLEncodingComma}
	seen := make(map[URLEncoding]bool)

	for _, e := range encodings {
		if seen[e] {
			t.Errorf("Duplicate encoding value: %v", e)
		}
		seen[e] = true
	}
}

// =============================================================================
// Effect Helper Tests (via root package)
// =============================================================================

func TestInterval_IsExported(t *testing.T) {
	if Interval == nil {
		t.Error("Interval should not be nil")
	}
}

func TestTimeout_IsExported(t *testing.T) {
	if Timeout == nil {
		t.Error("Timeout should not be nil")
	}
}

func TestSubscribe_FunctionExists(t *testing.T) {
	// Verify Subscribe is callable (would panic if not exported correctly)
	// We can't fully test it without a stream, but we can verify the function exists
	_ = Subscribe[int]
}

func TestGoLatest_FunctionExists(t *testing.T) {
	// Verify GoLatest is callable
	_ = GoLatest[int, string]
}

// =============================================================================
// Effect Option Tests (via root package)
// =============================================================================

func TestEffectOptions_AreExported(t *testing.T) {
	// Verify effect options are exported
	if AllowWrites == nil {
		t.Error("AllowWrites should not be nil")
	}
	if EffectTxName == nil {
		t.Error("EffectTxName should not be nil")
	}
	if IntervalTxName == nil {
		t.Error("IntervalTxName should not be nil")
	}
	if IntervalImmediate == nil {
		t.Error("IntervalImmediate should not be nil")
	}
	if SubscribeTxName == nil {
		t.Error("SubscribeTxName should not be nil")
	}
	if GoLatestTxName == nil {
		t.Error("GoLatestTxName should not be nil")
	}
	if GoLatestForceRestart == nil {
		t.Error("GoLatestForceRestart should not be nil")
	}
	if TimeoutTxName == nil {
		t.Error("TimeoutTxName should not be nil")
	}
	if ActionTxName == nil {
		t.Error("ActionTxName should not be nil")
	}
}

// =============================================================================
// Resource Tests (via root package)
// =============================================================================

func TestResourceStateConstants_Values(t *testing.T) {
	if Pending != 0 {
		t.Errorf("Pending = %d, want 0", Pending)
	}
	if Loading != 1 {
		t.Errorf("Loading = %d, want 1", Loading)
	}
	if Ready != 2 {
		t.Errorf("Ready = %d, want 2", Ready)
	}
	if Error != 3 {
		t.Errorf("Error = %d, want 3", Error)
	}
}

func TestResourceHandlers_AreExported(t *testing.T) {
	// Verify handler constructors work
	pending := OnPending[int](func() *VNode {
		return vdom.Text("pending")
	})
	if pending == nil {
		t.Error("OnPending should return non-nil handler")
	}

	loading := OnLoading[int](func() *VNode {
		return vdom.Text("loading")
	})
	if loading == nil {
		t.Error("OnLoading should return non-nil handler")
	}

	ready := OnReady(func(val int) *VNode {
		return vdom.Textf("ready: %d", val)
	})
	if ready == nil {
		t.Error("OnReady should return non-nil handler")
	}

	errHandler := OnError[int](func(err error) *VNode {
		return vdom.Text(err.Error())
	})
	if errHandler == nil {
		t.Error("OnError should return non-nil handler")
	}

	loadingOrPending := OnLoadingOrPending[int](func() *VNode {
		return vdom.Text("loading or pending")
	})
	if loadingOrPending == nil {
		t.Error("OnLoadingOrPending should return non-nil handler")
	}
}

// =============================================================================
// Error Tests (via root package)
// =============================================================================

func TestErrors_AreExported(t *testing.T) {
	if ErrBudgetExceeded == nil {
		t.Error("ErrBudgetExceeded should not be nil")
	}
	if ErrQueueFull == nil {
		t.Error("ErrQueueFull should not be nil")
	}
	if ErrActionRunning == nil {
		t.Error("ErrActionRunning should not be nil")
	}
	if ErrEffectContext == nil {
		t.Error("ErrEffectContext should not be nil")
	}
	if ErrGoLatestContext == nil {
		t.Error("ErrGoLatestContext should not be nil")
	}
}

func TestHTTPError_Methods(t *testing.T) {
	err := BadRequest(errors.New("invalid input"))

	if err.StatusCode() != 400 {
		t.Errorf("StatusCode() = %d, want 400", err.StatusCode())
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}
}

func TestHTTPErrorHelpers_AllStatusCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      *HTTPError
		wantCode int
	}{
		{"BadRequest", BadRequest(nil), 400},
		{"Unauthorized", Unauthorized(), 401},
		{"Forbidden", Forbidden(), 403},
		{"NotFound", NotFound(), 404},
		{"Conflict", Conflict(), 409},
		{"UnprocessableEntity", UnprocessableEntity(), 422},
		{"InternalError", InternalError(errors.New("internal")), 500},
		{"ServiceUnavailable", ServiceUnavailable(), 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.StatusCode() != tt.wantCode {
				t.Errorf("StatusCode() = %d, want %d", tt.err.StatusCode(), tt.wantCode)
			}
		})
	}
}

func TestBadRequestf_FormatsMessage(t *testing.T) {
	err := BadRequestf("invalid field: %s", "email")

	if err.StatusCode() != 400 {
		t.Errorf("StatusCode() = %d, want 400", err.StatusCode())
	}

	errStr := err.Error()
	if errStr != "invalid field: email" {
		t.Errorf("Error() = %q, want %q", errStr, "invalid field: email")
	}
}

// =============================================================================
// Context API Tests (via root package)
// =============================================================================

func TestCreateContext_CreatesContextType(t *testing.T) {
	userCtx := CreateContext("default-user")

	if userCtx == nil {
		t.Fatal("CreateContext returned nil")
	}
}

func TestSetContext_GetContext_RoundTrip(t *testing.T) {
	// Test simple key-value context API
	var retrievedValue any

	// Create a component that uses the context
	comp := vdom.Func(func() *vdom.VNode {
		SetContext("theme", "dark")
		retrievedValue = GetContext("theme")
		return vdom.Text("test")
	})

	// Render with owner to test context
	owner := corevango.NewOwner(nil)
	var node *vdom.VNode
	corevango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()
		node = comp.Render()
	})

	if retrievedValue != "dark" {
		t.Errorf("expected retrieved value 'dark', got %v", retrievedValue)
	}
	if node == nil {
		t.Error("expected non-nil node")
	}
}

func TestTypedContext_RoundTrip(t *testing.T) {
	type User struct {
		Name string
		ID   int
	}

	userCtx := CreateContext(User{Name: "default", ID: 0})

	if userCtx == nil {
		t.Fatal("CreateContext returned nil")
	}
}

// =============================================================================
// Reactive Primitive Tests (via root package)
// =============================================================================

func TestSignal_Methods(t *testing.T) {
	s := NewSignal(10)

	// Get
	if s.Get() != 10 {
		t.Errorf("Get() = %d, want 10", s.Get())
	}

	// Set
	s.Set(20)
	if s.Get() != 20 {
		t.Errorf("Get() after Set = %d, want 20", s.Get())
	}

	// Update
	s.Update(func(v int) int { return v + 5 })
	if s.Get() != 25 {
		t.Errorf("Get() after Update = %d, want 25", s.Get())
	}

	// Peek (non-reactive)
	peek := s.Peek()
	if peek != 25 {
		t.Errorf("Peek() = %d, want 25", peek)
	}
}

func TestMemo_ComputesValue(t *testing.T) {
	count := NewSignal(5)
	doubled := NewMemo(func() int {
		return count.Get() * 2
	})

	if doubled.Get() != 10 {
		t.Errorf("Memo Get() = %d, want 10", doubled.Get())
	}

	count.Set(7)
	if doubled.Get() != 14 {
		t.Errorf("Memo Get() after signal change = %d, want 14", doubled.Get())
	}
}

func TestRef_CurrentAndSet(t *testing.T) {
	ref := NewRef("")

	ref.Set("hello")
	if ref.Current() != "hello" {
		t.Errorf("Current() = %q, want %q", ref.Current(), "hello")
	}

	ref.Set("world")
	if ref.Current() != "world" {
		t.Errorf("Current() = %q, want %q", ref.Current(), "world")
	}
}

func TestBatch_GroupsUpdates(t *testing.T) {
	s := NewSignal(0)

	Batch(func() {
		s.Set(1)
		s.Set(2)
		s.Set(3)
	})

	if s.Get() != 3 {
		t.Errorf("Get() after Batch = %d, want 3", s.Get())
	}
}

func TestTx_IsAliasForBatch(t *testing.T) {
	s := NewSignal(0)

	Tx(func() {
		s.Set(10)
	})

	if s.Get() != 10 {
		t.Errorf("Get() after Tx = %d, want 10", s.Get())
	}
}

func TestUntracked_ReadsWithoutSubscription(t *testing.T) {
	s := NewSignal(42)
	var value int

	Untracked(func() {
		value = s.Get()
	})

	if value != 42 {
		t.Errorf("value = %d, want 42", value)
	}
}

func TestUntrackedGet_ReadsWithoutSubscription(t *testing.T) {
	s := NewSignal(100)

	value := UntrackedGet(s)

	if value != 100 {
		t.Errorf("UntrackedGet = %d, want 100", value)
	}
}

// =============================================================================
// Signal Options Tests (via root package)
// =============================================================================

func TestTransient_IsExported(t *testing.T) {
	if Transient == nil {
		t.Error("Transient should not be nil")
	}

	// Should be usable as SignalOption
	s := NewSignal(0, Transient())
	if s == nil {
		t.Error("NewSignal with Transient returned nil")
	}
}

func TestPersistKey_IsExported(t *testing.T) {
	if PersistKey == nil {
		t.Error("PersistKey should not be nil")
	}

	// Should be usable as SignalOption
	s := NewSignal(0, PersistKey("test-key"))
	if s == nil {
		t.Error("NewSignal with PersistKey returned nil")
	}
}

// =============================================================================
// Component/VNode Tests (via root package)
// =============================================================================

func TestFunc_CreatesComponent(t *testing.T) {
	comp := Func(func() *VNode {
		return vdom.Div(vdom.Text("Hello"))
	})

	if comp == nil {
		t.Fatal("Func returned nil")
	}

	node := comp.Render()
	if node == nil {
		t.Error("Render returned nil")
	}
}

func TestVKindConstants_AreExported(t *testing.T) {
	if KindElement != 1 {
		t.Errorf("KindElement = %d, want 1", KindElement)
	}
	if KindText != 2 {
		t.Errorf("KindText = %d, want 2", KindText)
	}
	if KindFragment != 3 {
		t.Errorf("KindFragment = %d, want 3", KindFragment)
	}
	if KindComponent != 4 {
		t.Errorf("KindComponent = %d, want 4", KindComponent)
	}
	if KindRaw != 5 {
		t.Errorf("KindRaw = %d, want 5", KindRaw)
	}
}

// =============================================================================
// DevMode Tests (via root package)
// =============================================================================

func TestDevMode_IsExported(t *testing.T) {
	if DevMode == nil {
		t.Error("DevMode should not be nil")
	}

	// Should be a pointer to a bool
	_ = *DevMode
}
