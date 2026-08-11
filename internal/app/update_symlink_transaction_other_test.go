//go:build !darwin && !linux

package app

import (
	"testing"

	"github.com/tacogips/ign/internal/template/generator"
)

func TestPrepareSymlinkTransitionTransactionsFailsClosedOutsideDarwinLinux(t *testing.T) {
	transactions, err := prepareSymlinkTransitionTransactions(t.TempDir(), map[string]generator.SymlinkTransition{
		"managed-directory": {
			Disposition: generator.SymlinkTransitionEligible,
			Target:      ".agents",
		},
	})
	if err == nil {
		t.Fatal("prepared eligible managed directory-to-symlink transition on an unsupported platform")
	}
	if transactions != nil {
		t.Fatalf("transactions = %#v, want nil after fail-closed rejection", transactions)
	}
}
