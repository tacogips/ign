//go:build darwin || linux

package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/tacogips/ign/internal/template/generator"
	"github.com/tacogips/ign/internal/template/model"
)

var persistCommittedSymlinkTransitionJournal = func(journal *symlinkTransitionJournal) error {
	return journal.persist()
}

// beforeSymlinkTransitionRename is a test seam for the narrow interval between
// plan validation and the descriptor-relative rename. Production leaves it as
// a no-op; the post-rename fingerprint check is the safety boundary.
var beforeSymlinkTransitionRename = func() {}

var afterSymlinkTransitionSnapshot = func() {}

// symlinkTransitionTransactions holds open parent descriptors for transitions
// that have replaced a managed directory but whose tracking artifacts have not
// yet committed. The descriptors keep every mutation relative to the verified
// output tree even if a path ancestor is swapped concurrently.
type symlinkTransitionTransactions struct {
	entries   []*symlinkTransitionTransaction
	journal   *symlinkTransitionJournal
	finalized bool
}

type symlinkTransitionTransaction struct {
	parentFD int
	path     string
	name     string
	backup   string
	target   string
	snapshot []symlinkTransitionSnapshotNode
	done     bool
}

func prepareSymlinkTransitionTransactions(outputDir string, transitions map[string]generator.SymlinkTransition, artifactPaths ...string) (*symlinkTransitionTransactions, error) {
	if err := validateTransitionArtifactPaths(artifactPaths...); err != nil {
		return nil, err
	}
	transactions := &symlinkTransitionTransactions{}
	for path, transition := range transitions {
		if transition.Disposition != generator.SymlinkTransitionEligible {
			continue
		}
		if transactions.journal == nil {
			journal, err := openSymlinkTransitionJournal(outputDir, artifactPaths...)
			if err != nil {
				return nil, err
			}
			transactions.journal = journal
			if err := journal.captureArtifactPreimages(); err != nil {
				journal.close()
				return nil, err
			}
		}
		entry, err := beginSymlinkTransition(outputDir, path, transition, transactions.journal)
		if err != nil {
			if len(transactions.journal.entries) > len(transactions.entries) {
				// The current entry could not be restored without overwriting a
				// concurrent destination. Keep its durable journal and backup for
				// a later safe recovery instead of clearing actionable state.
				transactions.close()
				return nil, err
			}
			_ = transactions.rollback()
			return nil, err
		}
		transactions.entries = append(transactions.entries, entry)
	}
	return transactions, nil
}

func (transactions *symlinkTransitionTransactions) rollback() error {
	if transactions == nil || transactions.finalized {
		return nil
	}
	// Once tracking artifacts are durably committed, an unsuccessful backup
	// cleanup is recovered by finalizing the committed journal on the next
	// update. Do not restore a now-committed project to a possibly partial tree.
	if transactions.journal != nil && transactions.journal.committed {
		transactions.close()
		return nil
	}
	var errors []error
	for i := len(transactions.entries) - 1; i >= 0; i-- {
		entry := transactions.entries[i]
		if entry.done {
			continue
		}
		if err := transactions.journal.archiveExpectedReplacementSymlinkAt(entry.parentFD, entry.name, entry.target); err != nil {
			errors = append(errors, fmt.Errorf("archive replacement symlink: %w", err))
			continue
		}
		if err := restoreTransitionDirectoryNoReplaceAt(entry.parentFD, entry.backup, entry.name); err != nil {
			errors = append(errors, fmt.Errorf("restore managed directory: %w", err))
			continue
		}
		entry.done = true
	}
	if transactions.journal != nil {
		if err := transactions.journal.restoreArtifactPreimages(); err != nil {
			errors = append(errors, fmt.Errorf("restore rollback tracking artifacts: %w", err))
		}
		if len(errors) == 0 {
			if err := transactions.journal.clear(); err != nil {
				errors = append(errors, fmt.Errorf("remove rollback transaction journal: %w", err))
			}
		}
	}
	transactions.close()
	return errorsJoin(errors)
}

