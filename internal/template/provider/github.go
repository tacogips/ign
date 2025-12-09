package provider

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tacogips/ign/internal/template/model"
)

// GitHubProvider implements Provider for GitHub repositories.
type GitHubProvider struct {
	// HTTPClient is the HTTP client for API requests.
	HTTPClient *http.Client
	// Token is the optional GitHub personal access token for private repos.
	Token string
}

// NewGitHubProvider creates a new GitHub provider.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewGitHubProviderWithToken creates a new GitHub provider with authentication.
func NewGitHubProviderWithToken(token string) *GitHubProvider {
	return &GitHubProvider{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Token: token,
	}
}

// Name returns the provider name.
func (p *GitHubProvider) Name() string {
	return "github"
}

// Resolve converts a URL string to a TemplateRef.
func (p *GitHubProvider) Resolve(url string) (model.TemplateRef, error) {
	ref, err := ParseGitHubURL(url)
	if err != nil {
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), url, err)
	}
	return *ref, nil
}

// Validate checks if a template reference is valid and accessible.
func (p *GitHubProvider) Validate(ctx context.Context, ref model.TemplateRef) error {
	// Construct API URL to check repository existence
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", ref.Owner, ref.Repo)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return NewFetchError(p.Name(), p.formatURL(ref), err)
	}

	// Add authentication if token is provided
	if p.Token != "" {
		req.Header.Set("Authorization", "token "+p.Token)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return NewFetchError(p.Name(), p.formatURL(ref), err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return NewNotFoundError(p.Name(), p.formatURL(ref))
	case http.StatusUnauthorized, http.StatusForbidden:
		return NewAuthError(p.Name(), p.formatURL(ref))
	default:
		return NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}
}

// Fetch downloads a template from GitHub.
func (p *GitHubProvider) Fetch(ctx context.Context, ref model.TemplateRef) (*model.Template, error) {
	// Download repository archive (tarball)
	archivePath, err := p.downloadArchive(ctx, ref)
	if err != nil {
		return nil, err
	}
	defer os.Remove(archivePath) // Clean up archive after extraction

	// Extract archive to temporary directory
	extractDir, err := p.extractArchive(archivePath)
	if err != nil {
		return nil, NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to extract archive: %w", err))
	}

	// Find template root (handle subdirectory path if specified)
	templateRoot := extractDir
	if ref.Path != "" {
		templateRoot = filepath.Join(extractDir, ref.Path)
		if _, err := os.Stat(templateRoot); err != nil {
			return nil, NewInvalidTemplateError(p.Name(), p.formatURL(ref),
				fmt.Sprintf("subdirectory '%s' not found in template", ref.Path), err)
		}
	}

	// Read and parse ign.json
	ignConfig, err := p.readIgnConfig(templateRoot)
	if err != nil {
		return nil, NewInvalidTemplateError(p.Name(), p.formatURL(ref),
			"failed to read ign.json", err)
	}

	// Collect all template files
	files, err := p.collectFiles(templateRoot)
	if err != nil {
		return nil, NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to collect template files: %w", err))
	}

	return &model.Template{
		Ref:      ref,
		Config:   *ignConfig,
		Files:    files,
		RootPath: templateRoot,
	}, nil
}

// downloadArchive downloads the repository archive (tarball) from GitHub.
func (p *GitHubProvider) downloadArchive(ctx context.Context, ref model.TemplateRef) (string, error) {
	// GitHub archive URL: https://github.com/owner/repo/archive/refs/heads/main.tar.gz
	// Or for tags: https://github.com/owner/repo/archive/refs/tags/v1.0.0.tar.gz
	archiveURL := fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz",
		ref.Owner, ref.Repo, ref.Ref)

	req, err := http.NewRequestWithContext(ctx, "GET", archiveURL, nil)
	if err != nil {
		return "", NewFetchError(p.Name(), p.formatURL(ref), err)
	}

	// Add authentication if token is provided
	if p.Token != "" {
		req.Header.Set("Authorization", "token "+p.Token)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return "", NewFetchError(p.Name(), p.formatURL(ref), err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Continue to download
	case http.StatusNotFound:
		return "", NewNotFoundError(p.Name(), p.formatURL(ref))
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", NewAuthError(p.Name(), p.formatURL(ref))
	default:
		return "", NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	// Create temporary file for archive
	tmpFile, err := os.CreateTemp("", "ign-github-*.tar.gz")
	if err != nil {
		return "", NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to create temp file: %w", err))
	}
	defer tmpFile.Close()

	// Download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to download archive: %w", err))
	}

	return tmpFile.Name(), nil
}

