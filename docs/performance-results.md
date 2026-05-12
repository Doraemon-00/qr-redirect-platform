# Performance Results

This file records local benchmark runs. Results are not cloud capacity claims; they are evidence that the benchmark harness, metrics, and cache behavior are working.

## 2026-05-12: Analytics Worker Recovery Benchmark

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

Commands:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-analytics-recovery.ps1 -Rate 500 -Duration 30s -VUs 100 -MaxVUs 500 -AnalyticsBatchSize 500 -AnalyticsBlockSeconds 2 -DrainTimeoutSeconds 240
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-analytics-recovery.ps1 -Rate 10 -Duration 2s -VUs 10 -MaxVUs 50 -AnalyticsBatchSize 500 -AnalyticsBlockSeconds 2 -DrainTimeoutSeconds 60
```

Scenario:

```text
Start API with ANALYTICS_WORKER_ENABLED=false.
Create fresh token and warm redirect cache.
Run redirect load while worker is disabled.
Verify ClickHouse write counters do not increase while disabled.
Restart API with worker enabled.
Wait for queued Redis Stream events to drain into ClickHouse.
```

500 RPS outage/backlog result:

| Metric | Value |
|---|---:|
| Request rate | `500.01 req/s` |
| Failed requests | `0.00%` |
| p95 redirect latency while worker disabled | `2.09 ms` |
| Redis stream length before load | `84039` |
| Redis stream length after load | `99040` |
| ClickHouse writes while worker disabled | `0` |
| Events written after worker restart | `15002` |
| Expected minimum batches | `31` |
| Actual batches | `31` |
| Pending after drain | `0` |
| Worker failures | `0` |

Low-traffic partial-batch result:

| Metric | Value |
|---|---:|
| Request rate | `10.48 req/s` |
| Failed requests | `0.00%` |
| p95 redirect latency while worker disabled | `2.88 ms` |
| Redis stream length before load | `99041` |
| Redis stream length after load | `99062` |
| ClickHouse writes while worker disabled | `0` |
| Events written after worker restart | `22` |
| Expected minimum batches | `1` |
| Actual batches | `1` |
| Pending after drain | `0` |
| Worker failures | `0` |

Interpretation:

- Redirects continue to succeed when the analytics worker is disabled.
- Events accumulate in Redis Stream while the worker is unavailable.
- After restart, the worker drains the backlog into ClickHouse with no pending messages or worker failures.
- The configured batch size is respected under backlog load: roughly 500 events per ClickHouse batch.
- Partial batches flush correctly when traffic is low, so low-volume reports do not wait for a full batch.

## 2026-05-12: 500 RPS Cache vs No-Cache Redirect Benchmark

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

Commands:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-redirect.ps1 -CacheEnabled true -Rate 500 -Duration 30s -VUs 100 -MaxVUs 500
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-redirect.ps1 -CacheEnabled false -Rate 500 -Duration 30s -VUs 100 -MaxVUs 500
```

Scenario:

```text
Fresh token per run.
API restarted before each run with REDIRECT_CACHE_ENABLED=true or false.
Cache-enabled run was warmed once before measuring metric deltas.
Target rate: 500 redirect requests/sec for 30 sec.
Total measured requests per run: 15,001.
```

k6 result:

| Cache | Request rate | Failed requests | Average latency | Median latency | p90 latency | p95 latency | Max latency |
|---|---:|---:|---:|---:|---:|---:|---:|
| Enabled | `500.01 req/s` | `0.00%` | `1.87 ms` | `1.74 ms` | `2.46 ms` | `2.90 ms` | `40.94 ms` |
| Disabled | `500.02 req/s` | `0.00%` | `2.14 ms` | `1.75 ms` | `2.88 ms` | `3.81 ms` | `45.94 ms` |

Prometheus counter deltas:

| Metric | Cache enabled | Cache disabled |
|---|---:|---:|
| `redirect_requests_total{result="redirect"}` | `15001` | `15001` |
| `redirect_cache_hits_total` | `15001` | `0` |
| `redirect_cache_misses_total` | `0` | `0` |
| `redirect_db_lookups_total` | `0` | `15001` |
| `analytics_enqueue_failures_total` | `0` | `0` |
| `redirect_latency_seconds_count` | `15001` | `15001` |

Container stats after returning API to cache-enabled config:

| Container | CPU | Memory |
|---|---:|---:|
| `qrcode-api-1` | `0.04%` | `3.43 MiB` |
| `qrcode-redis-1` | `0.38%` | `24.22 MiB` |
| `qrcode-postgres-1` | `0.00%` | `33.95 MiB` |
| `qrcode-clickhouse-1` | `60.48%` | `1.055 GiB` |

Redis stream length after both runs:

```text
scan_events XLEN = 69033
```

Interpretation:

- At 500 RPS, both paths still meet the local p95 target with zero failed requests.
- Cache-enabled latency was lower, but the larger design signal is backend load: warm cache avoided all measured PostgreSQL lookups, while cache-disabled performed one PostgreSQL lookup per redirect.
- This validates the cache as a DB protection layer even when local single-token latency remains low.

