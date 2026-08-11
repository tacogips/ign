//go:build linux

package app

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// restoreTransitionDirectoryNoReplaceAt restores a backup only when the
// destination is still absent. RENAME_NOREPLACE closes the check-to-rename
// interval without replacing a concurrently created entry.
func restoreTransitionDirectoryNoReplaceAt(parentFD int, backup, destination string) error {
	if err := renameNoReplaceAt(parentFD, backup, destination); err != nil {
		return fmt.Errorf("restore transaction backup without replacing destination: %w", err)
	}
	return nil
}

func renameNoReplaceAt(parentFD int, from, to string) error {
	return renameNoReplaceAcrossAt(parentFD, from, parentFD, to)
}

func renameNoReplaceAcrossAt(fromFD int, from string, toFD int, to string) error {
	return unix.Renameat2(fromFD, from, toFD, to, unix.RENAME_NOREPLACE)
}
