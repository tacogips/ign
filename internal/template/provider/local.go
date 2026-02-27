package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
)

// LocalProvider implements Provider for local filesystem templates.
type LocalProvider struct {
	// BaseDir is the base directory for resolving relative paths.
	// If empty, uses current working directory.
	BaseDir string
}

// NewLocalProvider creates a new local filesystem provider.
func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

// NewLocalProviderWithBase creates a new local provider with a base directory.
func NewLocalProviderWithBase(baseDir string) *LocalProvider {
	return &LocalProvider{
		BaseDir: baseDir,
	}
}

// Name returns the provider name.
func (p *LocalProvider) Name() string {
	return "local"
}

// Resolve converts a local path to a TemplateRef.
func (p *LocalProvider) Resolve(path string) (model.TemplateRef, error) {
	debug.Debug("[local] Resolving path: %s", path)

	// If it's a file:// URL, extract the path
	originalPath := path
	if strings.HasPrefix(path, "file://") {
		var err error
		path, err = ParseFileURL(path)
		if err != nil {
			debug.Debug("[local] Failed to parse file:// URL: %v", err)
			return model.TemplateRef{}, NewInvalidURLError(p.Name(), originalPath, err)
		}
		debug.Debug("[local] Extracted path from file:// URL: %s", path)
	}

	// Validate local path
	if err := ValidateLocalPath(path); err != nil {
		debug.Debug("[local] Path validation failed: %v", err)
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), originalPath, err)
	}

	// Normalize path
	normalized, err := NormalizeLocalPath(path)
	if err != nil {
		debug.Debug("[local] Path normalization failed: %v", err)
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), originalPath, err)
	}
	debug.Debug("[local] Normalized path: %s", normalized)

	// Resolve absolute path
	absPath, err := p.resolvePath(normalized)
	if err != nil {
		debug.Debug("[local] Path resolution failed: %v", err)
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), path, err)
	}
	debug.Debug("[local] Absolute path: %s", absPath)

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			debug.Debug("[local] Path does not exist: %s", absPath)
			return model.TemplateRef{}, NewNotFoundError(p.Name(), path)
		}
		debug.Debug("[local] Failed to stat path: %v", err)
		return model.TemplateRef{}, NewFetchError(p.Name(), path, err)
	}

	debug.Debug("[local] Path resolved successfully")
	return model.TemplateRef{
		Provider: "local",
		Owner:    "",
		Repo:     "",
		Path:     normalized,
		Ref:      "",
	}, nil
}

// Validate checks if a local path is valid and accessible.
func (p *LocalProvider) Validate(ctx context.Context, ref model.TemplateRef) error {
	debug.Debug("[local] Validating template reference: %s", ref.Path)

	if ref.Provider != "local" {
		debug.Debug("[local] Invalid provider: %s", ref.Provider)
		return NewInvalidURLError(p.Name(), ref.Path,
			fmt.Errorf("invalid provider: expected 'local', got '%s'", ref.Provider))
	}

	// Validate path security
	debug.Debug("[local] Validating path security...")
	if err := ValidateLocalPath(ref.Path); err != nil {
		debug.Debug("[local] Path security validation failed: %v", err)
		return NewInvalidURLError(p.Name(), ref.Path, err)
	}

	// Resolve to absolute path
	debug.Debug("[local] Resolving to absolute path...")
	absPath, err := p.resolvePath(ref.Path)
	if err != nil {
		debug.Debug("[local] Failed to resolve path: %v", err)
		return NewFetchError(p.Name(), ref.Path, err)
	}
	debug.Debug("[local] Absolute path: %s", absPath)

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			debug.Debug("[local] Path does not exist: %s", absPath)
			return NewNotFoundError(p.Name(), ref.Path)
		}
		debug.Debug("[local] Failed to stat path: %v", err)
		return NewFetchError(p.Name(), ref.Path, err)
	}

	// Must be a directory
	if !info.IsDir() {
		debug.Debug("[local] Path is not a directory")
		return NewInvalidTemplateError(p.Name(), ref.Path,
			"path must be a directory", nil)
	}
	debug.Debug("[local] Path is a directory")

	// Check if template config file exists
	ignPath := filepath.Join(absPath, model.IgnTemplateConfigFile)
	debug.Debug("[local] Checking for %s at: %s", model.IgnTemplateConfigFile, ignPath)
	if _, err := os.Stat(ignPath); err != nil {
		if os.IsNotExist(err) {
			debug.Debug("[local] %s not found", model.IgnTemplateConfigFile)
			return NewInvalidTemplateError(p.Name(), ref.Path,
				model.IgnTemplateConfigFile+" not found in template directory", nil)
		}
		debug.Debug("[local] Failed to check %s: %v", model.IgnTemplateConfigFile, err)
		return NewFetchError(p.Name(), ref.Path, err)
	}

	debug.Debug("[local] Validation successful")
	return nil
}

