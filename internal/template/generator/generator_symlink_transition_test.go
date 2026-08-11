package generator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/parser"
)

func TestGeneratorSymlinkTransitionEligibleReplacesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".claude")
	if err := os.Symlink(".agents", path); err != nil {
		t.Fatal(err)
	}

	result, err := NewGenerator().Generate(context.Background(), symlinkTransitionGenerateOptions(tempDir, map[string]SymlinkTransition{
		path: {Disposition: SymlinkTransitionEligible},
	}))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(result.Errors) != 0 || len(result.WrittenFiles) != 1 || result.FilesOverwritten != 1 {
		t.Fatalf("generation errors = %v", result.Errors)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("transition result = %v, %v; want symlink", info, err)
	}
}

func TestGeneratorSymlinkTransitionPreservedSkipsWrite(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "user.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := NewGenerator().Generate(context.Background(), symlinkTransitionGenerateOptions(tempDir, map[string]SymlinkTransition{
		path: {Disposition: SymlinkTransitionPreserved},
	}))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(result.WrittenFiles) != 0 || result.FilesSkipped != 1 {
		t.Fatalf("result = %#v, want one skipped and no writes", result)
	}
	if info, err := os.Lstat(path); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("transition result = %v, %v; want preserved directory", info, err)
	}
}

func TestGeneratorSymlinkTransitionFailureIsFatal(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := NewGenerator().Generate(context.Background(), symlinkTransitionGenerateOptions(tempDir, map[string]SymlinkTransition{
		path: {Disposition: SymlinkTransitionEligible},
	}))
	if err == nil {
		t.Fatal("generate succeeded despite eligible transition failure")
	}
	if len(result.Errors) != 0 {
		t.Fatalf("fatal transition error was downgraded into result errors: %v", result.Errors)
	}
	if info, statErr := os.Lstat(path); statErr != nil || !info.IsDir() {
		t.Fatalf("failed transition changed source directory: %v, %v", info, statErr)
	}
}

func TestGeneratorOrdinarySymlinkReplacementRemainsNonFatal(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}

	baseWriter := NewFileWriter(false)
	gen := &DefaultGenerator{parser: parser.NewParser(), writer: failingOrdinarySymlinkWriter{Writer: baseWriter}}
	result, err := gen.Generate(context.Background(), symlinkTransitionGenerateOptions(tempDir, nil))
	if err != nil {
		t.Fatalf("ordinary symlink replacement returned fatal error: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("ordinary replacement errors = %v, want one non-fatal error", result.Errors)
	}
}

func symlinkTransitionGenerateOptions(outputDir string, transitions map[string]SymlinkTransition) GenerateOptions {
	return GenerateOptions{
		Template: &model.Template{Files: []model.TemplateFile{{
			Path: ".claude", Mode: 0777 | os.ModeSymlink, SymlinkTarget: ".agents",
		}}},
		Variables:          parser.NewMapVariables(map[string]interface{}{}),
		OutputDir:          outputDir,
		Overwrite:          true,
		OverwriteMode:      OverwriteAll,
		SymlinkTransitions: transitions,
	}
}

type failingOrdinarySymlinkWriter struct {
	Writer
}

func (failingOrdinarySymlinkWriter) WriteSymlink(string, string) error {
	return errors.New("injected ordinary symlink failure")
}