func (transactions *symlinkTransitionTransactions) commit() error {
	if transactions == nil {
		return nil
	}
	if transactions.journal != nil && !transactions.journal.committed {
		return fmt.Errorf("tracking artifacts are not durably committed")
	}
	var errors []error
	for _, entry := range transactions.entries {
		if entry.done {
			continue
		}
		if err := transactions.journal.archiveBackup(entry.parentFD, entry.backup); err != nil {
			errors = append(errors, fmt.Errorf("archive transition backup: %w", err))
			continue
		}
		entry.done = true
	}
	if len(errors) > 0 {
		// The committed journal still identifies every backup. Recovery archives
		// extant backups and safely clears records whose backup was already moved.
		transactions.close()
		return errorsJoin(errors)
	}
	if transactions.journal != nil {
		if err := transactions.journal.clear(); err != nil {
			transactions.close()
			return fmt.Errorf("remove committed transaction journal: %w", err)
		}
	}
	transactions.finalized = true
	transactions.close()
	return errorsJoin(errors)
}

func (transactions *symlinkTransitionTransactions) close() {
	if transactions == nil {
		return
	}
	for _, entry := range transactions.entries {
		if entry.parentFD >= 0 {
			_ = unix.Close(entry.parentFD)
			entry.parentFD = -1
		}
	}
	if transactions.journal != nil {
		transactions.journal.close()
	}
}

