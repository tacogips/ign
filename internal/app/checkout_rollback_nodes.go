package app

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/generator"
)

func (r *checkoutGenerationRollback) captureGeneratedNodes(genResult *generator.GenerateResult) {
	if r == nil || genResult == nil {
		return
	}
	r.created = r.created[:0]
	for _, path := range genResult.CreatedFiles {
		if path == "" {
			continue
		}
		info, fingerprint, err := captureCheckoutRollbackNode(path)
		if err != nil {
			if !os.IsNotExist(err) {
				debug.Debug("[app] Failed to capture generated rollback node %s: %v", path, err)
			}
			continue
		}
		r.created = append(r.created, checkoutRollbackEntry{path: path, expectedInfo: info, expectedFingerprint: fingerprint})
	}
	for i := range r.overwritten {
		info, fingerprint, err := captureCheckoutRollbackNode(r.overwritten[i].path)
		if err != nil {
			if !os.IsNotExist(err) {
				debug.Debug("[app] Failed to capture overwritten rollback node %s: %v", r.overwritten[i].path, err)
			}
			continue
		}
		r.overwritten[i].expectedInfo = info
		r.overwritten[i].expectedFingerprint = fingerprint
	}
	for i := range r.createdDirs {
		info, _, err := captureCheckoutRollbackNode(r.createdDirs[i].path)
		if err != nil {
			if !os.IsNotExist(err) {
				debug.Debug("[app] Failed to capture generated rollback directory %s: %v", r.createdDirs[i].path, err)
			}
			continue
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			debug.Debug("[app] Generated rollback directory %s is no longer a directory", r.createdDirs[i].path)
			continue
		}
		r.createdDirs[i].expectedInfo = info
		// Children are rolled back before their generated parent directories.
		// The expected final state for each newly created directory is therefore
		// an empty directory, not its post-generation tree.
		r.createdDirs[i].expectedFingerprint = fingerprintCheckoutRollbackEmptyDirectory(info)
	}
}

// captureExpectedCheckoutRollbackEntry binds a rollback entry to the node that
// was successfully written after its preimage was captured. Callers must do
// this before a later operation can fail and require restoration.
func (r *checkoutGenerationRollback) captureExpectedCheckoutRollbackEntry(entry *checkoutRollbackEntry) error {
	if r == nil || entry == nil {
		return nil
	}
	info, fingerprint, err := captureCheckoutRollbackNode(entry.path)
	if err != nil {
		return err
	}
	entry.expectedInfo = info
	entry.expectedFingerprint = fingerprint
	return nil
}

// bindExpectedCheckoutRollbackRegularFile binds a rollback entry to the exact
// bytes and mode supplied to an atomic writer. Unlike inspecting the pathname
// after the write, this cannot adopt a concurrent replacement as generation
// output.
func (r *checkoutGenerationRollback) bindExpectedCheckoutRollbackRegularFile(entry *checkoutRollbackEntry, data []byte, mode os.FileMode) {
	if r == nil || entry == nil {
		return
	}
	entry.expectedInfo = nil
	entry.expectedFingerprint = fingerprintCheckoutRollbackRegularFile(data, mode)
}

func (r *checkoutGenerationRollback) capturePotentialGeneratedDirectories(paths []string) error {
	if r == nil {
		return nil
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		_, err := os.Lstat(path)
		if os.IsNotExist(err) {
			r.createdDirs = append(r.createdDirs, checkoutRollbackEntry{path: path})
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *checkoutGenerationRollback) rollbackCreatedFiles(genResult *generator.GenerateResult) {
	if r == nil || genResult == nil {
		return
	}
	for _, entry := range r.created {
		if err := r.removeExpectedCheckoutEntry(entry); err != nil {
			r.retainBackup = true
			debug.Debug("[app] Failed to rollback generated file %s: %v", entry.path, err)
		}
	}
	r.rollbackCreatedDirectories()
}

func (r *checkoutGenerationRollback) rollbackCreatedDirectories() {
	if r == nil {
		return
	}
	sort.Slice(r.createdDirs, func(i, j int) bool {
		return len(filepath.Clean(r.createdDirs[i].path)) > len(filepath.Clean(r.createdDirs[j].path))
	})
	for _, entry := range r.createdDirs {
		if entry.expectedInfo == nil {
			continue
		}
		if err := r.removeExpectedCheckoutEntry(entry); err != nil {
			r.retainBackup = true
			debug.Debug("[app] Failed to rollback generated directory %s: %v", entry.path, err)
		}
	}
}

func restoreCheckoutRollbackEntry(entry checkoutRollbackEntry) error {
	return restoreCheckoutRollbackEntryAtomically(entry, "")
}

func (r *checkoutGenerationRollback) removeExpectedCheckoutEntry(entry checkoutRollbackEntry) error {
	archiveDir, err := r.ensureBackupDir()
	if err != nil {
		return err
	}
	err = archiveExpectedCheckoutNode(entry.path, entry.expectedInfo, entry.expectedFingerprint, archiveDir)
	r.retainPrivateArchives()
	return err
}

func (r *checkoutGenerationRollback) restoreCheckoutRollbackEntry(entry checkoutRollbackEntry) error {
	archiveDir, err := r.ensureBackupDir()
	if err != nil {
		return err
	}
	err = restoreCheckoutRollbackEntryAtomically(entry, archiveDir)
	r.retainPrivateArchives()
	return err
}

// retainPrivateArchives prevents cleanup from deleting a node after its
// identity and contents were validated. A writer holding the archived inode
// can still change it after validation; keeping the private archive is the
// fail-closed outcome until a separate pruning policy can prove it is safe.
func (r *checkoutGenerationRollback) retainPrivateArchives() {
	if r == nil || r.backupDir == "" {
		return
	}
	entries, err := os.ReadDir(r.backupDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "archive-") {
			r.retainBackup = true
			return
		}
	}
}

func captureCheckoutRollbackNode(path string) (os.FileInfo, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, "", err
	}
	fingerprint, err := fingerprintCheckoutRollbackNode(path)
	if err != nil {
		return nil, "", err
	}
	return info, fingerprint, nil
}

func fingerprintCheckoutRollbackNode(path string) (string, error) {
	hash := sha256.New()
	if err := writeCheckoutRollbackFingerprint(hash, path); err != nil {
		return "", err
	}
	return string(hash.Sum(nil)), nil
}

func fingerprintCheckoutRollbackRegularFile(data []byte, mode os.FileMode) string {
	hash := sha256.New()
	_ = writeCheckoutRollbackFingerprintField(hash, []byte(mode.String()))
	_, _ = hash.Write(data)
	return string(hash.Sum(nil))
}

func fingerprintCheckoutRollbackEmptyDirectory(info os.FileInfo) string {
	hash := sha256.New()
	_ = writeCheckoutRollbackFingerprintField(hash, []byte(info.Mode().String()))
	return string(hash.Sum(nil))
}

func writeCheckoutRollbackFingerprint(dst io.Writer, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if err := writeCheckoutRollbackFingerprintField(dst, []byte(info.Mode().String())); err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return err
		}
		return writeCheckoutRollbackFingerprintField(dst, []byte(target))
	case info.IsDir():
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := writeCheckoutRollbackFingerprintField(dst, []byte(entry.Name())); err != nil {
				return err
			}
			if err := writeCheckoutRollbackFingerprint(dst, filepath.Join(path, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	default:
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		return nil
	}
}

func writeCheckoutRollbackFingerprintField(dst io.Writer, value []byte) error {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	if _, err := dst.Write(size[:]); err != nil {
		return err
	}
	_, err := dst.Write(value)
	return err
}