// extractArchive extracts a .tar.gz archive to a temporary directory.
func (p *GitHubProvider) extractArchive(archivePath string) (string, error) {
	// Create temporary directory for extraction
	extractDir, err := os.MkdirTemp("", "ign-template-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		os.RemoveAll(extractDir)
		return "", fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		os.RemoveAll(extractDir)
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Extract files
	var rootDir string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(extractDir)
			return "", fmt.Errorf("failed to read tar entry: %w", err)
		}

		// GitHub archives have a root directory like "repo-ref/"
		// We need to strip this prefix
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 {
			// Skip the root directory entry itself
			continue
		}
		if rootDir == "" {
			rootDir = parts[0]
		}
		relPath := parts[1]

		// Construct target path
		target := filepath.Join(extractDir, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				os.RemoveAll(extractDir)
				return "", fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				os.RemoveAll(extractDir)
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				os.RemoveAll(extractDir)
				return "", fmt.Errorf("failed to create file %s: %w", target, err)
			}

			// Copy content
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				os.RemoveAll(extractDir)
				return "", fmt.Errorf("failed to write file %s: %w", target, err)
			}
			outFile.Close()
		}
	}

	return extractDir, nil
}

// readIgnConfig reads and parses the ign.json file.
func (p *GitHubProvider) readIgnConfig(templateRoot string) (*model.IgnJson, error) {
	ignPath := filepath.Join(templateRoot, "ign.json")

	data, err := os.ReadFile(ignPath)
	if err != nil {
		return nil, fmt.Errorf("ign.json not found in template root: %w", err)
	}

	var config model.IgnJson
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse ign.json: %w", err)
	}

	// Basic validation
	if config.Name == "" {
		return nil, fmt.Errorf("ign.json missing required field: name")
	}
	if config.Version == "" {
		return nil, fmt.Errorf("ign.json missing required field: version")
	}

	return &config, nil
}

// collectFiles recursively collects all files in the template directory.
// Excludes ign.json as it's not part of the template output.
func (p *GitHubProvider) collectFiles(templateRoot string) ([]model.TemplateFile, error) {
	var files []model.TemplateFile

	err := filepath.Walk(templateRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip ign.json (config file, not template content)
		if filepath.Base(path) == "ign.json" {
			return nil
		}

		// Get relative path from template root
		relPath, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", relPath, err)
		}

		// Detect if binary (simple heuristic: check for null bytes)
		isBinary := p.isBinaryContent(content)

		files = append(files, model.TemplateFile{
			Path:     relPath,
			Content:  content,
			Mode:     info.Mode(),
			IsBinary: isBinary,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// isBinaryContent checks if content appears to be binary.
// Simple heuristic: check first 512 bytes for null bytes.
func (p *GitHubProvider) isBinaryContent(content []byte) bool {
	// Check first 512 bytes (or less if file is smaller)
	size := len(content)
	if size > 512 {
		size = 512
	}

	for i := 0; i < size; i++ {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

// formatURL formats a TemplateRef as a human-readable URL.
func (p *GitHubProvider) formatURL(ref model.TemplateRef) string {
	url := fmt.Sprintf("github.com/%s/%s", ref.Owner, ref.Repo)
	if ref.Path != "" {
		url = fmt.Sprintf("%s/%s", url, ref.Path)
	}
	if ref.Ref != "" && ref.Ref != "main" {
		url = fmt.Sprintf("%s@%s", url, ref.Ref)
	}
	return url
}
