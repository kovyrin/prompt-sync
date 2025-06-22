package system

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitScaffoldsFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Build the CLI binary in a temporary location so we can run it in an isolated dir.
	binaryPath := filepath.Join(tempDir, "prompt-sync-test-bin")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/prompt-sync")
	buildCmd.Dir = filepath.Join("..", "..", "..") // project root relative to internal/test/system
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build prompt-sync binary: %v\n%s", err, output)
	}

	cmd := exec.Command(binaryPath, "init")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("prompt-sync init failed: %v\nOutput: %s", err, output)
	}

	// Verify Promptsfile exists
	if _, err := os.Stat(filepath.Join(tempDir, "Promptsfile")); err != nil {
		t.Fatalf("Promptsfile not created: %v", err)
	}

	// Verify .gitignore has managed block
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf(".gitignore not created: %v", err)
	}
	if !strings.Contains(string(data), "BEGIN prompt-sync") {
		t.Fatalf(".gitignore missing managed block:\n%s", data)
	}

	// Idempotency: running init again without --force should fail.
	cmd2 := exec.Command(binaryPath, "init")
	cmd2.Dir = tempDir
	if err := cmd2.Run(); err == nil {
		t.Fatalf("expected init to fail when files already exist, but it succeeded")
	}

	// Now run with --force, expecting success.
	cmd3 := exec.Command(binaryPath, "init", "--force")
	cmd3.Dir = tempDir
	if output, err := cmd3.CombinedOutput(); err != nil {
		t.Fatalf("init with --force failed: %v\nOutput: %s", err, output)
	}
}
