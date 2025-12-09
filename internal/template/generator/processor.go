package generator

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

// Processor processes individual files during generation.
type Processor interface {
	// Process processes a single template file and returns the processed content.
	// For binary files, returns the content unchanged.
	// For text files, processes template directives using the parser.
	Process(ctx context.Context, file model.TemplateFile, vars parser.Variables, templateRoot string) ([]byte, error)

	// ShouldProcess determines if a file should be template-processed.
	// Returns false for binary files.
	ShouldProcess(file model.TemplateFile) bool
}

// FileProcessor implements Processor for file processing.
type FileProcessor struct {
	parser           parser.Parser
	binaryExtensions []string
}

// NewFileProcessor creates a new FileProcessor.
// binaryExtensions is a list of file extensions that should be treated as binary.
func NewFileProcessor(p parser.Parser, binaryExtensions []string) Processor {
	if binaryExtensions == nil {
		binaryExtensions = defaultBinaryExtensions()
	}
	return &FileProcessor{
		parser:           p,
		binaryExtensions: binaryExtensions,
	}
}

// defaultBinaryExtensions returns a default list of binary file extensions.
func defaultBinaryExtensions() []string {
	return []string{
		// Images
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".svg",
		// Archives
		".zip", ".tar", ".gz", ".bz2", ".xz", ".rar", ".7z",
		// Executables
		".exe", ".dll", ".so", ".dylib", ".bin",
		// Media
		".mp3", ".mp4", ".avi", ".mov", ".wav",
		// Documents
		".pdf", ".doc", ".docx", ".xls", ".xlsx",
		// Fonts
		".ttf", ".otf", ".woff", ".woff2",
	}
}

// ShouldProcess determines if a file should be template-processed.
// Returns false if:
// - file.IsBinary is true
// - file extension matches binary extensions
// - file content appears to be binary (contains null bytes in first 512 bytes)
func (p *FileProcessor) ShouldProcess(file model.TemplateFile) bool {
	// Check if explicitly marked as binary
	if file.IsBinary {
		return false
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Path))
	for _, binaryExt := range p.binaryExtensions {
		if ext == binaryExt {
			return false
		}
	}

	// Check content for binary markers (null bytes in first 512 bytes)
	if isBinaryContent(file.Content) {
		return false
	}

	return true
}

// isBinaryContent checks if content appears to be binary by looking for null bytes.
// Checks the first 512 bytes (or entire content if smaller).
func isBinaryContent(content []byte) bool {
	// Check up to 512 bytes for null bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}

	return bytes.IndexByte(content[:checkLen], 0) != -1
}

// Process processes a single template file.
// For binary files or files that should not be processed, returns content unchanged.
// For text files, processes template directives using the parser.
func (p *FileProcessor) Process(ctx context.Context, file model.TemplateFile, vars parser.Variables, templateRoot string) ([]byte, error) {
	// If file should not be processed, return unchanged
	if !p.ShouldProcess(file) {
		debug.Debug("[generator] Skipping template processing for binary/special file: %s (size: %d bytes)",
			file.Path, len(file.Content))
		return file.Content, nil
	}

	debug.Debug("[generator] Processing template content: %s (size: %d bytes)", file.Path, len(file.Content))

	// Create parse context with template root and current file
	pctx := &parser.ParseContext{
		Variables:    vars,
		IncludeDepth: 0,
		IncludeStack: []string{},
		TemplateRoot: templateRoot,
		CurrentFile:  file.Path,
	}

	// Process template directives
	processed, err := p.parser.ParseWithContext(ctx, file.Content, pctx)
	if err != nil {
		debug.Debug("[generator] Failed to process template: %s, error: %v", file.Path, err)
		return nil, newGeneratorError(GeneratorProcessFailed,
			"failed to process template",
			file.Path,
			err)
	}

	debug.Debug("[generator] Template processing complete: %s (input: %d bytes, output: %d bytes)",
		file.Path, len(file.Content), len(processed))

	return processed, nil
}
