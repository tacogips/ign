package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

// writeIgnTemplateConfig creates a minimal ign-template.json in the given directory.
func writeIgnTemplateConfig(t *testing.T, dir string) {
	t.Helper()
	configPath := filepath.Join(dir, model.IgnTemplateConfigFile)
	content := []byte(`{"name":"test","version":"1.0.0"}`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", model.IgnTemplateConfigFile, err)
	}
}

// TestCollectFiles_SymlinkToFile verifies that a symlink pointing to a regular file
// is collected as a symlink entry with SymlinkTarget set (not as a regular file
// with the target's content).
func TestCollectFiles_SymlinkToFile(t *testing.T) {
	templateDir := t.TempDir()
	writeIgnTemplateConfig(t, templateDir)

	// Create a regular file as the symlink target.
	targetContent := "hello from target file"
	targetPath := filepath.Join(templateDir, "target.txt")
	if err := os.WriteFile(targetPath, []byte(targetContent), 0644); err != nil {
		t.Fatalf("failed to write target file: %v", err)
	}

	// Create a symlink pointing to the target file (absolute path).
	linkPath := filepath.Join(templateDir, "link.txt")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Fetch via LocalProvider.
	provider := NewLocalProviderWithBase(templateDir)
	ref := model.TemplateRef{
		Provider: "local",
		Path:     templateDir,
	}
	tmpl, err := provider.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// We expect two entries: target.txt (regular file) and link.txt (symlink).
	foundTarget := false
	foundLink := false
	for _, f := range tmpl.Files {
		switch f.Path {
		case "target.txt":
			foundTarget = true
			if f.SymlinkTarget != "" {
				t.Errorf("target.txt should be a regular file, got SymlinkTarget=%q", f.SymlinkTarget)
			}
			if string(f.Content) != targetContent {
				t.Errorf("target.txt content = %q, want %q", string(f.Content), targetContent)
			}
		case "link.txt":
			foundLink = true
			// The symlink should be preserved with the target path.
			if f.SymlinkTarget == "" {
				t.Error("link.txt should be a symlink entry with SymlinkTarget set")
			} else if f.SymlinkTarget != targetPath {
				t.Errorf("link.txt SymlinkTarget = %q, want %q", f.SymlinkTarget, targetPath)
			}
		}
	}

	if !foundTarget {
		t.Error("target.txt not found in collected files")
	}
	if !foundLink {
		t.Error("link.txt (symlink) not found in collected files")
	}
}

// TestCollectFiles_SymlinkToDirectory verifies that a symlink pointing to a
// subdirectory is collected as a symlink entry. The real directory's files are
// also collected via the real directory path.
func TestCollectFiles_SymlinkToDirectory(t *testing.T) {
	templateDir := t.TempDir()
	writeIgnTemplateConfig(t, templateDir)

	// Create a subdirectory with a file.
	subDir := filepath.Join(templateDir, "realdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	fileContent := "file inside real directory"
	if err := os.WriteFile(filepath.Join(subDir, "inner.txt"), []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to write inner.txt: %v", err)
	}

	// Create a symlink pointing to the subdirectory.
	linkDir := filepath.Join(templateDir, "linkdir")
	if err := os.Symlink(subDir, linkDir); err != nil {
		t.Fatalf("failed to create directory symlink: %v", err)
	}

	// Fetch should succeed: the directory symlink is collected as a symlink entry,
	// and the real directory's files are also collected.
	provider := NewLocalProviderWithBase(templateDir)
	ref := model.TemplateRef{
		Provider: "local",
		Path:     templateDir,
	}
	tmpl, err := provider.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}

	// The file inside the real directory should be collected.
	foundInner := false
	foundLinkDir := false
	for _, f := range tmpl.Files {
		if f.Path == filepath.Join("realdir", "inner.txt") {
			foundInner = true
			if string(f.Content) != fileContent {
				t.Errorf("realdir/inner.txt content = %q, want %q", string(f.Content), fileContent)
			}
		}
		if f.Path == "linkdir" {
			foundLinkDir = true
			if f.SymlinkTarget == "" {
				t.Error("linkdir should be a symlink entry with SymlinkTarget set")
			} else if f.SymlinkTarget != subDir {
				t.Errorf("linkdir SymlinkTarget = %q, want %q", f.SymlinkTarget, subDir)
			}
		}
	}

	if !foundInner {
		t.Error("realdir/inner.txt not found in collected files")
	}
	if !foundLinkDir {
		t.Error("linkdir (directory symlink) not found in collected files")
	}
}

