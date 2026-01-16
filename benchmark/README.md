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

For usage, profiles, and JSON output schema, see `BENCHMARKS.md` in the repo root.
