//go:build darwin || linux

package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestRecoverSymlinkTransitionJournalRestoresArtifactPreimageModes(t *testing.T) {
	root := t.TempDir()
	ignDir := filepath.Join(root, model.IgnConfigDir)
	if err := os.Mkdir(ignDir, 0700); err != nil {
		t.Fatal(err)
	}
	artifacts := []struct {
		name string
		data string
		mode os.FileMode
	}{
		{name: model.IgnManifestFile, data: "manifest-before", mode: 0644},
		{name: "ign.json", data: "config-before", mode: 0600},
		{name: "ign-var.json", data: "variables-before", mode: 0640},
	}
	for _, artifact := range artifacts {
		if err := os.WriteFile(filepath.Join(ignDir, artifact.name), []byte(artifact.data), artifact.mode); err != nil {
			t.Fatal(err)
		}
	}

	backup := ".ign-symlink-transition-000000000000000000000000"
	if err := os.Mkdir(filepath.Join(root, backup), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, backup, "retired.txt"), []byte("prior tree"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".agents", filepath.Join(root, ".claude")); err != nil {
		t.Fatal(err)
	}

	journal, err := openSymlinkTransitionJournal(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.captureArtifactPreimages(); err != nil {
		journal.close()
		t.Fatal(err)
	}
	journal.entries = []symlinkTransitionJournalEntry{{
		Path: ".claude", Backup: backup, Target: ".agents", Phase: symlinkTransitionJournalReplaced,
	}}
	if err := journal.persist(); err != nil {
		journal.close()
		t.Fatal(err)
	}
	journal.close()

	for _, artifact := range artifacts {
		if err := os.WriteFile(filepath.Join(ignDir, artifact.name), []byte("changed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := recoverSymlinkTransitionJournal(root, filepath.Join(ignDir, model.IgnManifestFile)); err != nil {
		t.Fatal(err)
	}

	for _, artifact := range artifacts {
		path := filepath.Join(ignDir, artifact.name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != artifact.data {
			t.Fatalf("%s content = %q, want %q", artifact.name, data, artifact.data)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != artifact.mode.Perm() {
			t.Fatalf("%s mode = %04o, want %04o", artifact.name, info.Mode().Perm(), artifact.mode.Perm())
		}
	}
	if info, err := os.Lstat(filepath.Join(root, ".claude")); err != nil || !info.IsDir() {
		t.Fatalf("restored directory = %v, %v; want directory", info, err)
	}
}

func TestRecoverSymlinkTransitionJournalUsesExplicitTrackingArtifactDirectory(t *testing.T) {
	root := t.TempDir()
	trackingDir := filepath.Join(root, "tracking")
	if err := os.Mkdir(trackingDir, 0700); err != nil {
		t.Fatal(err)
	}
	artifacts := []struct {
		name string
		data string
	}{
		{name: model.IgnManifestFile, data: "manifest-before"},
		{name: "ign.json", data: "config-before"},
		{name: "ign-var.json", data: "variables-before"},
	}
	for _, artifact := range artifacts {
		if err := os.WriteFile(filepath.Join(trackingDir, artifact.name), []byte(artifact.data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	manifestPath := filepath.Join(trackingDir, model.IgnManifestFile)
	journal, err := openSymlinkTransitionJournal(root, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.captureArtifactPreimages(); err != nil {
		journal.close()
		t.Fatal(err)
	}
	if err := journal.restoreArtifactPreimages(); err != nil {
		journal.close()
		t.Fatal(err)
	}
	journal.close()
	for _, artifact := range artifacts {
		if err := os.WriteFile(filepath.Join(trackingDir, artifact.name), []byte("changed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	journal, err = loadSymlinkTransitionJournal(root, manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.restoreArtifactPreimages(); err != nil {
		journal.close()
		t.Fatal(err)
	}
	journal.close()
	for _, artifact := range artifacts {
		data, err := os.ReadFile(filepath.Join(trackingDir, artifact.name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != artifact.data {
			t.Fatalf("%s content = %q, want %q", artifact.name, data, artifact.data)
		}
	}
	if _, err := os.Stat(filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)); !os.IsNotExist(err) {
		t.Fatalf("default tracking artifact = %v; want no artifact outside explicit directory", err)
	}
}