func beginSymlinkTransition(outputDir, path string, transition generator.SymlinkTransition, journal *symlinkTransitionJournal) (*symlinkTransitionTransaction, error) {
	if transition.Target == "" {
		return nil, fmt.Errorf("missing symlink target for managed transition %s", path)
	}
	parentFD, name, err := openTransitionParent(outputDir, path)
	if err != nil {
		return nil, err
	}
	closeParent := true
	defer func() {
		if closeParent {
			_ = unix.Close(parentFD)
		}
	}()

	var stat unix.Stat_t
	if err := unix.Fstatat(parentFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return nil, fmt.Errorf("inspect managed transition source: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return nil, fmt.Errorf("managed transition source is not a directory")
	}
	sourceFingerprint := transition.SourceFingerprint
	if sourceFingerprint == "" {
		// Direct transaction callers used by package tests do not carry an
		// execution plan. Production transitions always provide the preview
		// fingerprint, while this fallback still verifies the rename boundary.
		sourceFingerprint, err = fingerprintTransitionTreeAt(parentFD, name)
		if err != nil {
			return nil, fmt.Errorf("fingerprint managed transition source: %w", err)
		}
	}
	backup, err := unusedBackupName(parentFD)
	if err != nil {
		return nil, err
	}
	entry := &symlinkTransitionTransaction{parentFD: parentFD, path: path, name: name, backup: backup, target: transition.Target}
	if err := journal.add(entry, symlinkTransitionJournalPrepared); err != nil {
		return nil, fmt.Errorf("persist transition recovery journal: %w", err)
	}
	beforeSymlinkTransitionRename()
	if err := unix.Renameat(parentFD, name, parentFD, backup); err != nil {
		_ = journal.remove(entry)
		return nil, fmt.Errorf("move managed directory to transaction backup: %w", err)
	}
	actualFingerprint, snapshot, err := captureTransitionTreeAt(parentFD, backup)
	// Package-level transaction tests may construct an eligible transition
	// directly without planner ownership data. Production plans always include
	// retired managed paths, which makes this additional ownership proof
	// mandatory on the real update path.
	if err == nil && len(transition.RetiredManagedPaths) > 0 {
		err = validateTransitionSnapshotOwnership(path, snapshot, transition.RetiredManagedPaths)
	}
	if err != nil || actualFingerprint != sourceFingerprint {
		if restoreErr := restoreTransitionDirectoryNoReplaceAt(parentFD, backup, name); restoreErr != nil {
			return nil, fmt.Errorf("validate renamed managed directory: %v (restore managed directory: %w)", err, restoreErr)
		}
		_ = journal.remove(entry)
		if err != nil {
			return nil, fmt.Errorf("fingerprint renamed managed directory: %w", err)
		}
		return nil, fmt.Errorf("managed transition source state changed before replacement")
	}
	entry.snapshot = snapshot
	if err := journal.setSnapshot(entry, snapshot); err != nil {
		if restoreErr := restoreTransitionDirectoryNoReplaceAt(parentFD, backup, name); restoreErr != nil {
			return nil, fmt.Errorf("persist transition cleanup snapshot: %v (restore managed directory: %w)", err, restoreErr)
		}
		_ = journal.remove(entry)
		return nil, fmt.Errorf("persist transition cleanup snapshot: %w", err)
	}
	afterSymlinkTransitionSnapshot()
	if err := unix.Symlinkat(transition.Target, parentFD, name); err != nil {
		if restoreErr := restoreTransitionDirectoryNoReplaceAt(parentFD, backup, name); restoreErr != nil {
			return nil, fmt.Errorf("create replacement symlink: %w (restore managed directory: %v)", err, restoreErr)
		}
		_ = journal.remove(entry)
		return nil, fmt.Errorf("create replacement symlink: %w", err)
	}
	if err := journal.setPhase(entry, symlinkTransitionJournalReplaced); err != nil {
		if removeErr := journal.archiveExpectedReplacementSymlinkAt(parentFD, name, transition.Target); removeErr == nil {
			if restoreErr := restoreTransitionDirectoryNoReplaceAt(parentFD, backup, name); restoreErr == nil {
				_ = journal.remove(entry)
			}
		}
		return nil, fmt.Errorf("persist replacement transition recovery journal: %w", err)
	}
	closeParent = false
	return entry, nil
}

func openTransitionParent(outputDir, path string) (int, string, error) {
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return -1, "", fmt.Errorf("resolve update output directory: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return -1, "", fmt.Errorf("resolve transition path: %w", err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return -1, "", fmt.Errorf("transition path is outside output directory")
	}
	parts := strings.Split(filepath.Clean(rel), string(filepath.Separator))
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, "", fmt.Errorf("open output directory without following symlinks: %w", err)
	}
	for _, part := range parts[:len(parts)-1] {
		if part == "" || part == "." || part == ".." {
			_ = unix.Close(fd)
			return -1, "", fmt.Errorf("invalid transition path component")
		}
		next, err := unix.Openat(fd, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if err != nil {
			return -1, "", fmt.Errorf("open transition ancestor without following symlinks: %w", err)
		}
		fd = next
	}
	return fd, parts[len(parts)-1], nil
}

func transitionPathIsSafe(outputDir, path string) bool {
	fd, name, err := openTransitionParent(outputDir, path)
	if err != nil {
		return false
	}
	defer func() { _ = unix.Close(fd) }()
	var stat unix.Stat_t
	return unix.Fstatat(fd, name, &stat, unix.AT_SYMLINK_NOFOLLOW) == nil
}

func isNotExist(err error) bool { return err == unix.ENOENT || err == fs.ErrNotExist }

func errorsJoin(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("symlink transition transaction: %v", errors)
}

const (
	symlinkTransitionJournalFile     = ".ign-symlink-transitions.json"
	symlinkTransitionJournalPrepared = "prepared"
	symlinkTransitionJournalReplaced = "replaced"
)

// symlinkTransitionJournal is a project-local, descriptor-relative recovery
// record. It is deliberately kept under .ign rather than beside the replaced
// tree so interrupted transactions cannot be mistaken for template content.
type symlinkTransitionJournal struct {
	rootDir    string
	journalFD  int
	artifactFD int
	entries    []symlinkTransitionJournalEntry
	artifacts  []symlinkTransitionJournalArtifact
	committed  bool
}

type symlinkTransitionJournalEntry struct {
	Path     string                          `json:"path"`
	Backup   string                          `json:"backup"`
	Target   string                          `json:"target"`
	Phase    string                          `json:"phase"`
	Snapshot []symlinkTransitionSnapshotNode `json:"snapshot"`
}

type symlinkTransitionJournalDocument struct {
	Entries   []symlinkTransitionJournalEntry    `json:"entries"`
	Artifacts []symlinkTransitionJournalArtifact `json:"artifacts"`
	Committed bool                               `json:"committed"`
}

func recoverSymlinkTransitionJournal(outputDir, manifestPath string, artifactPaths ...string) error {
	if err := validateTransitionArtifactPaths(append([]string{manifestPath}, artifactPaths...)...); err != nil {
		return err
	}
	journal, err := loadSymlinkTransitionJournal(outputDir, manifestPath)
	if err != nil {
		return err
	}
	if journal == nil {
		return nil
	}
	defer journal.close()

	if err := validateSymlinkTransitionJournal(journal); err != nil {
		return fmt.Errorf("invalid symlink transition recovery journal: %w", err)
	}
	records, err := preflightSymlinkTransitionRecovery(journal)
	if err != nil {
		return err
	}
	defer closeSymlinkTransitionRecoveryRecords(records)
	if !journal.committed {
		if err := journal.restoreArtifactPreimages(); err != nil {
			return fmt.Errorf("restore interrupted transition tracking artifacts: %w", err)
		}
	}
	for _, record := range records {
		if err := recoverSymlinkTransitionRecord(record, journal); err != nil {
			return err
		}
	}
	return journal.clear()
}

type symlinkTransitionRecoveryAction uint8

const (
	symlinkTransitionRecoveryNoop symlinkTransitionRecoveryAction = iota
	symlinkTransitionRecoveryRestore
	symlinkTransitionRecoveryFinalize
)

type symlinkTransitionRecoveryRecord struct {
	parentFD  int
	name      string
	record    symlinkTransitionJournalEntry
	action    symlinkTransitionRecoveryAction
	committed bool
}

func preflightSymlinkTransitionRecovery(journal *symlinkTransitionJournal) ([]symlinkTransitionRecoveryRecord, error) {
	records := make([]symlinkTransitionRecoveryRecord, 0, len(journal.entries))
	for _, entry := range journal.entries {
		path, err := journal.absolutePath(entry.Path)
		if err != nil {
			closeSymlinkTransitionRecoveryRecords(records)
			return nil, err
		}
		parentFD, name, err := openTransitionParent(journal.rootDir, path)
		if err != nil {
			closeSymlinkTransitionRecoveryRecords(records)
			return nil, fmt.Errorf("open interrupted transition parent: %w", err)
		}
		action, err := preflightSymlinkTransitionRecoveryRecord(parentFD, name, entry, journal.committed)
		if err != nil {
			_ = unix.Close(parentFD)
			closeSymlinkTransitionRecoveryRecords(records)
			return nil, err
		}
		records = append(records, symlinkTransitionRecoveryRecord{parentFD: parentFD, name: name, record: entry, action: action, committed: journal.committed})
	}
	return records, nil
}

func closeSymlinkTransitionRecoveryRecords(records []symlinkTransitionRecoveryRecord) {
	for _, record := range records {
		if record.parentFD >= 0 {
			_ = unix.Close(record.parentFD)
		}
	}
}

func preflightSymlinkTransitionRecoveryRecord(parentFD int, name string, record symlinkTransitionJournalEntry, committed bool) (symlinkTransitionRecoveryAction, error) {
	var destination unix.Stat_t
	destinationErr := unix.Fstatat(parentFD, name, &destination, unix.AT_SYMLINK_NOFOLLOW)
	var backup unix.Stat_t
	backupErr := unix.Fstatat(parentFD, record.Backup, &backup, unix.AT_SYMLINK_NOFOLLOW)
	backupExists := backupErr == nil
	if backupErr != nil && !isNotExist(backupErr) {
		return symlinkTransitionRecoveryNoop, fmt.Errorf("inspect interrupted transition backup: %w", backupErr)
	}
	if backupExists && backup.Mode&unix.S_IFMT != unix.S_IFDIR {
		return symlinkTransitionRecoveryNoop, fmt.Errorf("interrupted transition backup is not a directory")
	}

	if destinationErr == nil && destination.Mode&unix.S_IFMT == unix.S_IFLNK {
		target, err := readlinkAt(parentFD, name)
		if err != nil {
			return symlinkTransitionRecoveryNoop, fmt.Errorf("read interrupted replacement symlink: %w", err)
		}
		if target != record.Target {
			return symlinkTransitionRecoveryNoop, fmt.Errorf("interrupted replacement symlink target diverged from recovery journal")
		}
	}
	if committed {
		if record.Phase != symlinkTransitionJournalReplaced {
			return symlinkTransitionRecoveryNoop, fmt.Errorf("committed transition has not recorded a replacement")
		}
		if destinationErr == nil && destination.Mode&unix.S_IFMT == unix.S_IFLNK {
			if backupExists {
				return symlinkTransitionRecoveryFinalize, nil
			}
			return symlinkTransitionRecoveryNoop, nil
		}
		if destinationErr != nil && !isNotExist(destinationErr) {
			return symlinkTransitionRecoveryNoop, fmt.Errorf("inspect interrupted transition destination: %w", destinationErr)
		}
		return symlinkTransitionRecoveryNoop, fmt.Errorf("committed transition destination diverged from recovery journal")
	}
	if destinationErr == nil && destination.Mode&unix.S_IFMT == unix.S_IFLNK && backupExists {
		return symlinkTransitionRecoveryRestore, nil
	}
	if isNotExist(destinationErr) && backupExists {
		return symlinkTransitionRecoveryRestore, nil
	}
	if destinationErr == nil && destination.Mode&unix.S_IFMT == unix.S_IFDIR && !backupExists {
		// The rollback completed before interruption; only the journal removal
		// was left outstanding.
		return symlinkTransitionRecoveryNoop, nil
	}
	if destinationErr == nil && destination.Mode&unix.S_IFMT == unix.S_IFLNK && !backupExists && committed {
		return symlinkTransitionRecoveryNoop, nil
	}
	if destinationErr != nil && !isNotExist(destinationErr) {
		return symlinkTransitionRecoveryNoop, fmt.Errorf("inspect interrupted transition destination: %w", destinationErr)
	}
	return symlinkTransitionRecoveryNoop, fmt.Errorf("interrupted transition is not recoverable without manual inspection")
}

func recoverSymlinkTransitionRecord(record symlinkTransitionRecoveryRecord, journal *symlinkTransitionJournal) error {
	if record.action == symlinkTransitionRecoveryNoop {
		return nil
	}
	if record.action == symlinkTransitionRecoveryFinalize {
		if err := journal.archiveBackup(record.parentFD, record.record.Backup); err != nil {
			return fmt.Errorf("archive committed transition backup: %w", err)
		}
		return nil
	}
	if err := journal.archiveExpectedReplacementSymlinkAt(record.parentFD, record.name, record.record.Target); err != nil {
		return fmt.Errorf("archive uncommitted replacement symlink: %w", err)
	}
	if err := restoreTransitionDirectoryNoReplaceAt(record.parentFD, record.record.Backup, record.name); err != nil {
		return fmt.Errorf("restore interrupted managed directory: %w", err)
	}
	return nil
}

func openSymlinkTransitionJournal(outputDir string, manifestPaths ...string) (*symlinkTransitionJournal, error) {
	if err := validateTransitionArtifactPaths(manifestPaths...); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve journal output directory: %w", err)
	}
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open journal output directory without following symlinks: %w", err)
	}
	defer func() { _ = unix.Close(rootFD) }()
	var stat unix.Stat_t
	err = unix.Fstatat(rootFD, model.IgnConfigDir, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if isNotExist(err) {
		if err := unix.Mkdirat(rootFD, model.IgnConfigDir, 0700); err != nil && err != unix.EEXIST {
			return nil, fmt.Errorf("create journal directory: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("inspect journal directory: %w", err)
	} else if stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return nil, fmt.Errorf("journal directory is not a directory")
	}
	journalFD, err := unix.Openat(rootFD, model.IgnConfigDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open journal directory without following symlinks: %w", err)
	}
	manifestPath := filepath.Join(root, model.IgnConfigDir, model.IgnManifestFile)
	if len(manifestPaths) > 0 && manifestPaths[0] != "" {
		manifestPath = manifestPaths[0]
	}
	artifactDir, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		_ = unix.Close(journalFD)
		return nil, fmt.Errorf("resolve tracking artifact directory: %w", err)
	}
	artifactFD, err := unix.Open(artifactDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = unix.Close(journalFD)
		return nil, fmt.Errorf("open tracking artifact directory without following symlinks: %w", err)
	}
	return &symlinkTransitionJournal{rootDir: filepath.Clean(root), journalFD: journalFD, artifactFD: artifactFD}, nil
}

