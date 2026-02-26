package provider

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestGitHubProvider_ExtractArchive_PreservesSymlinkForCollectFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "template.tar.gz")
	if err := createTestTemplateArchiveWithSymlink(archivePath); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	p := NewGitHubProvider()
	extractDir, err := p.extractArchive(archivePath)
	if err != nil {
		t.Fatalf("extractArchive() error = %v", err)
	}
	defer func() { _ = os.RemoveAll(extractDir) }()

	// Ensure the symlink entry itself was extracted.
	linkPath := filepath.Join(extractDir, "CLAUDE.md")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat(%s) error = %v", linkPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is not a symlink", linkPath)
	}

	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink(%s) error = %v", linkPath, err)
	}
	if target != "AGENTS.md" {
		t.Fatalf("symlink target = %q, want %q", target, "AGENTS.md")
	}

	// Verify downstream collection sees both files (symlink + target content).
	files, err := p.collectFiles(extractDir)
	if err != nil {
		t.Fatalf("collectFiles() error = %v", err)
	}

	got := map[string]string{}
	for _, f := range files {
		got[f.Path] = string(f.Content)
	}

	if got["AGENTS.md"] != "agent instructions\n" {
		t.Fatalf("AGENTS.md content = %q, want %q", got["AGENTS.md"], "agent instructions\n")
	}
	if got["CLAUDE.md"] != "agent instructions\n" {
		t.Fatalf("CLAUDE.md content = %q, want %q", got["CLAUDE.md"], "agent instructions\n")
	}
}

func createTestTemplateArchiveWithSymlink(archivePath string) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gzw := gzip.NewWriter(f)
	defer func() { _ = gzw.Close() }()

	tw := tar.NewWriter(gzw)
	defer func() { _ = tw.Close() }()

	entries := []struct {
		header  *tar.Header
		content []byte
	}{
		{
			header: &tar.Header{
				Name:     "repo-main/",
				Typeflag: tar.TypeDir,
				Mode:     0755,
			},
		},
		{
			header: &tar.Header{
				Name:     "repo-main/ign-template.json",
				Typeflag: tar.TypeReg,
				Mode:     0644,
				Size:     int64(len([]byte(`{"name":"test-template","version":"1.0.0"}`))),
			},
			content: []byte(`{"name":"test-template","version":"1.0.0"}`),
		},
		{
			header: &tar.Header{
				Name:     "repo-main/AGENTS.md",
				Typeflag: tar.TypeReg,
				Mode:     0644,
				Size:     int64(len([]byte("agent instructions\n"))),
			},
			content: []byte("agent instructions\n"),
		},
		{
			header: &tar.Header{
				Name:     "repo-main/CLAUDE.md",
				Typeflag: tar.TypeSymlink,
				Mode:     0777,
				Linkname: "AGENTS.md",
			},
		},
	}

	for _, e := range entries {
		if err := tw.WriteHeader(e.header); err != nil {
			return err
		}
		if len(e.content) > 0 {
			if _, err := tw.Write(e.content); err != nil {
				return err
			}
		}
	}

	return nil
}
