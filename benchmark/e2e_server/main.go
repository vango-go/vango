// E2E Benchmark Server for Vango
// This server provides a real WebSocket-based counter component
// to measure actual Click → Server → DOM Update roundtrip latency.
package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/vango-dev/vango/v2/pkg/render"
	"github.com/vango-dev/vango/v2/pkg/server"
	"github.com/vango-dev/vango/v2/pkg/vango"
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

func main() {
	// Create the Vango server with WebSocket support
	srv := server.New(&server.ServerConfig{
		Address: ":8766",
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for benchmark
		},
	})

	// Set the root component factory - called for each WebSocket session
	srv.SetRootComponent(func() server.Component {
		return NewBenchmarkApp()
	})

	// Create HTTP mux for initial page load + static files
	mux := http.NewServeMux()

	// Serve Vango client JS
	mux.HandleFunc("/_vango/client.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, "../../templates/default/public/_vango/client.js")
	})

	// Home page - renders initial HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		app := NewBenchmarkApp()
		renderer := render.NewRenderer(render.RendererConfig{})

		var buf bytes.Buffer
		err := renderer.RenderPage(&buf, render.PageData{
			Title:  "Vango E2E Benchmark - Real Latency Test",
			Body:   app.Render(),
			Styles: []string{benchmarkCSS},
		})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		buf.Write([]byte(benchmarkScript))
		w.Write(buf.Bytes())
	})

	// Set HTTP handler (WebSocket handled at /_vango/ws automatically)
	srv.SetHandler(mux)

	fmt.Println("🚀 E2E Benchmark Server running at http://localhost:8766")
	fmt.Println("📡 WebSocket endpoint: ws://localhost:8766/_vango/ws")
	fmt.Println("   Click 'Increment' to measure real latency")

	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

// BenchmarkApp is the root application component
type BenchmarkApp struct {
	counter *CounterState
}

func NewBenchmarkApp() *BenchmarkApp {
	return &BenchmarkApp{
		counter: NewCounterState(0),
	}
}

func (a *BenchmarkApp) Render() *VNode {
	return Div(ID("app"),
		// Counter display and buttons
		a.counter.Render(),
	)
}

// CounterState holds the reactive state for the counter
type CounterState struct {
	count *vango.IntSignal
}

func NewCounterState(initial int) *CounterState {
	return &CounterState{
		count: vango.NewIntSignal(initial),
	}
}

func (c *CounterState) Render() *VNode {
	return Div(
		Div(Class("counter-display"),
			Textf("%d", c.count.Get()),
		),
		Button(
			Class("increment-btn"),
			OnClick(func() {
				c.count.Inc()
			}),
			"Increment (measures latency)",
		),
		Button(
			Class("reset-btn"),
			OnClick(func() {
				c.count.Set(0)
			}),
			"Reset",
		),
	)
}

const benchmarkCSS = `
body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	max-width: 900px;
	margin: 0 auto;
	padding: 2rem;
	background: #1a1a2e;
	color: #eee;
}
h1 { color: #6ee7b7; margin-bottom: 0.5rem; }
.subtitle { color: #888; margin-bottom: 2rem; }
.benchmark-section {
	background: #16213e;
	border-radius: 12px;
	padding: 1.5rem;
	margin-bottom: 1.5rem;
}
.counter-display {
	font-size: 4rem;
	font-weight: bold;
	color: #6ee7b7;
	text-align: center;
	padding: 2rem;
	background: #0f3460;
	border-radius: 12px;
	margin-bottom: 1rem;
}
button {
	background: linear-gradient(135deg, #6ee7b7, #3b82f6);
	border: none;
	color: #000;
	padding: 0.75rem 1.5rem;
	border-radius: 8px;
	font-weight: 600;
	cursor: pointer;
	margin-right: 0.5rem;
	margin-bottom: 0.5rem;
	transition: transform 0.1s;
}
button:hover { transform: scale(1.02); }
button:active { transform: scale(0.98); }
.reset-btn {
	background: #4a5568;
	color: #fff;
}
.result-value { color: #6ee7b7; font-weight: bold; }
.result-pass { color: #22c55e; }
.result-warn { color: #eab308; }
.result-fail { color: #ef4444; }
table { width: 100%; border-collapse: collapse; margin-top: 1rem; }
th, td { padding: 0.5rem; text-align: left; border-bottom: 1px solid #333; }
th { color: #6ee7b7; }
.latency-log {
	max-height: 200px;
	overflow-y: auto;
	font-size: 0.8rem;
	color: #888;
	margin-top: 1rem;
}
.status-connected { color: #22c55e; }
.status-disconnected { color: #ef4444; }
.instructions {
	background: #2d3748;
	border-left: 4px solid #6ee7b7;
	padding: 1rem;
	margin-bottom: 1.5rem;
	font-size: 0.9rem;
}
`

