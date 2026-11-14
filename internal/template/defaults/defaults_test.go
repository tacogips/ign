package defaults

import (
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestCurrentDirName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "relative current directory",
			input: ".",
			want:  filepath.Base(mustAbs(t, ".")),
		},
		{
			name:  "relative nested directory",
			input: "tmp/project-name",
			want:  "project-name",
		},
		{
			name:  "absolute directory",
			input: "/tmp/sample-app",
			want:  "sample-app",
		},
		{
			name:  "empty directory",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CurrentDirName(tt.input); got != tt.want {
				t.Fatalf("CurrentDirName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveValue(t *testing.T) {
	currentDir := filepath.Join(t.TempDir(), "my-app")

	tests := []struct {
		name  string
		value interface{}
		want  interface{}
	}{
		{
			name:  "string placeholder only",
			value: "{current_dir}",
			want:  "my-app",
		},
		{
			name:  "string placeholder embedded",
			value: "some/path/{current_dir}",
			want:  "some/path/my-app",
		},
		{
			name:  "string without placeholder",
			value: "plain",
			want:  "plain",
		},
		{
			name:  "non-string value",
			value: 8080,
			want:  8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveValue(tt.value, currentDir); got != tt.want {
				t.Fatalf("ResolveValue(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolveIgnJSON(t *testing.T) {
	ignJSON := &model.IgnJson{
		Variables: map[string]model.VarDef{
			"project_name": {
				Type:    model.VarTypeString,
				Default: "{current_dir}",
			},
			"module_path": {
				Type:    model.VarTypeString,
				Default: "github.com/acme/{current_dir}",
			},
			"port": {
				Type:    model.VarTypeInt,
				Default: 8080,
			},
		},
	}

	resolved := ResolveIgnJSON(ignJSON, filepath.Join(t.TempDir(), "sample-service"))

	if got := resolved.Variables["project_name"].Default; got != "sample-service" {
		t.Fatalf("project_name default = %v, want %q", got, "sample-service")
	}
	if got := resolved.Variables["module_path"].Default; got != "github.com/acme/sample-service" {
		t.Fatalf("module_path default = %v, want %q", got, "github.com/acme/sample-service")
	}
	if got := resolved.Variables["port"].Default; got != 8080 {
		t.Fatalf("port default = %v, want 8080", got)
	}
	if got := ignJSON.Variables["project_name"].Default; got != "{current_dir}" {
		t.Fatalf("original project_name default mutated: got %v", got)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", path, err)
	}
	return abs
}
