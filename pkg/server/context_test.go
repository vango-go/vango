package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/vango-dev/vango/v2/pkg/protocol"
)

func TestNewCtx(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()
	logger := slog.Default()

	c := newCtx(w, req, logger)

	if c == nil {
		t.Fatal("newCtx should not return nil")
	}
	if c.Request() != req {
		t.Error("Request should match")
	}
	if c.Path() != "/test" {
		t.Errorf("Path = %s, want /test", c.Path())
	}
	if c.Method() != "GET" {
		t.Errorf("Method = %s, want GET", c.Method())
	}
}

func TestCtxPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"/", "/"},
		{"/users", "/users"},
		{"/users/123", "/users/123"},
		{"/api/v1/data?query=test", "/api/v1/data"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.url, nil)
		c := newCtx(httptest.NewRecorder(), req, slog.Default())
		if c.Path() != tt.want {
			t.Errorf("Path(%s) = %s, want %s", tt.url, c.Path(), tt.want)
		}
	}
}

func TestCtxMethod(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/", nil)
		c := newCtx(httptest.NewRecorder(), req, slog.Default())
		if c.Method() != method {
			t.Errorf("Method = %s, want %s", c.Method(), method)
		}
	}
}

func TestCtxQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?foo=bar&count=5&multi=a&multi=b", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	query := c.Query()
	if query.Get("foo") != "bar" {
		t.Errorf("Query(foo) = %s, want bar", query.Get("foo"))
	}
	if query.Get("count") != "5" {
		t.Errorf("Query(count) = %s, want 5", query.Get("count"))
	}
	if len(query["multi"]) != 2 {
		t.Errorf("Query(multi) length = %d, want 2", len(query["multi"]))
	}
}

func TestCtxParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	c.setParams(map[string]string{
		"id":   "123",
		"name": "test",
	})

	if c.Param("id") != "123" {
		t.Errorf("Param(id) = %s, want 123", c.Param("id"))
	}
	if c.Param("name") != "test" {
		t.Errorf("Param(name) = %s, want test", c.Param("name"))
	}
	if c.Param("missing") != "" {
		t.Errorf("Param(missing) = %s, want empty", c.Param("missing"))
	}
}

func TestCtxHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("Authorization", "Bearer token123")

	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	if c.Header("X-Custom-Header") != "custom-value" {
		t.Errorf("Header(X-Custom-Header) = %s, want custom-value", c.Header("X-Custom-Header"))
	}
	if c.Header("Authorization") != "Bearer token123" {
		t.Errorf("Header(Authorization) = %s, want Bearer token123", c.Header("Authorization"))
	}
	if c.Header("Missing") != "" {
		t.Errorf("Header(Missing) = %s, want empty", c.Header("Missing"))
	}
}

func TestCtxCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc123"})

	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	cookie, err := c.Cookie("session_id")
	if err != nil {
		t.Fatalf("Cookie error: %v", err)
	}
	if cookie.Value != "abc123" {
		t.Errorf("Cookie value = %s, want abc123", cookie.Value)
	}

	_, err = c.Cookie("missing")
	if err == nil {
		t.Error("Cookie(missing) should return error")
	}
}

func TestCtxStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Default status
	if c.getStatus() != http.StatusOK {
		t.Errorf("Default status = %d, want %d", c.getStatus(), http.StatusOK)
	}

	// Set status
	c.Status(http.StatusCreated)
	if c.getStatus() != http.StatusCreated {
		t.Errorf("Status = %d, want %d", c.getStatus(), http.StatusCreated)
	}

	c.Status(http.StatusNotFound)
	if c.getStatus() != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", c.getStatus(), http.StatusNotFound)
	}
}

func TestCtxRedirect(t *testing.T) {
	req := httptest.NewRequest("GET", "/old", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, req, slog.Default())

	c.Redirect("/new", http.StatusMovedPermanently)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("Redirect status = %d, want %d", w.Code, http.StatusMovedPermanently)
	}
	if w.Header().Get("Location") != "/new" {
		t.Errorf("Redirect location = %s, want /new", w.Header().Get("Location"))
	}
	if !c.isWritten() {
		t.Error("isWritten should be true after redirect")
	}
}

func TestCtxSetHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, req, slog.Default())

	c.SetHeader("X-Custom", "value1")
	c.SetHeader("X-Another", "value2")

	if w.Header().Get("X-Custom") != "value1" {
		t.Error("SetHeader should set response header")
	}
	if w.Header().Get("X-Another") != "value2" {
		t.Error("SetHeader should set response header")
	}
}

func TestCtxSetCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, req, slog.Default())

	c.SetCookie(&http.Cookie{
		Name:  "session",
		Value: "xyz789",
	})

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "session" || cookies[0].Value != "xyz789" {
		t.Error("SetCookie should set response cookie")
	}
}

func TestCtxSession(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Initially nil
	if c.Session() != nil {
		t.Error("Session should be nil initially")
	}

	// Create mock session
	session := &Session{ID: "test-session"}
	c.setSession(session)

	if c.Session() != session {
		t.Error("Session should match after setSession")
	}
}

func TestCtxUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Initially nil
	if c.User() != nil {
		t.Error("User should be nil initially")
	}

	// Set user
	user := map[string]string{"id": "123", "name": "Test"}
	c.SetUser(user)

	if c.User() == nil {
		t.Error("User should not be nil after SetUser")
	}
	u := c.User().(map[string]string)
	if u["id"] != "123" {
		t.Error("User should match")
	}
}

func TestCtxLogger(t *testing.T) {
	logger := slog.Default()
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, logger)

	if c.Logger() != logger {
		t.Error("Logger should match")
	}
}

func TestCtxDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	done := c.Done()
	select {
	case <-done:
		t.Error("Done should not be closed yet")
	default:
		// Good
	}

	cancel()

	select {
	case <-done:
		// Good - should be closed now
	case <-time.After(100 * time.Millisecond):
		t.Error("Done should be closed after cancel")
	}
}

func TestCtxSetValueAndValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Initially nil
	if c.Value("key") != nil {
		t.Error("Value should be nil for unset key")
	}

	// Set values
	c.SetValue("string", "hello")
	c.SetValue("int", 42)
	c.SetValue("struct", struct{ Name string }{"test"})

	if c.Value("string") != "hello" {
		t.Error("Value(string) mismatch")
	}
	if c.Value("int") != 42 {
		t.Error("Value(int) mismatch")
	}
	if c.Value("missing") != nil {
		t.Error("Value(missing) should be nil")
	}
}

func TestCtxStdContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Default - returns request context
	stdCtx := c.StdContext()
	if stdCtx == nil {
		t.Error("StdContext should not be nil")
	}

	// With custom context
	customCtx := context.WithValue(context.Background(), "trace_id", "abc123")
	c2 := c.WithStdContext(customCtx)

	stdCtx2 := c2.StdContext()
	if stdCtx2.Value("trace_id") != "abc123" {
		t.Error("StdContext should return custom context")
	}
}

func TestCtxWithStdContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	customCtx := context.WithValue(context.Background(), "key", "value")
	c2 := c.WithStdContext(customCtx)

	// Original unchanged
	if c.StdContext().Value("key") != nil {
		t.Error("Original context should be unchanged")
	}

	// New context has value
	if c2.StdContext().Value("key") != "value" {
		t.Error("New context should have value")
	}
}

func TestCtxEvent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Initially nil
	if c.Event() != nil {
		t.Error("Event should be nil initially")
	}

	// Set event
	event := &Event{HID: "h1", Type: protocol.EventClick}
	c.setEvent(event)

	if c.Event() != event {
		t.Error("Event should match after setEvent")
	}
}

func TestCtxPatchCount(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Initially 0
	if c.PatchCount() != 0 {
		t.Error("PatchCount should be 0 initially")
	}

	// Add patches
	c.AddPatchCount(5)
	if c.PatchCount() != 5 {
		t.Errorf("PatchCount = %d, want 5", c.PatchCount())
	}

	c.AddPatchCount(3)
	if c.PatchCount() != 8 {
		t.Errorf("PatchCount = %d, want 8", c.PatchCount())
	}
}

