package app

import (
	"testing"
	"time"
)

func TestRedirectCacheTTL(t *testing.T) {
	now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)

	t.Run("active without expiry uses default ttl", func(t *testing.T) {
		got := redirectCacheTTL(qrCode{}, now)
		if got != 10*time.Minute {
			t.Fatalf("got %s, want 10m", got)
		}
	})

	t.Run("active with far expiry caps at default ttl", func(t *testing.T) {
		expiresAt := now.Add(time.Hour)
		got := redirectCacheTTL(qrCode{ExpiresAt: &expiresAt}, now)
		if got != 10*time.Minute {
			t.Fatalf("got %s, want 10m", got)
		}
	})

	t.Run("active with near expiry caps at expiry", func(t *testing.T) {
		expiresAt := now.Add(90 * time.Second)
		got := redirectCacheTTL(qrCode{ExpiresAt: &expiresAt}, now)
		if got != 90*time.Second {
			t.Fatalf("got %s, want 90s", got)
		}
	})

	t.Run("expired token uses tombstone ttl", func(t *testing.T) {
		expiresAt := now.Add(-time.Second)
		got := redirectCacheTTL(qrCode{ExpiresAt: &expiresAt}, now)
		if got != 5*time.Minute {
			t.Fatalf("got %s, want 5m", got)
		}
	})

	t.Run("deleted token uses tombstone ttl", func(t *testing.T) {
		deletedAt := now.Add(-time.Second)
		got := redirectCacheTTL(qrCode{DeletedAt: &deletedAt}, now)
		if got != 5*time.Minute {
			t.Fatalf("got %s, want 5m", got)
		}
	})
}
