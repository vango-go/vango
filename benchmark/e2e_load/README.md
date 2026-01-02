# E2E Load Benchmark (Server + WebSocket Clients)

This benchmark answers the production questions that microbenchmarks can’t:

- p50/p95/p99 roundtrip latency under concurrent load
- throughput (events/sec) under load
- allocation + GC behavior under sustained load

It runs the real Vango WebSocket server (`pkg/server`) and drives **N concurrent WebSocket clients** that speak the real Vango protocol:

`ClientHello` → `Event` frames → `Patches` frames

**What it measures**

Client send → kernel → server decode → handler → render → diff → patch encode → WS write → client read + decode.

It does **not** include browser DOM patch application. (That’s a separate benchmark.)

## Run

From `vango_v2/`:

```bash
mkdir -p .gocache
GOCACHE=$PWD/.gocache go run ./benchmark/e2e_load -clients=200 -duration=30s -rps=5 -list=50
```

Notes:
- The benchmark is response-gated (each client waits for the patch that echoes its token), so it naturally captures queueing/tail latency.
- `-list` controls how much VDOM is re-rendered/diffed per event.
- Requires binding a localhost TCP port (may be blocked in some sandboxed environments).