func TestCtxDispatchWithSession(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Create a mock session with dispatch channel
	session := &Session{
		ID:         "test",
		dispatchCh: make(chan func(), 10),
		done:       make(chan struct{}),
	}
	c.setSession(session)

	// Dispatch should queue the function
	called := false
	c.Dispatch(func() {
		called = true
	})

	// Function should be in the channel
	select {
	case fn := <-session.dispatchCh:
		fn()
		if !called {
			t.Error("Dispatched function should be callable")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Dispatch should queue function")
	}
}

func TestCtxDispatchWithoutSession(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// No session - should execute inline
	called := false
	c.Dispatch(func() {
		called = true
	})

	if !called {
		t.Error("Dispatch without session should execute inline")
	}
}

func TestCtxDispatchConcurrent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	session := &Session{
		ID:         "test",
		dispatchCh: make(chan func(), 100),
		done:       make(chan struct{}),
	}
	c.setSession(session)

	var wg sync.WaitGroup

	// Dispatch from multiple goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Dispatch(func() {
				// Just a no-op function
			})
		}()
	}

	wg.Wait()

	// Drain and count dispatched functions
	drained := 0
	for {
		select {
		case fn := <-session.dispatchCh:
			fn()
			drained++
		default:
			goto done
		}
	}
done:

	if drained != 50 {
		t.Errorf("Dispatched %d functions, want 50", drained)
	}
}

func TestNewTestContext(t *testing.T) {
	session := &Session{ID: "test-session"}
	c := NewTestContext(session)

	if c == nil {
		t.Fatal("NewTestContext should not return nil")
	}
	if c.Session() != session {
		t.Error("Session should match")
	}
	if c.StdContext() == nil {
		t.Error("StdContext should not be nil")
	}
}

func TestCtxResponseWriter(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, req, slog.Default())

	rw := c.ResponseWriter()
	if rw != w {
		t.Error("ResponseWriter should return the underlying writer")
	}
}

func TestCtxWithLogger(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	newLogger := slog.Default().With("custom", "field")
	c2 := c.WithLogger(newLogger)

	if c2.Logger() != newLogger {
		t.Error("WithLogger should return new context with updated logger")
	}
	// Original unchanged
	if c.Logger() == newLogger {
		t.Error("Original context logger should be unchanged")
	}
}

func TestCtxWriteStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	c := newCtx(w, req, slog.Default())

	c.Status(http.StatusCreated)
	c.writeStatus()

	if w.Code != http.StatusCreated {
		t.Errorf("writeStatus should write status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestEncodeJSON(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{"hello", `"hello"`},
		{123, "123"},
		{true, "true"},
		{[]int{1, 2, 3}, "[1,2,3]"},
		{map[string]int{"a": 1}, `{"a":1}`},
	}

	for _, tt := range tests {
		got, err := encodeJSON(tt.input)
		if err != nil {
			t.Errorf("encodeJSON(%v) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("encodeJSON(%v) = %s, want %s", tt.input, got, tt.want)
		}
	}
}

// Test for ctx implementing the vango.Ctx interface requirements
func TestCtxImplementsVangoCtx(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	session := &Session{
		ID:         "test",
		dispatchCh: make(chan func(), 10),
		done:       make(chan struct{}),
	}
	c.setSession(session)

	// Verify Dispatch is callable
	c.Dispatch(func() {})

	// Verify StdContext returns valid context
	stdCtx := c.StdContext()
	if stdCtx == nil {
		t.Error("StdContext should return valid context")
	}
}

func TestCtxQueryParsing(t *testing.T) {
	// Test URL encoding
	req := httptest.NewRequest("GET", "/search?q=hello+world&filter=a%26b", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	query := c.Query()
	if query.Get("q") != "hello world" {
		t.Errorf("Query(q) = %s, want 'hello world'", query.Get("q"))
	}
	if query.Get("filter") != "a&b" {
		t.Errorf("Query(filter) = %s, want 'a&b'", query.Get("filter"))
	}
}

func TestCtxEmptyQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	query := c.Query()
	if query == nil {
		t.Error("Query should not be nil for empty query string")
	}
	if len(query) != 0 {
		t.Error("Query should be empty")
	}
}

func TestCtxParamDefault(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	c := newCtx(httptest.NewRecorder(), req, slog.Default())

	// Without setParams, params map should be empty but not nil
	if c.Param("any") != "" {
		t.Error("Param should return empty string for missing key")
	}
}

func TestCtxGetQueryURL(t *testing.T) {
	u, _ := url.Parse("/test?a=1&b=2")
	req := &http.Request{URL: u}
	c := &ctx{request: req}

	if c.Query().Get("a") != "1" {
		t.Error("Query should work with minimal request")
	}
}
