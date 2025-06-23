package unit

import (
	"testing"

	"github.com/kovyrin/prompt-sync/internal/adapter/claude"
)

func TestClaudeAdapter_PrefixResolution(t *testing.T) {
	tests := []struct {
		name           string
		sourcePrefix   string // sources[i].claude_prefix
		configPrefix   string // config.claude_prefix
		sourceName     string // sources[i].name
		expectedPrefix string
	}{
		{
			name:           "explicit source prefix wins",
			sourcePrefix:   "shp",
			configPrefix:   "default",
			sourceName:     "shopify",
			expectedPrefix: "shp",
		},
		{
			name:           "config prefix used when no source prefix",
			sourcePrefix:   "",
			configPrefix:   "proj",
			sourceName:     "shopify",
			expectedPrefix: "proj",
		},
		{
			name:           "source name used as fallback",
			sourcePrefix:   "",
			configPrefix:   "",
			sourceName:     "personal",
			expectedPrefix: "personal",
		},
		{
			name:           "source name kebab-cased",
			sourcePrefix:   "",
			configPrefix:   "",
			sourceName:     "MyCompany",
			expectedPrefix: "my-company",
		},
		{
			name:           "source name with special chars",
			sourcePrefix:   "",
			configPrefix:   "",
			sourceName:     "user@host/repo",
			expectedPrefix: "user-host-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix := claude.ResolvePrefix(tt.sourcePrefix, tt.configPrefix, tt.sourceName)
			if prefix != tt.expectedPrefix {
				t.Errorf("ResolvePrefix() = %v, want %v", prefix, tt.expectedPrefix)
			}
		})
	}
}

func TestClaudeAdapter_FileNaming(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		fileName     string
		expectedName string
	}{
		{
			name:         "simple file name",
			prefix:       "test",
			fileName:     "coding-style.md",
			expectedName: "test-coding-style.md",
		},
		{
			name:         "file with path",
			prefix:       "proj",
			fileName:     "ruby/style-guide.md",
			expectedName: "proj-style-guide.md", // Only basename is used
		},
		{
			name:         "file with spaces",
			prefix:       "my",
			fileName:     "git workflow.md",
			expectedName: "my-git-workflow.md",
		},
		{
			name:         "file with underscores",
			prefix:       "app",
			fileName:     "test_utils.md",
			expectedName: "app-test-utils.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := claude.GenerateFileName(tt.prefix, tt.fileName)
			if result != tt.expectedName {
				t.Errorf("GenerateFileName() = %v, want %v", result, tt.expectedName)
			}
		})
	}
}

func TestClaudeAdapter_KebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CamelCase", "camel-case"},
		{"snake_case", "snake-case"},
		{"Mixed_CamelCase", "mixed-camel-case"},
		{"with spaces", "with-spaces"},
		{"with-dashes", "with-dashes"},
		{"UPPERCASE", "uppercase"},
		{"123numbers", "123numbers"},
		{"special@chars#here", "special-chars-here"},
		{"multiple___underscores", "multiple-underscores"},
		{"TrailingSpace ", "trailing-space"},
		{" LeadingSpace", "leading-space"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := claude.ToKebabCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToKebabCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
