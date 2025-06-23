package git

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// Options configures a GitFetcher instance.
type Options struct {
	CacheDir string // Base directory for git cache (default: $HOME/.prompt-sync/repos)
	Offline  bool   // If true, only use cached repos (no network access)
}

// Fetcher defines the interface for Git repository operations.
// Implementations may use go-git, exec git, or other backends.
type Fetcher interface {
	// Clone fetches a repository at the given ref and returns the local path.
	// If already cached, it may return the existing path.
	Clone(repoURL, ref string) (string, error)

	// Update pulls the latest changes for a repository at the given ref.
	// Returns error if the repo is not already cloned.
	Update(repoURL, ref string) error

	// CachedPath returns the local path for a cached repository, if it exists.
	// The bool indicates whether the repo is cached.
	CachedPath(repoURL, ref string) (string, bool)

	// CloneOrUpdate fetches a repository at the given ref and returns the local path and commit hash.
	// If already cached, it updates and returns the existing path.
	CloneOrUpdate(repoURL, ref string) (path string, commit string, err error)
}

// fetcher is the go-git implementation of the Fetcher interface.
type fetcher struct {
	options Options
}

// NewFetcher creates a new GitFetcher with the given options.
func NewFetcher(options ...Option) Fetcher {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}
	opts.CacheDir = ResolveCacheDir(opts.CacheDir)
	return &fetcher{options: opts}
}

// repoPath generates a deterministic local path for a repo URL.
func (f *fetcher) repoPath(repoURL string) string {
	// Create a hash of the URL for the directory name
	h := sha256.Sum256([]byte(repoURL))
	hash := hex.EncodeToString(h[:])[:12]

	// Extract a human-readable name from the URL
	name := repoURL
	name = strings.TrimSuffix(name, ".git")
	parts := strings.Split(name, "/")
	if len(parts) >= 2 {
		name = parts[len(parts)-2] + "-" + parts[len(parts)-1]
	}

	// Clean up the name
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "@", "-")

	return filepath.Join(f.options.CacheDir, name+"-"+hash)
}

// Clone fetches a repository at the given ref and returns the local path.
func (f *fetcher) Clone(repoURL, ref string) (string, error) {
	repoPath := f.repoPath(repoURL)

	// Check if already cloned
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Repository exists, checkout the ref
		repo, err := git.PlainOpen(repoPath)
		if err != nil {
			return "", fmt.Errorf("open existing repo: %w", err)
		}

		if err := f.checkoutRef(repo, ref); err != nil {
			return "", fmt.Errorf("checkout ref %s: %w", ref, err)
		}

		return repoPath, nil
	}

	// In offline mode, we can't clone new repos
	if f.options.Offline {
		return "", fmt.Errorf("offline mode: repository not cached")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(f.options.CacheDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	// Clone the repository
	cloneOpts := &git.CloneOptions{
		URL:      repoURL,
		Progress: nil, // Could add progress reporting later
	}

	// If ref looks like a branch, clone it directly
	if !strings.HasPrefix(ref, "v") && !isCommitHash(ref) {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ref)
	}

	repo, err := git.PlainClone(repoPath, false, cloneOpts)
	if err != nil {
		// Clean up partial clone
		os.RemoveAll(repoPath)
		return "", fmt.Errorf("clone repository: %w", err)
	}

	// Checkout the specific ref if needed
	if err := f.checkoutRef(repo, ref); err != nil {
		// Clean up failed clone
		os.RemoveAll(repoPath)
		return "", fmt.Errorf("checkout ref %s: %w", ref, err)
	}

	return repoPath, nil
}

// checkoutRef checks out a specific ref (branch, tag, or commit).
func (f *fetcher) checkoutRef(repo *git.Repository, ref string) error {
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	// Try as a tag first
	tagRef, err := repo.Tag(ref)
	if err == nil {
		return w.Checkout(&git.CheckoutOptions{
			Hash: tagRef.Hash(),
		})
	}

	// Try as a branch
	branchRef := plumbing.NewBranchReferenceName(ref)
	if _, err := repo.Reference(branchRef, false); err == nil {
		return w.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
		})
	}

	// Try as a commit hash
	if isCommitHash(ref) {
		return w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(ref),
		})
	}

	// Fetch all refs and try again
	if !f.options.Offline {
		remote, err := repo.Remote("origin")
		if err == nil {
			err = remote.Fetch(&git.FetchOptions{
				RefSpecs: []config.RefSpec{
					config.RefSpec("+refs/heads/*:refs/remotes/origin/*"),
					config.RefSpec("+refs/tags/*:refs/tags/*"),
				},
			})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				return fmt.Errorf("fetch refs: %w", err)
			}

			// Try tag again after fetch
			tagRef, err := repo.Tag(ref)
			if err == nil {
				return w.Checkout(&git.CheckoutOptions{
					Hash: tagRef.Hash(),
				})
			}
		}
	}

	return fmt.Errorf("ref not found: %s", ref)
}

// Update pulls the latest changes for a repository at the given ref.
func (f *fetcher) Update(repoURL, ref string) error {
	if f.options.Offline {
		return fmt.Errorf("offline mode: cannot update")
	}

	repoPath := f.repoPath(repoURL)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("repository not cloned: %w", err)
	}

	// Fetch latest changes
	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("get remote: %w", err)
	}

	err = remote.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/remotes/origin/*"),
			config.RefSpec("+refs/tags/*:refs/tags/*"),
		},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("fetch: %w", err)
	}

	// Checkout the ref again to get latest
	return f.checkoutRef(repo, ref)
}

// CachedPath returns the local path for a cached repository, if it exists.
func (f *fetcher) CachedPath(repoURL, ref string) (string, bool) {
	repoPath := f.repoPath(repoURL)

	// Check if the repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return "", false
	}

	// Verify the ref is checked out
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", false
	}

	// For now, just return true if the repo exists
	// Could verify the exact ref is checked out
	_ = repo

	return repoPath, true
}

// CloneOrUpdate fetches a repository at the given ref and returns the local path and commit hash.
func (f *fetcher) CloneOrUpdate(repoURL, ref string) (string, string, error) {
	// First try to clone (which will reuse existing if cached)
	path, err := f.Clone(repoURL, ref)
	if err != nil {
		return "", "", err
	}

	// If not offline and repo already existed, try to update
	if !f.options.Offline {
		if cached, _ := f.CachedPath(repoURL, ref); cached != "" {
			// Ignore update errors in case we're on a detached head
			_ = f.Update(repoURL, ref)
		}
	}

	// Get the current commit hash
	repo, err := git.PlainOpen(path)
	if err != nil {
		return path, "", fmt.Errorf("open repo to get commit: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return path, "", fmt.Errorf("get HEAD commit: %w", err)
	}

	return path, head.Hash().String(), nil
}

// isCommitHash checks if a string looks like a git commit hash.
func isCommitHash(s string) bool {
	if len(s) < 7 || len(s) > 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
