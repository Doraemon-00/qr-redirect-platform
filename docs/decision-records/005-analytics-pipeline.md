# 005: Analytics Pipeline

## Context

The product needs scan analytics, but redirect latency is the core user-facing SLA. Writing analytics synchronously during redirect couples a latency-sensitive path to a high-volume analytics workload.

## Options

1. Synchronously insert scan event into PostgreSQL during redirect.
2. Synchronously insert scan event into ClickHouse during redirect.
3. Enqueue scan event asynchronously and batch-write to analytics store.

## Decision

Use async scan event enqueue and a worker that writes to ClickHouse.

## Why

- Redirect path needs low latency.
- Analytics can be eventually consistent.
- Batch writes are more efficient for high event volume.
- Analytics failure should not break redirects.

## Trade-Offs

- Analytics may lag behind real time.
- Event loss policy must be explicit.
- Worker and queue add operational complexity.

## Failure Policy

If analytics enqueue fails, the redirect still succeeds. The system increments a metric and logs the failure. This chooses redirect availability over perfect analytics completeness.

## Interview Answer

Redirect and analytics have different SLAs. Redirect is latency-sensitive and must return quickly. Analytics is high-volume and can tolerate delay, so I decouple it with a queue/stream and batch writes to an OLAP store.
