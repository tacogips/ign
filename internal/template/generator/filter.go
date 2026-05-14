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

	// Check for template-side overwrite ignore file. Only the template root file is metadata;
	// nested files with the same name are generated normally.
	if path == model.IgnOverwriteIgnoreFile {
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

// ParseIgnoreFilePatterns parses gitignore-style pattern lines.
// Blank lines and comments are ignored. Escaped leading hashes are unescaped.
func ParseIgnoreFilePatterns(content []byte) []string {
	lines := strings.Split(string(content), "\n")
	patterns := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, `\#`) {
			line = strings.TrimPrefix(line, `\`)
		} else if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// MatchesGitIgnorePattern checks a path against the subset of gitignore syntax
// used by .ign-overwrite-ignore. Later negated patterns can re-include paths.
func MatchesGitIgnorePattern(path string, patterns []string) bool {
	path = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
	ignored := false
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(filepath.ToSlash(pattern))
		if pattern == "" || pattern == "." {
			continue
		}

		negated := strings.HasPrefix(pattern, "!")
		if negated {
			pattern = strings.TrimPrefix(pattern, "!")
		}
		if pattern == "" {
			continue
		}

		if matchesIgnorePattern(path, pattern) {
			ignored = !negated
		}
	}
	return ignored
}

func matchesIgnorePattern(path, pattern string) bool {
	anchored := strings.HasPrefix(pattern, "/")
	if anchored {
		pattern = strings.TrimPrefix(pattern, "/")
	}

	dirOnly := strings.HasSuffix(pattern, "/")
	if dirOnly {
		pattern = strings.TrimSuffix(pattern, "/")
	}
	if pattern == "" {
		return false
	}

	if dirOnly && matchDirectoryPattern(path, pattern, anchored) {
		return true
	}

	if strings.Contains(pattern, "/") || anchored {
		if matchSlashPattern(pattern, path) {
			return true
		}
		return strings.HasPrefix(path, pattern+"/")
	}

	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if ok, _ := filepath.Match(pattern, segment); ok {
			return true
		}
	}
	return false
}
func matchSlashPattern(pattern, path string) bool {
	patternSegments := strings.Split(pattern, "/")
	pathSegments := strings.Split(path, "/")
	return matchSlashPatternSegments(patternSegments, pathSegments)
}

func matchSlashPatternSegments(patternSegments, pathSegments []string) bool {
	if len(patternSegments) == 0 {
		return len(pathSegments) == 0
	}

	segmentPattern := patternSegments[0]
	if segmentPattern == "**" {
		if len(patternSegments) == 1 {
			return true
		}
		for i := 0; i <= len(pathSegments); i++ {
			if matchSlashPatternSegments(patternSegments[1:], pathSegments[i:]) {
				return true
			}
		}
		return false
	}

	if len(pathSegments) == 0 {
		return false
	}

	matched, err := filepath.Match(segmentPattern, pathSegments[0])
	if err != nil || !matched {
		return false
	}
	return matchSlashPatternSegments(patternSegments[1:], pathSegments[1:])
}

func matchDirectoryPattern(path, pattern string, anchored bool) bool {
	if anchored {
		return path == pattern || strings.HasPrefix(path, pattern+"/")
	}
	if path == pattern || strings.HasPrefix(path, pattern+"/") {
		return true
	}
	return strings.Contains(path, "/"+pattern+"/")
}