func loadSymlinkTransitionJournal(outputDir string, manifestPaths ...string) (*symlinkTransitionJournal, error) {
	journal, err := openSymlinkTransitionJournal(outputDir, manifestPaths...)
	if err != nil {
		return nil, err
	}
	fd, err := unix.Openat(journal.journalFD, symlinkTransitionJournalFile, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if isNotExist(err) {
		journal.close()
		return nil, nil
	}
	if err != nil {
		journal.close()
		return nil, fmt.Errorf("open symlink transition recovery journal without following symlinks: %w", err)
	}
	file := os.NewFile(uintptr(fd), symlinkTransitionJournalFile)
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		journal.close()
		return nil, fmt.Errorf("read symlink transition recovery journal: %w", readErr)
	}
	if closeErr != nil {
		journal.close()
		return nil, fmt.Errorf("close symlink transition recovery journal: %w", closeErr)
	}
	var document symlinkTransitionJournalDocument
	if err := json.Unmarshal(data, &document); err != nil {
		journal.close()
		return nil, fmt.Errorf("parse symlink transition recovery journal: %w", err)
	}
	journal.entries = document.Entries
	journal.artifacts = document.Artifacts
	journal.committed = document.Committed
	return journal, nil
}

// symlinkTransitionJournalPending checks for a recovery record without
// creating .ign or changing transaction state, so dry-run remains non-mutating.
func symlinkTransitionJournalPending(outputDir string) (bool, error) {
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return false, fmt.Errorf("resolve journal output directory: %w", err)
	}
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, fmt.Errorf("open journal output directory without following symlinks: %w", err)
	}
	defer func() { _ = unix.Close(rootFD) }()
	var stat unix.Stat_t
	err = unix.Fstatat(rootFD, model.IgnConfigDir, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if isNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect journal directory: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return false, fmt.Errorf("journal directory is not a directory")
	}
	journalFD, err := unix.Openat(rootFD, model.IgnConfigDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, fmt.Errorf("open journal directory without following symlinks: %w", err)
	}
	defer func() { _ = unix.Close(journalFD) }()
	err = unix.Fstatat(journalFD, symlinkTransitionJournalFile, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if isNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect symlink transition recovery journal: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return false, fmt.Errorf("symlink transition recovery journal is not a regular file")
	}
	return true, nil
}

