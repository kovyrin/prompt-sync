package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ExtendedConfig represents the full Promptsfile configuration
type ExtendedConfig struct {
	Sources  []string    `yaml:"sources"`
	Overlays []Overlay   `yaml:"overlays"`
	Adapters AdaptersCfg `yaml:"adapters"`
}

// Overlay represents a prompt pack with a specific scope
type Overlay struct {
	Scope  string `yaml:"scope"`
	Source string `yaml:"source"`
}

// AdaptersCfg holds configuration for all adapters
type AdaptersCfg struct {
	Cursor CursorCfg `yaml:"cursor"`
	Claude ClaudeCfg `yaml:"claude"`
}

// CursorCfg holds Cursor-specific configuration
type CursorCfg struct {
	Enabled bool `yaml:"enabled"`
}

// ClaudeCfg holds Claude-specific configuration
type ClaudeCfg struct {
	Enabled bool   `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`
}

// Loader handles configuration loading
type Loader struct {
	workspaceDir string
}

// NewLoader creates a new configuration loader
func NewLoader(workspaceDir string) *Loader {
	return &Loader{workspaceDir: workspaceDir}
}

// Load reads and parses the Promptsfile
func (l *Loader) Load() (*ExtendedConfig, error) {
	promptsfilePath := filepath.Join(l.workspaceDir, "Promptsfile")

	data, err := os.ReadFile(promptsfilePath)
	if err != nil {
		return nil, err
	}

	var cfg ExtendedConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if !cfg.Adapters.Cursor.Enabled && !cfg.Adapters.Claude.Enabled {
		// If no adapters are explicitly configured, enable Cursor by default
		cfg.Adapters.Cursor.Enabled = true
	}

	return &cfg, nil
}
