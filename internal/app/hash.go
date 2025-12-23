package app

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashableFile represents a file with path and content for hash calculation.
type HashableFile struct {
	// Path is the relative path of the file.
	Path string
	// Content is the file content.
	Content []byte
}

// HashTemplateFiles calculates SHA256 hash from file path/content pairs.
// The input files must already be sorted by path for deterministic results.
// Uses null byte separators between path and content, and between files,
// to prevent hash collisions from different file combinations.
func HashTemplateFiles(files []HashableFile) string {
	if len(files) == 0 {
		return ""
	}

	h := sha256.New()

	for _, file := range files {
		h.Write([]byte(file.Path))
		h.Write([]byte("\x00")) // Separator between path and content
		h.Write(file.Content)
		h.Write([]byte("\x00")) // Separator between files
	}

	return hex.EncodeToString(h.Sum(nil))
}
