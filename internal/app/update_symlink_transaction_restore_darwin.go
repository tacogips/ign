//go:build darwin

package app

import (
	"fmt"
)

// restoreTransitionDirectoryNoReplaceAt uses the opened parent descriptor so
// RenamexNp resolves both names in the verified directory. RENAME_EXCL makes
// restoration fail rather than replace an entry created concurrently.
func restoreTransitionDirectoryNoReplaceAt(parentFD int, backup, destination string) error {
	if err := renameNoReplaceAt(parentFD, backup, destination); err != nil {
		return fmt.Errorf("restore transaction backup without replacing destination: %w", err)
	}
	return nil
}
