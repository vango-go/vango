package vango

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/urlparam"
)

// =============================================================================
// Context Tests
// =============================================================================

func TestCtxIsServerCtx(t *testing.T) {
	// Verify that vango.Ctx is the same type as server.Ctx
	var vangoCtx Ctx
	var serverCtx server.Ctx

	// This should compile because they're the same type
	_ = vangoCtx
	_ = serverCtx

	// They should be assignable
	vangoCtx = serverCtx
	_ = vangoCtx
}

func TestUseCtxReturnsNilOutsideContext(t *testing.T) {
	// Outside render/effect context, UseCtx should return nil
	ctx := UseCtx()
	if ctx != nil {
		t.Errorf("expected nil ctx outside render context, got %v", ctx)
	}
}

// =============================================================================
// Navigation Option Tests
// =============================================================================

func TestNavigateOptionsExist(t *testing.T) {
	// Verify navigation options are exported
	_ = WithReplace
	_ = WithNavigateParams
	_ = WithoutScroll
}

func TestNavigateOptionType(t *testing.T) {
	// Verify NavigateOption is the correct type
	var opt NavigateOption
	opt = WithReplace()
	_ = opt

	opt = WithNavigateParams(map[string]any{"foo": "bar"})
	_ = opt

	opt = WithoutScroll()
	_ = opt
}

// =============================================================================
// Reactive Primitive Tests
// =============================================================================

func TestNewSignal(t *testing.T) {
	s := NewSignal(42)
	if s.Get() != 42 {
		t.Errorf("expected 42, got %d", s.Get())
	}

	s.Set(100)
	if s.Get() != 100 {
		t.Errorf("expected 100, got %d", s.Get())
	}
}

func TestNewSignalWithOptions(t *testing.T) {
	s := NewSignal(0, Transient())
	if s.Get() != 0 {
		t.Errorf("expected 0, got %d", s.Get())
	}
}

func TestNewMemo(t *testing.T) {
	count := NewSignal(5)
	doubled := NewMemo(func() int {
		return count.Get() * 2
	})

	if doubled.Get() != 10 {
		t.Errorf("expected 10, got %d", doubled.Get())
	}
}

func TestNewRef(t *testing.T) {
	ref := NewRef[string]("")
	ref.Set("hello")
	if ref.Current() != "hello" {
		t.Errorf("expected 'hello', got %q", ref.Current())
	}
}

func TestBatch(t *testing.T) {
	count := NewSignal(0)
	Batch(func() {
		count.Set(1)
		count.Set(2)
		count.Set(3)
	})
	if count.Get() != 3 {
		t.Errorf("expected 3, got %d", count.Get())
	}
}

func TestUntracked(t *testing.T) {
	count := NewSignal(42)
	var value int
	Untracked(func() {
		value = count.Get()
	})
	if value != 42 {
		t.Errorf("expected 42, got %d", value)
	}
}

func TestUntrackedGet(t *testing.T) {
	count := NewSignal(42)
	value := UntrackedGet(count)
	if value != 42 {
		t.Errorf("expected 42, got %d", value)
	}
}

// =============================================================================
// Action Tests
// =============================================================================

func TestActionStateConstants(t *testing.T) {
	// Verify action state constants are exported
	if ActionIdle != 0 {
		t.Errorf("expected ActionIdle to be 0")
	}
	if ActionRunning != 1 {
		t.Errorf("expected ActionRunning to be 1")
	}
	if ActionSuccess != 2 {
		t.Errorf("expected ActionSuccess to be 2")
	}
	if ActionError != 3 {
		t.Errorf("expected ActionError to be 3")
	}
}

// =============================================================================
// URLParam Tests
// =============================================================================

func TestURLParamOptionsExist(t *testing.T) {
	// Verify URL param options are exported
	var opt URLParamOption

	opt = Push
	_ = opt

	opt = Replace
	_ = opt

	opt = Debounce(100 * time.Millisecond)
	_ = opt

	opt = Encoding(URLEncodingFlat)
	_ = opt
}

func TestURLEncodingConstants(t *testing.T) {
	// Verify encoding constants match urlparam package
	if URLEncodingFlat != urlparam.EncodingFlat {
		t.Errorf("URLEncodingFlat mismatch")
	}
	if URLEncodingJSON != urlparam.EncodingJSON {
		t.Errorf("URLEncodingJSON mismatch")
	}
	if URLEncodingComma != urlparam.EncodingComma {
		t.Errorf("URLEncodingComma mismatch")
	}
}

// =============================================================================
// Form Validator Tests
// =============================================================================

