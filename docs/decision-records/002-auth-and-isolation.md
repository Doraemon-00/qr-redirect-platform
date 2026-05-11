# 002: Auth And Isolation

## Context

The redirect endpoint must be public because QR scanners should not log in. Owner management endpoints need isolation so one owner cannot read, update, delete, or inspect analytics for another owner's QR code.

## Options

1. No auth.
2. Full login/password/OAuth.
3. API key auth.
4. External auth provider.

## Decision

Use API key auth for owner APIs in V1. Public redirect remains unauthenticated.

## Why

- Demonstrates tenant isolation without building login UI.
- Fits an API-first backend showcase.
- Easy to test with curl and k6.
- Keeps focus on redirect performance, caching, analytics, and observability.

## Required Invariant

Every owner API query must be scoped by authenticated owner ID:

```sql
WHERE token = $1 AND owner_id = $2
```

## Trade-Offs

- API key auth is less user-friendly than a full login product.
- Key creation/revocation still needs safe handling.
- External OIDC can be added later without changing the core isolation rule.

## Interview Answer

Scanners use public redirects, but owners need authenticated management APIs. I chose API key auth because the project is API-first and performance-focused. The important production invariant is tenant isolation: every management query is scoped by owner ID.
