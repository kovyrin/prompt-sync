package unit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kovyrin/prompt-sync/internal/adapter/cursor"
)

func TestCursorAdapter_MetadataMerge(t *testing.T) {
	tests := []struct {
		name         string
		defaults     map[string]interface{}
		fileOverride map[string]interface{}
		frontMatter  map[string]interface{}
		expected     map[string]interface{}
	}{
		{
			name: "defaults only",
			defaults: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
			},
			expected: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
			},
		},
		{
			name: "file override replaces defaults",
			defaults: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
			},
			fileOverride: map[string]interface{}{
				"alwaysApply": false,
				"globs":       []string{"**/*.rb", "**/*.erb"},
			},
			expected: map[string]interface{}{
				"alwaysApply": false,
				"globs":       []string{"**/*.rb", "**/*.erb"},
			},
		},
		{
			name: "front-matter wins over all",
			defaults: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
			},
			fileOverride: map[string]interface{}{
				"alwaysApply": false,
				"globs":       []string{"**/*.rb", "**/*.erb"},
			},
			frontMatter: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.go"},
				"priority":    "high",
			},
			expected: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.go"},
				"priority":    "high",
			},
		},
		{
			name: "null value removes field",
			defaults: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
				"description": "Default description",
			},
			fileOverride: map[string]interface{}{
				"description": nil, // Remove description
			},
			expected: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
			},
		},
		{
			name: "partial override keeps other fields",
			defaults: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.rb"},
				"description": "Default description",
			},
			fileOverride: map[string]interface{}{
				"globs": []string{"**/*.go"}, // Only override globs
			},
			expected: map[string]interface{}{
				"alwaysApply": true,
				"globs":       []string{"**/*.go"},
				"description": "Default description",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cursor.MergeMetadata(tt.defaults, tt.fileOverride, tt.frontMatter)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("MergeMetadata() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCursorAdapter_ParseFrontMatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantMeta map[string]interface{}
		wantBody string
		wantErr  bool
	}{
		{
			name: "valid front-matter",
			content: `---
title: Test Rule
alwaysApply: true
globs: ["**/*.go"]
---

This is the body content.`,
			wantMeta: map[string]interface{}{
				"title":       "Test Rule",
				"alwaysApply": true,
				"globs":       []interface{}{"**/*.go"},
			},
			wantBody: "This is the body content.",
			wantErr:  false,
		},
		{
			name: "no front-matter",
			content: `This is just body content.
No front-matter here.`,
			wantMeta: map[string]interface{}{},
			wantBody: `This is just body content.
No front-matter here.`,
			wantErr: false,
		},
		{
			name: "empty front-matter",
			content: `---
---

Body content here.`,
			wantMeta: map[string]interface{}{},
			wantBody: "Body content here.",
			wantErr:  false,
		},
		{
			name: "invalid YAML in front-matter",
			content: `---
title: Test
invalid: [unclosed
---

Body`,
			wantMeta: nil,
			wantBody: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, body, err := cursor.ParseFrontMatter([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFrontMatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(meta, tt.wantMeta) {
					t.Errorf("ParseFrontMatter() meta = %v, want %v", meta, tt.wantMeta)
				}
				if string(body) != tt.wantBody {
					t.Errorf("ParseFrontMatter() body = %q, want %q", string(body), tt.wantBody)
				}
			}
		})
	}
}

func TestCursorAdapter_LoadMetadataFile(t *testing.T) {
	// Create a test directory with metadata.yaml
	testDir := t.TempDir()
	metadataPath := filepath.Join(testDir, "metadata.yaml")

	metadataContent := `defaults:
  alwaysApply: true
  globs: ["**/*.rb"]

files:
  coding-style.md:
    description: "Ruby style guide"
    globs: ["**/*.rb", "**/*.erb"]
    alwaysApply: false
  git-workflow.md:
    globs: ["**/*.md"]
`

	if err := os.WriteFile(metadataPath, []byte(metadataContent), 0644); err != nil {
		t.Fatalf("failed to write metadata file: %v", err)
	}

	metadata, err := cursor.LoadMetadataFile(metadataPath)
	if err != nil {
		t.Fatalf("LoadMetadataFile() error = %v", err)
	}

	// Check defaults
	expectedDefaults := map[string]interface{}{
		"alwaysApply": true,
		"globs":       []interface{}{"**/*.rb"},
	}
	if !reflect.DeepEqual(metadata.Defaults, expectedDefaults) {
		t.Errorf("LoadMetadataFile() defaults = %v, want %v", metadata.Defaults, expectedDefaults)
	}

	// Check file overrides
	if len(metadata.Files) != 2 {
		t.Errorf("LoadMetadataFile() files count = %d, want 2", len(metadata.Files))
	}

	// Check specific file override
	if override, ok := metadata.Files["coding-style.md"]; ok {
		if override["alwaysApply"] != false {
			t.Errorf("File override alwaysApply = %v, want false", override["alwaysApply"])
		}
	} else {
		t.Error("Missing file override for coding-style.md")
	}
}
