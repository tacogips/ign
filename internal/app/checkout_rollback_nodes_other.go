//go:build !darwin && !linux

package app

import (
	"fmt"
	"os"
)

func checkoutAtomicRollbackSupported() bool { return false }

// Non-Unix targets deliberately fail closed during rollback rather than using
// check-then-remove or truncating replacement paths.
func archiveExpectedCheckoutNode(_ string, _ os.FileInfo, _ string, _ string) error {
	return fmt.Errorf("atomic checkout rollback is unsupported on this platform")
}

func restoreCheckoutRollbackEntryAtomically(entry checkoutRollbackEntry, _ string) error {
	if !entry.existed && entry.expectedInfo == nil {
		return fmt.Errorf("atomic checkout rollback is unsupported on this platform")
	}
	return fmt.Errorf("atomic checkout rollback is unsupported on this platform")
}
