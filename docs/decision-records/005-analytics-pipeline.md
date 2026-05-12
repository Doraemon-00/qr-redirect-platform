# 005: Analytics Pipeline

## Context

The product needs scan analytics, but redirect latency is the core user-facing SLA. Writing analytics synchronously during redirect couples a latency-sensitive path to a high-volume analytics workload.

## Options

1. Synchronously insert scan event into PostgreSQL during redirect.
2. Synchronously insert scan event into ClickHouse during redirect.
3. Enqueue scan event asynchronously and batch-write to analytics store.

## Decision

Use async scan event enqueue and a worker that writes to ClickHouse.

V1 runs the worker in the API process. It consumes Redis Stream events with a consumer group, writes batches to ClickHouse, and acknowledges stream IDs only after a successful insert. Pending messages idle for more than `ANALYTICS_RECLAIM_IDLE_SECONDS` are reclaimed with `XAUTOCLAIM`.

## Why

- Redirect path needs low latency.
- Analytics can be eventually consistent.
- Batch writes are more efficient for high event volume.
- Analytics failure should not break redirects.

## Trade-Offs

- Analytics may lag behind real time.
- Event loss policy must be explicit.
- Worker and queue add operational complexity.
- At-least-once delivery can retry events. V1 uses deterministic event IDs from Redis Stream IDs and unique-event analytics queries to avoid inflated counts from duplicate inserts.

## Failure Policy

If analytics enqueue fails, the redirect still succeeds. The system increments a metric and logs the failure. This chooses redirect availability over perfect analytics completeness.

If the worker fails after reading an event but before acknowledging it, another worker pass can reclaim the pending message after the idle timeout. If ClickHouse insert succeeds but Redis ACK fails, the event can be retried; analytics queries count unique event IDs to keep reports stable.

## Interview Answer

Redirect and analytics have different SLAs. Redirect is latency-sensitive and must return quickly. Analytics is high-volume and can tolerate delay, so I decouple it with a queue/stream and batch writes to an OLAP store.
