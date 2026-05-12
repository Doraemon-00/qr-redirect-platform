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
