package config

import (
	"encoding/hex"
	"testing"
)

func TestLoadConfigSessionSecret(t *testing.T) {
	t.Run("missing secret is unpredictable", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "")
		first := LoadConfig().SessionSecret
		second := LoadConfig().SessionSecret
		if first == "secret" || first == second || len(first) != 64 {
			t.Fatalf("missing secret produced an unsafe fallback: first=%q second=%q", first, second)
		}
		if _, err := hex.DecodeString(first); err != nil {
			t.Fatalf("fallback is not a 32-byte hex key: %v", err)
		}
	})

	t.Run("configured secret is preserved", func(t *testing.T) {
		t.Setenv("SESSION_SECRET", "configured-secret")
		if got := LoadConfig().SessionSecret; got != "configured-secret" {
			t.Fatalf("configured secret changed: %q", got)
		}
	})
}
