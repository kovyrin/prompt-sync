package git

import (
	"os"
)

// Backend represents the type of Git implementation to use.
type Backend string

const (
	// BackendGoGit uses the pure-Go go-git library.
	BackendGoGit Backend = "go-git"
	// BackendExec uses the system git binary.
	BackendExec Backend = "exec"
	// BackendAuto automatically selects the best backend.
	BackendAuto Backend = "auto"
)

// NewFetcherWithBackend creates a GitFetcher with the specified backend.
func NewFetcherWithBackend(backend Backend, options ...Option) Fetcher {
	// Allow environment override
	if envBackend := os.Getenv("PROMPT_SYNC_GIT_BACKEND"); envBackend != "" {
		switch envBackend {
		case "go-git":
			backend = BackendGoGit
		case "exec":
			backend = BackendExec
		}
	}

	switch backend {
	case BackendExec:
		return NewExecFetcher(options...)
	case BackendGoGit:
		return NewFetcher(options...)
	case BackendAuto:
		// For now, default to go-git for better portability
		// In the future, could check repo size or other heuristics
		return NewFetcher(options...)
	default:
		return NewFetcher(options...)
	}
}
