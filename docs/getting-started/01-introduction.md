# Introduction to Vango

Vango is a **server-driven web framework** for Go that lets you build interactive web applications without writing JavaScript.

## Why Vango?

Modern web development is fragmented:
- Two languages (JavaScript + backend)
- Two state systems (client + server)
- Heavy JavaScript bundles (200KB+)
- Complex toolchains

**Vango's approach:**
1. Components run on the server
2. UI updates flow as binary patches over WebSocket
3. The client is a tiny renderer (~12KB)
4. You write Go everywhere

## Hello World

```go
package main

import (
    "vango/pkg/vango"
    . "vango/pkg/vdom"
)

func main() {
    app := vango.New()
    app.Route("/", HomePage)
    app.Run(":3000")
}

func HomePage() *vango.VNode {
    return Div(
        H1(Text("Hello, Vango!")),
        P(Text("Your first server-driven app.")),
    )
}
```

## What's Next?

- [Quickstart](./02-quickstart.md) — Set up your first project
- [Tutorial](./03-tutorial.md) — Build a complete app step-by-step
