# Benchmark Results (Local)

These results are from the new `cmd/vango-bench` macrobench harness. They are
environment-dependent and are intended for local regression tracking, not
absolute performance claims.

## fast profile

- Clients: 50
- Duration: 10s
- Target rps/client: 2
- Throughput: ~100 events/sec (2.00 per client), total events 1000, errors 0
- Latency (ms): min 0.18, p50 3.30, p95 6.53, p99 9.42, max 12.11
- Protocol: avg event bytes 34.0, avg patch bytes 64.7, patches/event 1.00
- GC: alloc 33.47 MB, heap live 2.87 MB, total pause 3.86 ms, GC CPU 0.07%

## standard profile

- Clients: 200
- Duration: 30s
- Target rps/client: 5
- Throughput: ~996 events/sec (4.98 per client), total events 29893, errors 0
- Latency (ms): min 0.17, p50 2.82, p95 11.34, p99 31.34, max 186.75
- Protocol: avg event bytes 34.15, avg patch bytes 65.05, patches/event 1.00
- GC: alloc 1816.41 MB, heap live 19.54 MB, total pause 18.53 ms, GC CPU 0.39%

## stress profile

- Clients: 500
- Duration: 60s
- Target rps/client: 10
- Throughput: ~4885 events/sec (9.77 per client), total events 147937, errors 500
- Latency (ms): min 0.22, p50 3.17, p95 39.88, p99 89.17, max 284.87
- Protocol: avg event bytes 34.6, avg patch bytes 65.6, patches/event 1.00
- GC: alloc 16634.17 MB, heap live 3.36 MB, total pause 13.74 ms, GC CPU 4.39%
- Note: server evicted all sessions due to MaxMemoryPerSession=200KB (errors reflect evictions).

## stress-highlimit profile

- Clients: 500
- Duration: 60s
- Target rps/client: 10
- Server limits: unlimited (sessions/mem)
- Connections: 500/500 ok
- Session memory: total 782.77 MB, per session 1603.12 KB
- Throughput: ~4724.9 events/sec (9.45 per client), total events 283533, errors 0
- Latency (ms): min 0.21, p50 4.15, p95 94.28, p99 226.67, max 599.79
- Protocol: avg event bytes 34.8, avg patch bytes 65.8, patches/event 1.00
- GC: alloc 31814.16 MB, heap live 83.15 MB, total pause 27.71 ms, GC CPU 4.35%

## density profile

- Clients: 2000
- Duration: 30s
- Target rps/client: 0 (idle)
- Server limits: unlimited (sessions/mem)
- Connections: 2000/2000 ok
- Session memory: total 124.76 MB, per session 63.88 KB
- Throughput: 0 events/sec (idle), total events 0, errors 0
- Latency: none recorded (idle)
- GC: alloc 125.40 MB, heap live 58.12 MB, total pause 2.38 ms, GC CPU 0.10%

For usage, profiles, and JSON output schema, see `BENCHMARKS.md` in the repo root.

## stress attempt #1

• Here’s what happened during the stress run:

  - The macrobench started and created hundreds of sessions. As load grew, the server’s cleanup loop hit memory-pressure logic and began
    evicting oversized sessions.
  - In that path, SessionManager.evictOversizedSessions() calls Session.MemoryUsage() for each session (see vango/pkg/server/
    manager.go), which iterates s.components and calls comp.MemoryUsage() on each component (see vango/pkg/server/session.go).
  - One of the entries in s.components was nil, so comp.MemoryUsage() dereferenced a nil pointer and panicked:
    panic: runtime error: invalid memory address or nil pointer dereference
    at vango/pkg/server/component.go:224 → called from vango/pkg/server/session.go:1540 → vango/pkg/server/manager.go:656.

  In short: the stress benchmark triggered memory-pressure eviction, and the server crashed because Session.MemoryUsage() doesn’t guard
  against nil component entries.
