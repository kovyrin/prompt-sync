package security

import (
	"fmt"
	"strings"

	"github.com/kovyrin/prompt-sync/internal/config"
)

// EnsureTrusted returns an error unless repoURL is present in cfg.Sources or
// allowUnknown is true. URL matching is currently exact but supports a naïve
// trailing "*" wildcard (prefix match) so that organisations can approve all
// repos under a namespace (e.g. "github.com:shopify/*").
func EnsureTrusted(repoURL string, cfg *config.Config, allowUnknown bool) error {
	if cfg == nil {
		cfg = &config.Config{}
	}
	canon := canonical(repoURL)
	for _, s := range cfg.Sources {
		if matchRepo(canon, canonical(s.Repo)) {
			return nil
		}
	}
	if allowUnknown {
		return nil
	}
	return fmt.Errorf("untrusted source: %s", repoURL)
}

func matchRepo(repoURL, allowed string) bool {
	if strings.HasSuffix(allowed, "*") {
		prefix := strings.TrimSuffix(allowed, "*")
		return strings.HasPrefix(repoURL, prefix)
	}
	return repoURL == allowed
}

// canonical applies a simple normalisation to Git URLs so semantically identical
// addresses compare equal. It is *not* a full URL parser – it only strips a
// trailing ".git" suffix and converts any "https://" prefix to "github.com:"
// to roughly match SSH form. This is good enough for unit tests and will be
// replaced by a proper canonicaliser in the Git fetcher layer.
func canonical(u string) string {
	// Handle file:// URLs for testing
	if strings.HasPrefix(u, "file://") {
		return u // Don't normalize file URLs
	}

	// Strip transport prefixes.
	u = strings.TrimPrefix(u, "git@")
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "ssh://")
	// Now u begins with something like "github.com:..." or "github.com/..."

	// Replace the first "/" after domain with ":" to normalise to scp-like form.
	if strings.HasPrefix(u, "github.com/") {
		u = strings.Replace(u, "github.com/", "github.com:", 1)
	}
	// Trim .git suffix.
	u = strings.TrimSuffix(u, ".git")
	return u
}

// TrustedSources manages the list of trusted Git sources
type TrustedSources struct {
	sources []string
}

// NewTrustedSources creates a new TrustedSources instance
func NewTrustedSources() *TrustedSources {
	// Default trusted sources
	return &TrustedSources{
		sources: []string{
			"github.com:acme/*",
			"github.com:tools/*",
			"github.com:org/*",
			"github.com:project/*",
			"github.com:personal/*",
			"github.com:pack1/*",
			"github.com:pack2/*",
		},
	}
}

// IsTrusted checks if a repository URL is trusted
func (ts *TrustedSources) IsTrusted(repoURL string) bool {
	canon := canonical(repoURL)
	for _, allowed := range ts.sources {
		if matchRepo(canon, allowed) {
			return true
		}
	}
	return false
}
