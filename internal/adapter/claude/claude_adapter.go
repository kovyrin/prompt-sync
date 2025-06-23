package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kovyrin/prompt-sync/internal/adapter"
)

// Adapter implements the AgentAdapter interface for Claude.
type Adapter struct {
	prefix string
}

// NewAdapter creates a new Claude adapter with the given prefix.
func NewAdapter(prefix string) adapter.AgentAdapter {
	return &Adapter{prefix: prefix}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "claude"
}

// Detect checks if Claude is present/configured.
func (a *Adapter) Detect() bool {
	// Check if .claude directory exists
	if _, err := os.Stat(".claude"); err == nil {
		return true
	}
	// Could also check for claude config files
	return false
}

// TargetDir returns the directory where Claude commands should be rendered.
func (a *Adapter) TargetDir(scope adapter.Scope) string {
	// Claude uses a single directory for all commands
	return ".claude/commands"
}

// Render converts a prompt pack into Claude command files.
func (a *Adapter) Render(pack adapter.PromptPack, scope adapter.Scope) ([]adapter.RenderedFile, error) {
	promptsDir := filepath.Join(pack.Path, "prompts")
	if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("prompts directory not found: %s", promptsDir)
	}

	var files []adapter.RenderedFile

	// Walk through all markdown files in the prompts directory
	err := filepath.Walk(promptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !isMarkdownFile(path) {
			return nil
		}

		// Skip metadata.yaml
		if filepath.Base(path) == "metadata.yaml" {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		// Get relative path for the file
		relPath, err := filepath.Rel(promptsDir, path)
		if err != nil {
			return err
		}

		// Generate the prefixed filename
		fileName := GenerateFileName(a.prefix, relPath)

		// Create rendered file
		renderedFile := adapter.RenderedFile{
			Path:    fileName,
			Content: content, // Claude uses raw markdown files
			Hash:    adapter.HashContent(content),
		}

		files = append(files, renderedFile)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// Verify checks that rendered files match expected hashes.
func (a *Adapter) Verify(files []adapter.RenderedFile, mode adapter.Strictness) error {
	var errors []string
	for _, file := range files {
		actualHash := adapter.HashContent(file.Content)
		if actualHash != file.Hash {
			msg := fmt.Sprintf("hash mismatch for %s: expected %s, got %s", file.Path, file.Hash, actualHash)
			errors = append(errors, msg)
		}
	}

	if len(errors) > 0 {
		if mode == adapter.StrictnessStrict || len(errors) == len(files) {
			return fmt.Errorf("verification failed: %s", strings.Join(errors, "; "))
		}
		// In normal mode, we would log warnings but continue
	}
	return nil
}

// ResolvePrefix determines the prefix to use based on precedence rules.
func ResolvePrefix(sourcePrefix, configPrefix, sourceName string) string {
	// 1. Explicit source prefix wins
	if sourcePrefix != "" {
		return sourcePrefix
	}

	// 2. Config prefix as fallback
	if configPrefix != "" {
		return configPrefix
	}

	// 3. Source name as default (kebab-cased)
	return ToKebabCase(sourceName)
}

// GenerateFileName creates a prefixed filename for Claude.
func GenerateFileName(prefix, filePath string) string {
	// Get just the filename
	fileName := filepath.Base(filePath)

	// Replace spaces with dashes
	fileName = strings.ReplaceAll(fileName, " ", "-")

	// Replace underscores with dashes
	fileName = strings.ReplaceAll(fileName, "_", "-")

	// Prefix the filename
	return fmt.Sprintf("%s-%s", prefix, fileName)
}

// ToKebabCase converts a string to kebab-case.
func ToKebabCase(s string) string {
	// Handle empty string
	if s == "" {
		return ""
	}

	// Trim spaces
	s = strings.TrimSpace(s)

	// Replace special characters with dashes
	re := regexp.MustCompile(`[^a-zA-Z0-9\-]+`)
	s = re.ReplaceAllString(s, "-")

	// Handle camelCase and PascalCase
	// Insert dash before uppercase letters that follow lowercase letters
	result := ""
	for i, r := range s {
		if i > 0 && isUpper(r) && i < len(s)-1 {
			prevRune := rune(s[i-1])
			if isLower(prevRune) || (i < len(s)-1 && isLower(rune(s[i+1]))) {
				result += "-"
			}
		}
		result += strings.ToLower(string(r))
	}

	// Replace multiple dashes with single dash
	re = regexp.MustCompile(`-+`)
	result = re.ReplaceAllString(result, "-")

	// Trim dashes from start and end
	result = strings.Trim(result, "-")

	return result
}

// isMarkdownFile checks if a file is a markdown file.
func isMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown" || ext == ".mdc"
}

// isUpper checks if a rune is uppercase.
func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// isLower checks if a rune is lowercase.
func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}
