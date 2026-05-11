# Performance Plan

## Primary SLA

The primary user-facing SLA is redirect latency:

```text
GET /r/{token} p95 < 100ms
```

## First Benchmark

Use k6 against a cached token:

```bash
k6 run -e BASE_URL=http://localhost:8080 -e TOKEN=<token> load-tests/redirect.k6.js
```

## Metrics To Watch

- `redirect_latency_seconds`
- `redirect_requests_total`
- `redirect_cache_hits_total`
- `redirect_cache_misses_total`
- `analytics_enqueue_failures_total`

## Experiments

1. Warm-cache redirect latency.
2. Cold-cache redirect latency.
3. Redis unavailable: service should fall back to PostgreSQL.
4. Analytics enqueue unavailable: redirect should still succeed.
5. Expired/deleted tokens: return `410 Gone`.

## Interview Framing

Do not only say "add cache." Quantify the effect:

```text
50K QPS with 95% cache hit rate -> 2.5K DB QPS.
50K QPS with 99% cache hit rate -> 500 DB QPS.
```

Then discuss failure modes:

```text
Redis failure should degrade to slower DB reads, not user-facing 5xx after a successful DB lookup.
Analytics failure should affect analytics completeness, not redirect availability.
```
