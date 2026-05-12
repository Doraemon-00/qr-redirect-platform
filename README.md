# High-Performance Dynamic QR Redirect Platform

Production-shaped backend project for dynamic QR codes, short redirects, cache strategy, async analytics, and measurable performance.

## Goal

This project intentionally fills the gap between a tutorial prototype and a production backend:

- Low-latency public redirect path.
- Owner-only QR management APIs.
- Redis-backed redirect cache.
- Async scan analytics pipeline.
- PostgreSQL for QR metadata.
- ClickHouse for high-volume analytics.
- Prometheus metrics and k6 load tests.
- Architecture decision records explaining trade-offs.

## Core API

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/api/qr/create` | API key | Create a short URL and QR code |
| `GET` | `/r/{token}` | Public | Redirect to current target URL |
| `GET` | `/api/qr/{token}` | API key | Get QR metadata |
| `PATCH` | `/api/qr/{token}` | API key | Update target URL or expiration |
| `DELETE` | `/api/qr/{token}` | API key | Soft delete QR code |
| `GET` | `/api/qr/{token}/image` | Public | Generate QR PNG |
| `GET` | `/api/qr/{token}/analytics` | API key | Scan count and daily breakdown |
| `GET` | `/healthz` | Public | Liveness |
| `GET` | `/readyz` | Public | Readiness |
| `GET` | `/metrics` | Public | Prometheus metrics |

## Stack

- Go API service.
- PostgreSQL for OLTP QR metadata.
- Redis for redirect cache and event stream.
- ClickHouse for OLAP scan analytics.
- Docker Compose for local infrastructure.
- Prometheus metrics.
- k6 load tests.
- Redis-backed owner API rate limiting.

## Current Scope

V1 deliberately excludes custom aliases, full login UI, S3 image storage, and frontend. The product assumption is QR-first: scanners scan QR images; owners manage QR codes through authenticated APIs.

## Docs

- [Product Requirements](docs/product-requirements.md)
- [Architecture](docs/architecture.md)
- [Performance Plan](docs/performance-plan.md)
- [Decision Records](docs/decision-records)

## Local Run

Prerequisites:

- Docker Desktop running with Linux containers.

Build and start:

```bash
docker compose up --build
```

Create a QR code:

```bash
curl -X POST http://localhost:8080/api/qr/create \
  -H "Authorization: Bearer qk_demo_local_dev_key" \
  -H "Content-Type: application/json" \
  -d '{"targetUrl":"https://example.com"}'
```

Check metrics:

```bash
curl http://localhost:8080/metrics
```

Run redirect smoke test after creating a token:

```bash
k6 run -e TOKEN=<created-token> load-tests/redirect.k6.js
```
