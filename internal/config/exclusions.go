package config

// FileExclusions contains patterns for files/directories to exclude from stats calculations
var FileExclusions = ExclusionConfig{
	// Patterns support SQL LIKE syntax (% for wildcard)
	Patterns: []string{
		// Package manager lock files
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"Gemfile.lock",
		"Pipfile.lock",
		"poetry.lock",
		"go.sum",

		// Generated/compiled files
		"%.pb.go",
		"%.gen.go",
		"%.generated.%",
		"%.min.js",
		"%.min.css",
		"%.bundle.js",

		// Vendor/dependency directories
		"vendor/%",
		"node_modules/%",
		".git/%",

		// Build artifacts
		"dist/%",
		"build/%",
		"out/%",
		"bin/%",

		// IDE/editor config
		".vscode/%",
		".idea/%",

		// Documentation/config
		"%.md",
		"%.json",
		"%.yaml",
		"%.yml",
		"%.toml",
	},

	// File extensions to always exclude
	Extensions: []string{
		".lock",
		".sum",
		".map",
	},
}

// ExclusionConfig holds file exclusion patterns
type ExclusionConfig struct {
	Patterns   []string `json:"patterns"`
	Extensions []string `json:"extensions"`
}

// GetExclusionPatterns returns all patterns for SQL LIKE filtering
func GetExclusionPatterns() []string {
	return FileExclusions.Patterns
}
