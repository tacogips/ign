package cli

import (
	"testing"
)

// TestValidateGitHubURL tests URL validation and normalization
func TestValidateGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "full HTTPS URL",
			url:  "https://github.com/owner/repo",
			want: "https://github.com/owner/repo",
		},
		{
			name: "HTTPS URL with subdirectory",
			url:  "https://github.com/owner/repo/templates/go-basic",
			want: "https://github.com/owner/repo/templates/go-basic",
		},
		{
			name: "git SSH URL",
			url:  "git@github.com:owner/repo.git",
			want: "git@github.com:owner/repo.git",
		},
		{
			name: "short form",
			url:  "github.com/owner/repo",
			want: "https://github.com/owner/repo",
		},
		{
			name: "short form with subdirectory",
			url:  "github.com/owner/repo/templates/go-basic",
			want: "https://github.com/owner/repo/templates/go-basic",
		},
		{
			name: "owner/repo format",
			url:  "owner/repo",
			want: "https://github.com/owner/repo",
		},
		{
			name: "owner/repo with subdirectory",
			url:  "owner/repo/templates/go-basic",
			want: "https://github.com/owner/repo/templates/go-basic",
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateGitHubURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateGitHubURL() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateGitHubURL() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateGitHubURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestValidateGitRef tests git reference validation
func TestValidateGitRef(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		wantErr bool
	}{
		{
			name: "branch name",
			ref:  "main",
		},
		{
			name: "branch with slash",
			ref:  "feature/new-feature",
		},
		{
			name: "semantic version tag",
			ref:  "v1.2.3",
		},
		{
			name: "semantic version without v",
			ref:  "1.2.3",
		},
		{
			name: "semantic version with prerelease",
			ref:  "v1.2.3-beta.1",
		},
		{
			name: "full commit SHA",
			ref:  "abc123def456789012345678901234567890abcd",
		},
		{
			name: "short commit SHA",
			ref:  "abc123d",
		},
		{
			name:    "empty ref",
			ref:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateGitRef() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateGitRef() unexpected error: %v", err)
			}
		})
	}
}

// TestValidateOutputPath tests output path validation
func TestValidateOutputPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name: "simple path",
			path: "./output",
		},
		{
			name: "current directory",
			path: ".",
		},
		{
			name: "nested path",
			path: "./my-project/subdir",
		},
		{
			name:    "path traversal",
			path:    "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateOutputPath() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateOutputPath() unexpected error: %v", err)
			}
		})
	}
}

// TestVersionCommand tests version command output
func TestVersionCommand(t *testing.T) {
	// Set test version info
	Version = "1.0.0-test"
	GitCommit = "abc123"
	BuildDate = "2025-12-09"

	t.Run("normal output", func(t *testing.T) {
		// Reset flags
		versionShort = false
		versionJSON = false

		// Run command - output goes to stdout which we can't easily capture
		// Just test that it doesn't error
		err := runVersion(versionCmd, []string{})
		if err != nil {
			t.Errorf("runVersion() unexpected error: %v", err)
		}
	})

	t.Run("short output", func(t *testing.T) {
		versionShort = true
		versionJSON = false

		err := runVersion(versionCmd, []string{})
		if err != nil {
			t.Errorf("runVersion() unexpected error: %v", err)
		}
	})

	t.Run("JSON output", func(t *testing.T) {
		versionShort = false
		versionJSON = true

		err := runVersion(versionCmd, []string{})
		if err != nil {
			t.Errorf("runVersion() unexpected error: %v", err)
		}
	})
}

// TestFormatBytes tests byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "bytes",
			bytes: 512,
			want:  "512 B",
		},
		{
			name:  "kilobytes",
			bytes: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "megabytes",
			bytes: 1048576,
			want:  "1.0 MB",
		},
		{
			name:  "gigabytes",
			bytes: 1073741824,
			want:  "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