// Fetch loads a template from the local filesystem.
func (p *LocalProvider) Fetch(ctx context.Context, ref model.TemplateRef) (*model.Template, error) {
	debug.Debug("[local] Starting fetch for: %s", ref.Path)

	if ref.Provider != "local" {
		debug.Debug("[local] Invalid provider: %s", ref.Provider)
		return nil, NewInvalidURLError(p.Name(), ref.Path,
			fmt.Errorf("invalid provider: expected 'local', got '%s'", ref.Provider))
	}

	// Validate path
	debug.Debug("[local] Validating path security...")
	if err := ValidateLocalPath(ref.Path); err != nil {
		debug.Debug("[local] Path validation failed: %v", err)
		return nil, NewInvalidURLError(p.Name(), ref.Path, err)
	}

	// Resolve to absolute path
	debug.Debug("[local] Resolving to absolute path...")
	absPath, err := p.resolvePath(ref.Path)
	if err != nil {
		debug.Debug("[local] Path resolution failed: %v", err)
		return nil, NewFetchError(p.Name(), ref.Path, err)
	}
	debug.Debug("[local] Template root: %s", absPath)

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			debug.Debug("[local] Path does not exist: %s", absPath)
			return nil, NewNotFoundError(p.Name(), ref.Path)
		}
		debug.Debug("[local] Failed to stat path: %v", err)
		return nil, NewFetchError(p.Name(), ref.Path, err)
	}

	if !info.IsDir() {
		debug.Debug("[local] Path is not a directory")
		return nil, NewInvalidTemplateError(p.Name(), ref.Path,
			"path must be a directory", nil)
	}

	// Read and parse template config file
	debug.Debug("[local] Reading %s...", model.IgnTemplateConfigFile)
	ignConfig, err := p.readIgnConfig(absPath)
	if err != nil {
		debug.Debug("[local] Failed to read %s: %v", model.IgnTemplateConfigFile, err)
		return nil, NewInvalidTemplateError(p.Name(), ref.Path,
			"failed to read "+model.IgnTemplateConfigFile, err)
	}
	debug.Debug("[local] Template name: %s, version: %s", ignConfig.Name, ignConfig.Version)

	// Collect all template files
	debug.Debug("[local] Collecting template files...")
	files, err := p.collectFiles(absPath)
	if err != nil {
		debug.Debug("[local] Failed to collect files: %v", err)
		return nil, NewFetchError(p.Name(), ref.Path,
			fmt.Errorf("failed to collect template files: %w", err))
	}
	debug.Debug("[local] Collected %d template files", len(files))

	debug.Debug("[local] Fetch completed successfully")
	return &model.Template{
		Ref:      ref,
		Config:   *ignConfig,
		Files:    files,
		RootPath: absPath,
	}, nil
}

// resolvePath resolves a path to an absolute path.
// If the path is already absolute, it returns it directly.
// If the path is relative, it resolves it relative to BaseDir or current working directory.
func (p *LocalProvider) resolvePath(path string) (string, error) {
	// If path is already absolute, use it directly
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	// For relative paths, resolve relative to base directory
	baseDir := p.BaseDir
	if baseDir == "" {
		// Use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		baseDir = cwd
	}

	absPath := filepath.Join(baseDir, path)

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Verify it doesn't escape base directory (security check for relative paths only)
	if !filepath.IsAbs(absPath) {
		return "", fmt.Errorf("resolved path is not absolute: %s", absPath)
	}

	// Ensure it's under base directory (no ".." escaping)
	if !isSubPath(baseDir, absPath) {
		return "", fmt.Errorf("path escapes base directory: %s", path)
	}

	return absPath, nil
}

// isSubPath checks if child is under parent directory.
func isSubPath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	// Make both absolute for comparison
	if !filepath.IsAbs(parent) {
		return false
	}
	if !filepath.IsAbs(child) {
		return false
	}

	// Check if child starts with parent
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	// If rel starts with "..", it's outside parent
	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}

// readIgnConfig reads and parses the template config file.
func (p *LocalProvider) readIgnConfig(templateRoot string) (*model.IgnJson, error) {
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
func (p *LocalProvider) collectFiles(templateRoot string) ([]model.TemplateFile, error) {
	var files []model.TemplateFile
	var totalBytes int64

	err := filepath.Walk(templateRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Handle errors from Walk (e.g., broken symlinks encountered by Lstat).
			// If the entry is a broken symlink, skip it gracefully instead of failing.
			if os.IsNotExist(err) {
				debug.Debug("[local] Skipping broken symlink: %s", path)
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
				debug.Debug("[local] Skipping unreadable symlink: %s: %v", path, err)
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

			debug.Debug("[local] Collecting symlink: %s -> %s", relPath, linkTarget)
			files = append(files, model.TemplateFile{
				Path:          relPath,
				Mode:          info.Mode(),
				SymlinkTarget: linkTarget,
			})
			return nil
		}

		// Skip non-regular, non-symlink files (devices, sockets, named pipes, etc.)
		if info.Mode()&os.ModeSymlink == 0 && !info.Mode().IsRegular() {
			debug.Debug("[local] Skipping non-regular file: %s", path)
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

	debug.Debug("[local] Collected %d files, total size: %d bytes", len(files), totalBytes)
	return files, nil
}

// isBinaryContent checks if content appears to be binary.
// Simple heuristic: check first 512 bytes for null bytes.
func (p *LocalProvider) isBinaryContent(content []byte) bool {
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
