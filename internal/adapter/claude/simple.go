package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kovyrin/prompt-sync/internal/adapter"
)

// SimpleAdapter implements the workflow Adapter interface for Claude
type SimpleAdapter struct{}

// NewSimpleAdapter creates a new simple Claude adapter
func NewSimpleAdapter() adapter.Adapter {
	return &SimpleAdapter{}
}

// DiscoverFiles finds all prompt files in the given source directory
func (a *SimpleAdapter) DiscoverFiles(sourceDir string) ([]string, error) {
	var files []string
	promptsDir := filepath.Join(sourceDir, "prompts")

	// Check if prompts directory exists
	if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
		// Also check commands directory as an alternative
		commandsDir := filepath.Join(sourceDir, "commands")
		if _, err := os.Stat(commandsDir); err == nil {
			promptsDir = commandsDir
		} else {
			return files, nil // No prompts found
		}
	}

	err := filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !isMDFile(path) {
			return nil
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// RenderFile processes a single prompt file and returns the rendered content
func (a *SimpleAdapter) RenderFile(filePath string, content []byte, config adapter.Config) ([]byte, error) {
	// For Claude, we just pass through the content as-is
	// The actual Claude adapter does slash command processing, but for MVP we keep it simple
	return content, nil
}

// GetOutputPath returns the output path for a given input file
func (a *SimpleAdapter) GetOutputPath(inputPath string, config adapter.Config) string {
	// Remove prompts/ or commands/ prefix
	basename := filepath.Base(inputPath)

	// Add prefix if configured
	if config.Prefix != "" {
		basename = fmt.Sprintf("%s-%s", config.Prefix, basename)
	}

	return filepath.Join(".claude/commands", basename)
}

// GetGitignorePatterns returns patterns to add to .gitignore for this adapter
func (a *SimpleAdapter) GetGitignorePatterns(config adapter.Config) []string {
	if config.Prefix != "" {
		return []string{fmt.Sprintf(".claude/commands/%s-*", config.Prefix)}
	}
	return []string{".claude/commands/*"}
}

// GetBaseOutputDir returns the base output directory for this adapter
func (a *SimpleAdapter) GetBaseOutputDir(config adapter.Config) string {
	return ".claude/commands"
}

// isMDFile checks if a file is a markdown file
func isMDFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown" || ext == ".mdc"
}
