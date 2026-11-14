package provider

import (
	"fmt"
	"os"
)

// NewProvider creates the appropriate provider based on the URL/path.
// Automatically detects whether the input is a GitHub URL or local path.
func NewProvider(url string) (Provider, error) {
	if url == "" {
		return nil, fmt.Errorf("URL or path cannot be empty")
	}

	// Check if it's a local path
	if IsLocalPath(url) {
		return NewLocalProvider(), nil
	}

	// Default to GitHub provider
	return NewGitHubProvider(), nil
}

// NewProviderWithToken creates a provider with optional GitHub token.
// If the URL is a GitHub URL and token is provided, uses authenticated GitHub provider.
// For local paths, token is ignored.
func NewProviderWithToken(url, token string) (Provider, error) {
	if url == "" {
		return nil, fmt.Errorf("URL or path cannot be empty")
	}

	// Check if it's a local path
	if IsLocalPath(url) {
		return NewLocalProvider(), nil
	}

	// GitHub provider with token
	if token != "" {
		return NewGitHubProviderWithToken(token), nil
	}

	// Default GitHub provider without token
	return NewGitHubProvider(), nil
}

// NewProviderWithConfig creates a provider with configuration options.
type ProviderConfig struct {
	// GitHubToken is the optional GitHub personal access token.
	GitHubToken string
	// BaseDir is the base directory for resolving local paths.
	BaseDir string
}

// NewProviderWithConfig creates a provider with advanced configuration.
func NewProviderWithConfig(url string, config ProviderConfig) (Provider, error) {
	if url == "" {
		return nil, fmt.Errorf("URL or path cannot be empty")
	}

	// Check if it's a local path
	if IsLocalPath(url) {
		if config.BaseDir != "" {
			return NewLocalProviderWithBase(config.BaseDir), nil
		}
		return NewLocalProvider(), nil
	}

	// GitHub provider
	if config.GitHubToken != "" {
		return NewGitHubProviderWithToken(config.GitHubToken), nil
	}

	return NewGitHubProvider(), nil
}

// GetGitHubTokenFromEnv retrieves the GitHub token from environment variables.
// Checks GITHUB_TOKEN first, then falls back to GH_TOKEN.
func GetGitHubTokenFromEnv() string {
	// Try GITHUB_TOKEN first (common in CI/CD)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}

	// Fall back to GH_TOKEN (used by GitHub CLI)
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}

	return ""
}
