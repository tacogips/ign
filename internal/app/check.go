package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/template/parser"
)

// CheckTemplateOptions holds options for template validation.
type CheckTemplateOptions struct {
	// Path is the file or directory path to check.
	Path string
	// Recursive indicates whether to check subdirectories.
	Recursive bool
	// Verbose indicates whether to show detailed output.
	Verbose bool
}

// CheckResult holds the results of template validation.
type CheckResult struct {
	// FilesChecked is the number of files checked.
	FilesChecked int
	// FilesWithErrors is the number of files with validation errors.
	FilesWithErrors int
	// Errors is the list of validation errors found.
	Errors []CheckError
}

// CheckError represents a validation error in a template file.
type CheckError struct {
	// File is the file path where the error occurred.
	File string
	// Line is the line number (0 if not applicable).
	Line int
	// Message is the error message.
	Message string
	// Directive is the directive that caused the error (if applicable).
	Directive string
}

// binaryFileExtensions contains common binary file extensions to skip.
var binaryFileExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".ico": true, ".svg": true, ".webp": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".7z": true, ".rar": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mkv": true, ".mov": true, ".wav": true,
	".bin": true, ".dat": true, ".db": true, ".sqlite": true,
}

// CheckTemplate validates template files for syntax errors.
func CheckTemplate(ctx context.Context, opts CheckTemplateOptions) (*CheckResult, error) {
	result := &CheckResult{
		FilesChecked:    0,
		FilesWithErrors: 0,
		Errors:          []CheckError{},
	}

	p := parser.NewParser()

	// Get absolute path
	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, NewValidationError("failed to get absolute path", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, NewValidationError(fmt.Sprintf("path not found: %s", absPath), err)
	}

	// Process based on file or directory
	if info.IsDir() {
		err = checkDirectory(ctx, p, absPath, opts.Recursive, result)
	} else {
		err = checkFile(ctx, p, absPath, result)
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

// checkDirectory recursively checks all files in a directory.
func checkDirectory(ctx context.Context, p parser.Parser, dirPath string, recursive bool, result *CheckResult) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return NewValidationError(fmt.Sprintf("failed to read directory: %s", dirPath), err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		// Skip hidden files and directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.IsDir() {
			if recursive {
				if err := checkDirectory(ctx, p, fullPath, recursive, result); err != nil {
					return err
				}
			}
			continue
		}

		// Check if it's a file
		if entry.Type().IsRegular() {
			if err := checkFile(ctx, p, fullPath, result); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkFile validates a single template file.
func checkFile(ctx context.Context, p parser.Parser, filePath string, result *CheckResult) error {
	// Skip binary files
	if isBinaryFile(filePath) {
		return nil
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		// Skip files that cannot be read (permissions, etc.)
		return nil
	}

	// Check if file contains any @ign- directives
	if !containsIgnDirective(content) {
		return nil
	}

	// File contains directives, so validate it
	result.FilesChecked++

	// Validate template syntax
	if err := p.Validate(ctx, content); err != nil {
		result.FilesWithErrors++

		// Extract error details
		checkErr := CheckError{
			File:    filePath,
			Line:    0,
			Message: err.Error(),
		}

		// Try to extract directive from error message if present
		if strings.Contains(err.Error(), "@ign-") {
			// Extract directive from error message
			parts := strings.Split(err.Error(), "@ign-")
			if len(parts) > 1 {
				directivePart := strings.Split(parts[1], "@")
				if len(directivePart) > 0 {
					checkErr.Directive = "@ign-" + directivePart[0] + "@"
				}
			}
		}

		result.Errors = append(result.Errors, checkErr)
	}

	return nil
}

// isBinaryFile checks if a file is likely a binary file based on extension.
func isBinaryFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return binaryFileExtensions[ext]
}

// containsIgnDirective checks if content contains any @ign- directive.
func containsIgnDirective(content []byte) bool {
	return strings.Contains(string(content), "@ign-")
}