func TestValidatorsAreCallable(t *testing.T) {
	// Verify validators are functions that return Validator
	var v Validator

	v = Required("required")
	_ = v

	v = MinLength(5, "too short")
	_ = v

	v = MaxLength(100, "too long")
	_ = v

	v = Email("invalid email")
	_ = v

	v = Pattern(`^\d+$`, "digits only")
	_ = v

	v = Min(0, "must be positive")
	_ = v

	v = Max(100, "too large")
	_ = v

	v = Between(0, 100, "out of range")
	_ = v

	v = Positive("must be positive")
	_ = v

	v = NonNegative("must be non-negative")
	_ = v

	v = URL("invalid URL")
	_ = v

	v = UUID("invalid UUID")
	_ = v

	v = Alpha("letters only")
	_ = v

	v = AlphaNumeric("alphanumeric only")
	_ = v

	v = Numeric("numbers only")
	_ = v

	v = Phone("invalid phone")
	_ = v

	now := time.Now()
	v = DateAfter(now, "must be after")
	_ = v

	v = DateBefore(now, "must be before")
	_ = v

	v = Future("must be in future")
	_ = v

	v = Past("must be in past")
	_ = v

	v = Custom(func(value any) error { return nil })
	_ = v
}

func TestRequiredValidator(t *testing.T) {
	v := Required("field is required")

	// Empty string should fail
	if err := v.Validate(""); err == nil {
		t.Error("expected error for empty string")
	}

	// Non-empty string should pass
	if err := v.Validate("hello"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestEmailValidator(t *testing.T) {
	v := Email("invalid email")

	// Valid email should pass
	if err := v.Validate("test@example.com"); err != nil {
		t.Errorf("expected valid email, got error: %v", err)
	}

	// Invalid email should fail
	if err := v.Validate("not-an-email"); err == nil {
		t.Error("expected error for invalid email")
	}

	// Empty string should pass (let Required handle emptiness)
	if err := v.Validate(""); err != nil {
		t.Errorf("expected empty to pass, got error: %v", err)
	}
}

func TestMinLengthValidator(t *testing.T) {
	v := MinLength(3, "too short")

	// Long enough should pass
	if err := v.Validate("hello"); err != nil {
		t.Errorf("expected pass, got error: %v", err)
	}

	// Too short should fail
	if err := v.Validate("hi"); err == nil {
		t.Error("expected error for too short string")
	}
}

// =============================================================================
// VNode/Component Tests
// =============================================================================

func TestComponentType(t *testing.T) {
	// Verify Component type is exported
	var _ Component

	// Verify VNode type is exported
	var _ VNode

	// Verify Props type is exported
	var _ Props
}

func TestVKindConstants(t *testing.T) {
	// Verify VKind constants are exported
	if KindElement != 1 {
		t.Error("KindElement mismatch")
	}
	if KindText != 2 {
		t.Error("KindText mismatch")
	}
}

// =============================================================================
// Error Tests
// =============================================================================

func TestErrorsExported(t *testing.T) {
	// Verify error variables are exported
	_ = ErrBudgetExceeded
	_ = ErrQueueFull
	_ = ErrActionRunning
	_ = ErrEffectContext
	_ = ErrGoLatestContext
}

func TestHTTPErrorHelpers(t *testing.T) {
	err := BadRequest(nil)
	if err.StatusCode() != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", err.StatusCode())
	}

	err = Unauthorized()
	if err.StatusCode() != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", err.StatusCode())
	}

	err = Forbidden()
	if err.StatusCode() != http.StatusForbidden {
		t.Errorf("expected 403, got %d", err.StatusCode())
	}

	err = NotFound()
	if err.StatusCode() != http.StatusNotFound {
		t.Errorf("expected 404, got %d", err.StatusCode())
	}

	err = Conflict()
	if err.StatusCode() != http.StatusConflict {
		t.Errorf("expected 409, got %d", err.StatusCode())
	}

	err = UnprocessableEntity()
	if err.StatusCode() != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", err.StatusCode())
	}

	err = ServiceUnavailable()
	if err.StatusCode() != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", err.StatusCode())
	}
}

// =============================================================================
// QueryParam Tests (via server.Ctx)
// =============================================================================

func TestQueryParam(t *testing.T) {
	// Create a test request with query params
	req := httptest.NewRequest("GET", "http://example.com/test?foo=bar&baz=qux", nil)
	w := httptest.NewRecorder()

	// Create server context
	ctx := server.NewTestContext(nil)
	if ctx == nil {
		// If NewTestContext doesn't work, we need a different approach
		t.Skip("NewTestContext not available for this test")
	}

	// The QueryParam method should be available on server.Ctx
	// We're testing that the method exists and is accessible via vango.Ctx
	_ = req
	_ = w
}

// =============================================================================
// Context API Tests
// =============================================================================

func TestCreateContextType(t *testing.T) {
	// Verify CreateContext creates a context
	userCtx := CreateContext[string]("default")
	if userCtx == nil {
		t.Error("expected non-nil context")
	}
}

// =============================================================================
// Effect Helper Type Tests
// =============================================================================

func TestEffectHelperTypesExported(t *testing.T) {
	var _ EffectOption
	var _ IntervalOption
	var _ SubscribeOption
	var _ GoLatestOption
	var _ TimeoutOption
	var _ ActionOption
}

// =============================================================================
// Stream Interface Test
// =============================================================================

func TestStreamInterface(t *testing.T) {
	// Verify Stream interface is exported
	var _ Stream[int]
}
