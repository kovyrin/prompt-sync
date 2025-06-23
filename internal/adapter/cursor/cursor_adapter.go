package cursor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kovyrin/prompt-sync/internal/adapter"
	"gopkg.in/yaml.v3"
)

// Adapter implements the AgentAdapter interface for Cursor.
type Adapter struct{}

// NewAdapter creates a new Cursor adapter.
func NewAdapter() adapter.AgentAdapter {
	return &Adapter{}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "cursor"
}

// Detect checks if Cursor is present/configured.
func (a *Adapter) Detect() bool {
	// Check if .cursor directory exists
	if _, err := os.Stat(".cursor"); err == nil {
		return true
	}
	// Could also check for cursor config files
	return false
}

// TargetDir returns the directory where Cursor rules should be rendered.
func (a *Adapter) TargetDir(scope adapter.Scope) string {
	// Cursor uses a single directory for all rules
	return ".cursor/rules/_active"
}

// Render converts a prompt pack into Cursor rule files.
func (a *Adapter) Render(pack adapter.PromptPack, scope adapter.Scope) ([]adapter.RenderedFile, error) {
	promptsDir := filepath.Join(pack.Path, "prompts")
	if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("prompts directory not found: %s", promptsDir)
	}

	// Load metadata.yaml if it exists
	metadataPath := filepath.Join(promptsDir, "metadata.yaml")
	metadata, _ := LoadMetadataFile(metadataPath) // Ignore error if file doesn't exist

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

		// Parse front-matter
		frontMatter, body, err := ParseFrontMatter(content)
		if err != nil {
			return fmt.Errorf("parse front-matter in %s: %w", path, err)
		}

		// Get relative path for the file
		relPath, err := filepath.Rel(promptsDir, path)
		if err != nil {
			return err
		}

		// Merge metadata (defaults -> file overrides -> front-matter)
		fileOverride := metadata.Files[filepath.Base(path)]
		effectiveMeta := MergeMetadata(metadata.Defaults, fileOverride, frontMatter)

		// Render the file with merged metadata
		rendered := renderCursorRule(effectiveMeta, body)

		// Create rendered file
		renderedFile := adapter.RenderedFile{
			Path:    relPath,
			Content: rendered,
			Hash:    adapter.HashContent(rendered),
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

// Metadata represents the structure of metadata.yaml
type Metadata struct {
	Defaults map[string]interface{}            `yaml:"defaults"`
	Files    map[string]map[string]interface{} `yaml:"files"`
}

// LoadMetadataFile loads and parses a metadata.yaml file.
func LoadMetadataFile(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty metadata if file doesn't exist
			return &Metadata{
				Defaults: make(map[string]interface{}),
				Files:    make(map[string]map[string]interface{}),
			}, nil
		}
		return nil, err
	}

	var metadata Metadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata.yaml: %w", err)
	}

	// Initialize maps if nil
	if metadata.Defaults == nil {
		metadata.Defaults = make(map[string]interface{})
	}
	if metadata.Files == nil {
		metadata.Files = make(map[string]map[string]interface{})
	}

	return &metadata, nil
}

// ParseFrontMatter extracts YAML front-matter from markdown content.
func ParseFrontMatter(content []byte) (map[string]interface{}, []byte, error) {
	// Check if content starts with ---
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		// No front-matter
		return make(map[string]interface{}), content, nil
	}

	// Find the closing ---
	lines := bytes.Split(content, []byte("\n"))
	var endIndex int
	for i := 1; i < len(lines); i++ {
		line := bytes.TrimSpace(lines[i])
		if bytes.Equal(line, []byte("---")) {
			endIndex = i
			break
		}
	}

	if endIndex == 0 {
		// No closing ---, treat as no front-matter
		return make(map[string]interface{}), content, nil
	}

	// Extract front-matter
	frontMatterBytes := bytes.Join(lines[1:endIndex], []byte("\n"))

	// Parse YAML
	var frontMatter map[string]interface{}
	if err := yaml.Unmarshal(frontMatterBytes, &frontMatter); err != nil {
		return nil, nil, fmt.Errorf("parse YAML front-matter: %w", err)
	}

	if frontMatter == nil {
		frontMatter = make(map[string]interface{})
	}

	// Extract body (everything after the closing ---)
	bodyLines := lines[endIndex+1:]
	body := bytes.Join(bodyLines, []byte("\n"))
	body = bytes.TrimLeft(body, "\n\r")

	return frontMatter, body, nil
}

// MergeMetadata merges metadata from three sources with proper precedence.
func MergeMetadata(defaults, fileOverride, frontMatter map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Start with defaults
	for k, v := range defaults {
		result[k] = v
	}

	// Apply file overrides
	for k, v := range fileOverride {
		if v == nil {
			// nil means remove the field
			delete(result, k)
		} else {
			result[k] = v
		}
	}

	// Apply front-matter (highest priority)
	for k, v := range frontMatter {
		if v == nil {
			delete(result, k)
		} else {
			result[k] = v
		}
	}

	return result
}

// renderCursorRule renders a Cursor rule with metadata as front-matter.
func renderCursorRule(metadata map[string]interface{}, body []byte) []byte {
	var buf bytes.Buffer

	// Write front-matter if there's metadata
	if len(metadata) > 0 {
		buf.WriteString("---\n")
		// Marshal metadata to YAML
		yamlBytes, _ := yaml.Marshal(metadata)
		buf.Write(yamlBytes)
		buf.WriteString("---\n\n")
	}

	// Write body
	buf.Write(body)

	return buf.Bytes()
}

// isMarkdownFile checks if a file is a markdown file.
func isMarkdownFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown" || ext == ".mdc"
}
