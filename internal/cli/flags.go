package cli

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Common flag names and descriptions
const (
	// Flag names
	FlagOutput    = "output"
	FlagOverwrite = "overwrite"
	FlagConfig    = "config"
	FlagRef       = "ref"
	FlagForce     = "force"
	FlagDryRun    = "dry-run"
	FlagVerbose   = "verbose"
	FlagNoColor   = "no-color"
	FlagQuiet     = "quiet"
	FlagDebug     = "debug"

	// Flag descriptions
	DescOutput    = "Output directory"
	DescOverwrite = "Overwrite existing files"
	DescConfig    = "Path to config file"
	DescRef       = "Git branch, tag, or commit SHA"
	DescForce     = "Force overwrite"
	DescDryRun    = "Show actions without execution"
	DescVerbose   = "Verbose output"
	DescNoColor   = "Disable colored output"
	DescQuiet     = "Suppress output"
	DescDebug     = "Enable debug logging"
)

// URL validation patterns
var (
	// GitHub URL patterns
	githubHTTPSPattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)(.*)$`)
	githubSSHPattern   = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+)\.git$`)
	githubShortPattern = regexp.MustCompile(`^github\.com/([^/]+)/([^/]+)(.*)$`)
	ownerRepoPattern   = regexp.MustCompile(`^([^/]+)/([^/]+)(.*)$`)

	// Git ref patterns
	refBranchPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-/\.]+$`)
	refTagPattern    = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9\-\.]+)?$`)
	refCommitPattern = regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
)

// ValidateGitHubURL validates and normalizes a GitHub URL
func ValidateGitHubURL(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}

	// Already valid HTTPS URL
	if githubHTTPSPattern.MatchString(url) {
		return url, nil
	}

	// Git SSH URL
	if githubSSHPattern.MatchString(url) {
		return url, nil
	}

	// Short form: github.com/owner/repo
	if githubShortPattern.MatchString(url) {
		return "https://" + url, nil
	}

	// Owner/repo format
	if ownerRepoPattern.MatchString(url) {
		return "https://github.com/" + url, nil
	}

	return "", fmt.Errorf("invalid GitHub URL format: %s", url)
}

// ValidateGitRef validates a git reference (branch, tag, or commit)
func ValidateGitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git reference cannot be empty")
	}

	// Check if it matches any valid pattern
	if refBranchPattern.MatchString(ref) ||
		refTagPattern.MatchString(ref) ||
		refCommitPattern.MatchString(ref) {
		return nil
	}

	return fmt.Errorf("invalid git reference: %s", ref)
}

// ValidateOutputPath validates an output directory path
func ValidateOutputPath(path string) error {
	if path == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	// Check for path traversal attempts
	if containsPathTraversal(path) {
		return fmt.Errorf("output path contains invalid traversal: %s", path)
	}

	return nil
}

// containsPathTraversal checks if path contains .. or other suspicious patterns
func containsPathTraversal(path string) bool {
	// Simple check for now - can be enhanced
	return regexp.MustCompile(`\.\.`).MatchString(path)
}

// getGitHubToken retrieves GitHub token from environment or gh CLI.
// Priority: GITHUB_TOKEN env > GH_TOKEN env > gh auth token command
func getGitHubToken(configPath string) string {
	// Try environment variables first (highest priority)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}

	// Try gh CLI auth token (uses gh's secure credential storage)
	// Only attempt if gh command is available
	if _, err := exec.LookPath("gh"); err == nil {
		cmd := exec.Command("gh", "auth", "token")
		output, err := cmd.Output()
		if err == nil {
			token := strings.TrimSpace(string(output))
			if token != "" {
				return token
			}
		}
	}

	return ""
}
