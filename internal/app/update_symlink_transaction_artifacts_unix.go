//go:build darwin || linux

package app

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"

	"github.com/tacogips/ign/internal/template/model"
)

type symlinkTransitionJournalArtifact struct {
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
	Data   []byte `json:"data,omitempty"`
	Mode   uint32 `json:"mode,omitempty"`
}

func (journal *symlinkTransitionJournal) captureArtifactPreimages() error {
	if len(journal.artifacts) != 0 {
		return nil
	}
	for _, name := range []string{model.IgnManifestFile, "ign.json", "ign-var.json"} {
		fd, err := unix.Openat(journal.artifactFD, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if isNotExist(err) {
			journal.artifacts = append(journal.artifacts, symlinkTransitionJournalArtifact{Name: name})
			continue
		}
		if err != nil {
			return fmt.Errorf("open %s preimage without following symlinks: %w", name, err)
		}
		var stat unix.Stat_t
		if err := unix.Fstat(fd, &stat); err != nil {
			_ = unix.Close(fd)
			return fmt.Errorf("inspect %s preimage: %w", name, err)
		}
		if stat.Mode&unix.S_IFMT != unix.S_IFREG {
			_ = unix.Close(fd)
			return fmt.Errorf("%s preimage is not a regular file", name)
		}
		file := os.NewFile(uintptr(fd), name)
		data, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return fmt.Errorf("read %s preimage: %w", name, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s preimage: %w", name, closeErr)
		}
		journal.artifacts = append(journal.artifacts, symlinkTransitionJournalArtifact{
			Name: name, Exists: true, Data: data, Mode: uint32(stat.Mode & 0777),
		})
	}
	return journal.persist()
}

func (journal *symlinkTransitionJournal) syncArtifacts() error {
	for _, artifact := range journal.artifacts {
		fd, err := unix.Openat(journal.artifactFD, artifact.Name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return fmt.Errorf("open committed %s without following symlinks: %w", artifact.Name, err)
		}
		if err := unix.Fsync(fd); err != nil {
			_ = unix.Close(fd)
			return fmt.Errorf("sync committed %s: %w", artifact.Name, err)
		}
		if err := unix.Close(fd); err != nil {
			return fmt.Errorf("close committed %s: %w", artifact.Name, err)
		}
	}
	return unix.Fsync(journal.artifactFD)
}

func (journal *symlinkTransitionJournal) restoreArtifactPreimages() error {
	for _, artifact := range journal.artifacts {
		if !artifact.Exists {
			if err := unix.Unlinkat(journal.artifactFD, artifact.Name, 0); err != nil && !isNotExist(err) {
				return fmt.Errorf("remove uncommitted %s: %w", artifact.Name, err)
			}
			continue
		}
		if err := writeJournalArtifact(journal.artifactFD, artifact.Name, artifact.Data, artifact.Mode); err != nil {
			return err
		}
	}
	return unix.Fsync(journal.artifactFD)
}

func writeJournalArtifact(journalFD int, name string, data []byte, mode uint32) error {
	tmp, err := unusedJournalTempName(journalFD)
	if err != nil {
		return err
	}
	fd, err := unix.Openat(journalFD, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, mode)
	if err != nil {
		return fmt.Errorf("create %s recovery temporary file: %w", name, err)
	}
	if err := unix.Fchmod(fd, mode); err != nil {
		_ = unix.Close(fd)
		_ = unix.Unlinkat(journalFD, tmp, 0)
		return fmt.Errorf("set %s recovery preimage mode: %w", name, err)
	}
	file := os.NewFile(uintptr(fd), tmp)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = unix.Unlinkat(journalFD, tmp, 0)
		return fmt.Errorf("write %s recovery preimage: %w", name, err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = unix.Unlinkat(journalFD, tmp, 0)
		return fmt.Errorf("sync %s recovery preimage: %w", name, err)
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(journalFD, tmp, 0)
		return fmt.Errorf("close %s recovery preimage: %w", name, err)
	}
	if err := unix.Renameat(journalFD, tmp, journalFD, name); err != nil {
		_ = unix.Unlinkat(journalFD, tmp, 0)
		return fmt.Errorf("restore %s preimage: %w", name, err)
	}
	return nil
}
