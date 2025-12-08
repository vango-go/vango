# Elements Reference

All HTML elements are available as functions in `pkg/vdom`.

## Basic Usage

```go
import . "vango/pkg/vdom"

Div(Class("container"),
    H1(Text("Hello")),
    P(Text("World")),
)
```

> [!NOTE]
> Vango provides concrete helper functions (`Div`, `Span`, etc.) for all standard HTML elements.
> There is **no generic `El()` shorthand function**. Use the specific element helpers or `CustomElement()` for non-standard tags.

## Common Elements

| Element | Description |
|---------|-------------|
| `Div`, `Span`, `P` | Block/inline containers |
| `H1`-`H6` | Headings |
| `A` | Links |
| `Button` | Buttons |
| `Input`, `Textarea`, `Select` | Form inputs |
| `Form` | Form container |
| `Ul`, `Ol`, `Li` | Lists |
| `Table`, `Tr`, `Td`, `Th` | Tables |
| `Img`, `Video`, `Audio` | Media |

## Attributes

```go
// Classes
Div(Class("card primary"))
Div(ClassIf(isActive, "active"))

// IDs and data
Div(Id("main"), Data("id", "123"))

// Styles
Div(Style("color: red; font-size: 16px"))

// Generic attributes
Input(Attr("autocomplete", "off"))
```

## Event Handlers

```go
Button(OnClick(handleClick), Text("Click"))
Input(OnInput(setValue), OnChange(handleChange))
Form(OnSubmit(handleSubmit))
```

## Text Content

```go
Text("Hello")                  // Static text
Textf("Count: %d", count)      // Formatted text
```

## Keys

Use `Key` for list reconciliation:

```go
Li(Key(item.ID), Text(item.Name))
```

## Void Elements

These cannot have children: `Input`, `Img`, `Br`, `Hr`, `Meta`, `Link`, etc.
