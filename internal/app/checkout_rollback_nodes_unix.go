//go:build darwin || linux

package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func checkoutAtomicRollbackSupported() bool { return true }

// The hooks make the narrow rollback races deterministic in regression tests.
// Production keeps both as no-ops. Tests that use them must not run in
// parallel.
var beforeCheckoutRollbackArchive = func() {}
var beforeCheckoutRollbackRestore = func() {}

// archiveExpectedCheckoutNode removes a generated node only by atomically
// moving it into a private, same-filesystem archive. The moved node is then
// verified against the post-generation identity and fingerprint. A concurrent
// substitution or an in-place change is restored without replacing a newly
// created destination, and rollback fails closed.
func archiveExpectedCheckoutNode(path string, expected os.FileInfo, fingerprint, privateArchiveDir string) error {
	if fingerprint == "" {
		return fmt.Errorf("rollback destination has no expected node fingerprint")
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	archiveDir, err := checkoutRollbackArchiveDirectory(path, privateArchiveDir)
	if err != nil {
		return fmt.Errorf("create rollback archive: %w", err)
	}
	archivedPath := filepath.Join(archiveDir, "node")

	beforeCheckoutRollbackArchive()
	if err := rollbackRenameNoReplace(path, archivedPath); err != nil {
		return fmt.Errorf("archive rollback destination: %w", err)
	}

	actual, actualFingerprint, err := captureCheckoutRollbackNode(archivedPath)
	if err == nil && (expected == nil || os.SameFile(actual, expected)) && actualFingerprint == fingerprint {
		// Keep the private archive instead of unlinking by pathname. A later
		// in-place mutation cannot be distinguished from the generated node at
		// unlink time, so retaining this bounded failure artifact is safer than
		// deleting a concurrently changed node.
		return nil
	}
	if err == nil {
		err = fmt.Errorf("rollback destination changed concurrently")
	}
	if restoreErr := rollbackRenameNoReplace(archivedPath, path); restoreErr != nil {
		return fmt.Errorf("%v; retain archived concurrent node: %w", err, restoreErr)
	}
	return err
}

func restoreCheckoutRollbackEntryAtomically(entry checkoutRollbackEntry, privateArchiveDir string) error {
	if !entry.existed {
		return archiveExpectedCheckoutNode(entry.path, entry.expectedInfo, entry.expectedFingerprint, privateArchiveDir)
	}

	if _, err := os.Lstat(entry.path); err == nil {
		if err := archiveExpectedCheckoutNode(entry.path, entry.expectedInfo, entry.expectedFingerprint, privateArchiveDir); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	parent := filepath.Dir(entry.path)
	stageDir, err := os.MkdirTemp(parent, ".ign-rollback-restore-")
	if err != nil {
		return fmt.Errorf("create rollback restore staging directory: %w", err)
	}
	stagedPath := filepath.Join(stageDir, filepath.Base(entry.path))
	if err := materializeCheckoutRollbackEntry(entry, stagedPath); err != nil {
		return err
	}

	beforeCheckoutRollbackRestore()
	if err := rollbackRenameNoReplace(stagedPath, entry.path); err != nil {
		return fmt.Errorf("restore rollback preimage without replacing destination: %w", err)
	}
	if err := os.Remove(stageDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove empty rollback staging directory: %w", err)
	}
	return nil
}

func checkoutRollbackArchiveDirectory(path, privateArchiveDir string) (string, error) {
	if privateArchiveDir == "" {
		return os.MkdirTemp(filepath.Dir(path), ".ign-rollback-archive-")
	}
	return os.MkdirTemp(privateArchiveDir, "archive-")
}

func materializeCheckoutRollbackEntry(entry checkoutRollbackEntry, destination string) error {
	switch {
	case entry.isSymlink:
		return os.Symlink(entry.linkTarget, destination)
	case entry.isDir:
		if err := os.Mkdir(destination, entry.mode.Perm()); err != nil {
			return err
		}
		if err := os.Chmod(destination, entry.mode.Perm()); err != nil {
			return err
		}
		for _, child := range entry.children {
			if err := materializeCheckoutRollbackEntry(child, filepath.Join(destination, filepath.Base(child.path))); err != nil {
				return err
			}
		}
		return nil
	default:
		return materializeCheckoutRollbackFile(entry.backupPath, destination, entry.mode.Perm())
	}
}

func materializeCheckoutRollbackFile(source, destination string, mode os.FileMode) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}

func rollbackRenameNoReplace(from, to string) error {
	fromParent, err := unix.Open(filepath.Dir(from), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(fromParent) }()
	toParent, err := unix.Open(filepath.Dir(to), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(toParent) }()
	return renameNoReplaceAcrossAt(fromParent, filepath.Base(from), toParent, filepath.Base(to))
}
