//go:build darwin || linux

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

func TestCompleteUpdateRejectsLegacyDelimiterFingerprintCollision(t *testing.T) {
	tempDir, prep, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	legacyMode := "-rw-r--r--"
	if err := os.WriteFile(retiredPath, []byte("prefix\x00z-untracked\x00"+legacyMode+"\x00suffix"), 0644); err != nil {
		t.Fatal(err)
	}
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retiredPath, []byte("prefix"), 0644); err != nil {
		t.Fatal(err)
	}
	untracked := filepath.Join(managedDir, "z-untracked")
	if err := os.WriteFile(untracked, []byte("suffix"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err == nil {
		t.Fatal("update succeeded after a delimiter-collision source mutation")
	}
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; mutation ran despite fingerprint divergence", info, err)
	}
	if content, err := os.ReadFile(untracked); err != nil || string(content) != "suffix" {
		t.Fatalf("untracked file = %q, %v; want preserved", content, err)
	}
}

func TestCompleteUpdateRejectsUntrackedAdditionBetweenOwnershipAndFingerprint(t *testing.T) {
	tempDir, prep, managedDir, _, _, _ := setupSymlinkTransitionUpdate(t)
	originalHook := afterOwnedDirectoryTree
	t.Cleanup(func() { afterOwnedDirectoryTree = originalHook })
	added := false
	latePath := filepath.Join(managedDir, "late-user-file")
	afterOwnedDirectoryTree = func() {
		if added {
			return
		}
		added = true
		if err := os.WriteFile(latePath, []byte("preserve me"), 0600); err != nil {
			t.Fatalf("create late user file: %v", err)
		}
	}

	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatalf("preview update: %v", err)
	}
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err == nil {
		t.Fatal("update succeeded after an untracked addition between ownership and fingerprint")
	}
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want preserved directory", info, err)
	}
	assertFileContentUnchanged(t, latePath, []byte("preserve me"))
}

func TestRestoreCheckoutRollbackEntryDoesNotRecursivelyDeleteRecreatedDirectory(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "generated-file")
	if err := os.Mkdir(destination, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(destination, "user-file")
	if err := os.WriteFile(sentinel, []byte("preserve me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := restoreCheckoutRollbackEntry(checkoutRollbackEntry{path: destination}); err == nil {
		t.Fatal("rollback removed a recreated non-empty directory")
	}
	assertFileContentUnchanged(t, sentinel, []byte("preserve me"))
}

func TestMarkArtifactsCommittedPersistenceFailureRollsBack(t *testing.T) {
	root, _, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	originalPersist := persistCommittedSymlinkTransitionJournal
	t.Cleanup(func() { persistCommittedSymlinkTransitionJournal = originalPersist })
	persistCommittedSymlinkTransitionJournal = func(*symlinkTransitionJournal) error {
		return errors.New("injected committed-journal persistence failure")
	}
	if err := transactions.markArtifactsCommitted(); err == nil {
		t.Fatal("markArtifactsCommitted succeeded despite injected persistence failure")
	}
	persistCommittedSymlinkTransitionJournal = originalPersist
	if transactions.journal.committed {
		t.Fatal("in-memory journal remained committed after persistence failure")
	}
	durableJournal, err := loadSymlinkTransitionJournal(root)
	if err != nil {
		t.Fatal(err)
	}
	if durableJournal == nil {
		t.Fatal("durable journal disappeared after persistence failure")
	}
	if durableJournal.committed {
		durableJournal.close()
		t.Fatal("durable journal was marked committed after persistence failure")
	}
	durableJournal.close()
	if err := transactions.rollback(); err != nil {
		t.Fatalf("rollback after committed-journal persistence failure: %v", err)
	}
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want restored directory", info, err)
	}
	if content, err := os.ReadFile(retiredPath); err != nil || string(content) != "old settings" {
		t.Fatalf("retired file = %q, %v; want restored content", content, err)
	}
}

