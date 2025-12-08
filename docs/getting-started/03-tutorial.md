# Tutorial: Build a Todo App

This tutorial walks you through building a complete Todo application with Vango.

## What You'll Learn

- Creating components with state
- Handling user input
- Working with lists
- Connecting to a database

## Step 1: Create the Project

```bash
vango create todo-app
cd todo-app
vango dev
```

## Step 2: Define the Todo Model

Create `db/models.go`:

```go
package db

type Todo struct {
    ID   int
    Text string
    Done bool
}

var todos = []Todo{}
var nextID = 1

func All() []Todo       { return todos }
func Create(text string) Todo {
    t := Todo{ID: nextID, Text: text}
    nextID++
    todos = append(todos, t)
    return t
}
func Toggle(id int) {
    for i := range todos {
        if todos[i].ID == id {
            todos[i].Done = !todos[i].Done
        }
    }
}
```

## Step 3: Build the UI

Edit `app/routes/index.go`:

```go
package routes

import (
    "todo-app/db"
    "vango/pkg/vango"
    . "vango/pkg/vdom"
)

func Page() vango.Component {
    return vango.Func(func() *vango.VNode {
        todos := vango.Signal(db.All())
        newTodo := vango.Signal("")

        add := func() {
            if newTodo() == "" { return }
            db.Create(newTodo())
            todos.Set(db.All())
            newTodo.Set("")
        }

        toggle := func(id int) func() {
            return func() {
                db.Toggle(id)
                todos.Set(db.All())
            }
        }

        return Div(Class("todo-app"),
            H1(Text("My Todos")),

            Form(OnSubmit(add),
                Input(
                    Type("text"),
                    Value(newTodo()),
                    OnInput(newTodo.Set),
                    Placeholder("What needs to be done?"),
                ),
                Button(Type("submit"), Text("Add")),
            ),

            Ul(
                Range(todos(), func(t db.Todo, i int) *vango.VNode {
                    return Li(
                        Key(t.ID),
                        ClassIf(t.Done, "completed"),
                        Input(Type("checkbox"), Checked(t.Done), OnChange(toggle(t.ID))),
                        Span(Text(t.Text)),
                    )
                }),
            ),
        )
    })
}
```

## Step 4: Add Styling

Create `public/styles.css`:

```css
.todo-app { max-width: 500px; margin: 2rem auto; }
.completed span { text-decoration: line-through; color: #888; }
```

## What's Happening?

1. `vango.Signal` creates reactive state
2. `OnSubmit`, `OnInput`, `OnChange` bind server handlers
3. When state changes, Vango re-renders and sends patches
4. The browser updates without a full reload

## Next Steps

- [Components](../concepts/03-components.md) — Deeper dive into the component model
- [State](../reference/06-state.md) — Advanced state patterns