func (journal *symlinkTransitionJournal) add(entry *symlinkTransitionTransaction, phase string) error {
	path, err := journal.relativePath(entry.path)
	if err != nil {
		return err
	}
	journal.entries = append(journal.entries, symlinkTransitionJournalEntry{Path: path, Backup: entry.backup, Target: entry.target, Phase: phase})
	return journal.persist()
}

func (transactions *symlinkTransitionTransactions) markArtifactsCommitted() error {
	if transactions == nil || transactions.journal == nil {
		return nil
	}
	if err := transactions.journal.syncArtifacts(); err != nil {
		return err
	}
	return transactions.journal.markCommitted()
}

func (journal *symlinkTransitionJournal) markCommitted() error {
	journal.committed = true
	if err := persistCommittedSymlinkTransitionJournal(journal); err != nil {
		journal.committed = false
		return err
	}
	return nil
}

func (journal *symlinkTransitionJournal) clear() error {
	entries := journal.entries
	artifacts := journal.artifacts
	committed := journal.committed
	journal.entries = nil
	journal.artifacts = nil
	journal.committed = false
	if err := journal.persist(); err != nil {
		journal.entries = entries
		journal.artifacts = artifacts
		journal.committed = committed
		return err
	}
	return nil
}

func (journal *symlinkTransitionJournal) setPhase(entry *symlinkTransitionTransaction, phase string) error {
	path, err := journal.relativePath(entry.path)
	if err != nil {
		return err
	}
	for i := range journal.entries {
		if journal.entries[i].Path == path && journal.entries[i].Backup == entry.backup {
			journal.entries[i].Phase = phase
			return journal.persist()
		}
	}
	return fmt.Errorf("transition journal entry is missing")
}

