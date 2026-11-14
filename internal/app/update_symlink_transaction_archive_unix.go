//go:build darwin || linux

package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/sys/unix"
)

var selectSymlinkTransitionArchiveName = unusedSymlinkTransitionArchiveName

// afterSymlinkTransitionReplacementValidation is a test seam for the narrow
// interval before the replacement node is atomically moved into the durable
// archive. Production leaves it as a no-op.
var afterSymlinkTransitionReplacementValidation = func() {}

// archiveBackup moves a committed backup into .ign instead of deleting it.
// The journal is cleared only after this same-filesystem rename succeeds.
func (journal *symlinkTransitionJournal) archiveBackup(parentFD int, backup string) error {
	var stat unix.Stat_t
	err := unix.Fstatat(parentFD, backup, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if isNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect committed transition backup: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR {
		return fmt.Errorf("committed transition backup is not a directory")
	}
	archive, err := selectSymlinkTransitionArchiveName(journal.journalFD)
	if err != nil {
		return err
	}
	if err := renameNoReplaceAcrossAt(parentFD, backup, journal.journalFD, archive); err != nil {
		return fmt.Errorf("archive committed transition backup: %w", err)
	}
	if err := unix.Fsync(parentFD); err != nil {
		return fmt.Errorf("sync transition backup parent after archive: %w", err)
	}
	if err := unix.Fsync(journal.journalFD); err != nil {
		return fmt.Errorf("sync transition archive directory: %w", err)
	}
	return nil
}

// archiveExpectedReplacementSymlinkAt removes a replacement only by moving it
// to durable journal storage with a no-replace rename. It validates the moved
// node's identity afterwards; if another process substituted the destination,
// that node remains archived and the caller retains the backup and journal for
// later recovery instead of deleting or overwriting the concurrent node.
func (journal *symlinkTransitionJournal) archiveExpectedReplacementSymlinkAt(parentFD int, name, expectedTarget string) error {
	var stat unix.Stat_t
	err := unix.Fstatat(parentFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if isNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect replacement symlink: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFLNK {
		return fmt.Errorf("replacement destination is not the expected symlink")
	}
	target, err := readlinkAt(parentFD, name)
	if err != nil {
		return fmt.Errorf("read replacement symlink: %w", err)
	}
	if target != expectedTarget {
		return fmt.Errorf("replacement symlink target changed")
	}

	afterSymlinkTransitionReplacementValidation()
	archive, err := selectSymlinkTransitionArchiveName(journal.journalFD)
	if err != nil {
		return err
	}
	if err := renameNoReplaceAcrossAt(parentFD, name, journal.journalFD, archive); err != nil {
		return fmt.Errorf("archive replacement symlink: %w", err)
	}
	if err := unix.Fsync(parentFD); err != nil {
		return fmt.Errorf("sync replacement parent after archive: %w", err)
	}
	if err := unix.Fsync(journal.journalFD); err != nil {
		return fmt.Errorf("sync replacement archive directory: %w", err)
	}

	var archived unix.Stat_t
	if err := unix.Fstatat(journal.journalFD, archive, &archived, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return fmt.Errorf("inspect archived replacement symlink: %w", err)
	}
	if archived.Mode&unix.S_IFMT != unix.S_IFLNK || uint64(archived.Dev) != uint64(stat.Dev) || uint64(archived.Ino) != uint64(stat.Ino) {
		return fmt.Errorf("replacement destination changed during archival")
	}
	archivedTarget, err := readlinkAt(journal.journalFD, archive)
	if err != nil {
		return fmt.Errorf("read archived replacement symlink: %w", err)
	}
	if archivedTarget != expectedTarget {
		return fmt.Errorf("replacement destination changed during archival")
	}
	return nil
}

func unusedSymlinkTransitionArchiveName(journalFD int) (string, error) {
	for range 32 {
		bytes := make([]byte, 12)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate transition archive name: %w", err)
		}
		name := ".ign-symlink-archive-" + hex.EncodeToString(bytes)
		var stat unix.Stat_t
		err := unix.Fstatat(journalFD, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
		if isNotExist(err) {
			return name, nil
		}
		if err != nil {
			return "", fmt.Errorf("check transition archive path: %w", err)
		}
	}
	return "", fmt.Errorf("allocate unique transition archive name")
}
