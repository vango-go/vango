# Routing Reference

Vango uses file-based routing.

## Directory Structure

```
app/routes/
├── index.go              → /
├── about.go              → /about
├── projects/
│   ├── index.go          → /projects
│   ├── new.go            → /projects/new
│   └── _id_.go           → /projects/:id
├── users/
│   └── _id_/
│       └── posts.go      → /users/:id/posts
├── api/
│   └── users.go          → /api/users
└── _layout.go            → Wraps all routes
```

## Dynamic Route Conventions

Vango supports two conventions for dynamic route parameters:

### Underscore Notation (Recommended)

The underscore notation is Go-friendly and works on all systems:

- `_id_.go` → `:id` (dynamic parameter)
- `_id_/posts.go` → `:id/posts` (dynamic directory)
- `_slug___.go` → `*slug` (catch-all, triple underscore)

### Bracket Notation (Next.js/Remix Style)

The bracket notation may not work on Windows or with some Go toolchains:

- `[id].go` → `:id`
- `[...slug].go` → `*slug` (catch-all)

**Note:** We recommend using underscore notation for maximum compatibility.

## Page Components

```go
// app/routes/projects/_id_.go
package routes

type Params struct {
    ID int `param:"id"`
}

func ShowPage(ctx vango.Ctx, p Params) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Access p.ID
    })
}
```

## Layouts

`_layout.go` wraps child routes:

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(
        Head(Title(Text("My App"))),
        Body(
            Navbar(),
            children,
            Footer(),
        ),
    )
}
```

## Route Middleware

Add a `Middleware()` function to apply middleware to routes:

```go
// app/routes/admin/_layout.go
func Middleware() []router.Middleware {
    return []router.Middleware{
        auth.RequireAuth,
        auth.RequireRole(func(u *models.User) bool {
            return u.Role == "admin"
        }),
    }
}
```

Middleware runs on every event for matching routes. See [Middleware Reference](12-middleware.md).

## Navigation

```go
// Programmatic
vango.Navigate("/dashboard")
vango.Navigate("/projects/123")

// With query params
vango.Navigate("/search?q=hello")

// Replace history (no back)
vango.Replace("/login")

// Links
A(Href("/projects"), Text("Projects"))
A(Href("/projects"), Prefetch(), Text("Projects"))  // Preload on hover
```

## API Routes

```go
// app/routes/api/users.go
func GET(ctx vango.Ctx) ([]User, error) {
    return db.Users.All()
}

func POST(ctx vango.Ctx, input CreateUserInput) (*User, error) {
    return db.Users.Create(input)
}
```

## External Router Integration

Mount Vango in Chi, Gorilla, or stdlib mux:

```go
r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(myauth.JWTMiddleware)

// API routes
r.Post("/api/webhook", webhookHandler)

// Mount Vango for all other routes
r.Handle("/*", app.Handler())
```

