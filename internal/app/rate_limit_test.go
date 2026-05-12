package app

import (
	"testing"
	"time"
)

func TestOwnerRateLimitKey(t *testing.T) {
	now := time.Date(2026, 5, 12, 10, 15, 42, 0, time.UTC)
	key, resetAt := ownerRateLimitKey("owner-1", now)

	if key != "rate:owner:owner-1:1778580900" {
		t.Fatalf("got key %q", key)
	}

	wantReset := time.Date(2026, 5, 12, 10, 16, 0, 0, time.UTC)
	if !resetAt.Equal(wantReset) {
		t.Fatalf("got reset %s, want %s", resetAt, wantReset)
	}
}
