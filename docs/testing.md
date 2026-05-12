# Testing

## Prerequisites

Start the local stack:

```powershell
docker compose up -d
```

Run Go tests:

```powershell
& 'C:\Program Files\Go\bin\go.exe' test ./...
```

## Smoke Test

The smoke script verifies the full product flow:

```text
readyz
create QR
image endpoint
redirect
active Redis TTL
update target
redirect updated target
delete
410 Gone
tombstone Redis TTL
ClickHouse analytics count and daily breakdown
rate-limit headers
```

Run:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\smoke.ps1
```

The script checks Redis TTL directly, so it verifies the 10-minute cache policy without waiting 10 minutes.

## k6 Benchmarks

If k6 is installed locally:

```powershell
k6 run -e BASE_URL=http://localhost:8080 -e RATE=10 load-tests/create.k6.js
k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=<token> -e RATE=100 load-tests/redirect.k6.js
k6 run -e BASE_URL=http://localhost:8080 load-tests/owner-rate-limit.k6.js
```

If k6 is not installed, run through Docker:

```powershell
docker run --rm -v "${PWD}/load-tests:/scripts" -e BASE_URL=http://host.docker.internal:8080 -e RATE=10 grafana/k6 run /scripts/create.k6.js
docker run --rm -v "${PWD}/load-tests:/scripts" -e BASE_URL=http://host.docker.internal:8080 -e TOKEN=<token> -e RATE=100 grafana/k6 run /scripts/redirect.k6.js
docker run --rm -v "${PWD}/load-tests:/scripts" -e BASE_URL=http://host.docker.internal:8080 grafana/k6 run /scripts/owner-rate-limit.k6.js
```

Use the benchmark helper for isolated redirect metric deltas:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-redirect.ps1 -CacheEnabled true -Rate 100 -Duration 30s
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-redirect.ps1 -CacheEnabled false -Rate 100 -Duration 30s
```

The helper restarts the API with `REDIRECT_CACHE_ENABLED`, creates a fresh QR code, warms the cache when enabled, runs k6 through Docker, and prints before/after deltas for redirect counters.

Use the analytics helper to measure async worker drain behavior under redirect load:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-analytics.ps1 -Rate 500 -Duration 30s -VUs 100 -MaxVUs 500
```

The helper warms one token, runs redirect load, waits for `analytics_events_written_total` to catch up to the new redirect count, and prints stream/pending/batch metric deltas.

Use the recovery helper to verify analytics outage/backlog behavior:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\benchmark-analytics-recovery.ps1 -Rate 500 -Duration 30s -VUs 100 -MaxVUs 500 -AnalyticsBatchSize 500 -AnalyticsBlockSeconds 2
```

The helper first runs redirect load with `ANALYTICS_WORKER_ENABLED=false`, verifies no ClickHouse writes happen while the worker is disabled, then restarts the worker and waits for the queued events to drain.

## Result Format

Record benchmark results in `docs/performance-results.md` with:

```text
date
machine
docker resources
command
scenario
RPS
p50/p95/p99 latency
error rate
cache hit ratio
notes
```
