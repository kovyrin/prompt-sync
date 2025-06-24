package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kovyrin/prompt-sync/internal/adapter"
	"github.com/kovyrin/prompt-sync/internal/adapter/claude"
	"github.com/kovyrin/prompt-sync/internal/adapter/cursor"
	"github.com/kovyrin/prompt-sync/internal/config"
	"github.com/kovyrin/prompt-sync/internal/conflict"
	"github.com/kovyrin/prompt-sync/internal/git"
	"github.com/kovyrin/prompt-sync/internal/gitignore"
	"github.com/kovyrin/prompt-sync/internal/lock"
	"github.com/kovyrin/prompt-sync/internal/security"
)

// InstallOptions contains options for the install workflow
type InstallOptions struct {
	WorkspaceDir string
	StrictMode   bool
	VerifyOnly   bool
	Offline      bool
	CacheDir     string
	AllowUnknown bool
}

// Installer orchestrates the installation workflow
type Installer struct {
	opts             InstallOptions
	configLoader     *config.Loader
	gitFetcher       git.Fetcher
	gitignoreManager *gitignore.Manager
	lockWriter       *lock.Writer
	conflictDetector *conflict.Detector
	adapters         map[string]adapter.Adapter
	trustedSources   *security.TrustedSources
}

// New creates a new installer
func New(opts InstallOptions) (*Installer, error) {
	// Initialize components
	configLoader := config.NewLoader(opts.WorkspaceDir)

	// Initialize trusted sources
	trustedSources := security.NewTrustedSources()

	// Create git fetcher with options
	gitOpts := []git.Option{
		git.WithCacheDir(opts.CacheDir),
	}
	if opts.Offline {
		gitOpts = append(gitOpts, git.WithOfflineMode())
	}
	gitFetcher := git.NewFetcher(gitOpts...)

	// Initialize other components
	gitignoreManager := gitignore.New(opts.WorkspaceDir)
	lockWriter := lock.New(opts.WorkspaceDir)
	conflictDetector := conflict.New(opts.StrictMode)

	// Initialize adapters
	adapters := map[string]adapter.Adapter{
		"cursor": cursor.NewSimpleAdapter(),
		"claude": claude.NewSimpleAdapter(),
	}

	return &Installer{
		opts:             opts,
		configLoader:     configLoader,
		gitFetcher:       gitFetcher,
		gitignoreManager: gitignoreManager,
		lockWriter:       lockWriter,
		conflictDetector: conflictDetector,
		adapters:         adapters,
		trustedSources:   trustedSources,
	}, nil
}

// SetGitFetcher allows replacing the git fetcher (primarily for testing)
func (i *Installer) SetGitFetcher(fetcher git.Fetcher) {
	i.gitFetcher = fetcher
}

