package unit_test

import (
	"testing"

	"github.com/kovyrin/prompt-sync/internal/adapter"
	"github.com/kovyrin/prompt-sync/internal/adapter/cursor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleCursorAdapter_PreservesFrontmatter(t *testing.T) {
	cursorAdapter := cursor.NewSimpleAdapter()
	config := adapter.Config{}

	t.Run("preserves YAML frontmatter", func(t *testing.T) {
		content := []byte(`---
title: Code Review Guidelines
tags: [review, quality]
priority: high
custom_field: some_value
---

# Code Review Guidelines

Always review code thoroughly.`)

		rendered, err := cursorAdapter.RenderFile("rules/review.md", content, config)
		require.NoError(t, err)

		// Should preserve the content exactly as-is
		assert.Equal(t, string(content), string(rendered))
	})

	t.Run("preserves empty frontmatter", func(t *testing.T) {
		content := []byte(`---
---

# Empty Frontmatter

This file has empty frontmatter.`)

		rendered, err := cursorAdapter.RenderFile("rules/empty.md", content, config)
		require.NoError(t, err)

		assert.Equal(t, string(content), string(rendered))
	})

	t.Run("preserves content without frontmatter", func(t *testing.T) {
		content := []byte(`# No Frontmatter

This file has no frontmatter at all.`)

		rendered, err := cursorAdapter.RenderFile("rules/plain.md", content, config)
		require.NoError(t, err)

		assert.Equal(t, string(content), string(rendered))
	})

	t.Run("preserves complex frontmatter with nested structures", func(t *testing.T) {
		content := []byte(`---
title: Complex Rule
metadata:
  author: Test Author
  version: 1.2.3
  tags:
    - testing
    - complex
settings:
  enabled: true
  level: strict
---

# Complex Rule

This has complex nested frontmatter.`)

		rendered, err := cursorAdapter.RenderFile("rules/complex.md", content, config)
		require.NoError(t, err)

		assert.Equal(t, string(content), string(rendered))
	})

	t.Run("preserves MDC files with frontmatter", func(t *testing.T) {
		content := []byte(`---
title: MDC Rule
description: This is an MDC file
---

# MDC Content

::alert{type="info"}
This is an MDC component
::

Some regular content.`)

		rendered, err := cursorAdapter.RenderFile("rules/example.mdc", content, config)
		require.NoError(t, err)

		assert.Equal(t, string(content), string(rendered))
	})
}

func TestSimpleCursorAdapter_GetOutputPath(t *testing.T) {
	cursorAdapter := cursor.NewSimpleAdapter()
	config := adapter.Config{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "file in prompts directory",
			input:    "prompts/coding.md",
			expected: ".cursor/rules/_active/coding.md",
		},
		{
			name:     "file in rules directory",
			input:    "rules/review.md",
			expected: ".cursor/rules/_active/review.md",
		},
		{
			name:     "MDC file",
			input:    "prompts/example.mdc",
			expected: ".cursor/rules/_active/example.mdc",
		},
		{
			name:     "nested path",
			input:    "prompts/team/guidelines.md",
			expected: ".cursor/rules/_active/guidelines.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := cursorAdapter.GetOutputPath(tt.input, config)
			assert.Equal(t, tt.expected, output)
		})
	}
}
