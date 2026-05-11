# 004: Redirect Cache Strategy

## Context

At high redirect QPS, PostgreSQL token lookup becomes the first bottleneck. Redis is added in front of PostgreSQL to reduce DB reads.

## Options

1. No cache.
2. Local in-process cache.
3. Shared Redis cache.
4. CDN edge cache.
5. Redis plus CDN.

## Decision

Use Redis cache-aside in V1. Discuss CDN as a later multi-region or extreme-QPS optimization.

## Why

- Redis is shared across API instances.
- It avoids local-cache inconsistency and poor hit rate without sticky routing.
- It keeps the implementation realistic without requiring CDN purge integration.

## Cache Miss Flow

```text
Redis miss -> PostgreSQL lookup -> return response -> best-effort Redis fill
```

Cache fill failure should not fail the redirect.

## Update/Delete Flow

On update, delete, or expiration-sensitive changes, invalidate the Redis key.

## Trade-Offs

- Redis adds network dependency.
- Cache hit ratio must be measured.
- Cache invalidation needs care.
- CDN can reduce origin traffic further but can create analytics gaps if edge hits do not reach the API.

## Metrics

- `redirect_cache_hits_total`
- `redirect_cache_misses_total`
- `redirect_cache_fill_failures_total`
- `redirect_latency_seconds`

## Interview Answer

The redirect path is read-heavy, and hot campaign QR codes can drive very high repeated lookups. Redis cache-aside protects PostgreSQL while keeping DB as source of truth. Cache refill is best-effort because cache is an optimization layer and Redis failure should degrade the system to slower, not broken.
