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

	"github.com/tacogips/ign/internal/debug"
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
	debug.Debug("[github] Resolving URL: %s", url)
	ref, err := ParseGitHubURL(url)
	if err != nil {
		debug.Debug("[github] Failed to parse URL: %v", err)
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), url, err)
	}
	debug.Debug("[github] Resolved to: owner=%s, repo=%s, path=%s, ref=%s",
		ref.Owner, ref.Repo, ref.Path, ref.Ref)
	return *ref, nil
}

// Validate checks if a template reference is valid and accessible.
func (p *GitHubProvider) Validate(ctx context.Context, ref model.TemplateRef) error {
	// Construct API URL to check repository existence
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", ref.Owner, ref.Repo)
	debug.Debug("[github] Validating repository: %s", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		debug.Debug("[github] Failed to create request: %v", err)
		return NewFetchError(p.Name(), p.formatURL(ref), err)
	}

	// Add authentication if token is provided
	if p.Token != "" {
		req.Header.Set("Authorization", "token "+p.Token)
		debug.Debug("[github] Using authenticated request")
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		debug.Debug("[github] Validation request failed: %v", err)
		return NewFetchError(p.Name(), p.formatURL(ref), err)
	}
	defer func() { _ = resp.Body.Close() }()

	debug.Debug("[github] Validation response status: %d", resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusOK:
		debug.Debug("[github] Repository validated successfully")
		return nil
	case http.StatusNotFound:
		debug.Debug("[github] Repository not found")
		return NewNotFoundError(p.Name(), p.formatURL(ref))
	case http.StatusUnauthorized, http.StatusForbidden:
		debug.Debug("[github] Authentication required or forbidden")
		return NewAuthError(p.Name(), p.formatURL(ref))
	default:
		debug.Debug("[github] Unexpected status code: %d", resp.StatusCode)
		return NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}
}

