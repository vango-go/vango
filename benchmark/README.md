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
