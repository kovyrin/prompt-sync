package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/config"
)

func TestLoadConfig_Precedence(t *testing.T) {
	tempDir := t.TempDir()

	// User config path (override via env var for isolation).
	userConfigPath := filepath.Join(tempDir, "user.yaml")
	os.Setenv("PROMPT_SYNC_USER_CONFIG", userConfigPath)
	t.Cleanup(func() { os.Unsetenv("PROMPT_SYNC_USER_CONFIG") })

	writeFile(t, userConfigPath, `sources:
  - name: user
    repo: git@github.com:user/prompts.git
  - name: shared
    repo: git@github.com:user/shared-prompts.git
`)

	// Project Promptsfile with an overriding "shared" source and a new one.
	promptsfilePath := filepath.Join(tempDir, "Promptsfile")
	writeFile(t, promptsfilePath, `sources:
  - name: project
    repo: git@github.com:proj/prompts.git
  - name: shared
    repo: git@github.com:proj/shared-prompts.git
`)

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := map[string]string{
		"user":    "git@github.com:user/prompts.git",
		"project": "git@github.com:proj/prompts.git",
		"shared":  "git@github.com:proj/shared-prompts.git", // project overrides user
	}
	if len(cfg.Sources) != len(want) {
		// Provide helpful diff.
		t.Fatalf("unexpected number of sources: got %d want %d", len(cfg.Sources), len(want))
	}
	for _, s := range cfg.Sources {
		if repo, ok := want[s.Name]; !ok || repo != s.Repo {
			t.Errorf("source %s repo mismatch: got %s want %s", s.Name, s.Repo, want[s.Name])
		}
		delete(want, s.Name)
	}
	if len(want) != 0 {
		t.Errorf("missing expected sources: %v", want)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