// TestCollectFiles_DanglingSymlink verifies that a dangling symlink (pointing to a
// non-existent target) is collected as a symlink entry. The target path is preserved
// even though the target does not exist. This allows the generator to recreate the
// symlink in the output (the target might be created during generation).
func TestCollectFiles_DanglingSymlink(t *testing.T) {
	templateDir := t.TempDir()
	writeIgnTemplateConfig(t, templateDir)

	// Create a regular file so we have something to collect.
	regularContent := "regular file content"
	if err := os.WriteFile(filepath.Join(templateDir, "regular.txt"), []byte(regularContent), 0644); err != nil {
		t.Fatalf("failed to write regular.txt: %v", err)
	}

	// Create a symlink pointing to a non-existent target.
	danglingTarget := filepath.Join(templateDir, "nonexistent-target")
	linkPath := filepath.Join(templateDir, "broken-link.txt")
	if err := os.Symlink(danglingTarget, linkPath); err != nil {
		t.Fatalf("failed to create dangling symlink: %v", err)
	}

	// Fetch should succeed: the dangling symlink is collected as a symlink entry.
	provider := NewLocalProviderWithBase(templateDir)
	ref := model.TemplateRef{
		Provider: "local",
		Path:     templateDir,
	}
	tmpl, err := provider.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}

	// Both the regular file and the dangling symlink should be collected.
	foundRegular := false
	foundBrokenLink := false
	for _, f := range tmpl.Files {
		if f.Path == "regular.txt" {
			foundRegular = true
			if string(f.Content) != regularContent {
				t.Errorf("regular.txt content = %q, want %q", string(f.Content), regularContent)
			}
		}
		if f.Path == "broken-link.txt" {
			foundBrokenLink = true
			if f.SymlinkTarget != danglingTarget {
				t.Errorf("broken-link.txt SymlinkTarget = %q, want %q", f.SymlinkTarget, danglingTarget)
			}
		}
	}

	if !foundRegular {
		t.Error("regular.txt not found in collected files")
	}
	if !foundBrokenLink {
		t.Error("broken-link.txt (dangling symlink) not found in collected files")
	}
}

// TestCollectFiles_SymlinkOutsideTemplate verifies that a symlink pointing to a file
// outside the template directory is collected as a symlink entry with the target path
// preserved.
func TestCollectFiles_SymlinkOutsideTemplate(t *testing.T) {
	// Create a file outside the template directory.
	externalDir := t.TempDir()
	externalContent := "content from outside template"
	externalFile := filepath.Join(externalDir, "external.txt")
	if err := os.WriteFile(externalFile, []byte(externalContent), 0644); err != nil {
		t.Fatalf("failed to write external file: %v", err)
	}

	// Create the template directory.
	templateDir := t.TempDir()
	writeIgnTemplateConfig(t, templateDir)

	// Create a symlink inside the template directory pointing outside.
	linkPath := filepath.Join(templateDir, "external-link.txt")
	if err := os.Symlink(externalFile, linkPath); err != nil {
		t.Fatalf("failed to create symlink to external file: %v", err)
	}

	// Fetch via LocalProvider.
	provider := NewLocalProviderWithBase(templateDir)
	ref := model.TemplateRef{
		Provider: "local",
		Path:     templateDir,
	}
	tmpl, err := provider.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// The symlink should be collected with its target path preserved.
	found := false
	for _, f := range tmpl.Files {
		if f.Path == "external-link.txt" {
			found = true
			if f.SymlinkTarget != externalFile {
				t.Errorf("external-link.txt SymlinkTarget = %q, want %q", f.SymlinkTarget, externalFile)
			}
		}
	}

	if !found {
		t.Error("external-link.txt (symlink to outside) not found in collected files")
	}
}

// TestCollectFiles_RelativeSymlink verifies that a relative symlink within the template
// directory is collected as a symlink entry with the relative target path preserved.
func TestCollectFiles_RelativeSymlink(t *testing.T) {
	templateDir := t.TempDir()
	writeIgnTemplateConfig(t, templateDir)

	// Create a subdirectory with a target file.
	subDir := filepath.Join(templateDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	targetContent := "target via relative symlink"
	if err := os.WriteFile(filepath.Join(subDir, "target.txt"), []byte(targetContent), 0644); err != nil {
		t.Fatalf("failed to write target.txt: %v", err)
	}

	// Create a relative symlink in the template root pointing to subdir/target.txt.
	// The relative path is relative to the directory containing the symlink.
	relTarget := filepath.Join("subdir", "target.txt")
	linkPath := filepath.Join(templateDir, "relative-link.txt")
	if err := os.Symlink(relTarget, linkPath); err != nil {
		t.Fatalf("failed to create relative symlink: %v", err)
	}

	// Fetch via LocalProvider.
	provider := NewLocalProviderWithBase(templateDir)
	ref := model.TemplateRef{
		Provider: "local",
		Path:     templateDir,
	}
	tmpl, err := provider.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// We expect subdir/target.txt (regular file) and relative-link.txt (symlink).
	foundTarget := false
	foundLink := false
	for _, f := range tmpl.Files {
		switch f.Path {
		case filepath.Join("subdir", "target.txt"):
			foundTarget = true
			if f.SymlinkTarget != "" {
				t.Errorf("subdir/target.txt should be a regular file, got SymlinkTarget=%q", f.SymlinkTarget)
			}
			if string(f.Content) != targetContent {
				t.Errorf("subdir/target.txt content = %q, want %q", string(f.Content), targetContent)
			}
		case "relative-link.txt":
			foundLink = true
			if f.SymlinkTarget != relTarget {
				t.Errorf("relative-link.txt SymlinkTarget = %q, want %q", f.SymlinkTarget, relTarget)
			}
		}
	}

	if !foundTarget {
		t.Error("subdir/target.txt not found in collected files")
	}
	if !foundLink {
		t.Error("relative-link.txt (relative symlink) not found in collected files")
	}
}