## 2026-05-12: Analytics Worker Throughput Benchmark

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
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-analytics.ps1 -Rate 500 -Duration 30s -VUs 100 -MaxVUs 500
```

Scenario:

```text
Fresh token.
Cache enabled and warmed once before the measured run.
Redirect endpoint enqueues one analytics event per successful 302.
Worker writes Redis Stream events to ClickHouse in batches of up to 500.
Target rate: 500 redirect requests/sec for 30 sec.
Total measured requests: 15,000.
```

k6 result:

| Metric | Value |
|---|---:|
| Request rate | `499.99 req/s` |
| Failed requests | `0.00%` |
| Average latency | `2.16 ms` |
| Median latency | `2.01 ms` |
| p90 latency | `2.89 ms` |
| p95 latency | `3.34 ms` |
| Max latency | `39.67 ms` |

Prometheus counter and gauge deltas:

| Metric | Delta |
|---|---:|
| `redirect_requests_total{result="redirect"}` | `15000` |
| `redirect_cache_hits_total` | `15000` |
| `redirect_db_lookups_total` | `0` |
| `analytics_enqueue_failures_total` | `0` |
| `analytics_events_written_total` | `15000` |
| `analytics_batches_written_total` | `30` |
| `analytics_events_reclaimed_total` | `0` |
| `analytics_worker_failures_total` | `0` |
| `analytics_events_pending` | `0` |
| `analytics_stream_length` | `15000` |
| `analytics_batch_write_duration_seconds_count` | `30` |

Drain result:

```text
WrittenDelta = 15000
Pending = 0
Failures = 0
```

Container stats after run:

| Container | CPU | Memory |
|---|---:|---:|
| `qrcode-api-1` | `0.08%` | `14.26 MiB` |
| `qrcode-redis-1` | `0.38%` | `17.16 MiB` |
| `qrcode-postgres-1` | `0.00%` | `35.85 MiB` |
| `qrcode-clickhouse-1` | `9.30%` | `1.101 GiB` |

Redis stream length after run:

```text
scan_events XLEN = 39027
```

Interpretation:

- Redirect latency stayed far below the `p95 < 100ms` target while enqueueing analytics events.
- The worker drained all 15,000 new events into ClickHouse without pending backlog or worker failures.
- The coalesced batch writer produced 30 ClickHouse batches for 15,000 events, matching the configured 500-event batch size.
- Redis Stream length grows because V1 acknowledges processed messages but does not trim by consumer progress beyond the approximate max length configured on enqueue.

## 2026-05-12: Cache vs No-Cache Redirect Benchmark

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

Commands:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-redirect.ps1 -CacheEnabled true -Rate 100 -Duration 30s
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-redirect.ps1 -CacheEnabled false -Rate 100 -Duration 30s
```

Scenario:

```text
Fresh token per run.
API restarted before each run with REDIRECT_CACHE_ENABLED=true or false.
Cache-enabled run was warmed once before measuring metric deltas.
Target rate: 100 redirect requests/sec for 30 sec.
Total measured requests per run: 3,001.
```

k6 result:

| Cache | Request rate | Failed requests | Average latency | Median latency | p90 latency | p95 latency | Max latency |
|---|---:|---:|---:|---:|---:|---:|---:|
| Enabled | `100.03 req/s` | `0.00%` | `1.64 ms` | `1.40 ms` | `2.12 ms` | `2.70 ms` | `18.16 ms` |
| Disabled | `100.03 req/s` | `0.00%` | `1.73 ms` | `1.50 ms` | `2.33 ms` | `2.86 ms` | `16.30 ms` |

Prometheus counter deltas:

| Metric | Cache enabled | Cache disabled |
|---|---:|---:|
| `redirect_requests_total{result="redirect"}` | `3001` | `3001` |
| `redirect_cache_hits_total` | `3001` | `0` |
| `redirect_cache_misses_total` | `0` | `0` |
| `redirect_db_lookups_total` | `0` | `3001` |
| `analytics_enqueue_failures_total` | `0` | `0` |
| `redirect_latency_seconds_count` | `3001` | `3001` |

Container stats after the second run:

| Container | CPU | Memory |
|---|---:|---:|
| `qrcode-api-1` | `0.00%` | `10.14 MiB` |
| `qrcode-redis-1` | `2.35%` | `10.34 MiB` |
| `qrcode-postgres-1` | `0.01%` | `35.19 MiB` |
| `qrcode-clickhouse-1` | `7.05%` | `571.8 MiB` |

Redis stream length after both runs:

```text
scan_events XLEN = 9013
```

Interpretation:

- At 100 RPS on this laptop, PostgreSQL lookups are still fast enough that p95 latency is nearly the same for cache-enabled and cache-disabled runs.
- The benchmark now isolates the architectural effect: warm cache served all measured redirects without PostgreSQL lookups, while cache-disabled performed one PostgreSQL lookup per redirect.
- The next useful benchmark is 500 RPS x 30s to see when the DB-backed path begins to diverge.

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
