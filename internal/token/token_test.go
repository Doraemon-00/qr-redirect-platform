package token

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	got, err := Generate(DefaultLength)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(got) != DefaultLength {
		t.Fatalf("got length %d, want %d", len(got), DefaultLength)
	}
	for _, r := range got {
		if !strings.ContainsRune(Alphabet, r) {
			t.Fatalf("token contains non-base62 rune %q in %q", r, got)
		}
	}
}

func TestGenerateUsesDefaultLengthForInvalidLength(t *testing.T) {
	got, err := Generate(0)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(got) != DefaultLength {
		t.Fatalf("got length %d, want %d", len(got), DefaultLength)
	}
}
