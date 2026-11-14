package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

// TestIsSpecialFile tests special file detection.
func TestIsSpecialFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"ign-template.json root", model.IgnTemplateConfigFile, true},
		{"ign-template.json in subdir", "subdir/" + model.IgnTemplateConfigFile, true},
		{".ign exact", ".ign", true},
		{".ign with slash", ".ign/", true},
		{".ign subpath", ".ign/ign-var.json", true},
		{"regular file", "main.go", false},
		{"regular subdir file", "src/main.go", false},
		{"similar name", "ign-template.json.bak", false},
		{"similar prefix", "ign-build-backup", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSpecialFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsSpecialFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestMatchesPattern tests glob pattern matching.
func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"exact match", "test.txt", "test.txt", true},
		{"wildcard extension", "file.txt", "*.txt", true},
		{"wildcard all", "anything", "*", true},
		{"subdir wildcard", "dir/file.txt", "*.txt", true},
		{"pattern with slash", "dir/file.txt", "dir/*.txt", true},
		{"no match", "file.go", "*.txt", false},
		{"case sensitive", "FILE.txt", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("MatchesPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestShouldIgnoreFile tests file filtering logic.
func TestShouldIgnoreFile(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		ignorePatterns []string
		expected       bool
	}{
		{"special file ign-template.json", model.IgnTemplateConfigFile, []string{}, true},
		{"special file .ign", ".ign/ign-var.json", []string{}, true},
		{"ignored by pattern", "test.log", []string{"*.log"}, true},
		{"not ignored", "main.go", []string{"*.txt"}, false},
		{"multiple patterns match", "temp.tmp", []string{"*.log", "*.tmp"}, true},
		{"multiple patterns no match", "main.go", []string{"*.log", "*.tmp"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldIgnoreFile(tt.path, tt.ignorePatterns)
			if result != tt.expected {
				t.Errorf("ShouldIgnoreFile(%q, %v) = %v, want %v", tt.path, tt.ignorePatterns, result, tt.expected)
			}
		})
	}
}

// TestFileProcessor_ShouldProcess tests binary file detection.
func TestFileProcessor_ShouldProcess(t *testing.T) {
	p := NewFileProcessor(parser.NewParser(), []string{".bin", ".exe"})

	tests := []struct {
		name     string
		file     model.TemplateFile
		expected bool
	}{
		{
			name: "text file",
			file: model.TemplateFile{
				Path:     "main.go",
				Content:  []byte("package main"),
				IsBinary: false,
			},
			expected: true,
		},
		{
			name: "binary marked",
			file: model.TemplateFile{
				Path:     "data",
				Content:  []byte("data"),
				IsBinary: true,
			},
			expected: false,
		},
		{
			name: "binary extension",
			file: model.TemplateFile{
				Path:     "prog.exe",
				Content:  []byte("text"),
				IsBinary: false,
			},
			expected: false,
		},
		{
			name: "binary content with null",
			file: model.TemplateFile{
				Path:     "file",
				Content:  []byte{0x00, 0x01, 0x02},
				IsBinary: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ShouldProcess(tt.file)
			if result != tt.expected {
				t.Errorf("ShouldProcess(%s) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

// TestFileProcessor_Process tests file processing.
func TestFileProcessor_Process(t *testing.T) {
	p := NewFileProcessor(parser.NewParser(), []string{".bin"})
	ctx := context.Background()
	vars := parser.NewMapVariables(map[string]interface{}{
		"name": "test",
		"port": 8080,
	})

	tests := []struct {
		name     string
		file     model.TemplateFile
		expected string
	}{
		{
			name: "simple variable substitution",
			file: model.TemplateFile{
				Path:     "config.txt",
				Content:  []byte("name: @ign-var:name@"),
				IsBinary: false,
			},
			expected: "name: test",
		},
		{
			name: "binary file unchanged",
			file: model.TemplateFile{
				Path:     "data.bin",
				Content:  []byte{0x00, 0x01, 0x02},
				IsBinary: true,
			},
			expected: string([]byte{0x00, 0x01, 0x02}),
		},
		{
			name: "multiple variables",
			file: model.TemplateFile{
				Path:     "config.txt",
				Content:  []byte("@ign-var:name@ on port @ign-var:port@"),
				IsBinary: false,
			},
			expected: "test on port 8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Process(ctx, tt.file, vars, "/tmp/template")
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Process() = %q, want %q", string(result), tt.expected)
			}
		})
	}
}

// TestFileWriter tests file writing operations.
func TestFileWriter(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewFileWriter(false)

	t.Run("WriteFile creates file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")

		err := writer.WriteFile(path, content, 0644)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		// Verify file exists
		if !writer.Exists(path) {
			t.Error("File was not created")
		}

		// Verify content
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("File content = %q, want %q", string(data), string(content))
		}
	})

	t.Run("WriteFile creates parent dirs", func(t *testing.T) {
		path := filepath.Join(tmpDir, "subdir", "nested", "file.txt")
		content := []byte("nested content")

		err := writer.WriteFile(path, content, 0644)
		if err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		if !writer.Exists(path) {
			t.Error("File was not created in nested directory")
		}
	})

	t.Run("CreateDir creates directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "newdir", "nested")

		err := writer.CreateDir(path)
		if err != nil {
			t.Fatalf("CreateDir() error = %v", err)
		}

		if !writer.Exists(path) {
			t.Error("Directory was not created")
		}
	})

	t.Run("Exists returns false for non-existent", func(t *testing.T) {
		path := filepath.Join(tmpDir, "does-not-exist")
		if writer.Exists(path) {
			t.Error("Exists() returned true for non-existent path")
		}
	})
}

// TestGenerator_Generate tests basic generation.
func TestGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	// Create test template
	template := &model.Template{
		Ref: model.TemplateRef{},
		Config: model.IgnJson{
			Name:    "test",
			Version: "1.0.0",
		},
		Files: []model.TemplateFile{
			{
				Path:     "main.go",
				Content:  []byte("package @ign-var:pkg@"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "README.md",
				Content:  []byte("# @ign-var:title@"),
				Mode:     0644,
				IsBinary: false,
			},
		},
		RootPath: tmpDir,
	}

	vars := parser.NewMapVariables(map[string]interface{}{
		"pkg":   "main",
		"title": "Test Project",
	})

	opts := GenerateOptions{
		Template:  template,
		Variables: vars,
		OutputDir: tmpDir,
		Overwrite: false,
		Verbose:   false,
	}

	result, err := gen.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check result statistics
	if result.FilesCreated != 2 {
		t.Errorf("FilesCreated = %d, want 2", result.FilesCreated)
	}
	if result.FilesSkipped != 0 {
		t.Errorf("FilesSkipped = %d, want 0", result.FilesSkipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}

	// Verify file content
	mainGo, err := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}
	if string(mainGo) != "package main" {
		t.Errorf("main.go content = %q, want %q", string(mainGo), "package main")
	}

	readme, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}
	if string(readme) != "# Test Project" {
		t.Errorf("README.md content = %q, want %q", string(readme), "# Test Project")
	}
}

