# Hooks Reference

Client Hooks provide 60fps interactions while keeping state on the server.

## Using Hooks

```go
Div(
    Hook("Sortable", map[string]any{
        "animation": 150,
        "group":     "items",
    }),
    OnEvent("reorder", func(e vango.HookEvent) {
        from := e.Int("fromIndex")
        to := e.Int("toIndex")
        db.Items.Reorder(from, to)
    }),
    // children...
)
```

## Standard Hooks

| Hook | Purpose | Events |
|------|---------|--------|
| `Sortable` | Drag-to-reorder lists | `reorder` |
| `Draggable` | Free-form dragging | `dragend` |
| `Droppable` | Drop zones | `drop` |
| `Resizable` | Resize handles | `resize` |
| `Tooltip` | Hover tooltips | (visual only) |
| `Dropdown` | Click-outside-to-close | `close` |
| `Collapsible` | Expand/collapse | `toggle` |

## HookEvent API

```go
e.String("key")    // string value
e.Int("key")       // int value
e.Float("key")     // float64 value
e.Bool("key")      // bool value
e.Strings("key")   // []string value
e.Revert()         // Revert visual change on error
```

## Custom Hooks

Create custom hooks in JavaScript:

```javascript
// public/js/hooks.js
export default {
    ColorPicker: {
        mounted(el, config, pushEvent) {
            this.picker = new Pickr({el, default: config.color});
            this.picker.on('change', color => {
                pushEvent('color-changed', {color: color.toHEXA()});
            });
        },
        destroyed(el) {
            this.picker.destroy();
        }
    }
}
```

Register in `vango.json`:
```json
{ "hooks": "./public/js/hooks.js" }
```

Use like standard hooks:
```go
Div(
    Hook("ColorPicker", map[string]any{"color": "#ff0000"}),
    OnEvent("color-changed", func(e vango.HookEvent) {
        setColor(e.String("color"))
    }),
)
```
