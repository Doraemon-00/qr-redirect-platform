# 006: Prototype To Production Gaps

## Context

The tutorial prototype demonstrates the core idea but leaves production gaps. This project intentionally addresses those gaps.

## Gaps And Responses

| Gap | Production Response |
|---|---|
| Error handling | Typed validation errors and HTTP responses instead of crashes |
| Rate limiting | Redis-backed rate limits for owner APIs; redirect strategy discussed separately |
| Auth & isolation | API key auth and owner-scoped QR queries |
| Monitoring | Health checks, readiness checks, Prometheus metrics, structured logs |
| Data cleanup | Soft delete plus retention/cleanup job plan |
| Caching/CDN | Redis cache-aside for redirect path; CDN discussed as scale option |

## Decision

Implement these gaps incrementally, starting with error handling, auth isolation, Redis caching, and metrics.

## Interview Answer

The prototype proves the product behavior. The production version adds the controls needed under real traffic: input validation, rate limiting, tenant isolation, observability, cleanup, and cache design.
