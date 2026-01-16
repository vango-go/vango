<p align="center">
  <img src="assets/vango-logo.svg" alt="Vango" width="120" />
</p>

<h1 align="center">Vango</h1>

<p align="center">
  <strong>Server-driven UI for Go.</strong><br>
  Build modern web apps with a single language, a single binary, and no client/server state synchronization.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/vango-go/vango"><img src="https://pkg.go.dev/badge/github.com/vango-go/vango.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/vango-go/vango"><img src="https://goreportcard.com/badge/github.com/vango-go/vango" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://discord.gg/vango"><img src="https://img.shields.io/discord/xxxxx?color=7389D8&label=discord" alt="Discord"></a>
</p>

<p align="center">
  <a href="https://vango.dev/docs">Documentation</a> •
  <a href="https://vango.dev/examples">Examples</a> •
  <a href="https://vango.cloud">Managed Hosting</a> •
  <a href="https://discord.gg/vango">Discord</a>
</p>

---
# WIP / NOT LAUNCHED

## Why Vango?

Solid's reactivity meets LiveView's architecture—in Go.

**Vango takes a different approach.** Your UI is a projection of server state. The browser runs a thin client (~12KB) that captures events, sends them to the server, and applies binary patches to the DOM. No hydration. No client-side state management. No synchronization bugs.

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)

        return Div(
            Button(OnClick(count.Dec), Text("-")),
            Span(Textf("%d", count.Get())),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}
```

That's a fully interactive counter. No JavaScript. No build step. Just Go.

## Features

- **Single Language** — Write your entire app in Go. Database queries, business logic, and UI in one place.
- **Single Binary** — Deploy a single executable. No Node.js, no npm, no webpack.
- **Reactive Signals** — Fine-grained reactivity with automatic dependency tracking.
- **Server-Side State** — No client/server sync. Server is the source of truth.
- **Instant Interactivity** — SSR for initial load, WebSocket for updates. SPA feel without SPA complexity.
- **Type-Safe Routing** — File-based routing with typed parameters.
- **Built-in Tailwind** — Zero-config CSS pipeline. No Node.js required.
- **Escape Hatches** — Hooks for client-side behavior, islands for third-party widgets, WASM when you need it.

## Quick Start

```bash
# Install the CLI
go install github.com/vango-go/vango/cmd/vango@latest

# Create a new project
vango create myapp
cd myapp

# Start developing
vango dev
```

Open [http://localhost:8080](http://localhost:8080) and start building.

## Examples

### Reactive State

Signals are mutable values that trigger UI updates when they change.

```go
func TodoApp() vango.Component {
    return vango.Func(func() *vango.VNode {
        todos := vango.NewSignal([]Todo{})
        input := vango.NewSignal("")

        return Div(
            Form(
                OnSubmit(vango.PreventDefault(func() {
                    if v := input.Get(); v != "" {
                        todos.Set(append(todos.Get(), Todo{Text: v}))
                        input.Set("")
                    }
                })),
                Input(Value(input.Get()), OnInput(input.Set)),
                Button(Type("submit"), Text("Add")),
            ),
            Ul(Range(todos.Get(), func(t Todo, i int) *vango.VNode {
                return Li(Key(i), Text(t.Text))
            })),
        )
    })
}
```

### Data Loading

Resources handle async data fetching with built-in loading and error states.

```go
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()

        user := vango.NewResource(func() (*User, error) {
            return db.Users.FindByID(ctx.StdContext(), userID)
        })

        return user.Match(
            vango.OnLoading(func() *vango.VNode {
                return Div(Text("Loading..."))
            }),
            vango.OnError(func(err error) *vango.VNode {
                return Div(Class("text-red-600"), Text(err.Error()))
            }),
            vango.OnReady(func(u *User) *vango.VNode {
                return Div(
                    H1(Text(u.Name)),
                    P(Text(u.Email)),
                )
            }),
        )
    })
}
```

### Mutations

Actions handle async mutations with explicit state and concurrency control.

```go
func SaveButton(data FormData) vango.Component {
    return vango.Func(func() *vango.VNode {
        save := vango.NewAction(
            func(ctx context.Context, d FormData) (*Result, error) {
                return api.Save(ctx, d)
            },
            vango.DropWhileRunning(), // Prevent double-submit
        )

        return Button(
            Disabled(save.State() == vango.ActionRunning),
            OnClick(func() { save.Run(data) }),
            Text(save.State() == vango.ActionRunning ? "Saving..." : "Save"),
        )
    })
}
```

### File-Based Routing

```
app/routes/
├── layout.go              # Root layout
├── index.go               # /
├── about.go               # /about
└── projects/
    ├── layout.go          # Nested layout
    ├── index.go           # /projects
    └── [id:int]/
        ├── index.go       # /projects/:id
        └── edit.go        # /projects/:id/edit
```

```go
// app/routes/projects/[id:int]/index.go
type Params struct {
    ID int `param:"id"`
}

func ProjectPage(ctx vango.Ctx, p Params) *vango.VNode {
    return ProjectView(p.ID)
}
```

## When to Use Vango

**Vango excels at:**
- Dashboards and admin panels
- Internal tools and B2B applications
- CRUD-heavy applications
- Real-time collaborative features
- Teams that want a single-language stack

**Consider alternatives for:**
- Offline-first applications
- Latency-critical interactions (<50ms required)
- Heavy client-side computation (use Vango's WASM escape hatch)

## Ecosystem

| Package | Description |
|---------|-------------|
| [vango-ui](https://github.com/vango-go/vango-ui) | ShadCN-style component library |
| [vango-clerk](https://github.com/vango-go/vango-clerk) | Clerk authentication adapter |
| [vango-auth0](https://github.com/vango-go/vango-auth0) | Auth0 authentication adapter |

## Deployment

### Managed Hosting

[Vango Cloud](https://vango.cloud) handles WebSocket infrastructure, session persistence, and scaling automatically.

```bash
vango deploy
```

### Self-Hosted

Deploy anywhere that runs Go. We provide guides for:
- [Fly.io](https://vango.dev/docs/deploy/fly)
- [Railway](https://vango.dev/docs/deploy/railway)
- [Docker](https://vango.dev/docs/deploy/docker)
- [Kubernetes](https://vango.dev/docs/deploy/kubernetes)

```bash
vango build
./dist/server
```

## Documentation

- [Getting Started](https://vango.dev/docs/getting-started)
- [Core Concepts](https://vango.dev/docs/concepts)
- [API Reference](https://vango.dev/docs/api)
- [Examples](https://vango.dev/examples)
- [Deployment Guide](https://vango.dev/docs/deploy)

## Community

- [Discord](https://discord.gg/vango) — Chat with the community
- [GitHub Discussions](https://github.com/vango-go/vango/discussions) — Ask questions, share ideas
- [Twitter](https://twitter.com/vaborgo) — Updates and announcements

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
git clone https://github.com/vango-go/vango
cd vango
go test ./...
```

## License

MIT License. See [LICENSE](LICENSE) for details.

---

<p align="center">
  Built with care by the Vango team.
</p>