// Fetch downloads a template from GitHub.
func (p *GitHubProvider) Fetch(ctx context.Context, ref model.TemplateRef) (*model.Template, error) {
	debug.Debug("[github] Starting fetch for %s", p.formatURL(ref))

	// Download repository archive (tarball)
	debug.Debug("[github] Downloading archive...")
	archivePath, err := p.downloadArchive(ctx, ref)
	if err != nil {
		debug.Debug("[github] Archive download failed: %v", err)
		return nil, err
	}
	defer func() { _ = os.Remove(archivePath) }() // Clean up archive after extraction
	debug.Debug("[github] Archive downloaded to: %s", archivePath)

	// Extract archive to temporary directory
	debug.Debug("[github] Extracting archive...")
	extractDir, err := p.extractArchive(archivePath)
	if err != nil {
		debug.Debug("[github] Archive extraction failed: %v", err)
		return nil, NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to extract archive: %w", err))
	}
	debug.Debug("[github] Archive extracted to: %s", extractDir)

	// Find template root (handle subdirectory path if specified)
	templateRoot := extractDir
	if ref.Path != "" {
		templateRoot = filepath.Join(extractDir, ref.Path)
		debug.Debug("[github] Looking for subdirectory: %s", ref.Path)
		if _, err := os.Stat(templateRoot); err != nil {
			debug.Debug("[github] Subdirectory not found: %v", err)
			return nil, NewInvalidTemplateError(p.Name(), p.formatURL(ref),
				fmt.Sprintf("subdirectory '%s' not found in template", ref.Path), err)
		}
	}
	debug.Debug("[github] Template root: %s", templateRoot)

	// Read and parse template config file
	debug.Debug("[github] Reading %s...", model.IgnTemplateConfigFile)
	ignConfig, err := p.readIgnConfig(templateRoot)
	if err != nil {
		debug.Debug("[github] Failed to read %s: %v", model.IgnTemplateConfigFile, err)
		return nil, NewInvalidTemplateError(p.Name(), p.formatURL(ref),
			"failed to read "+model.IgnTemplateConfigFile, err)
	}
	debug.Debug("[github] Template name: %s, version: %s", ignConfig.Name, ignConfig.Version)

	// Collect all template files
	debug.Debug("[github] Collecting template files...")
	files, err := p.collectFiles(templateRoot)
	if err != nil {
		debug.Debug("[github] Failed to collect files: %v", err)
		return nil, NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to collect template files: %w", err))
	}
	debug.Debug("[github] Collected %d template files", len(files))

	debug.Debug("[github] Fetch completed successfully")
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
	debug.Debug("[github] Archive URL: %s", archiveURL)

	req, err := http.NewRequestWithContext(ctx, "GET", archiveURL, nil)
	if err != nil {
		debug.Debug("[github] Failed to create download request: %v", err)
		return "", NewFetchError(p.Name(), p.formatURL(ref), err)
	}

	// Add authentication if token is provided
	if p.Token != "" {
		req.Header.Set("Authorization", "token "+p.Token)
	}

	downloadStart := time.Now()
	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		debug.Debug("[github] Download request failed: %v", err)
		return "", NewFetchError(p.Name(), p.formatURL(ref), err)
	}
	defer func() { _ = resp.Body.Close() }()

	debug.Debug("[github] Download response status: %d", resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusOK:
		// Continue to download
	case http.StatusNotFound:
		debug.Debug("[github] Archive not found")
		return "", NewNotFoundError(p.Name(), p.formatURL(ref))
	case http.StatusUnauthorized, http.StatusForbidden:
		debug.Debug("[github] Authentication required for download")
		return "", NewAuthError(p.Name(), p.formatURL(ref))
	default:
		debug.Debug("[github] Unexpected download status: %d", resp.StatusCode)
		return "", NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}

	// Create temporary file for archive
	tmpFile, err := os.CreateTemp("", "ign-github-*.tar.gz")
	if err != nil {
		debug.Debug("[github] Failed to create temp file: %v", err)
		return "", NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to create temp file: %w", err))
	}
	defer func() { _ = tmpFile.Close() }()

	// Download to temp file
	bytesWritten, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		debug.Debug("[github] Failed to write archive: %v", err)
		return "", NewFetchError(p.Name(), p.formatURL(ref),
			fmt.Errorf("failed to download archive: %w", err))
	}

	downloadDuration := time.Since(downloadStart)
	debug.Debug("[github] Downloaded %d bytes in %v", bytesWritten, downloadDuration)

	return tmpFile.Name(), nil
}

// extractArchive extracts a .tar.gz archive to a temporary directory.
func (p *GitHubProvider) extractArchive(archivePath string) (string, error) {
	debug.Debug("[github] Extracting archive: %s", archivePath)

	// Create temporary directory for extraction
	extractDir, err := os.MkdirTemp("", "ign-template-*")
	if err != nil {
		debug.Debug("[github] Failed to create extraction directory: %v", err)
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	debug.Debug("[github] Extraction directory: %s", extractDir)

	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		_ = os.RemoveAll(extractDir)
		debug.Debug("[github] Failed to open archive: %v", err)
		return "", fmt.Errorf("failed to open archive: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		_ = os.RemoveAll(extractDir)
		debug.Debug("[github] Failed to create gzip reader: %v", err)
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Extract files
	var rootDir string
	fileCount := 0
	dirCount := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.RemoveAll(extractDir)
			debug.Debug("[github] Failed to read tar entry: %v", err)
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
			debug.Debug("[github] Archive root directory: %s", rootDir)
		}
		relPath := parts[1]

		// Construct target path
		target := filepath.Join(extractDir, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				_ = os.RemoveAll(extractDir)
				debug.Debug("[github] Failed to create directory %s: %v", target, err)
				return "", fmt.Errorf("failed to create directory %s: %w", target, err)
			}
			dirCount++
		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				_ = os.RemoveAll(extractDir)
				debug.Debug("[github] Failed to create parent directory: %v", err)
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				_ = os.RemoveAll(extractDir)
				debug.Debug("[github] Failed to create file %s: %v", target, err)
				return "", fmt.Errorf("failed to create file %s: %w", target, err)
			}

			// Copy content
			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				_ = os.RemoveAll(extractDir)
				debug.Debug("[github] Failed to write file %s: %v", target, err)
				return "", fmt.Errorf("failed to write file %s: %w", target, err)
			}
			_ = outFile.Close()
			fileCount++
		case tar.TypeSymlink:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				_ = os.RemoveAll(extractDir)
				debug.Debug("[github] Failed to create parent directory for symlink: %v", err)
				return "", fmt.Errorf("failed to create parent directory for symlink: %w", err)
			}

			// Create symbolic link (Linkname is stored as-is in tar archive)
			if err := os.Symlink(header.Linkname, target); err != nil {
				_ = os.RemoveAll(extractDir)
				debug.Debug("[github] Failed to create symlink %s -> %s: %v", target, header.Linkname, err)
				return "", fmt.Errorf("failed to create symlink %s -> %s: %w", target, header.Linkname, err)
			}
			fileCount++
		}
	}

	debug.Debug("[github] Extracted %d files and %d directories", fileCount, dirCount)
	return extractDir, nil
}

