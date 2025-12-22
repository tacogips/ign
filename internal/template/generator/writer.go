package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tacogips/ign/internal/debug"
)

// Writer writes files to the filesystem.
type Writer interface {
	// WriteFile writes content to a file with the specified permissions.
	WriteFile(path string, content []byte, mode os.FileMode) error

	// CreateDir creates a directory and any necessary parent directories.
	CreateDir(path string) error

	// Exists checks if a file or directory exists at the given path.
	Exists(path string) bool
}

// FileWriter implements Writer for filesystem operations.
type FileWriter struct {
	preserveExecutable bool
}

// NewFileWriter creates a new FileWriter.
// If preserveExecutable is true, executable permissions from the source will be preserved.
// Otherwise, files are created with default permissions (0644).
func NewFileWriter(preserveExecutable bool) Writer {
	return &FileWriter{
		preserveExecutable: preserveExecutable,
	}
}

// WriteFile writes content to a file with the specified permissions.
// Creates parent directories if they don't exist.
// Writes atomically using a temporary file and rename.
func (w *FileWriter) WriteFile(path string, content []byte, mode os.FileMode) error {
	debug.Debug("[generator] Writing file: %s (size: %d bytes, mode: %o)", path, len(content), mode)

	// Create parent directories if needed
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := w.CreateDir(dir); err != nil {
			return newGeneratorError(GeneratorWriteFailed,
				"failed to create parent directory",
				path,
				err)
		}
	}

	// Determine file mode
	fileMode := mode
	if !w.preserveExecutable {
		// Use default mode 0644 for regular files
		fileMode = 0644
	} else {
		// Preserve executable bit if set
		// Ensure at least read/write for owner
		if fileMode&0600 == 0 {
			fileMode = fileMode | 0600
		}
	}

	// Write atomically using temporary file
	tempFile := path + ".tmp"
	debug.Debug("[generator] Creating temporary file: %s", tempFile)
	f, err := os.OpenFile(tempFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return newGeneratorError(GeneratorWriteFailed,
			"failed to create temporary file",
			path,
			err)
	}

	// Write content
	_, err = f.Write(content)
	closeErr := f.Close()

	if err != nil {
		_ = os.Remove(tempFile) // Clean up temp file
		return newGeneratorError(GeneratorWriteFailed,
			"failed to write file content",
			path,
			err)
	}

	if closeErr != nil {
		_ = os.Remove(tempFile) // Clean up temp file
		return newGeneratorError(GeneratorWriteFailed,
			"failed to close file",
			path,
			closeErr)
	}

	// Atomic rename
	debug.Debug("[generator] Renaming temporary file: %s -> %s", tempFile, path)
	if err := os.Rename(tempFile, path); err != nil {
		_ = os.Remove(tempFile) // Clean up temp file
		return newGeneratorError(GeneratorWriteFailed,
			"failed to rename temporary file",
			path,
			err)
	}

	debug.Debug("[generator] File written successfully: %s", path)
	return nil
}

// CreateDir creates a directory and any necessary parent directories.
// Uses 0755 permissions for created directories.
func (w *FileWriter) CreateDir(path string) error {
	debug.Debug("[generator] Creating directory: %s", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return newGeneratorError(GeneratorWriteFailed,
			"failed to create directory",
			path,
			err)
	}
	debug.Debug("[generator] Directory created: %s", path)
	return nil
}

// Exists checks if a file or directory exists at the given path.
func (w *FileWriter) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CopyFile is a utility function to copy a file from src to dst.
// This is useful for binary files that should be copied as-is.
func CopyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	// Create parent directories
	dir := filepath.Dir(dst)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}
