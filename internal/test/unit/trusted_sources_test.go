package unit

import (
	"testing"

	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/security"
)

func TestEnsureTrusted(t *testing.T) {
	cfg := &config.Config{Sources: []config.Source{{Name: "shopify", Repo: "git@github.com:shopify/ai-prompts.git"}}}

	t.Run("allows trusted repo", func(t *testing.T) {
		if err := security.EnsureTrusted("git@github.com:shopify/ai-prompts.git", cfg, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects unknown repo", func(t *testing.T) {
		if err := security.EnsureTrusted("git@github.com:evil/hack.git", cfg, false); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("allows unknown repo with override", func(t *testing.T) {
		if err := security.EnsureTrusted("git@github.com:evil/hack.git", cfg, true); err != nil {
			t.Fatalf("expected nil error with allowUnknown, got %v", err)
		}
	})

	t.Run("allows wildcard prefix", func(t *testing.T) {
		cfg := &config.Config{Sources: []config.Source{{Name: "shopify", Repo: "github.com:shopify/*"}}}
		if err := security.EnsureTrusted("github.com:shopify/ai-prompts.git", cfg, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects wildcard miss", func(t *testing.T) {
		cfg := &config.Config{Sources: []config.Source{{Name: "shopify", Repo: "github.com:shopify/*"}}}
		if err := security.EnsureTrusted("github.com:evil/ai-prompts.git", cfg, false); err == nil {
			t.Fatalf("expected error for non-matching wildcard, got nil")
		}
	})

	t.Run("allows HTTPS vs SSH equivalence", func(t *testing.T) {
		cfg := &config.Config{Sources: []config.Source{{Name: "shopify", Repo: "git@github.com:shopify/ai-prompts.git"}}}
		if err := security.EnsureTrusted("https://github.com/shopify/ai-prompts.git", cfg, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
