package adapter

// Config holds adapter-specific configuration
type Config struct {
	Enabled bool
	Prefix  string
}

// Adapter is the simplified interface that adapters implement for the workflow
type Adapter interface {
	// DiscoverFiles finds all prompt files in the given source directory
	DiscoverFiles(sourceDir string) ([]string, error)

	// RenderFile processes a single prompt file and returns the rendered content
	RenderFile(filePath string, content []byte, config Config) ([]byte, error)

	// GetOutputPath returns the output path for a given input file
	GetOutputPath(inputPath string, config Config) string

	// GetGitignorePatterns returns patterns to add to .gitignore for this adapter
	GetGitignorePatterns(config Config) []string

	// GetBaseOutputDir returns the base output directory for this adapter
	GetBaseOutputDir(config Config) string
}
