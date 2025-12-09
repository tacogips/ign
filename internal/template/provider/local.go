package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
	// Validate local path
	if err := ValidateLocalPath(path); err != nil {
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), path, err)
	}

	// Normalize path
	normalized, err := NormalizeLocalPath(path)
	if err != nil {
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), path, err)
	}

	// Resolve absolute path
	absPath, err := p.resolvePath(normalized)
	if err != nil {
		return model.TemplateRef{}, NewInvalidURLError(p.Name(), path, err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return model.TemplateRef{}, NewNotFoundError(p.Name(), path)
		}
		return model.TemplateRef{}, NewFetchError(p.Name(), path, err)
	}

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
	if ref.Provider != "local" {
		return NewInvalidURLError(p.Name(), ref.Path,
			fmt.Errorf("invalid provider: expected 'local', got '%s'", ref.Provider))
	}

	// Validate path security
	if err := ValidateLocalPath(ref.Path); err != nil {
		return NewInvalidURLError(p.Name(), ref.Path, err)
	}

	// Resolve to absolute path
	absPath, err := p.resolvePath(ref.Path)
	if err != nil {
		return NewFetchError(p.Name(), ref.Path, err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewNotFoundError(p.Name(), ref.Path)
		}
		return NewFetchError(p.Name(), ref.Path, err)
	}

	// Must be a directory
	if !info.IsDir() {
		return NewInvalidTemplateError(p.Name(), ref.Path,
			"path must be a directory", nil)
	}

	// Check if ign.json exists
	ignPath := filepath.Join(absPath, "ign.json")
	if _, err := os.Stat(ignPath); err != nil {
		if os.IsNotExist(err) {
			return NewInvalidTemplateError(p.Name(), ref.Path,
				"ign.json not found in template directory", nil)
		}
		return NewFetchError(p.Name(), ref.Path, err)
	}

	return nil
}

// Fetch loads a template from the local filesystem.
func (p *LocalProvider) Fetch(ctx context.Context, ref model.TemplateRef) (*model.Template, error) {
	if ref.Provider != "local" {
		return nil, NewInvalidURLError(p.Name(), ref.Path,
			fmt.Errorf("invalid provider: expected 'local', got '%s'", ref.Provider))
	}

	// Validate path
	if err := ValidateLocalPath(ref.Path); err != nil {
		return nil, NewInvalidURLError(p.Name(), ref.Path, err)
	}

	// Resolve to absolute path
	absPath, err := p.resolvePath(ref.Path)
	if err != nil {
		return nil, NewFetchError(p.Name(), ref.Path, err)
	}

	// Check if directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewNotFoundError(p.Name(), ref.Path)
		}
		return nil, NewFetchError(p.Name(), ref.Path, err)
	}

	if !info.IsDir() {
		return nil, NewInvalidTemplateError(p.Name(), ref.Path,
			"path must be a directory", nil)
	}

	// Read and parse ign.json
	ignConfig, err := p.readIgnConfig(absPath)
	if err != nil {
		return nil, NewInvalidTemplateError(p.Name(), ref.Path,
			"failed to read ign.json", err)
	}

	// Collect all template files
	files, err := p.collectFiles(absPath)
	if err != nil {
		return nil, NewFetchError(p.Name(), ref.Path,
			fmt.Errorf("failed to collect template files: %w", err))
	}

	return &model.Template{
		Ref:      ref,
		Config:   *ignConfig,
		Files:    files,
		RootPath: absPath,
	}, nil
}

// resolvePath resolves a relative path to an absolute path.
func (p *LocalProvider) resolvePath(relPath string) (string, error) {
	baseDir := p.BaseDir
	if baseDir == "" {
		// Use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		baseDir = cwd
	}

	absPath := filepath.Join(baseDir, relPath)

	// Clean the path
	absPath = filepath.Clean(absPath)

	// Verify it doesn't escape base directory (security check)
	if !filepath.IsAbs(absPath) {
		return "", fmt.Errorf("resolved path is not absolute: %s", absPath)
	}

	// Ensure it's under base directory (no ".." escaping)
	if !isSubPath(baseDir, absPath) {
		return "", fmt.Errorf("path escapes base directory: %s", relPath)
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
	return !filepath.IsAbs(rel) && !filepath.HasPrefix(rel, "..")
}

// readIgnConfig reads and parses the ign.json file.
func (p *LocalProvider) readIgnConfig(templateRoot string) (*model.IgnJson, error) {
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
func (p *LocalProvider) collectFiles(templateRoot string) ([]model.TemplateFile, error) {
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
