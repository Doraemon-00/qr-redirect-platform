# 003: QR Image Storage

## Context

The QR image encodes the short URL, not the original target URL. Updating the target URL should not require regenerating or redistributing the QR image.

## Options

1. Store QR PNG in PostgreSQL.
2. Store QR PNG in S3-compatible object storage.
3. Generate QR PNG on demand from token.

## Decision

Generate QR PNG on demand in V1 and use long HTTP cache headers.

## Why

- QR image is deterministic from `{PUBLIC_BASE_URL}/r/{token}`.
- No object storage lifecycle or cleanup needed.
- Avoids storing duplicate binary data.
- Target URL updates do not change the QR image.

## Trade-Offs

- API does CPU work when uncached QR image is requested.
- S3/CDN could be useful for very high image download volume.
- Redirect scan traffic is the hot path, not image generation.

## Interview Answer

I do not store QR images in V1 because the image is deterministic from the stable short URL. I generate it on demand and cache it aggressively. S3 is useful if image download volume becomes a product bottleneck, but it does not help the redirect hot path.
