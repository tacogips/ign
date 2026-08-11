//go:build darwin || linux

package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestRestoreCheckoutRollbackEntryRetainsSubstitutionBeforeArchive(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "managed-file")
	entry := checkoutRollbackSnapshotForGeneratedFile(t, destination, []byte("old"), []byte("generated"))

	originalHook := beforeCheckoutRollbackArchive
	t.Cleanup(func() { beforeCheckoutRollbackArchive = originalHook })
	beforeCheckoutRollbackArchive = func() {
		if err := os.Remove(destination); err != nil {
			t.Fatalf("remove generated file: %v", err)
		}
		if err := os.WriteFile(destination, []byte("concurrent"), 0600); err != nil {
			t.Fatalf("create concurrent file: %v", err)
		}
	}

	if err := restoreCheckoutRollbackEntry(entry); err == nil {
		t.Fatal("rollback restored over a substituted file")
	}
	assertFileContentUnchanged(t, destination, []byte("concurrent"))
}

func TestRestoreCheckoutRollbackEntryRetainsRecreationBeforeRestore(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "managed-file")
	entry := checkoutRollbackSnapshotForGeneratedFile(t, destination, []byte("old"), []byte("generated"))

	originalHook := beforeCheckoutRollbackRestore
	t.Cleanup(func() { beforeCheckoutRollbackRestore = originalHook })
	beforeCheckoutRollbackRestore = func() {
		if err := os.WriteFile(destination, []byte("concurrent"), 0600); err != nil {
			t.Fatalf("recreate destination: %v", err)
		}
	}

	if err := restoreCheckoutRollbackEntry(entry); err == nil {
		t.Fatal("rollback restored over a recreated file")
	}
	assertFileContentUnchanged(t, destination, []byte("concurrent"))
}

func TestRestoreCheckoutRollbackEntryRetainsInPlaceModification(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "managed-file")
	entry := checkoutRollbackSnapshotForGeneratedFile(t, destination, []byte("old"), []byte("generated"))

	originalHook := beforeCheckoutRollbackArchive
	t.Cleanup(func() { beforeCheckoutRollbackArchive = originalHook })
	beforeCheckoutRollbackArchive = func() {
		if err := os.WriteFile(destination, []byte("changed-in-place"), 0600); err != nil {
			t.Fatalf("modify generated file: %v", err)
		}
	}

	if err := restoreCheckoutRollbackEntry(entry); err == nil {
		t.Fatal("rollback restored over an in-place modification")
	}
	assertFileContentUnchanged(t, destination, []byte("changed-in-place"))
}

func TestCompleteCheckoutManifestRollbackRetainsPrivateArchive(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	manifestPath := filepath.Join(model.IgnConfigDir, model.IgnManifestFile)
	if err := os.MkdirAll(manifestPath, 0755); err != nil {
		t.Fatalf("create manifest failure fixture: %v", err)
	}
	if _, err := CompleteCheckout(context.Background(), CompleteCheckoutOptions{
		PrepareResult: singleFilePreparedCheckout(),
		Variables:     map[string]interface{}{},
		OutputDir:     "output",
	}); err == nil {
		t.Fatal("CompleteCheckout succeeded despite manifest failure fixture")
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read checkout parent: %v", err)
	}
	archiveFound := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ign-checkout-rollback-") {
			archiveFound = true
		}
	}
	if !archiveFound {
		t.Fatal("successful rollback removed its private archive")
	}
}

func TestCheckoutRollbackRetainsArchiveChangedAfterValidation(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "output")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatalf("create output root: %v", err)
	}
	destination := filepath.Join(root, "managed-file")
	entry := checkoutRollbackSnapshotForGeneratedFile(t, destination, []byte("old"), []byte("generated"))
	rollback := &checkoutGenerationRollback{outputDir: root}

	if err := rollback.restoreCheckoutRollbackEntry(entry); err != nil {
		t.Fatalf("restore rollback entry: %v", err)
	}
	archiveNodes, err := filepath.Glob(filepath.Join(rollback.backupDir, "archive-*", "node"))
	if err != nil || len(archiveNodes) != 1 {
		t.Fatalf("find private archive node: nodes=%v err=%v", archiveNodes, err)
	}
	if err := os.WriteFile(archiveNodes[0], []byte("changed-after-validation"), 0600); err != nil {
		t.Fatalf("modify private archive: %v", err)
	}

	rollback.cleanup()
	assertFileContentUnchanged(t, archiveNodes[0], []byte("changed-after-validation"))
}

func checkoutRollbackSnapshotForGeneratedFile(t *testing.T, destination string, old, generated []byte) checkoutRollbackEntry {
	t.Helper()
	if err := os.WriteFile(destination, old, 0644); err != nil {
		t.Fatal(err)
	}
	// Anchor the private archive under the test-owned tree. A zero outputDir
	// resolves the archive parent to the process working directory, which
	// leaks retained backups into the package source directory.
	rollback := &checkoutGenerationRollback{outputDir: filepath.Dir(destination)}
	entry, err := rollback.snapshotPath(destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, generated, 0644); err != nil {
		t.Fatal(err)
	}
	info, fingerprint, err := captureCheckoutRollbackNode(destination)
	if err != nil {
		t.Fatal(err)
	}
	entry.expectedInfo = info
	entry.expectedFingerprint = fingerprint
	return entry
}
