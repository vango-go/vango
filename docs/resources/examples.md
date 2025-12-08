# Examples

## Todo App

```go
func TodoPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        todos := vango.Signal([]Todo{})
        input := vango.Signal("")

        add := func() {
            if input() == "" { return }
            todos.Update(vango.Append(Todo{Text: input()}))
            input.Set("")
        }

        return Div(
            Form(OnSubmit(add),
                Input(Value(input()), OnInput(input.Set)),
                Button(Text("Add")),
            ),
            Ul(Range(todos(), func(t Todo, i int) *vango.VNode {
                return Li(Key(i), Text(t.Text))
            })),
        )
    })
}
```

## Real-time Chat

```go
func ChatRoom() vango.Component {
    return vango.Func(func() *vango.VNode {
        // GlobalSignal: shared across ALL users
        messages := vango.GlobalSignal([]Message{})
        input := vango.Signal("")

        send := func() {
            messages.Update(vango.Append(Message{
                User: currentUser(),
                Text: input(),
            }))
            input.Set("")
        }

        return Div(
            Div(Range(messages(), MessageItem)),
            Form(OnSubmit(send),
                Input(Value(input()), OnInput(input.Set)),
                Button(Text("Send")),
            ),
        )
    })
}
```

## Dashboard with Charts

```go
func Dashboard() vango.Component {
    return vango.Func(func() *vango.VNode {
        stats := vango.Resource(analytics.GetStats)

        if stats.Loading() {
            return Spinner()
        }

        return Div(
            StatCard("Revenue", stats.Data().Revenue),
            JSIsland("chart", JSModule("/js/charts.js"),
                JSProps{"data": stats.Data().History}),
        )
    })
}
```
