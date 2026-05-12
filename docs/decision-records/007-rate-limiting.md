# 007: Rate Limiting

## Context

The prototype has no protection against scripts repeatedly calling create/update/delete APIs. Public redirects can legitimately spike during campaigns, so rate limiting must treat owner APIs and scanner redirects differently.

## Options

1. No rate limiting.
2. In-process limiter.
3. Redis-backed application limiter.
4. Edge/CDN/WAF rate limiting.

## Decision

Use a Redis-backed fixed-window limiter for owner APIs in V1. Do not aggressively rate-limit public redirects in V1.

Default owner limit:

```text
60 requests per owner per minute
```

## Why

- Owner APIs are authenticated, so we can rate-limit by owner ID instead of only IP.
- Redis makes the limiter shared across API instances.
- Fixed window is simple and sufficient for a first production-shaped implementation.
- Public redirects may have legitimate event spikes, so blunt origin-level limits could break real QR campaigns.

## Failure Policy

If Redis is unavailable, the owner API limiter fails open and increments a metric:

```text
owner_rate_limit_failures_total
```

This keeps management APIs usable during cache/rate-limit backend issues. The trade-off is weaker abuse protection while Redis is unavailable.

## Metrics

- `owner_rate_limited_total`
- `owner_rate_limit_failures_total`

## Trade-Offs

- Fixed window can allow short bursts around window boundaries.
- Token bucket or sliding window is smoother but more complex.
- Edge/WAF limits are cheaper for abusive traffic but cannot see owner ID after authentication.
- Redirect rate limiting should be handled carefully with CDN/WAF and campaign-specific expectations.

## Interview Answer

I rate-limit owner APIs first because they are authenticated and abuse-prone. I avoid aggressive redirect rate limits because a marketing campaign can produce legitimate spikes. Redis gives a distributed limiter across API instances. I use fixed window in V1 for simplicity, and I would move to token bucket or sliding window if burst smoothness becomes important.
