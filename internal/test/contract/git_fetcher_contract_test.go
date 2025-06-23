package contract

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/git"
)

// createTestRepoWithCommit creates a local test repo with a known commit hash
func createTestRepoWithCommit(t *testing.T) (repoPath string, commitHash string) {
	t.Helper()

	// Create a bare repo
	bareDir := filepath.Join(t.TempDir(), "test-repo.git")
	cmd := exec.Command("git", "init", "--bare", bareDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create bare repo: %v\n%s", err, output)
	}

	// Create a working directory
	workDir := filepath.Join(t.TempDir(), "test-repo-work")
	cmd = exec.Command("git", "clone", bareDir, workDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone repo: %v\n%s", err, output)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	cmd.Run()

	// Create a file and commit
	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to add files: %v\n%s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Test commit")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to commit: %v\n%s", err, output)
	}

	// Get the commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get commit hash: %v", err)
	}
	commitHash = strings.TrimSpace(string(output)) // Full hash, trim newline

	// Push to bare repo
	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to push: %v\n%s", err, output)
	}

	return bareDir, commitHash
}

// TestGitFetcherContract defines the expected behavior for any GitFetcher
// implementation. This test can be run against multiple backends (go-git, exec).
func TestGitFetcherContract(t *testing.T) {
	backends := []struct {
		name    string
		backend git.Backend
	}{
		{"go-git", git.BackendGoGit},
		{"exec", git.BackendExec},
	}

	for _, tc := range backends {
		t.Run(tc.name, func(t *testing.T) {
			testFetcherContract(t, tc.backend)
		})
	}
}

func testFetcherContract(t *testing.T, backend git.Backend) {
	fetcher := git.NewFetcherWithBackend(git.Options{
		CacheDir: t.TempDir(),
	}, backend)

	t.Run("Clone creates local cache", func(t *testing.T) {
		repoURL := "https://github.com/golang/example.git"
		ref := "master"

		path, err := fetcher.Clone(repoURL, ref)
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		// Verify the path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Clone did not create expected path: %s", path)
		}

		// Verify it's a git repository
		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			t.Fatalf("Clone did not create a git repository at: %s", path)
		}
	})

	t.Run("CachedPath returns existing clone", func(t *testing.T) {
		repoURL := "https://github.com/golang/example.git"
		ref := "master"

		// First clone
		path1, err := fetcher.Clone(repoURL, ref)
		if err != nil {
			t.Fatalf("Initial clone failed: %v", err)
		}

		// Check cached path
		path2, exists := fetcher.CachedPath(repoURL, ref)
		if !exists {
			t.Fatalf("CachedPath returned false for existing clone")
		}
		if path1 != path2 {
			t.Fatalf("CachedPath returned different path: got %s, want %s", path2, path1)
		}
	})

	t.Run("Update fetches latest changes", func(t *testing.T) {
		repoURL := "https://github.com/golang/example.git"
		ref := "master"

		// Clone first
		_, err := fetcher.Clone(repoURL, ref)
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		// Update should succeed (even if no new commits)
		if err := fetcher.Update(repoURL, ref); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
	})

	t.Run("Clone with specific commit", func(t *testing.T) {
		// Create a local test repo with a known commit
		repoURL, commitHash := createTestRepoWithCommit(t)
		ref := commitHash

		path, err := fetcher.Clone(repoURL, ref)
		if err != nil {
			t.Fatalf("Clone with commit failed: %v", err)
		}

		// Verify the path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Clone did not create expected path: %s", path)
		}

		// Verify it's a git repository
		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			t.Fatalf("Clone did not create a git repository at: %s", path)
		}
	})

	t.Run("Offline mode prevents network access", func(t *testing.T) {
		offlineFetcher := git.NewFetcherWithBackend(git.Options{
			CacheDir: t.TempDir(),
			Offline:  true,
		}, backend)

		repoURL := "https://github.com/some/nonexistent-repo.git"
		ref := "main"

		_, err := offlineFetcher.Clone(repoURL, ref)
		if err == nil {
			t.Fatalf("Expected error in offline mode for non-cached repo")
		}
	})
}
