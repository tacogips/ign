package generator

import (
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
)

// IsSpecialFile checks if a file is a special file that should be excluded from generation.
// Returns true for:
// - ign-template.json (template configuration file)
// - Paths starting with ".ign/" or exactly ".ign"
func IsSpecialFile(path string) bool {
	// Normalize path separators
	path = filepath.ToSlash(path)

	// Check for template config file
	if path == model.IgnTemplateConfigFile || strings.HasSuffix(path, "/"+model.IgnTemplateConfigFile) {
		return true
	}

	// Check for .ign directory
	if path == model.IgnConfigDir || strings.HasPrefix(path, model.IgnConfigDir+"/") {
		return true
	}

	return false
}

// ShouldIgnoreFile checks if a file should be ignored during generation based on ignore patterns.
// Returns true if:
// - File is a special file (template config file, .ign/*)
// - File matches any of the ignore patterns (glob matching)
func ShouldIgnoreFile(path string, ignorePatterns []string) bool {
	// First check if it's a special file
	if IsSpecialFile(path) {
		debug.Debug("[generator] Ignoring special file: %s", path)
		return true
	}

	// Check against ignore patterns
	for _, pattern := range ignorePatterns {
		if MatchesPattern(path, pattern) {
			debug.Debug("[generator] Ignoring file: %s (matched pattern: %s)", path, pattern)
			return true
		}
	}

	return false
}

// MatchesPattern checks if a file path matches a glob pattern.
// Uses filepath.Match for glob matching.
func MatchesPattern(path, pattern string) bool {
	// Normalize path separators for consistent matching
	path = filepath.ToSlash(path)
	pattern = filepath.ToSlash(pattern)

	// Try matching the full path
	matched, err := filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Try matching just the filename
	filename := filepath.Base(path)
	matched, err = filepath.Match(pattern, filename)
	if err == nil && matched {
		return true
	}

	// Try matching with path prefix (e.g., "*.txt" should match "dir/file.txt")
	if !strings.Contains(pattern, "/") {
		// Pattern has no path separator, try matching against basename
		matched, err := filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}
	}

	return false
}
