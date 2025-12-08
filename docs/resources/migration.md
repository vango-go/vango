# Migration Guide

## From React

| React | Vango |
|-------|-------|
| `useState` | `vango.Signal` |
| `useEffect` | `vango.Effect` |
| `useMemo` | `vango.Memo` |
| JSX | Function calls |
| Runs in browser | Runs on server |

**React:**
```jsx
function Counter() {
    const [count, setCount] = useState(0);
    return <button onClick={() => setCount(c => c+1)}>{count}</button>;
}
```

**Vango:**
```go
func Counter() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(0)
        return Button(OnClick(count.Inc), Textf("%d", count()))
    })
}
```

## From Vue

| Vue | Vango |
|-----|-------|
| `ref` | `vango.Signal` |
| `computed` | `vango.Memo` |
| Template | Function composition |

## Gradual Migration

1. Add Vango alongside existing app
2. Use `JSIsland` to embed existing React/Vue widgets
3. Incrementally rewrite components in Go
