package git

import (
	"os"
	"path/filepath"
)

// Option is a function that configures a Fetcher
type Option func(*Options)

// WithCacheDir sets a custom cache directory
func WithCacheDir(dir string) Option {
	return func(o *Options) {
		o.CacheDir = dir
	}
}

// WithOfflineMode enables offline mode
func WithOfflineMode() Option {
	return func(o *Options) {
		o.Offline = true
	}
}

// ResolveCacheDir determines the cache directory to use based on:
// 1. Explicit cacheDir parameter (highest priority)
// 2. PROMPT_SYNC_CACHE_DIR environment variable
// 3. Default: $HOME/.prompt-sync/repos
func ResolveCacheDir(cacheDir string) string {
	if cacheDir != "" {
		return cacheDir
	}

	if envDir := os.Getenv("PROMPT_SYNC_CACHE_DIR"); envDir != "" {
		return envDir
	}

	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prompt-sync", "repos")
}
