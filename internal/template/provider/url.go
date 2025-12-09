package provider

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

// ParseGitHubURL parses a GitHub URL into a TemplateRef.
// Supported formats:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo/tree/branch/path
//   - git@github.com:owner/repo.git
//   - github.com/owner/repo
//   - github.com/owner/repo/path
//   - owner/repo
//   - owner/repo/path
func ParseGitHubURL(url string) (*model.TemplateRef, error) {
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Normalize URL
	url = strings.TrimSpace(url)

	// Handle git@github.com:owner/repo.git format
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		url = strings.TrimSuffix(url, ".git")
		return parseOwnerRepoPath(url)
	}

	// Handle https:// URLs
	if strings.HasPrefix(url, "https://github.com/") {
		url = strings.TrimPrefix(url, "https://github.com/")
		// Remove /tree/branch/ if present
		if idx := strings.Index(url, "/tree/"); idx != -1 {
			// Extract: owner/repo/tree/branch/path -> owner/repo, branch, path
			parts := strings.Split(url, "/tree/")
			ownerRepo := parts[0]
			if len(parts) > 1 {
				branchPath := parts[1]
				slashIdx := strings.Index(branchPath, "/")
				if slashIdx != -1 {
					ref := branchPath[:slashIdx]
					path := branchPath[slashIdx+1:]
					ref2, err := parseOwnerRepoPath(ownerRepo)
					if err != nil {
						return nil, err
					}
					ref2.Ref = ref
					ref2.Path = path
					return ref2, nil
				}
			}
		}
		return parseOwnerRepoPath(url)
	}

	// Handle http:// URLs (convert to https://)
	if strings.HasPrefix(url, "http://github.com/") {
		url = strings.TrimPrefix(url, "http://github.com/")
		return parseOwnerRepoPath(url)
	}

	// Handle github.com/ prefix
	if strings.HasPrefix(url, "github.com/") {
		url = strings.TrimPrefix(url, "github.com/")
		return parseOwnerRepoPath(url)
	}

	// Handle owner/repo format
	return parseOwnerRepoPath(url)
}

// parseOwnerRepoPath parses "owner/repo" or "owner/repo/path" format.
func parseOwnerRepoPath(s string) (*model.TemplateRef, error) {
	parts := strings.Split(s, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format, expected owner/repo: %s", s)
	}

	owner := parts[0]
	repo := parts[1]

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo cannot be empty: %s", s)
	}

	ref := &model.TemplateRef{
		Provider: "github",
		Owner:    owner,
		Repo:     repo,
		Ref:      "main", // default
	}

	// Extract subdirectory path if present
	if len(parts) > 2 {
		ref.Path = strings.Join(parts[2:], "/")
	}

	return ref, nil
}

// IsLocalPath checks if a path is a local filesystem path.
// Returns true for relative paths starting with "./" or "../".
// Returns false for GitHub-style URLs.
func IsLocalPath(path string) bool {
	if path == "" {
		return false
	}

	// Absolute paths are not considered local for portability
	if filepath.IsAbs(path) {
		return false
	}

	// Relative paths starting with "./" or "../"
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		return true
	}

	// Check if it looks like a GitHub URL
	if strings.Contains(path, "github.com") || strings.Contains(path, "git@") {
		return false
	}

	// owner/repo pattern (at least one slash, no dot prefix)
	if strings.Contains(path, "/") && !strings.HasPrefix(path, ".") {
		// Could be owner/repo format, not local
		return false
	}

	// Single path component without slashes - ambiguous, treat as non-local
	return false
}

// ValidateLocalPath validates a local filesystem path for security.
// Returns an error if:
//   - Path contains ".." (traversal)
//   - Path is absolute (portability)
func ValidateLocalPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for ".." components (path traversal)
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains '..' which is not allowed for security: %s", path)
	}

	// Check for absolute paths (portability)
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths are not allowed, use relative paths: %s", path)
	}

	return nil
}

// NormalizeLocalPath normalizes a local path by cleaning and validating it.
func NormalizeLocalPath(path string) (string, error) {
	if err := ValidateLocalPath(path); err != nil {
		return "", err
	}

	// Clean the path (removes redundant slashes, etc.)
	cleaned := filepath.Clean(path)

	// Verify cleaned path doesn't escape via ".."
	if strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("path escapes current directory: %s", path)
	}

	return cleaned, nil
}
