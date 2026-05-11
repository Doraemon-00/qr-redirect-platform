# 001: Token Generation

## Context

The system needs public route tokens for QR codes. The product target includes up to 1B QR codes, and scanners are expected to scan the QR rather than manually type the token.

Plain `SHA256(target_url)` is deterministic. It would produce the same token for the same URL, which breaks use cases where different owners or campaigns need independent QR codes pointing to the same target URL.

## Options

1. `SHA256(target_url)` truncated to Base62.
2. `SHA256(target_url + owner_id + nonce)` truncated to Base62.
3. Random Base62 token with DB unique constraint and retry.
4. Sequential ID encoded as Base62.

## Decision

Use a random 12-character Base62 token in V1.

## Why

- Same URL can create multiple independent QR codes.
- No dependency on auth being present during token generation.
- Tokens are not predictable.
- Implementation is simple.
- DB unique constraint provides correctness.

## Collision Math

Base62 alphabet size:

```text
26 lowercase + 26 uppercase + 10 digits = 62
```

Keyspace:

```text
7 chars:  62^7  = 3.52e12
10 chars: 62^10 = 8.39e17
12 chars: 62^12 = 3.22e21
```

Approximate probability of at least one collision:

```text
1 - exp(-(n^2 / 2N))
```

For 1B QR codes and 12-char Base62:

```text
n = 1e9
N = 3.22e21
n^2 / 2N ~= 0.000155
collision probability ~= 0.0155%
```

The probability is low, but not zero. PostgreSQL unique constraint and retry are still required.

## Trade-Offs

- Longer tokens are less memorable.
- For QR-first scanning, memorability is secondary.
- Custom aliases are intentionally excluded from V1.
- Sequential IDs avoid random collision but are predictable unless obfuscated.

## Interview Answer

I avoided plain SHA-256 of the URL because it is deterministic and prevents multiple independent QR codes for the same destination. I use a 12-character random Base62 token because the QR is scanned, not manually typed, and the keyspace is large enough for a 1B-code target. A unique DB constraint plus retry gives correctness.
