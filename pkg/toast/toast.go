package toast

import "github.com/vango-go/vango/pkg/server"

// EventName is the event name dispatched for toasts.
// Client-side code should listen for this event.
const EventName = "vango:toast"

// Type represents the toast notification type.
type Type string

const (
	TypeSuccess Type = "success"
	TypeError   Type = "error"
	TypeWarning Type = "warning"
	TypeInfo    Type = "info"
)

// Show displays a toast notification to the user.
// Uses ctx.Emit to send a custom event to the client.
//
// The client receives a CustomEvent with:
//   - event.type = "vango:toast"
//   - event.detail = { level: "success|error|warning|info", message: "..." }
func Show(ctx server.Ctx, level Type, message string) {
	ctx.Emit(EventName, map[string]any{
		"level":   string(level),
		"message": message,
	})
}

// Success shows a success toast.
//
//	toast.Success(ctx, "Changes saved!")
func Success(ctx server.Ctx, message string) {
	Show(ctx, TypeSuccess, message)
}

// Error shows an error toast.
//
//	toast.Error(ctx, "Failed to delete item")
func Error(ctx server.Ctx, message string) {
	Show(ctx, TypeError, message)
}

// Warning shows a warning toast.
//
//	toast.Warning(ctx, "This action cannot be undone")
func Warning(ctx server.Ctx, message string) {
	Show(ctx, TypeWarning, message)
}

// Info shows an info toast.
//
//	toast.Info(ctx, "New features available")
func Info(ctx server.Ctx, message string) {
	Show(ctx, TypeInfo, message)
}

// WithTitle shows a toast with a title and message.
//
//	toast.WithTitle(ctx, toast.TypeSuccess, "Settings", "Your changes have been saved.")
func WithTitle(ctx server.Ctx, level Type, title, message string) {
	ctx.Emit(EventName, map[string]any{
		"level":   string(level),
		"title":   title,
		"message": message,
	})
}

// WithAction shows a toast with an action button.
//
//	toast.WithAction(ctx, toast.TypeInfo, "Undo available", "Click to undo", "undo")
func WithAction(ctx server.Ctx, level Type, message, actionLabel, actionID string) {
	ctx.Emit(EventName, map[string]any{
		"level":       string(level),
		"message":     message,
		"actionLabel": actionLabel,
		"actionID":    actionID,
	})
}

// Custom shows a toast with custom data.
// Use this for advanced toast configurations.
func Custom(ctx server.Ctx, data map[string]any) {
	ctx.Emit(EventName, data)
}
