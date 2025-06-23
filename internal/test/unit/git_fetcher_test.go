package unit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/git"
)

// createTestRepo creates a bare git repository for testing.
func createTestRepo(t *testing.T, name string) string {
	t.Helper()

	repoDir := filepath.Join(t.TempDir(), name)

	// Create a bare repo
	cmd := exec.Command("git", "init", "--bare", repoDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create test repo: %v\n%s", err, output)
	}

	// Create a non-bare working directory to add content
	workDir := filepath.Join(t.TempDir(), name+"-work")
	cmd = exec.Command("git", "clone", repoDir, workDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone test repo: %v\n%s", err, output)
	}

	// Add a test file
	testFile := filepath.Join(workDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Configure git user for the test repo
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to configure git email: %v\n%s", err, output)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to configure git name: %v\n%s", err, output)
	}

	// Commit the file
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to add files: %v\n%s", err, output)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to commit: %v\n%s", err, output)
	}

	// Push to bare repo
	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to push: %v\n%s", err, output)
	}

	// Create a tag
	cmd = exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create tag: %v\n%s", err, output)
	}

	cmd = exec.Command("git", "push", "origin", "v1.0.0")
	cmd.Dir = workDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to push tag: %v\n%s", err, output)
	}

	return repoDir
}

func TestGitFetcher_LocalRepo(t *testing.T) {
	// Create a test repository
	testRepo := createTestRepo(t, "test-repo")

	fetcher := git.NewFetcher(
		git.WithCacheDir(t.TempDir()),
	)

	t.Run("Clone local repository", func(t *testing.T) {
		path, err := fetcher.Clone(testRepo, "master")
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		// Verify the clone exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Clone did not create expected path: %s", path)
		}

		// Verify it contains the expected file
		readmeFile := filepath.Join(path, "README.md")
		if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
			t.Fatalf("Clone did not contain expected README.md file")
		}
	})

	t.Run("Clone with tag", func(t *testing.T) {
		path, err := fetcher.Clone(testRepo, "v1.0.0")
		if err != nil {
			t.Fatalf("Clone with tag failed: %v", err)
		}

		// Verify the clone exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Clone did not create expected path: %s", path)
		}
	})

	t.Run("CachedPath returns true for cloned repo", func(t *testing.T) {
		// First clone
		path1, err := fetcher.Clone(testRepo, "master")
		if err != nil {
			t.Fatalf("Initial clone failed: %v", err)
		}

		// Check cached path
		path2, exists := fetcher.CachedPath(testRepo, "master")
		if !exists {
			t.Fatalf("CachedPath returned false for existing clone")
		}
		if path1 != path2 {
			t.Fatalf("CachedPath returned different path: got %s, want %s", path2, path1)
		}
	})

	t.Run("Offline mode uses cache", func(t *testing.T) {
		// First clone with online fetcher
		cacheDir := t.TempDir()
		onlineFetcher := git.NewFetcher(
			git.WithCacheDir(cacheDir),
		)
		path1, err := onlineFetcher.Clone(testRepo, "master")
		if err != nil {
			t.Fatalf("Online clone failed: %v", err)
		}

		// Now try offline mode with same cache dir
		offlineFetcher := git.NewFetcher(
			git.WithCacheDir(cacheDir),
			git.WithOfflineMode(),
		)

		path2, err := offlineFetcher.Clone(testRepo, "master")
		if err != nil {
			t.Fatalf("Offline clone of cached repo failed: %v", err)
		}

		if path1 != path2 {
			t.Fatalf("Offline clone returned different path: got %s, want %s", path2, path1)
		}
	})
}

func TestGitFetcher_CacheDir(t *testing.T) {
	testRepo := createTestRepo(t, "cache-test-repo")

	t.Run("Uses PROMPT_SYNC_CACHE_DIR env var", func(t *testing.T) {
		customCache := t.TempDir()
		os.Setenv("PROMPT_SYNC_CACHE_DIR", customCache)
		t.Cleanup(func() { os.Unsetenv("PROMPT_SYNC_CACHE_DIR") })

		fetcher := git.NewFetcher()
		path, err := fetcher.Clone(testRepo, "master")
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		// Verify the clone is in the custom cache dir
		if !strings.HasPrefix(path, customCache) {
			t.Fatalf("Clone path %s does not start with custom cache %s", path, customCache)
		}
	})

	t.Run("Explicit cache dir overrides env var", func(t *testing.T) {
		envCache := t.TempDir()
		explicitCache := t.TempDir()

		os.Setenv("PROMPT_SYNC_CACHE_DIR", envCache)
		t.Cleanup(func() { os.Unsetenv("PROMPT_SYNC_CACHE_DIR") })

		fetcher := git.NewFetcher(
			git.WithCacheDir(explicitCache),
		)
		path, err := fetcher.Clone(testRepo, "master")
		if err != nil {
			t.Fatalf("Clone failed: %v", err)
		}

		// Verify the clone is in the explicit cache dir, not env
		if !strings.HasPrefix(path, explicitCache) {
			t.Fatalf("Clone path %s does not start with explicit cache %s", path, explicitCache)
		}
		if strings.HasPrefix(path, envCache) {
			t.Fatalf("Clone path %s incorrectly uses env cache %s", path, envCache)
		}
	})
}
