# VangoUI Component Library

VangoUI is a component library built on Vango using the **Copy-Paste Ownership** model. Components are added to your project via `vango add` and become part of your codebase.

## Installation

```bash
vango add button
vango add dialog card input
```

Components are copied to `app/components/ui/`.

## Architecture

VangoUI uses a **Generic Option Pattern** for type-safe, consistent APIs:

```go
import "myapp/app/components/ui"

ui.Button(
    ui.Variant(ui.ButtonVariantPrimary),
    ui.Size(ui.ButtonSizeLg),
    ui.Class[*ui.ButtonConfig]("my-custom-class"),
    ui.Child[*ui.ButtonConfig](vdom.Text("Click me")),
)
```

## Core Patterns

### Typed Enums (No Magic Strings)

```go
type ButtonVariant string
const (
    ButtonVariantDefault   ButtonVariant = "default"
    ButtonVariantPrimary   ButtonVariant = "primary"
    ButtonVariantDestructive ButtonVariant = "destructive"
)
```

### Generic Base Options

All components support:
- `Class[T]("...")` — Add CSS classes
- `Attr[T](attr)` — Pass raw attributes
- `Child[T](nodes...)` — Add children

### Component-Specific Options

Each component has typed options:
```go
ui.Variant(ui.ButtonVariantPrimary)
ui.Size(ui.ButtonSizeSm)
ui.DialogOpen(openSignal)
```

## Available Components

| Component | Options | Interactive |
|-----------|---------|-------------|
| `Button` | Variant, Size | No |
| `Card` | - | No |
| `Input` | Type, Placeholder | No |
| `Label` | For | No |
| `Dialog` | Open, OnClose, CloseOnEscape | Yes (Hook) |
| `Kanban` | Columns, OnReorder | Yes (Hook) |

## Interactive Components

Interactive components use **Client Hooks** for 60fps interactions:

```go
open := vango.Signal(false)

ui.Dialog(
    ui.DialogOpen(&open),
    ui.DialogCloseOnEscape(true),
    ui.Child[*ui.DialogConfig](
        vdom.H2(vdom.Text("Title")),
        vdom.P(vdom.Text("Content")),
    ),
)
```

The Dialog hook handles:
- Focus trapping
- Escape key to close
- Click-outside to close
- Animations

## Styling

Components use **Tailwind CSS tokens** (not hardcoded colors):

```css
/* Components reference semantic tokens */
bg-primary        /* var(--primary) */
text-foreground   /* var(--foreground) */
border-input      /* var(--input) */
```

Customize by editing CSS variables in your stylesheet.

## Ownership Model

After `vango add`, you own the code:
- Modify components freely
- No external dependencies
- Upgrade selectively via `vango upgrade`

Files include version headers for diff-based upgrades:
```go
// Source: vango.dev/ui/button
// Version: 1.0.2
```