func TestCompleteUpdateMarkArtifactsCommittedFailureRollsBackOrdinaryGeneration(t *testing.T) {
	root, prep, managedDir, retiredPath, _, manifestPath := setupSymlinkTransitionUpdate(t)
	overwrittenPath := filepath.Join(root, "ordinary-overwritten.txt")
	if err := os.WriteFile(overwrittenPath, []byte("old ordinary content"), 0600); err != nil {
		t.Fatal(err)
	}
	createdPath := filepath.Join(root, "ordinary-created.txt")
	prep.Template.Files = append(prep.Template.Files,
		model.TemplateFile{Path: "ordinary-overwritten.txt", Content: []byte("new ordinary content"), Mode: 0644},
		model.TemplateFile{Path: "ordinary-created.txt", Content: []byte("new ordinary file"), Mode: 0644},
	)
	originalPersist := persistCommittedSymlinkTransitionJournal
	persistCommittedSymlinkTransitionJournal = func(*symlinkTransitionJournal) error {
		return errors.New("injected committed-journal persistence failure")
	}
	t.Cleanup(func() { persistCommittedSymlinkTransitionJournal = originalPersist })

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true,
	}); err == nil {
		t.Fatal("update succeeded despite committed-journal persistence failure")
	}
	if _, err := os.Lstat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("created ordinary file = %v; want rollback removal", err)
	}
	assertFileContentUnchanged(t, overwrittenPath, []byte("old ordinary content"))
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want restored directory", info, err)
	}
	assertFileContentUnchanged(t, retiredPath, []byte("old settings"))
	manifest, err := config.LoadIgnManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(manifest.Files, createdPath) || slices.Contains(manifest.Files, managedDir) {
		t.Fatalf("manifest paths = %v; want pre-commit tracking state", manifest.Files)
	}
}

func TestRollbackArchivesSubstitutedReplacementAndRetainsRecoveryState(t *testing.T) {
	root, _, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transactions.close)
	originalHook := afterSymlinkTransitionReplacementValidation
	t.Cleanup(func() { afterSymlinkTransitionReplacementValidation = originalHook })
	afterSymlinkTransitionReplacementValidation = func() {
		if err := os.Remove(managedDir); err != nil {
			t.Fatalf("remove expected replacement symlink: %v", err)
		}
		if err := os.WriteFile(managedDir, []byte("concurrent replacement"), 0600); err != nil {
			t.Fatalf("create concurrent replacement: %v", err)
		}
	}

	if err := transactions.rollback(); err == nil {
		t.Fatal("rollback succeeded after replacement substitution")
	}
	if _, err := os.Lstat(managedDir); !os.IsNotExist(err) {
		t.Fatalf("replacement path = %v; want absent after concurrent node archival", err)
	}
	archive := findTransitionArchive(t, root)
	assertFileContentUnchanged(t, archive, []byte("concurrent replacement"))
	backup := findTransitionBackup(t, root)
	if info, err := os.Lstat(backup); err != nil || !info.IsDir() {
		t.Fatalf("backup = %v, %v; want retained directory", info, err)
	}
	journalPath := filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)
	if _, err := os.Lstat(journalPath); err != nil {
		t.Fatalf("recovery journal = %v; want retained", err)
	}

	if err := recoverSymlinkTransitionJournal(root, filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)); err != nil {
		t.Fatalf("recover archived concurrent replacement: %v", err)
	}
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want restored directory", info, err)
	}
	assertFileContentUnchanged(t, retiredPath, []byte("old settings"))
	assertFileContentUnchanged(t, archive, []byte("concurrent replacement"))
	if _, err := os.Lstat(journalPath); !os.IsNotExist(err) {
		t.Fatalf("recovery journal = %v; want removed after successful recovery", err)
	}
}

