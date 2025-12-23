package cli

import (
	"testing"
)

func TestTruncateHash(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		expected string
	}{
		{
			name:     "empty string",
			hash:     "",
			expected: "",
		},
		{
			name:     "short hash (less than 16 chars)",
			hash:     "abc123",
			expected: "abc123",
		},
		{
			name:     "exactly 16 chars - no truncation",
			hash:     "0123456789abcdef",
			expected: "0123456789abcdef",
		},
		{
			name:     "17 chars - triggers truncation",
			hash:     "0123456789abcdef0",
			expected: "01234567...9abcdef0",
		},
		{
			name:     "normal git hash (40 chars)",
			hash:     "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
			expected: "a1b2c3d4...q7r8s9t0",
		},
		{
			name:     "long hash (64 chars)",
			hash:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: "01234567...89abcdef",
		},
		{
			name:     "very long hash (100 chars)",
			hash:     "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			expected: "01234567...23456789",
		},
		{
			name:     "single character",
			hash:     "a",
			expected: "a",
		},
		{
			name:     "exactly at boundary (8 chars)",
			hash:     "12345678",
			expected: "12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateHash(tt.hash)
			if result != tt.expected {
				t.Errorf("truncateHash(%q) = %q, want %q", tt.hash, result, tt.expected)
			}
		})
	}
}