const benchmarkScript = `
<h1>Vango E2E Benchmark - Real Latency</h1>
<p class="subtitle">Measuring actual Click -> Server -> DOM Update roundtrip</p>

<div class="instructions">
	<strong>Status:</strong> <span id="connection-status" class="status-disconnected">Connecting...</span><br>
	<strong>How it works:</strong> Click "Increment" to send a real event through WebSocket to the server.
	The server updates state and sends a binary patch back. We measure the full roundtrip.
</div>

<div class="benchmark-section">
	<h3>Latency Results (Real E2E)</h3>
	<button onclick="runBurstTest(10)">Burst Test (10 clicks)</button>
	<button onclick="runBurstTest(50)">Burst Test (50 clicks)</button>
	<button onclick="clearLatencies()">Clear</button>
	<table>
		<thead>
			<tr>
				<th>Metric</th>
				<th>Value</th>
				<th>Target</th>
				<th>Status</th>
			</tr>
		</thead>
		<tbody id="metrics-body">
			<tr><td colspan="4">Click the Increment button to measure...</td></tr>
		</tbody>
	</table>
	<div id="latency-log" class="latency-log"></div>
</div>

<script>
(function() {
	const latencies = [];
	const TARGET_LATENCY = 50;
	let pendingTimestamp = null;

	// Wait for Vango client to be ready
	function waitForVango(callback) {
		if (window.__vango__ && window.__vango__.connected) {
			callback();
		} else {
			setTimeout(() => waitForVango(callback), 100);
		}
	}

	// Intercept click events to record timestamp
	document.addEventListener('click', function(e) {
		const target = e.target.closest('.increment-btn');
		if (target) {
			pendingTimestamp = performance.now();
		}
	}, true);

	// Create a MutationObserver to detect when the counter updates
	function setupObserver() {
		const counterEl = document.querySelector('.counter-display');
		if (counterEl) {
			const observer = new MutationObserver(function(mutations) {
				if (pendingTimestamp !== null) {
					const rtt = performance.now() - pendingTimestamp;
					pendingTimestamp = null;
					recordLatency(rtt);
				}
			});
			observer.observe(counterEl, { childList: true, characterData: true, subtree: true });
		}
	}

	function recordLatency(rtt) {
		latencies.push(rtt);
		updateMetrics();
		logLatency(rtt);
	}

	function updateMetrics() {
		if (latencies.length === 0) return;

		const sorted = [...latencies].sort((a, b) => a - b);
		const min = sorted[0];
		const max = sorted[sorted.length - 1];
		const avg = latencies.reduce((a, b) => a + b, 0) / latencies.length;
		const p50 = sorted[Math.floor(sorted.length * 0.5)];
		const p95 = sorted[Math.floor(sorted.length * 0.95)];
		const p99 = sorted[Math.floor(sorted.length * 0.99)];

		const metricsBody = document.getElementById('metrics-body');
		if (metricsBody) {
			const metrics = [
				{ name: 'Min', value: min, target: TARGET_LATENCY },
				{ name: 'Average', value: avg, target: TARGET_LATENCY },
				{ name: 'P50', value: p50, target: TARGET_LATENCY },
				{ name: 'P95', value: p95, target: TARGET_LATENCY },
				{ name: 'P99', value: p99, target: TARGET_LATENCY * 1.5 },
				{ name: 'Max', value: max, target: TARGET_LATENCY * 2 },
				{ name: 'Samples', value: latencies.length, target: null },
			];

			metricsBody.innerHTML = metrics.map(m => {
				if (m.target === null) {
					return '<tr><td>' + m.name + '</td><td>' + m.value + '</td><td>-</td><td>-</td></tr>';
				}
				const status = m.value <= m.target ? 'PASS' : m.value <= m.target * 1.5 ? 'WARN' : 'FAIL';
				const statusClass = status === 'PASS' ? 'result-pass' : status === 'WARN' ? 'result-warn' : 'result-fail';
				return '<tr><td>' + m.name + '</td><td class="result-value">' + m.value.toFixed(2) + ' ms</td><td>&lt; ' + m.target + ' ms</td><td class="' + statusClass + '">' + status + '</td></tr>';
			}).join('');
		}
	}

	function logLatency(rtt) {
		const log = document.getElementById('latency-log');
		if (log) {
			const statusClass = rtt <= TARGET_LATENCY ? 'result-pass' : rtt <= TARGET_LATENCY * 1.5 ? 'result-warn' : 'result-fail';
			log.innerHTML += '<span class="' + statusClass + '">[' + latencies.length + '] ' + rtt.toFixed(2) + ' ms</span><br>';
			log.scrollTop = log.scrollHeight;
		}
	}

	// Burst test function
	window.runBurstTest = async function(count) {
		const btn = document.querySelector('.increment-btn');
		if (!btn) return;
		for (let i = 0; i < count; i++) {
			btn.click();
			await new Promise(r => setTimeout(r, 100));
		}
	};

	// Clear function
	window.clearLatencies = function() {
		latencies.length = 0;
		const metricsBody = document.getElementById('metrics-body');
		if (metricsBody) {
			metricsBody.innerHTML = '<tr><td colspan="4">Click the Increment button to measure...</td></tr>';
		}
		const log = document.getElementById('latency-log');
		if (log) log.innerHTML = '';
	};

	// Connection status update
	waitForVango(function() {
		const statusEl = document.getElementById('connection-status');
		if (statusEl) {
			statusEl.textContent = 'Connected';
			statusEl.className = 'status-connected';
		}
		// Re-setup observer after Vango is connected (in case DOM was replaced)
		setupObserver();
	});

	// Initial observer setup
	setupObserver();
})();
</script>
`
