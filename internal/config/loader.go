package config

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// Source represents a trusted prompt source declared in configuration files.
// Only the fields required for tasks 2.* are included here.
type Source struct {
	Name         string `yaml:"name"`
	Repo         string `yaml:"repo"`
	ClaudePrefix string `yaml:"claude_prefix,omitempty"`
}

// Config aggregates all prompt-sync configuration that the application cares
// about at load-time. For the MVP, we only expose the list of trusted sources.
type Config struct {
	Sources []Source
}

// Load reads configuration from the following locations (lowest precedence → highest):
//  1. User-level config (~/.prompt-sync/config.yaml or path from $PROMPT_SYNC_USER_CONFIG)
//  2. Project Promptsfile (<projectDir>/Promptsfile)
//  3. Local overrides (<projectDir>/Promptsfile.local)
//
// Later files override earlier ones when they declare a source with the same
// name. Duplicate names are considered the same logical source – the entry
// appearing later in the precedence chain wins.
func Load(projectDir string) (*Config, error) {
	paths := []string{userConfigPath(), filepath.Join(projectDir, "Promptsfile"), filepath.Join(projectDir, "Promptsfile.local")}

	sourceMap := make(map[string]Source)
	for _, p := range paths {
		if err := readSourcesFromFile(p, sourceMap); err != nil {
			return nil, err
		}
	}

	// Convert map → slice with predictable order (sorted by name for stability)
	var cfg Config
	for _, src := range sourceMap {
		cfg.Sources = append(cfg.Sources, src)
	}
	// Stable ordering to avoid nondeterministic test failures.
	sort.Slice(cfg.Sources, func(i, j int) bool { return cfg.Sources[i].Name < cfg.Sources[j].Name })

	return &cfg, nil
}

// userConfigPath resolves the user-level configuration path, allowing override
// via $PROMPT_SYNC_USER_CONFIG for easier testing.
func userConfigPath() string {
	if p := os.Getenv("PROMPT_SYNC_USER_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // will be ignored by the caller
	}
	return filepath.Join(home, ".prompt-sync", "config.yaml")
}

// readSourcesFromFile parses a YAML config file and merges its sources into dst.
// Missing files are silently ignored so tests don't need to create every file.
func readSourcesFromFile(path string, dst map[string]Source) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errorsIsNotExist(err) {
			return nil // ignored
		}
		return err
	}
	var parsed struct {
		Sources []Source `yaml:"sources"`
	}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return err
	}
	for _, s := range parsed.Sources {
		dst[s.Name] = s // higher precedence overwrites
	}
	return nil
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}
