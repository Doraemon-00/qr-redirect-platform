package app

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestParseScanEvent(t *testing.T) {
	message := redis.XMessage{
		ID: "1747022400000-0",
		Values: map[string]any{
			"token":           "abc123abc123",
			"scanned_at":      "2026-05-12T10:00:00.123456789Z",
			"user_agent_hash": "ua",
			"ip_hash":         "ip",
		},
	}

	got, ok := parseScanEvent(message)
	if !ok {
		t.Fatal("parseScanEvent returned false")
	}

	if got.StreamID != message.ID {
		t.Fatalf("got stream id %q, want %q", got.StreamID, message.ID)
	}
	if got.Token != "abc123abc123" {
		t.Fatalf("got token %q, want abc123abc123", got.Token)
	}
	wantScannedAt := time.Date(2026, 5, 12, 10, 0, 0, 123456789, time.UTC)
	if !got.ScannedAt.Equal(wantScannedAt) {
		t.Fatalf("got scanned at %s, want %s", got.ScannedAt, wantScannedAt)
	}
	if got.UserAgentHash != "ua" {
		t.Fatalf("got user agent hash %q, want ua", got.UserAgentHash)
	}
	if got.IPHash != "ip" {
		t.Fatalf("got ip hash %q, want ip", got.IPHash)
	}
}

func TestParseScanEventRejectsInvalidEvent(t *testing.T) {
	message := redis.XMessage{
		ID: "1747022400000-0",
		Values: map[string]any{
			"token":      "abc123abc123",
			"scanned_at": "not-a-time",
		},
	}

	if _, ok := parseScanEvent(message); ok {
		t.Fatal("parseScanEvent returned true for invalid event")
	}
}

func TestScanEventUUIDIsDeterministic(t *testing.T) {
	first := scanEventUUID("1747022400000-0")
	second := scanEventUUID("1747022400000-0")
	other := scanEventUUID("1747022400000-1")

	if first != second {
		t.Fatalf("same stream id produced different UUIDs: %s and %s", first, second)
	}
	if first == other {
		t.Fatalf("different stream ids produced same UUID: %s", first)
	}
}
