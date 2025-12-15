package app

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tacogips/ign/internal/debug"
)

//go:embed scaffolds/*
var scaffoldsFS embed.FS

// NewTemplateOptions holds options for creating a new template.
type NewTemplateOptions struct {
	// Path is the destination directory for the new template.
	Path string
	// Type is the scaffold type to use (e.g., "default", "go", "web").
	Type string
	// Force overwrites existing files if true.
	Force bool
}

// NewTemplateResult holds the result of template creation.
type NewTemplateResult struct {
	// Path is the created template directory path.
	Path string
	// FilesCreated is the number of files created.
	FilesCreated int
	// Files is the list of created file paths.
	Files []string
}

// AvailableScaffoldTypes returns the list of available scaffold types.
func AvailableScaffoldTypes() ([]string, error) {
	entries, err := scaffoldsFS.ReadDir("scaffolds")
	if err != nil {
		return nil, fmt.Errorf("failed to read scaffolds directory: %w", err)
	}

	var types []string
	for _, entry := range entries {
		if entry.IsDir() {
			types = append(types, entry.Name())
		}
	}
	return types, nil
}

// NewTemplate creates a new template from scaffold.
func NewTemplate(ctx context.Context, opts NewTemplateOptions) (*NewTemplateResult, error) {
	debug.DebugSection("[app] NewTemplate workflow start")
	debug.DebugValue("[app] Target path", opts.Path)
	debug.DebugValue("[app] Scaffold type", opts.Type)
	debug.DebugValue("[app] Force overwrite", opts.Force)

	// Validate scaffold type
	scaffoldPath := filepath.Join("scaffolds", opts.Type)
	if _, err := scaffoldsFS.ReadDir(scaffoldPath); err != nil {
		availableTypes, _ := AvailableScaffoldTypes()
		return nil, NewValidationError(
			fmt.Sprintf("unknown scaffold type: %s (available: %v)", opts.Type, availableTypes),
			err,
		)
	}

	// Resolve target path
	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, NewValidationError("failed to resolve target path", err)
	}
	debug.DebugValue("[app] Absolute target path", absPath)

	// Check if target exists
	if info, err := os.Stat(absPath); err == nil {
		if !info.IsDir() {
			return nil, NewValidationError(
				fmt.Sprintf("target path exists and is not a directory: %s", absPath),
				nil,
			)
		}
		// Directory exists, check if it's empty or force is enabled
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, NewValidationError("failed to read target directory", err)
		}
		if len(entries) > 0 && !opts.Force {
			return nil, NewValidationError(
				fmt.Sprintf("target directory is not empty: %s (use --force to overwrite)", absPath),
				nil,
			)
		}
	}

	// Create target directory
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, NewValidationError("failed to create target directory", err)
	}

	result := &NewTemplateResult{
		Path:         absPath,
		FilesCreated: 0,
		Files:        []string{},
	}

	// Copy scaffold files to target
	err = fs.WalkDir(scaffoldsFS, scaffoldPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from scaffold root
		relPath, err := filepath.Rel(scaffoldPath, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(absPath, relPath)
		debug.DebugValue("[app] Processing", relPath)

		if d.IsDir() {
			// Create subdirectory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			return nil
		}

		// Read file content from embedded FS
		content, err := scaffoldsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read scaffold file %s: %w", path, err)
		}

		// Write file to target
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		result.FilesCreated++
		result.Files = append(result.Files, relPath)
		debug.DebugValue("[app] Created file", targetPath)

		return nil
	})

	if err != nil {
		return nil, NewValidationError("failed to copy scaffold files", err)
	}

	debug.Debug("[app] NewTemplate workflow completed")
	debug.DebugValue("[app] Files created", result.FilesCreated)

	return result, nil
}
