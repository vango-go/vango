# Toast Notifications

Show feedback to users without page reloads or flash cookies.

## Basic Usage

```go
import "github.com/vango-dev/vango/v2/pkg/toast"

func SaveProject(ctx vango.Ctx, input ProjectInput) error {
    if err := db.Projects.Create(input); err != nil {
        toast.Error(ctx, "Failed to save project")
        return err
    }
    
    toast.Success(ctx, "Project saved!")
    return nil
}
```

## Toast Types

```go
toast.Success(ctx, "Item created")
toast.Error(ctx, "Something went wrong")
toast.Warning(ctx, "This action cannot be undone")
toast.Info(ctx, "FYI: New features available")
```

## With Title

```go
toast.WithTitle(ctx, toast.TypeSuccess, "Settings", "Your preferences have been saved")
```

## With Action

```go
toast.WithAction(ctx, toast.TypeInfo, "Item deleted", "Undo", "undo-item-123")
```

## Custom Toast

```go
toast.Custom(ctx, map[string]any{
    "level":    "success",
    "message":  "Custom toast",
    "duration": 5000,
    "position": "top-right",
})
```

## Client-Side Setup

Toasts use `ctx.Emit()` which dispatches a custom event. Add a listener in your app:

```javascript
// Using Sonner
window.addEventListener("vango:toast", (e) => {
    const { level, message, title } = e.detail;
    toast[level](message, { description: title });
});

// Using Toastify
window.addEventListener("vango:toast", (e) => {
    Toastify({ text: e.detail.message, className: e.detail.level }).showToast();
});

// Vanilla JS
window.addEventListener("vango:toast", (e) => {
    showNotification(e.detail);
});
```