// TestGenerator_GenerateWithOverwrite tests overwrite behavior.
func TestGenerator_GenerateWithOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	// Create existing file
	existingPath := filepath.Join(tmpDir, "existing.txt")
	_ = os.WriteFile(existingPath, []byte("old content"), 0644)

	template := &model.Template{
		Ref: model.TemplateRef{},
		Config: model.IgnJson{
			Name:    "test",
			Version: "1.0.0",
		},
		Files: []model.TemplateFile{
			{
				Path:     "existing.txt",
				Content:  []byte("new content"),
				Mode:     0644,
				IsBinary: false,
			},
		},
		RootPath: tmpDir,
	}

	vars := parser.NewMapVariables(map[string]interface{}{})

	// First try without overwrite
	opts := GenerateOptions{
		Template:  template,
		Variables: vars,
		OutputDir: tmpDir,
		Overwrite: false,
	}

	result, err := gen.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.FilesSkipped != 1 {
		t.Errorf("FilesSkipped = %d, want 1", result.FilesSkipped)
	}

	// Verify content unchanged
	content, _ := os.ReadFile(existingPath)
	if string(content) != "old content" {
		t.Errorf("File was modified without overwrite flag")
	}

	// Now try with overwrite
	opts.Overwrite = true
	result, err = gen.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.FilesOverwritten != 1 {
		t.Errorf("FilesOverwritten = %d, want 1", result.FilesOverwritten)
	}

	// Verify content changed
	content, _ = os.ReadFile(existingPath)
	if string(content) != "new content" {
		t.Errorf("File content = %q, want %q", string(content), "new content")
	}
}

// TestGenerator_DryRun tests dry run mode.
func TestGenerator_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	template := &model.Template{
		Ref: model.TemplateRef{},
		Config: model.IgnJson{
			Name:    "test",
			Version: "1.0.0",
		},
		Files: []model.TemplateFile{
			{
				Path:     "file.txt",
				Content:  []byte("content"),
				Mode:     0644,
				IsBinary: false,
			},
		},
		RootPath: tmpDir,
	}

	vars := parser.NewMapVariables(map[string]interface{}{})

	opts := GenerateOptions{
		Template:  template,
		Variables: vars,
		OutputDir: tmpDir,
		Overwrite: false,
	}

	result, err := gen.DryRun(ctx, opts)
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}

	// Check that it would create the file
	if result.FilesCreated != 1 {
		t.Errorf("DryRun FilesCreated = %d, want 1", result.FilesCreated)
	}

	// Verify file was NOT actually created
	filePath := filepath.Join(tmpDir, "file.txt")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("DryRun created file when it shouldn't")
	}
}

// TestGenerator_FilterSpecialFiles tests that special files are filtered.
func TestGenerator_FilterSpecialFiles(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator()
	ctx := context.Background()

	template := &model.Template{
		Ref: model.TemplateRef{},
		Config: model.IgnJson{
			Name:    "test",
			Version: "1.0.0",
		},
		Files: []model.TemplateFile{
			{
				Path:     model.IgnTemplateConfigFile,
				Content:  []byte("{}"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     ".ign/ign-var.json",
				Content:  []byte("{}"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "main.go",
				Content:  []byte("package main"),
				Mode:     0644,
				IsBinary: false,
			},
		},
		RootPath: tmpDir,
	}

	vars := parser.NewMapVariables(map[string]interface{}{})

	opts := GenerateOptions{
		Template:  template,
		Variables: vars,
		OutputDir: tmpDir,
		Overwrite: false,
	}

	result, err := gen.Generate(ctx, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should only create main.go (not ign-template.json or .ign/ign-var.json)
	if result.FilesCreated != 1 {
		t.Errorf("FilesCreated = %d, want 1 (only main.go)", result.FilesCreated)
	}

	// Verify special files were not created
	if _, err := os.Stat(filepath.Join(tmpDir, model.IgnTemplateConfigFile)); !os.IsNotExist(err) {
		t.Errorf("%s was created when it should be filtered", model.IgnTemplateConfigFile)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, ".ign/ign-var.json")); !os.IsNotExist(err) {
		t.Error(".ign/ign-var.json was created when it should be filtered")
	}
}
