package toast_test

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/server"
	"github.com/vango-dev/vango/v2/pkg/toast"
)

// mockCtx implements server.Ctx for testing.
// It captures emitted events for verification.
type mockCtx struct {
	server.Ctx
	emittedEvents []emittedEvent
}

type emittedEvent struct {
	name string
	data any
}

func (m *mockCtx) Emit(name string, data any) {
	m.emittedEvents = append(m.emittedEvents, emittedEvent{name, data})
}

func (m *mockCtx) Session() *server.Session {
	return nil
}

func newMockCtx() *mockCtx {
	return &mockCtx{
		emittedEvents: make([]emittedEvent, 0),
	}
}

func TestSuccess(t *testing.T) {
	ctx := newMockCtx()

	toast.Success(ctx, "Item saved!")

	if len(ctx.emittedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(ctx.emittedEvents))
	}

	event := ctx.emittedEvents[0]
	if event.name != toast.EventName {
		t.Errorf("expected event name %q, got %q", toast.EventName, event.name)
	}

	data := event.data.(map[string]any)
	if data["level"] != "success" {
		t.Errorf("expected level success, got %v", data["level"])
	}
	if data["message"] != "Item saved!" {
		t.Errorf("expected message 'Item saved!', got %v", data["message"])
	}
}

func TestError(t *testing.T) {
	ctx := newMockCtx()

	toast.Error(ctx, "Something went wrong")

	if len(ctx.emittedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(ctx.emittedEvents))
	}

	data := ctx.emittedEvents[0].data.(map[string]any)
	if data["level"] != "error" {
		t.Errorf("expected level error, got %v", data["level"])
	}
}

func TestWarning(t *testing.T) {
	ctx := newMockCtx()

	toast.Warning(ctx, "Be careful!")

	data := ctx.emittedEvents[0].data.(map[string]any)
	if data["level"] != "warning" {
		t.Errorf("expected level warning, got %v", data["level"])
	}
}

func TestInfo(t *testing.T) {
	ctx := newMockCtx()

	toast.Info(ctx, "FYI")

	data := ctx.emittedEvents[0].data.(map[string]any)
	if data["level"] != "info" {
		t.Errorf("expected level info, got %v", data["level"])
	}
}

func TestWithTitle(t *testing.T) {
	ctx := newMockCtx()

	toast.WithTitle(ctx, toast.TypeSuccess, "Settings", "Changes saved")

	data := ctx.emittedEvents[0].data.(map[string]any)
	if data["level"] != "success" {
		t.Errorf("expected level success, got %v", data["level"])
	}
	if data["title"] != "Settings" {
		t.Errorf("expected title Settings, got %v", data["title"])
	}
	if data["message"] != "Changes saved" {
		t.Errorf("expected message 'Changes saved', got %v", data["message"])
	}
}

func TestWithAction(t *testing.T) {
	ctx := newMockCtx()

	toast.WithAction(ctx, toast.TypeInfo, "Item deleted", "Undo", "undo-123")

	data := ctx.emittedEvents[0].data.(map[string]any)
	if data["actionLabel"] != "Undo" {
		t.Errorf("expected actionLabel Undo, got %v", data["actionLabel"])
	}
	if data["actionID"] != "undo-123" {
		t.Errorf("expected actionID undo-123, got %v", data["actionID"])
	}
}

func TestCustom(t *testing.T) {
	ctx := newMockCtx()

	toast.Custom(ctx, map[string]any{
		"level":    "success",
		"message":  "Custom toast",
		"duration": 5000,
		"position": "top-right",
	})

	data := ctx.emittedEvents[0].data.(map[string]any)
	if data["duration"] != 5000 {
		t.Errorf("expected duration 5000, got %v", data["duration"])
	}
	if data["position"] != "top-right" {
		t.Errorf("expected position top-right, got %v", data["position"])
	}
}

func TestMultipleToasts(t *testing.T) {
	ctx := newMockCtx()

	toast.Success(ctx, "First")
	toast.Error(ctx, "Second")
	toast.Info(ctx, "Third")

	if len(ctx.emittedEvents) != 3 {
		t.Errorf("expected 3 events, got %d", len(ctx.emittedEvents))
	}
}
