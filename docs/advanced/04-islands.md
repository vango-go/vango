# JavaScript Islands

Embed third-party JavaScript libraries within Vango apps.

## When to Use

- Chart libraries (Chart.js, D3)
- Rich text editors (Quill, TipTap)
- Maps (Leaflet, Mapbox)
- Existing React/Vue widgets during migration

## Basic Usage

```go
JSIsland("chart-1",
    JSModule("/js/charts.js"),
    JSProps{
        "data": chartData,
        "type": "line",
    },
)
```

## JavaScript Side

```javascript
// public/js/charts.js
import { Chart } from 'chart.js';

export function mount(container, props) {
    const chart = new Chart(container, {
        type: props.type,
        data: props.data,
    });
    return () => chart.destroy();  // Cleanup
}

export function update(container, props, instance) {
    // Called when props change
}
```

## Communication

**Server → Island:**
```go
vango.SendToIsland("chart-1", map[string]any{"action": "update"})
```

**Island → Server:**
```javascript
import { sendToVango } from '@vango/bridge';
sendToVango('chart-1', { event: 'click', x: 100 });
```

```go
vango.OnIslandMessage("chart-1", func(msg map[string]any) {
    // Handle message
})
```
