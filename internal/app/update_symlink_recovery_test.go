//go:build darwin || linux

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestRecoverSymlinkTransitionJournalRefusesSymlinkedAncestor(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.MkdirAll(filepath.Join(external, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(external, ".claude", "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}

	journal, err := openSymlinkTransitionJournal(root)
	if err != nil {
		t.Fatal(err)
	}
	journal.entries = []symlinkTransitionJournalEntry{{
		Path: "linked/.claude", Backup: ".ign-symlink-transition-000000000000000000000000", Target: ".agents", Phase: symlinkTransitionJournalReplaced,
	}}
	journal.artifacts = []symlinkTransitionJournalArtifact{
		{Name: "ign-files.json"},
		{Name: "ign.json"},
		{Name: "ign-var.json"},
	}
	if err := journal.persist(); err != nil {
		journal.close()
		t.Fatal(err)
	}
	journalPath := filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)
	journalBefore, err := os.ReadFile(journalPath)
	if err != nil {
		journal.close()
		t.Fatal(err)
	}
	journal.close()

	err = recoverSymlinkTransitionJournal(root, filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile))
	if err == nil {
		t.Fatal("recovered transition through symlinked ancestor")
	}
	if strings.Contains(err.Error(), "invalid symlink transition recovery journal") {
		t.Fatalf("recovery rejected valid journal: %v", err)
	}
	if !strings.Contains(err.Error(), "open interrupted transition parent: open transition ancestor without following symlinks") {
		t.Fatalf("recovery error = %v; want no-follow symlinked-ancestor error", err)
	}
	assertFileContentUnchanged(t, sentinel, []byte("keep"))
	assertFileContentUnchanged(t, journalPath, journalBefore)
}
