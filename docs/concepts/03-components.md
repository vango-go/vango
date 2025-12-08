# Components

Components are the building blocks of Vango applications. They describe UI as a tree of elements.

## Elements

Elements are functions that return VNodes. Import from `pkg/vdom`:

```go
import . "vango/pkg/vdom"

// Basic element
Div(Class("card"), Text("Hello"))

// Nested elements
Div(Class("container"),
    H1(Text("Title")),
    P(Text("Description")),
    Button(OnClick(handler), Text("Click me")),
)
```

**Why this syntax?**
- Pure Go — standard tooling works
- Type-safe — compiler catches errors
- Flexible — attributes and children mix freely

## Stateless Components

Simple functions returning VNodes:

```go
func Greeting(name string) *vango.VNode {
    return H1(Textf("Hello, %s!", name))
}
```

## Stateful Components

Functions returning `vango.Component` with internal state:

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(initial)  // Local state

        return Div(
            Text(fmt.Sprintf("Count: %d", count())),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}
```

## Components with Children

Use variadic `...any` to accept children:

```go
func Card(title string, children ...any) *vango.VNode {
    return Div(Class("card"),
        H2(Text(title)),
        Div(Class("card-body"), children...),
    )
}

// Usage
Card("Settings",
    Input(Type("text")),
    Button(Text("Save")),
)
```

## Conditional Rendering

```go
If(isLoggedIn, UserMenu())

IfElse(isLoggedIn, UserMenu(), LoginButton())
```

## List Rendering

Use `Range` with keys for efficient updates:

```go
Ul(
    Range(items, func(item Item, i int) *vango.VNode {
        return Li(Key(item.ID), Text(item.Name))
    }),
)
```
