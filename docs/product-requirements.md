# Product Requirements

## Product Positioning

Build a dynamic QR redirect and analytics platform, not only a QR image generator. The implementation should demonstrate backend engineering decisions that matter under high read traffic.

## Users

Owner:

- Creates QR codes.
- Receives a short URL and QR image endpoint.
- Updates target URL.
- Soft deletes QR codes.
- Views analytics.

Scanner:

- Scans or clicks the public short URL.
- Does not authenticate.
- Receives a redirect or an appropriate error status.

## Functional Requirements

- Owners submit a long URL and receive a short URL token plus QR code image endpoint.
- The QR code encodes a short URL routed through this service.
- Owners can modify the target URL after creation.
- Owners can soft delete a QR code.
- Owners can optionally set or update expiration timestamp.
- Deleted or expired links return `410 Gone` on redirect.
- Missing tokens return `404 Not Found`.
- URL validation includes format checks, normalization, and malicious/internal URL blocking.
- Analytics endpoint returns scan count and daily breakdown.

## Non-Functional Requirements

- Redirect p95 latency target: under `100ms`.
- Cached redirect p95 target: under `20ms` in local benchmark where feasible.
- Design target: 1B QR codes and event traffic that can spike from 5K to 50K redirect QPS.
- Redirect path should degrade to slower DB fallback if Redis is unavailable.
- Analytics failure must not break redirect.
- Management APIs require tenant isolation.
- Metrics must expose latency, cache hit ratio, error counts, and analytics enqueue failures.

## Explicit V1 Assumptions

- QR-first product: scanners are not expected to manually type tokens.
- No custom aliases in V1.
- No QR image object storage in V1.
- No full login UI in V1.
- API key auth is enough to prove owner isolation.
- CDN is discussed in architecture but not required for local implementation.