func TestRecoverSymlinkTransitionJournalPreflightsEveryRecordBeforeMutation(t *testing.T) {
	root := t.TempDir()
	setupTestTemplate(t, root, testHash1)
	for _, name := range []string{".claude", ".other"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "settings.json"), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	manifestPath := filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)
	if err := config.SaveIgnManifest(manifestPath, &model.IgnManifest{Files: []string{}}); err != nil {
		t.Fatal(err)
	}
	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		filepath.Join(root, ".claude"): {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
		filepath.Join(root, ".other"):  {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transactions.close)
	manifestBefore, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	diverged := filepath.Join(root, ".other")
	if err := os.Remove(diverged); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".manual", diverged); err != nil {
		t.Fatal(err)
	}

	if err := recoverSymlinkTransitionJournal(root, manifestPath); err == nil {
		t.Fatal("recovery accepted a divergent replacement symlink")
	}
	assertFileContentUnchanged(t, manifestPath, manifestBefore)
	if target, err := os.Readlink(diverged); err != nil || target != ".manual" {
		t.Fatalf("divergent replacement = %q, %v; want preserved manual symlink", target, err)
	}
	for _, entry := range transactions.entries {
		backup := filepath.Join(root, entry.backup)
		if _, err := os.Lstat(backup); err != nil {
			t.Fatalf("backup %s = %v; want retained until valid recovery", backup, err)
		}
	}
}

func TestCompleteUpdateConfigOnlyRecoversPendingSymlinkTransition(t *testing.T) {
	tempDir, prep, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	transactions, err := prepareSymlinkTransitionTransactions(tempDir, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transactions.close)
	prep.NewHash = prep.CurrentHash
	prep.Template.Config.Hash = prep.CurrentHash
	prep.HashChanged = false
	prep.RefChanged = true
	prep.RefOverrideRequested = true
	prep.RequestedRef = "recovered-ref"

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, DryRun: true,
	}); err == nil {
		t.Fatal("config-only dry run succeeded with a pending transition journal")
	}
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir,
	}); err != nil {
		t.Fatalf("config-only update did not recover pending transition: %v", err)
	}
	if info, err := os.Lstat(managedDir); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want recovered directory", info, err)
	}
	if content, err := os.ReadFile(retiredPath); err != nil || string(content) != "old settings" {
		t.Fatalf("retired file = %q, %v; want recovered content", content, err)
	}
	configPath := filepath.Join(tempDir, model.IgnConfigDir, model.IgnProjectConfigFile)
	loaded, err := config.LoadIgnConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Template.Ref != "recovered-ref" {
		t.Fatalf("tracked ref = %q, want recovered-ref", loaded.Template.Ref)
	}
}

func TestCompleteUpdateRestoresDirectoryWhenSourceChangesBeforeTransitionRename(t *testing.T) {
	tempDir, prep, managedDir, _, _, _ := setupSymlinkTransitionUpdate(t)
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalHook := beforeSymlinkTransitionRename
	t.Cleanup(func() { beforeSymlinkTransitionRename = originalHook })
	added := filepath.Join(managedDir, "added-after-preview.txt")
	beforeSymlinkTransitionRename = func() {
		if err := os.WriteFile(added, []byte("user content"), 0644); err != nil {
			t.Fatalf("add concurrent descendant: %v", err)
		}
	}

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: tempDir, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err == nil {
		t.Fatal("update succeeded after a descendant appeared before transition rename")
	}
	info, err := os.Lstat(managedDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("managed directory = %v, %v; want restored directory", info, err)
	}
	assertFileContentUnchanged(t, added, []byte("user content"))
}