func (journal *symlinkTransitionJournal) setSnapshot(entry *symlinkTransitionTransaction, snapshot []symlinkTransitionSnapshotNode) error {
	path, err := journal.relativePath(entry.path)
	if err != nil {
		return err
	}
	if err := validateTransitionSnapshot(snapshot); err != nil {
		return err
	}
	for i := range journal.entries {
		if journal.entries[i].Path == path && journal.entries[i].Backup == entry.backup {
			journal.entries[i].Snapshot = append([]symlinkTransitionSnapshotNode(nil), snapshot...)
			return journal.persist()
		}
	}
	return fmt.Errorf("transition journal entry is missing")
}

func (journal *symlinkTransitionJournal) remove(entry *symlinkTransitionTransaction) error {
	path, err := journal.relativePath(entry.path)
	if err != nil {
		return err
	}
	for i := range journal.entries {
		if journal.entries[i].Path == path && journal.entries[i].Backup == entry.backup {
			journal.entries = append(journal.entries[:i], journal.entries[i+1:]...)
			return journal.persist()
		}
	}
	return nil
}

func (journal *symlinkTransitionJournal) persist() error {
	if len(journal.entries) == 0 && len(journal.artifacts) == 0 {
		if err := unix.Unlinkat(journal.journalFD, symlinkTransitionJournalFile, 0); err != nil && !isNotExist(err) {
			return fmt.Errorf("remove empty symlink transition recovery journal: %w", err)
		}
		if err := unix.Fsync(journal.journalFD); err != nil {
			return fmt.Errorf("sync empty symlink transition journal directory: %w", err)
		}
		return nil
	}
	data, err := json.Marshal(symlinkTransitionJournalDocument{Entries: journal.entries, Artifacts: journal.artifacts, Committed: journal.committed})
	if err != nil {
		return fmt.Errorf("encode symlink transition recovery journal: %w", err)
	}
	tmp, err := unusedJournalTempName(journal.journalFD)
	if err != nil {
		return err
	}
	fd, err := unix.Openat(journal.journalFD, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return fmt.Errorf("create temporary symlink transition recovery journal: %w", err)
	}
	file := os.NewFile(uintptr(fd), tmp)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = unix.Unlinkat(journal.journalFD, tmp, 0)
		return fmt.Errorf("write symlink transition recovery journal: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = unix.Unlinkat(journal.journalFD, tmp, 0)
		return fmt.Errorf("sync symlink transition recovery journal: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(journal.journalFD, tmp, 0)
		return fmt.Errorf("close symlink transition recovery journal: %w", err)
	}
	if err := unix.Renameat(journal.journalFD, tmp, journal.journalFD, symlinkTransitionJournalFile); err != nil {
		_ = unix.Unlinkat(journal.journalFD, tmp, 0)
		return fmt.Errorf("replace symlink transition recovery journal: %w", err)
	}
	if err := unix.Fsync(journal.journalFD); err != nil {
		return fmt.Errorf("sync symlink transition journal directory: %w", err)
	}
	return nil
}

func (journal *symlinkTransitionJournal) relativePath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve transition journal path: %w", err)
	}
	rel, err := filepath.Rel(journal.rootDir, absolute)
	if err != nil {
		return "", fmt.Errorf("relativize transition journal path: %w", err)
	}
	return cleanJournalRelativePath(rel)
}

