# Quickstart

Get a Vango app running in under 5 minutes.

## Prerequisites

- Go 1.21+
- A terminal

## Installation

```bash
# Install the Vango CLI
go install vango.dev/cli/vango@latest

# Create a new project
vango create my-app
cd my-app

# Start the dev server
vango dev
```

Open `http://localhost:3000` in your browser.

## Project Structure

```
my-app/
├── app/
│   ├── routes/           # File-based routing
│   │   └── index.go      # → /
│   └── components/       # Shared components
├── public/               # Static assets
├── go.mod
└── vango.json            # Configuration
```

## Your First Component

Edit `app/routes/index.go`:

```go
package routes

import (
    "vango/pkg/vango"
    . "vango/pkg/vdom"
)

func Page() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(0)

        return Div(
            H1(Textf("Count: %d", count())),
            Button(OnClick(count.Inc), Text("+")),
            Button(OnClick(count.Dec), Text("-")),
        )
    })
}
```

Save the file—the browser updates instantly via hot reload.

## Add UI Components

Use VangoUI for pre-built, styled components:

```bash
vango add button card input
```

```go
import "myapp/app/components/ui"

ui.Button(
    ui.Variant(ui.ButtonVariantPrimary),
    ui.Child[*ui.ButtonConfig](vdom.Text("Save")),
)
```

See [UI Components Reference](../reference/08-ui-components.md) for the full library.

## Next Steps

- [Tutorial](./03-tutorial.md) — Build a Todo app
- [Concepts](../concepts/01-philosophy.md) — Understand how Vango works
