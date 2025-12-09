package provider

import (
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		want      *model.TemplateRef
		wantErr   bool
		errSubstr string
	}{
		{
			name: "full https URL",
			url:  "https://github.com/owner/repo",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name: "https URL with subdirectory",
			url:  "https://github.com/owner/repo/tree/main/templates/go",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Path:     "templates/go",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name: "git@ SSH URL",
			url:  "git@github.com:owner/repo.git",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name: "github.com prefix",
			url:  "github.com/owner/repo",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name: "github.com with subdirectory",
			url:  "github.com/owner/repo/templates/python",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Path:     "templates/python",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name: "owner/repo format",
			url:  "owner/repo",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name: "owner/repo/path format",
			url:  "owner/repo/path/to/template",
			want: &model.TemplateRef{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Path:     "path/to/template",
				Ref:      "main",
			},
			wantErr: false,
		},
		{
			name:      "empty URL",
			url:       "",
			want:      nil,
			wantErr:   true,
			errSubstr: "empty",
		},
		{
			name:      "invalid format - no owner",
			url:       "repo",
			want:      nil,
			wantErr:   true,
			errSubstr: "owner/repo",
		},
		{
			name:      "invalid format - empty owner",
			url:       "/repo",
			want:      nil,
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitHubURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGitHubURL() error = nil, want error containing %q", tt.errSubstr)
					return
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseGitHubURL() error = %v, want error containing %q", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseGitHubURL() unexpected error = %v", err)
				return
			}

			if got.Provider != tt.want.Provider {
				t.Errorf("ParseGitHubURL() Provider = %v, want %v", got.Provider, tt.want.Provider)
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("ParseGitHubURL() Owner = %v, want %v", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("ParseGitHubURL() Repo = %v, want %v", got.Repo, tt.want.Repo)
			}
			if got.Path != tt.want.Path {
				t.Errorf("ParseGitHubURL() Path = %v, want %v", got.Path, tt.want.Path)
			}
			if got.Ref != tt.want.Ref {
				t.Errorf("ParseGitHubURL() Ref = %v, want %v", got.Ref, tt.want.Ref)
			}
		})
	}
}

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "relative path with ./",
			path: "./templates",
			want: true,
		},
		{
			name: "relative path with ../",
			path: "../templates",
			want: true,
		},
		{
			name: "github.com URL",
			path: "github.com/owner/repo",
			want: false,
		},
		{
			name: "owner/repo format",
			path: "owner/repo",
			want: false,
		},
		{
			name: "git@ URL",
			path: "git@github.com:owner/repo.git",
			want: false,
		},
		{
			name: "https URL",
			path: "https://github.com/owner/repo",
			want: false,
		},
		{
			name: "absolute path",
			path: "/home/user/templates",
			want: false, // Absolute paths not allowed for portability
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "single component",
			path: "templates",
			want: false, // Ambiguous, treated as non-local
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLocalPath(tt.path)
			if got != tt.want {
				t.Errorf("IsLocalPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateLocalPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid relative path",
			path:    "./templates",
			wantErr: false,
		},
		{
			name:    "valid nested relative path",
			path:    "./my-templates/go-basic",
			wantErr: false,
		},
		{
			name:      "invalid - contains ..",
			path:      "./templates/../../../etc",
			wantErr:   true,
			errSubstr: "..",
		},
		{
			name:      "invalid - starts with ..",
			path:      "../templates",
			wantErr:   true,
			errSubstr: "..",
		},
		{
			name:      "invalid - absolute path",
			path:      "/home/user/templates",
			wantErr:   true,
			errSubstr: "absolute",
		},
		{
			name:      "invalid - empty path",
			path:      "",
			wantErr:   true,
			errSubstr: "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLocalPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateLocalPath() error = nil, want error containing %q", tt.errSubstr)
					return
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ValidateLocalPath() error = %v, want error containing %q", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateLocalPath() unexpected error = %v", err)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantProvider string
		wantErr      bool
	}{
		{
			name:         "GitHub URL",
			url:          "github.com/owner/repo",
			wantProvider: "github",
			wantErr:      false,
		},
		{
			name:         "local path",
			url:          "./templates",
			wantProvider: "local",
			wantErr:      false,
		},
		{
			name:         "owner/repo format",
			url:          "owner/repo",
			wantProvider: "github",
			wantErr:      false,
		},
		{
			name:         "empty URL",
			url:          "",
			wantProvider: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProvider(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewProvider() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("NewProvider() unexpected error = %v", err)
				return
			}

			if got.Name() != tt.wantProvider {
				t.Errorf("NewProvider().Name() = %v, want %v", got.Name(), tt.wantProvider)
			}
		})
	}
}

func TestProviderError(t *testing.T) {
	tests := []struct {
		name      string
		err       *ProviderError
		wantSubst []string
	}{
		{
			name: "fetch error with cause",
			err:  NewFetchError("github", "owner/repo", &testError{"network timeout"}),
			wantSubst: []string{
				"github",
				"FetchFailed",
				"owner/repo",
				"network timeout",
			},
		},
		{
			name: "not found error",
			err:  NewNotFoundError("github", "owner/repo"),
			wantSubst: []string{
				"github",
				"NotFound",
				"owner/repo",
			},
		},
		{
			name: "auth error",
			err:  NewAuthError("github", "private/repo"),
			wantSubst: []string{
				"github",
				"AuthFailed",
				"private/repo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.wantSubst {
				if !contains(errStr, substr) {
					t.Errorf("ProviderError.Error() = %v, want substring %q", errStr, substr)
				}
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && containsAt(s, substr, 0)
}

func containsAt(s, substr string, start int) bool {
	if start < 0 || start >= len(s) {
		return false
	}
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// testError is a simple error implementation for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
