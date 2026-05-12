# Performance Results

This file records local benchmark runs. Results are not cloud capacity claims; they are evidence that the benchmark harness, metrics, and cache behavior are working.

## 2026-05-12: Warm Redirect Smoke Benchmark

Environment:

```text
Machine: local Windows development machine
Docker memory limit visible to containers: 15.46 GiB
API: Go service in Docker
PostgreSQL: postgres:17-alpine
Redis: redis:7-alpine
ClickHouse: clickhouse/clickhouse-server:24.12-alpine
Load generator: grafana/k6 Docker image
```

Command:

```powershell
docker run --rm `
  -v "${PWD}/load-tests:/scripts" `
  -e BASE_URL=http://host.docker.internal:8080 `
  -e TOKEN=mist8mzNLlLq `
  -e RATE=100 `
  -e DURATION=30s `
  -e VUS=50 `
  grafana/k6 run /scripts/redirect.k6.js
```

Scenario:

```text
Warm redirect benchmark.
Token was created and scanned once before the run to populate Redis.
Target rate: 100 redirect requests/sec for 30 sec.
Total requests: 3,000.
```

k6 result:

| Metric | Value |
|---|---:|
| Request rate | `100.00 req/s` |
| Failed requests | `0.00%` |
| Average latency | `2.14 ms` |
| Median latency | `2.03 ms` |
| p90 latency | `2.62 ms` |
| p95 latency | `2.98 ms` |
| Max latency | `8.91 ms` |

Prometheus counters after run:

```text
redirect_cache_hits_total 3001
redirect_cache_misses_total 4
redirect_latency_seconds_count 3005
redirect_requests_total{result="redirect"} 3004
redirect_requests_total{result="gone"} 1
analytics_enqueue_failures_total 0
```

Approximate observed cache hit ratio from current counters:

```text
3001 / (3001 + 4) = 99.87%
```

Note: counters included earlier smoke-test traffic in the same process, so this is a local run observation rather than an isolated benchmark dataset.

Container stats after run:

| Container | CPU | Memory |
|---|---:|---:|
| `qrcode-api-1` | `0.00%` | `10.08 MiB` |
| `qrcode-redis-1` | `0.36%` | `7.10 MiB` |
| `qrcode-postgres-1` | `2.44%` | `35.95 MiB` |
| `qrcode-clickhouse-1` | `7.83%` | `538 MiB` |

Redis stream length after run:

```text
scan_events XLEN = 3010
```

Interpretation:

- The warm redirect path is comfortably below the `p95 < 100ms` target at this small load.
- Cache behavior is visible: Redis hit count dominates DB miss count.
- Analytics enqueue did not fail, but events are not yet consumed into ClickHouse by a worker.
- Next benchmark should isolate counters by restarting the API/Redis or capturing before/after counter deltas.

Next tests:

```text
1. 500 RPS warm redirect.
2. Cold-cache redirect comparison.
3. Create API benchmark with owner rate limit adjusted or disabled for the test owner.
4. Analytics worker throughput once implemented.
```