// readIgnConfig reads and parses the template config file.
func (p *GitHubProvider) readIgnConfig(templateRoot string) (*model.IgnJson, error) {
	ignPath := filepath.Join(templateRoot, model.IgnTemplateConfigFile)

	data, err := os.ReadFile(ignPath)
	if err != nil {
		return nil, fmt.Errorf("%s not found in template root: %w", model.IgnTemplateConfigFile, err)
	}

	var config model.IgnJson
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", model.IgnTemplateConfigFile, err)
	}

	// Basic validation
	if config.Name == "" {
		return nil, fmt.Errorf("%s missing required field: name", model.IgnTemplateConfigFile)
	}
	if config.Version == "" {
		return nil, fmt.Errorf("%s missing required field: version", model.IgnTemplateConfigFile)
	}

	return &config, nil
}

// collectFiles recursively collects all files in the template directory.
// Excludes template config file as it's not part of the template output.
func (p *GitHubProvider) collectFiles(templateRoot string) ([]model.TemplateFile, error) {
	var files []model.TemplateFile
	var totalBytes int64

	err := filepath.Walk(templateRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Handle errors from Walk (e.g., broken symlinks encountered by Lstat).
			// If the entry is a broken symlink, skip it gracefully instead of failing.
			if os.IsNotExist(err) {
				debug.Debug("[github] Skipping broken symlink: %s", path)
				return nil
			}
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Handle symlinks: filepath.Walk uses os.Lstat, so symlinks appear as
		// non-directory, non-regular entries. Preserve symlinks as symlink entries
		// so the generator can recreate them in the output.
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				debug.Debug("[github] Skipping unreadable symlink: %s: %v", path, err)
				return nil
			}

			relPath, err := filepath.Rel(templateRoot, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path for symlink: %w", err)
			}

			// Skip template config file even if it's a symlink
			if filepath.Base(path) == model.IgnTemplateConfigFile {
				return nil
			}

			debug.Debug("[github] Collecting symlink: %s -> %s", relPath, linkTarget)
			files = append(files, model.TemplateFile{
				Path:          relPath,
				Mode:          info.Mode(),
				SymlinkTarget: linkTarget,
			})
			return nil
		}

		// Skip non-regular, non-symlink files (devices, sockets, named pipes, etc.)
		if info.Mode()&os.ModeSymlink == 0 && !info.Mode().IsRegular() {
			debug.Debug("[github] Skipping non-regular file: %s", path)
			return nil
		}

		// Skip template config file (config file, not template content)
		if filepath.Base(path) == model.IgnTemplateConfigFile {
			return nil
		}

		// Get relative path from template root
		relPath, err := filepath.Rel(templateRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Read file content (use os.ReadFile which follows symlinks)
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

		totalBytes += int64(len(content))

		return nil
	})

	if err != nil {
		return nil, err
	}

	debug.Debug("[github] Collected %d files, total size: %d bytes", len(files), totalBytes)
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