func (journal *symlinkTransitionJournal) absolutePath(path string) (string, error) {
	rel, err := cleanJournalRelativePath(path)
	if err != nil {
		return "", err
	}
	return filepath.Join(journal.rootDir, rel), nil
}

func (journal *symlinkTransitionJournal) close() {
	if journal == nil {
		return
	}
	if journal.artifactFD >= 0 {
		_ = unix.Close(journal.artifactFD)
		journal.artifactFD = -1
	}
	if journal.journalFD >= 0 {
		_ = unix.Close(journal.journalFD)
		journal.journalFD = -1
	}
}

func validateSymlinkTransitionJournalEntry(entry symlinkTransitionJournalEntry) error {
	if _, err := cleanJournalRelativePath(entry.Path); err != nil {
		return fmt.Errorf("invalid transition path: %w", err)
	}
	if !isTransactionBackupName(entry.Backup) {
		return fmt.Errorf("invalid transition backup name")
	}
	if entry.Target == "" || strings.IndexByte(entry.Target, 0) >= 0 {
		return fmt.Errorf("invalid replacement symlink target")
	}
	if entry.Phase != symlinkTransitionJournalPrepared && entry.Phase != symlinkTransitionJournalReplaced {
		return fmt.Errorf("invalid transition phase")
	}
	return nil
}

