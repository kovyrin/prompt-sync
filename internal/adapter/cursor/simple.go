package cursor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kovyrin/prompt-sync/internal/adapter"
)

// SimpleAdapter implements the workflow Adapter interface for Cursor
type SimpleAdapter struct{}

// NewSimpleAdapter creates a new simple Cursor adapter
func NewSimpleAdapter() adapter.Adapter {
	return &SimpleAdapter{}
}

// DiscoverFiles finds all prompt files in the given source directory
func (a *SimpleAdapter) DiscoverFiles(sourceDir string) ([]string, error) {
	var files []string
	promptsDir := filepath.Join(sourceDir, "prompts")

	// Check if prompts directory exists
	if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
		// Also check rules directory as an alternative
		rulesDir := filepath.Join(sourceDir, "rules")
		if _, err := os.Stat(rulesDir); err == nil {
			promptsDir = rulesDir
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
	// For Cursor, we just pass through the content as-is
	// The actual Cursor adapter does metadata merging, but for MVP we keep it simple
	return content, nil
}

// GetOutputPath returns the output path for a given input file
func (a *SimpleAdapter) GetOutputPath(inputPath string, config adapter.Config) string {
	// Remove prompts/ or rules/ prefix and place in active rules directory
	basename := filepath.Base(inputPath)
	return filepath.Join(".cursor/rules/_active", basename)
}

// GetGitignorePatterns returns patterns to add to .gitignore for this adapter
func (a *SimpleAdapter) GetGitignorePatterns(config adapter.Config) []string {
	return []string{".cursor/rules/_active/"}
}

// GetBaseOutputDir returns the base output directory for this adapter
func (a *SimpleAdapter) GetBaseOutputDir(config adapter.Config) string {
	return ".cursor/rules/_active"
}

// isMDFile checks if a file is a markdown file
func isMDFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown" || ext == ".mdc"
}