func TestRecoverCommittedSymlinkTransitionRejectsDivergentStateWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, managedDir string, transaction *symlinkTransitionTransaction)
		check  func(t *testing.T, managedDir string, transaction *symlinkTransitionTransaction)
	}{
		{
			name: "missing destination",
			mutate: func(t *testing.T, managedDir string, _ *symlinkTransitionTransaction) {
				t.Helper()
				if err := os.Remove(managedDir); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, managedDir string, transaction *symlinkTransitionTransaction) {
				t.Helper()
				if _, err := os.Lstat(managedDir); !os.IsNotExist(err) {
					t.Fatalf("destination = %v; want missing", err)
				}
				info, err := os.Lstat(filepath.Join(filepath.Dir(managedDir), transaction.backup))
				if err != nil || !info.IsDir() {
					t.Fatalf("backup = %v, %v; want retained directory", info, err)
				}
			},
		},
		{
			name: "directory destination",
			mutate: func(t *testing.T, managedDir string, _ *symlinkTransitionTransaction) {
				t.Helper()
				if err := os.Remove(managedDir); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(managedDir, 0755); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, managedDir string, _ *symlinkTransitionTransaction) {
				t.Helper()
				info, err := os.Lstat(managedDir)
				if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					t.Fatalf("destination = %v, %v; want retained directory", info, err)
				}
			},
		},
		{
			name: "non-directory backup",
			mutate: func(t *testing.T, managedDir string, transaction *symlinkTransitionTransaction) {
				t.Helper()
				backup := filepath.Join(filepath.Dir(managedDir), transaction.backup)
				if err := os.RemoveAll(backup); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(backup, []byte("not a directory"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, managedDir string, transaction *symlinkTransitionTransaction) {
				t.Helper()
				if target, err := os.Readlink(managedDir); err != nil || target != ".agents" {
					t.Fatalf("replacement target = %q, %v; want .agents", target, err)
				}
				assertFileContentUnchanged(t, filepath.Join(filepath.Dir(managedDir), transaction.backup), []byte("not a directory"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, _, managedDir, _, _, _ := setupSymlinkTransitionUpdate(t)
			transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
				managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(transactions.close)
			if err := transactions.markArtifactsCommitted(); err != nil {
				t.Fatal(err)
			}
			transaction := transactions.entries[0]
			tt.mutate(t, managedDir, transaction)
			journalPath := filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)
			journalBefore, err := os.ReadFile(journalPath)
			if err != nil {
				t.Fatal(err)
			}

			if err := recoverSymlinkTransitionJournal(root, filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)); err == nil {
				t.Fatal("committed recovery accepted divergent state")
			}
			tt.check(t, managedDir, transaction)
			assertFileContentUnchanged(t, journalPath, journalBefore)
		})
	}
}

func TestCompleteUpdateArchivesCommittedTransitionBackup(t *testing.T) {
	root, prep, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if target, err := os.Readlink(managedDir); err != nil || target != ".agents" {
		t.Fatalf("replacement symlink = %q, %v; want .agents", target, err)
	}
	archive := findTransitionArchive(t, root)
	assertFileContentUnchanged(t, filepath.Join(archive, filepath.Base(retiredPath)), []byte("old settings"))
	if _, err := os.Stat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); !os.IsNotExist(err) {
		t.Fatalf("transaction journal = %v; want removed after archival", err)
	}
}