// Execute runs the installation workflow
func (i *Installer) Execute() error {
	// Load configuration
	cfg, err := i.configLoader.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// In verify mode, check if lock file exists
	if i.opts.VerifyOnly && !i.lockWriter.Exists() {
		return fmt.Errorf("lock file not found, run install first")
	}

	// Read existing lock file to track old files for cleanup
	oldLock, err := i.lockWriter.Read()
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	// Create a map of old files per source for efficient lookup
	oldFilesBySource := make(map[string][]lock.File)
	if oldLock != nil {
		for _, source := range oldLock.Sources {
			baseURL := strings.Split(source.URL, "#")[0]
			oldFilesBySource[baseURL] = source.Files
		}
	}

	// Validate sources against trusted list
	for _, source := range cfg.Sources {
		url := strings.Split(source, "#")[0] // Remove ref if present
		if !i.trustedSources.IsTrusted(url) && !i.opts.AllowUnknown {
			return fmt.Errorf("untrusted source: %s", url)
		}
	}

	// Process overlays if configured
	allSources := append([]string{}, cfg.Sources...)
	for _, overlay := range cfg.Overlays {
		url := strings.Split(overlay.Source, "#")[0]
		if !i.trustedSources.IsTrusted(url) && !i.opts.AllowUnknown {
			return fmt.Errorf("untrusted overlay source: %s", url)
		}
		allSources = append(allSources, overlay.Source)
	}

	// Clone/update repositories
	var lockSources []lock.Source
	renderedFiles := make(map[string]string) // path -> source URL

	for _, source := range allSources {
		parts := strings.Split(source, "#")
		url := parts[0]
		ref := ""
		if len(parts) > 1 {
			ref = parts[1]
		}

		// Clone or update the repository
		repoPath, commit, err := i.gitFetcher.CloneOrUpdate(url, ref)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %w", url, err)
		}

		// Process each enabled adapter
		var lockFiles []lock.File

		for name, adapterImpl := range i.adapters {
			if !i.isAdapterEnabled(cfg, name) {
				continue
			}

			adapterCfg := i.getAdapterConfig(cfg, name)

			// Discover prompt files
			files, err := adapterImpl.DiscoverFiles(repoPath)
			if err != nil {
				return fmt.Errorf("failed to discover files for %s: %w", name, err)
			}

			// Render files
			for _, file := range files {
				outputPath := adapterImpl.GetOutputPath(file, adapterCfg)
				fullOutputPath := filepath.Join(i.opts.WorkspaceDir, outputPath)

				// Track for conflict detection
				if existing, exists := renderedFiles[outputPath]; exists {
					if !i.opts.VerifyOnly {
						return fmt.Errorf("conflict: %s would be rendered by both %s and %s", outputPath, existing, url)
					}
				}
				renderedFiles[outputPath] = url

				if !i.opts.VerifyOnly {
					// Read file content
					content, err := os.ReadFile(filepath.Join(repoPath, file))
					if err != nil {
						return fmt.Errorf("failed to read %s: %w", file, err)
					}

					// Render the file
					rendered, err := adapterImpl.RenderFile(file, content, adapterCfg)
					if err != nil {
						return fmt.Errorf("failed to render %s: %w", file, err)
					}

					// Create output directory
					outputDir := filepath.Dir(fullOutputPath)
					if err := os.MkdirAll(outputDir, 0755); err != nil {
						return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
					}

					// Write rendered file
					if err := os.WriteFile(fullOutputPath, rendered, 0644); err != nil {
						return fmt.Errorf("failed to write %s: %w", fullOutputPath, err)
					}
				}

				// Calculate hash for lock file
				hash, err := i.lockWriter.CalculateFileHash(fullOutputPath)
				if err != nil {
					if i.opts.VerifyOnly && os.IsNotExist(err) {
						// File doesn't exist in verify mode
						hash = "missing"
					} else {
						return fmt.Errorf("failed to calculate hash for %s: %w", fullOutputPath, err)
					}
				}

				lockFiles = append(lockFiles, lock.File{
					Path: outputPath,
					Hash: hash,
				})
			}
		}

		// Clean up orphaned files if this source was previously installed
		if oldFiles, exists := oldFilesBySource[url]; exists && !i.opts.VerifyOnly {
			orphanedFiles := i.findOrphanedFiles(oldFiles, lockFiles)
			for _, orphan := range orphanedFiles {
				fullPath := filepath.Join(i.opts.WorkspaceDir, orphan)
				if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
					// Log warning but don't fail the entire operation
					fmt.Printf("Warning: could not remove orphaned file %s: %v\n", orphan, err)
				}
			}
		}

		lockSources = append(lockSources, lock.Source{
			URL:    url,
			Ref:    ref,
			Commit: commit,
			Files:  lockFiles,
		})
	}

	// Check for conflicts
	for name := range i.adapters {
		if !i.isAdapterEnabled(cfg, name) {
			continue
		}

		adapterCfg := i.getAdapterConfig(cfg, name)
		outputDir := i.adapters[name].GetBaseOutputDir(adapterCfg)
		fullOutputDir := filepath.Join(i.opts.WorkspaceDir, outputDir)

		if info, err := os.Stat(fullOutputDir); err == nil && info.IsDir() {
			issues, err := i.conflictDetector.ScanDirectory(fullOutputDir)
			if err != nil {
				return fmt.Errorf("failed to scan for conflicts: %w", err)
			}

			if len(issues) > 0 && i.opts.StrictMode {
				return fmt.Errorf("conflicts detected: %v", issues)
			}
		}
	}

	// In verify mode, check for drift
	if i.opts.VerifyOnly {
		existingHashes, err := i.lockWriter.GetFileHashes()
		if err != nil {
			return fmt.Errorf("failed to read lock file: %w", err)
		}

		issues, err := i.conflictDetector.CheckDrift(existingHashes)
		if err != nil {
			return fmt.Errorf("failed to check drift: %w", err)
		}

		if len(issues) > 0 {
			return fmt.Errorf("drift detected: %v", issues)
		}

		return nil // Verification passed
	}

	// Update .gitignore
	var ignorePatterns []string
	for name, adapterImpl := range i.adapters {
		if !i.isAdapterEnabled(cfg, name) {
			continue
		}

		adapterCfg := i.getAdapterConfig(cfg, name)
		patterns := adapterImpl.GetGitignorePatterns(adapterCfg)
		ignorePatterns = append(ignorePatterns, patterns...)
	}

	if err := i.gitignoreManager.Update(ignorePatterns); err != nil {
		return fmt.Errorf("failed to update .gitignore: %w", err)
	}

	// Write lock file
	if err := i.lockWriter.Write(lockSources); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

func (i *Installer) isAdapterEnabled(cfg *config.ExtendedConfig, name string) bool {
	switch name {
	case "cursor":
		return cfg.Adapters.Cursor.Enabled
	case "claude":
		return cfg.Adapters.Claude.Enabled
	default:
		return false
	}
}

func (i *Installer) getAdapterConfig(cfg *config.ExtendedConfig, name string) adapter.Config {
	switch name {
	case "cursor":
		return adapter.Config{
			Enabled: cfg.Adapters.Cursor.Enabled,
		}
	case "claude":
		return adapter.Config{
			Enabled: cfg.Adapters.Claude.Enabled,
			Prefix:  cfg.Adapters.Claude.Prefix,
		}
	default:
		return adapter.Config{}
	}
}

// findOrphanedFiles returns files that exist in oldFiles but not in newFiles
func (i *Installer) findOrphanedFiles(oldFiles, newFiles []lock.File) []string {
	// Create a set of new file paths
	newPaths := make(map[string]bool)
	for _, f := range newFiles {
		newPaths[f.Path] = true
	}

	// Find files that exist in old but not in new
	var orphaned []string
	for _, f := range oldFiles {
		if !newPaths[f.Path] {
			orphaned = append(orphaned, f.Path)
		}
	}

	return orphaned
}
