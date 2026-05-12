package app

import "testing"

func TestBoolEnv(t *testing.T) {
	t.Run("uses fallback when unset", func(t *testing.T) {
		t.Setenv("BOOL_ENV_TEST", "")
		if got := boolEnv("BOOL_ENV_TEST", true); !got {
			t.Fatalf("got false, want fallback true")
		}
	})

	t.Run("parses false", func(t *testing.T) {
		t.Setenv("BOOL_ENV_TEST", "false")
		if got := boolEnv("BOOL_ENV_TEST", true); got {
			t.Fatalf("got true, want false")
		}
	})

	t.Run("uses fallback for invalid value", func(t *testing.T) {
		t.Setenv("BOOL_ENV_TEST", "nope")
		if got := boolEnv("BOOL_ENV_TEST", true); !got {
			t.Fatalf("got false, want fallback true")
		}
	})
}