func TestCommittedTransitionArchiveCollisionRetainsBackupAndJournal(t *testing.T) {
	root, _, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	transactions, err := prepareSymlinkTransitionTransactions(root, map[string]generator.SymlinkTransition{
		managedDir: {Disposition: generator.SymlinkTransitionEligible, Target: ".agents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(transactions.close)
	if err := transactions.markArtifactsCommitted(); err != nil {
		t.Fatal(err)
	}
	const collisionName = ".ign-symlink-archive-collision"
	collisionPath := filepath.Join(root, model.IgnConfigDir, collisionName)
	if err := os.WriteFile(collisionPath, []byte("archive sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	originalSelector := selectSymlinkTransitionArchiveName
	t.Cleanup(func() { selectSymlinkTransitionArchiveName = originalSelector })
	selectSymlinkTransitionArchiveName = func(int) (string, error) { return collisionName, nil }

	if err := transactions.commit(); err == nil {
		t.Fatal("commit succeeded despite occupied archive destination")
	}
	assertFileContentUnchanged(t, collisionPath, []byte("archive sentinel"))
	backup := filepath.Join(root, transactions.entries[0].backup)
	assertFileContentUnchanged(t, filepath.Join(backup, filepath.Base(retiredPath)), []byte("old settings"))
	if _, err := os.Stat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); err != nil {
		t.Fatalf("transaction journal = %v; want retained", err)
	}
}

func TestCompleteUpdateArchivesUnexpectedDescendantAddedThroughPreopenedDescriptor(t *testing.T) {
	root, prep, managedDir, _, _, _ := setupSymlinkTransitionUpdate(t)
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalBefore := beforeSymlinkTransitionRename
	originalAfter := afterSymlinkTransitionSnapshot
	t.Cleanup(func() {
		beforeSymlinkTransitionRename = originalBefore
		afterSymlinkTransitionSnapshot = originalAfter
	})
	preopenedFD := -1
	beforeSymlinkTransitionRename = func() {
		var openErr error
		preopenedFD, openErr = unix.Open(managedDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			t.Fatalf("open source directory before rename: %v", openErr)
		}
	}
	afterSymlinkTransitionSnapshot = func() {
		fd, openErr := unix.Openat(preopenedFD, "added-after-snapshot.txt", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
		if openErr != nil {
			t.Fatalf("create descendant through pre-opened directory: %v", openErr)
		}
		if _, writeErr := unix.Write(fd, []byte("user content")); writeErr != nil {
			_ = unix.Close(fd)
			t.Fatalf("write descendant through pre-opened directory: %v", writeErr)
		}
		if closeErr := unix.Close(fd); closeErr != nil {
			t.Fatalf("close descendant through pre-opened directory: %v", closeErr)
		}
		if closeErr := unix.Close(preopenedFD); closeErr != nil {
			t.Fatalf("close pre-opened source directory: %v", closeErr)
		}
		preopenedFD = -1
	}
	t.Cleanup(func() {
		if preopenedFD >= 0 {
			_ = unix.Close(preopenedFD)
		}
	})

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err != nil {
		t.Fatalf("update with retained backup descendant: %v", err)
	}
	if target, err := os.Readlink(managedDir); err != nil || target != ".agents" {
		t.Fatalf("replacement symlink = %q, %v; want .agents", target, err)
	}
	archive := findTransitionArchive(t, root)
	assertFileContentUnchanged(t, filepath.Join(archive, "added-after-snapshot.txt"), []byte("user content"))
	if _, err := os.Stat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); !os.IsNotExist(err) {
		t.Fatalf("transaction journal = %v; want removed after archival", err)
	}
}

func TestCompleteUpdateArchivesRegularFileMutatedThroughPreopenedDescriptor(t *testing.T) {
	root, prep, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalBefore := beforeSymlinkTransitionRename
	originalAfter := afterSymlinkTransitionSnapshot
	t.Cleanup(func() {
		beforeSymlinkTransitionRename = originalBefore
		afterSymlinkTransitionSnapshot = originalAfter
	})
	preopenedFD := -1
	beforeSymlinkTransitionRename = func() {
		var openErr error
		preopenedFD, openErr = unix.Open(retiredPath, unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			t.Fatalf("open source file before rename: %v", openErr)
		}
	}
	afterSymlinkTransitionSnapshot = func() {
		mutation := []byte("user mutation")
		if _, writeErr := unix.Pwrite(preopenedFD, mutation, 0); writeErr != nil {
			t.Fatalf("mutate source file through pre-opened descriptor: %v", writeErr)
		}
		if truncateErr := unix.Ftruncate(preopenedFD, int64(len(mutation))); truncateErr != nil {
			t.Fatalf("truncate mutated source file: %v", truncateErr)
		}
		if closeErr := unix.Close(preopenedFD); closeErr != nil {
			t.Fatalf("close pre-opened source file: %v", closeErr)
		}
		preopenedFD = -1
	}
	t.Cleanup(func() {
		if preopenedFD >= 0 {
			_ = unix.Close(preopenedFD)
		}
	})

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err != nil {
		t.Fatalf("update with retained mutated backup file: %v", err)
	}
	if target, err := os.Readlink(managedDir); err != nil || target != ".agents" {
		t.Fatalf("replacement symlink = %q, %v; want .agents", target, err)
	}
	archive := findTransitionArchive(t, root)
	assertFileContentUnchanged(t, filepath.Join(archive, filepath.Base(retiredPath)), []byte("user mutation"))
	if _, err := os.Stat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); !os.IsNotExist(err) {
		t.Fatalf("transaction journal = %v; want removed after archival", err)
	}
}

func TestCompleteUpdateArchivesLeafReplacedThroughPreopenedDirectoryDescriptor(t *testing.T) {
	root, prep, managedDir, retiredPath, _, _ := setupSymlinkTransitionUpdate(t)
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalBefore := beforeSymlinkTransitionRename
	originalAfter := afterSymlinkTransitionSnapshot
	t.Cleanup(func() {
		beforeSymlinkTransitionRename = originalBefore
		afterSymlinkTransitionSnapshot = originalAfter
	})
	preopenedFD := -1
	beforeSymlinkTransitionRename = func() {
		var openErr error
		preopenedFD, openErr = unix.Open(managedDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			t.Fatalf("open source directory before rename: %v", openErr)
		}
	}
	replaced := false
	afterSymlinkTransitionSnapshot = func() {
		if replaced {
			return
		}
		replaced = true
		fd, openErr := unix.Openat(preopenedFD, "concurrent-replacement", unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
		if openErr != nil {
			t.Fatalf("create concurrent replacement: %v", openErr)
		}
		if _, writeErr := unix.Write(fd, []byte("concurrent leaf")); writeErr != nil {
			_ = unix.Close(fd)
			t.Fatalf("write concurrent replacement: %v", writeErr)
		}
		if closeErr := unix.Close(fd); closeErr != nil {
			t.Fatalf("close concurrent replacement: %v", closeErr)
		}
		if renameErr := unix.Renameat(preopenedFD, "concurrent-replacement", preopenedFD, filepath.Base(retiredPath)); renameErr != nil {
			t.Fatalf("replace snapshot leaf through pre-opened directory: %v", renameErr)
		}
		if closeErr := unix.Close(preopenedFD); closeErr != nil {
			t.Fatalf("close pre-opened source directory: %v", closeErr)
		}
		preopenedFD = -1
	}
	t.Cleanup(func() {
		if preopenedFD >= 0 {
			_ = unix.Close(preopenedFD)
		}
	})

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err != nil {
		t.Fatalf("update with retained replacement leaf: %v", err)
	}
	archive := findTransitionArchive(t, root)
	assertFileContentUnchanged(t, filepath.Join(archive, filepath.Base(retiredPath)), []byte("concurrent leaf"))
	if _, err := os.Stat(filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)); !os.IsNotExist(err) {
		t.Fatalf("transaction journal = %v; want removed after archival", err)
	}
}

func TestCompleteUpdateDoesNotOverwriteConcurrentDestinationBeforeRestore(t *testing.T) {
	root, prep, managedDir, _, _, _ := setupSymlinkTransitionUpdate(t)
	preview, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalHook := afterSymlinkTransitionSnapshot
	t.Cleanup(func() { afterSymlinkTransitionSnapshot = originalHook })
	afterSymlinkTransitionSnapshot = func() {
		if err := os.WriteFile(managedDir, []byte("concurrent destination"), 0600); err != nil {
			t.Fatalf("recreate transition destination: %v", err)
		}
	}

	if _, err := CompleteUpdate(context.Background(), CompleteUpdateOptions{
		PrepareResult: prep, NewVariables: map[string]interface{}{}, OutputDir: root, Overwrite: true, ExecutionPlan: preview.ExecutionPlan,
	}); err == nil {
		t.Fatal("update succeeded after destination recreation")
	}
	assertFileContentUnchanged(t, managedDir, []byte("concurrent destination"))
	backup := findTransitionBackup(t, root)
	if info, err := os.Lstat(backup); err != nil || !info.IsDir() {
		t.Fatalf("backup = %v, %v; want retained directory", info, err)
	}
	journalPath := filepath.Join(root, model.IgnConfigDir, symlinkTransitionJournalFile)
	journalBefore, err := os.ReadFile(journalPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := recoverSymlinkTransitionJournal(root, filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)); err == nil {
		t.Fatal("recovery overwrote recreated destination")
	}
	assertFileContentUnchanged(t, managedDir, []byte("concurrent destination"))
	assertFileContentUnchanged(t, journalPath, journalBefore)
}

func findTransitionBackup(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if isTransactionBackupName(entry.Name()) {
			return filepath.Join(root, entry.Name())
		}
	}
	t.Fatal("transition backup was not retained")
	return ""
}

func findTransitionArchive(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, model.IgnConfigDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".ign-symlink-archive-") {
			return filepath.Join(root, model.IgnConfigDir, entry.Name())
		}
	}
	t.Fatal("transition backup archive was not retained")
	return ""
}