func validateSymlinkTransitionJournal(journal *symlinkTransitionJournal) error {
	if journal == nil {
		return fmt.Errorf("missing transition journal")
	}
	if len(journal.artifacts) != 3 {
		return fmt.Errorf("unexpected transition artifact count")
	}
	expectedArtifacts := map[string]struct{}{
		"ign-files.json": {},
		"ign.json":       {},
		"ign-var.json":   {},
	}
	for _, artifact := range journal.artifacts {
		if _, ok := expectedArtifacts[artifact.Name]; !ok {
			return fmt.Errorf("invalid transition artifact name %q", artifact.Name)
		}
		if !artifact.Exists && (len(artifact.Data) != 0 || artifact.Mode != 0) {
			return fmt.Errorf("missing transition artifact %s has preimage data or mode", artifact.Name)
		}
		if artifact.Exists && artifact.Mode&^uint32(0777) != 0 {
			return fmt.Errorf("invalid transition artifact %s permission mode", artifact.Name)
		}
		delete(expectedArtifacts, artifact.Name)
	}
	if len(expectedArtifacts) != 0 {
		return fmt.Errorf("missing transition artifact preimage")
	}

	paths := make(map[string]struct{}, len(journal.entries))
	backups := make(map[string]struct{}, len(journal.entries))
	for _, entry := range journal.entries {
		if err := validateSymlinkTransitionJournalEntry(entry); err != nil {
			return err
		}
		if len(entry.Snapshot) == 0 {
			if journal.committed {
				return fmt.Errorf("committed transition is missing cleanup snapshot")
			}
		} else if err := validateTransitionSnapshot(entry.Snapshot); err != nil {
			return err
		}
		if _, exists := paths[entry.Path]; exists {
			return fmt.Errorf("duplicate transition path")
		}
		if _, exists := backups[entry.Backup]; exists {
			return fmt.Errorf("duplicate transition backup")
		}
		paths[entry.Path] = struct{}{}
		backups[entry.Backup] = struct{}{}
	}
	return nil
}

func cleanJournalRelativePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute transition journal path")
	}
	path = filepath.Clean(path)
	if path == "." || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("outside-root transition journal path")
	}
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid transition journal path component")
		}
	}
	return path, nil
}

func isTransactionBackupName(name string) bool {
	const prefix = ".ign-symlink-transition-"
	if !strings.HasPrefix(name, prefix) || strings.ContainsRune(name, filepath.Separator) {
		return false
	}
	encoded := strings.TrimPrefix(name, prefix)
	if len(encoded) != 24 {
		return false
	}
	_, err := hex.DecodeString(encoded)
	return err == nil
}

func unusedJournalTempName(journalFD int) (string, error) {
	for range 32 {
		bytes := make([]byte, 12)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate transaction journal temporary name: %w", err)
		}
		name := ".ign-symlink-journal-" + hex.EncodeToString(bytes)
		var stat unix.Stat_t
		err := unix.Fstatat(journalFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
		if err == unix.ENOENT {
			return name, nil
		}
		if err != nil {
			return "", fmt.Errorf("check transaction journal temporary path: %w", err)
		}
	}
	return "", fmt.Errorf("allocate unique transaction journal temporary name")
}

func readlinkAt(parentFD int, name string) (string, error) {
	buffer := make([]byte, 256)
	for {
		n, err := unix.Readlinkat(parentFD, name, buffer)
		if err != nil {
			return "", err
		}
		if n < len(buffer) {
			return string(buffer[:n]), nil
		}
		buffer = make([]byte, len(buffer)*2)
	}
}
