package git

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// execFetcher uses the system git binary for operations.
// This is useful for large repos where go-git might be slower.
type execFetcher struct {
	options Options
}

// NewExecFetcher creates a GitFetcher that shells out to system git.
func NewExecFetcher(options ...Option) Fetcher {
	opts := Options{}
	for _, opt := range options {
		opt(&opts)
	}
	opts.CacheDir = ResolveCacheDir(opts.CacheDir)
	return &execFetcher{options: opts}
}

// repoPath generates a deterministic local path for a repo URL.
func (f *execFetcher) repoPath(repoURL string) string {
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

// runGit runs a git command and returns the output.
func (f *execFetcher) runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Clone fetches a repository at the given ref and returns the local path.
func (f *execFetcher) Clone(repoURL, ref string) (string, error) {
	repoPath := f.repoPath(repoURL)

	// Check if already cloned
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Repository exists, fetch and checkout the ref
		if !f.options.Offline {
			// Fetch latest
			if _, err := f.runGit(repoPath, "fetch", "--all", "--tags"); err != nil {
				// Ignore fetch errors, might still work with local refs
			}
		}

		// Checkout the ref
		if _, err := f.runGit(repoPath, "checkout", ref); err != nil {
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

	// Clone the repository with shallow clone for performance
	args := []string{"clone", "--depth", "1"}

	// If ref is a branch, clone it directly
	if !isCommitHash(ref) && !strings.HasPrefix(ref, "v") {
		args = append(args, "--branch", ref)
	}

	args = append(args, repoURL, repoPath)

	if _, err := f.runGit("", args...); err != nil {
		// Clean up partial clone
		os.RemoveAll(repoPath)

		// Try full clone if shallow clone failed
		args = []string{"clone", repoURL, repoPath}
		if _, err := f.runGit("", args...); err != nil {
			return "", fmt.Errorf("clone repository: %w", err)
		}
	}

	// Fetch all tags
	if _, err := f.runGit(repoPath, "fetch", "--tags"); err != nil {
		// Ignore tag fetch errors
	}

	// Checkout the specific ref if needed
	if _, err := f.runGit(repoPath, "checkout", ref); err != nil {
		// If checkout fails, try fetching the ref first
		if _, err := f.runGit(repoPath, "fetch", "origin", ref); err == nil {
			if _, err := f.runGit(repoPath, "checkout", ref); err != nil {
				// Clean up failed clone
				os.RemoveAll(repoPath)
				return "", fmt.Errorf("checkout ref %s: %w", ref, err)
			}
		} else {
			// Clean up failed clone
			os.RemoveAll(repoPath)
			return "", fmt.Errorf("checkout ref %s: %w", ref, err)
		}
	}

	return repoPath, nil
}

// Update pulls the latest changes for a repository at the given ref.
func (f *execFetcher) Update(repoURL, ref string) error {
	if f.options.Offline {
		return fmt.Errorf("offline mode: cannot update")
	}

	repoPath := f.repoPath(repoURL)

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return fmt.Errorf("repository not cloned")
	}

	// Fetch latest changes
	if _, err := f.runGit(repoPath, "fetch", "--all", "--tags"); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	// Checkout the ref to get latest
	if _, err := f.runGit(repoPath, "checkout", ref); err != nil {
		// Try to reset to origin/ref if it's a branch
		if _, err := f.runGit(repoPath, "reset", "--hard", "origin/"+ref); err != nil {
			return fmt.Errorf("checkout ref %s: %w", ref, err)
		}
	}

	return nil
}

// CachedPath returns the local path for a cached repository, if it exists.
func (f *execFetcher) CachedPath(repoURL, ref string) (string, bool) {
	repoPath := f.repoPath(repoURL)

	// Check if the repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return "", false
	}

	// Verify we can get the current ref
	if _, err := f.runGit(repoPath, "rev-parse", "HEAD"); err != nil {
		return "", false
	}

	return repoPath, true
}

// CloneOrUpdate fetches a repository at the given ref and returns the local path and commit hash.
func (f *execFetcher) CloneOrUpdate(repoURL, ref string) (string, string, error) {
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
	commitHash, err := f.runGit(path, "rev-parse", "HEAD")
	if err != nil {
		return path, "", fmt.Errorf("get HEAD commit: %w", err)
	}

	return path, commitHash, nil
}
