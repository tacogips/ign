//go:build !darwin && !linux

package app

import (
	"fmt"

	"github.com/tacogips/ign/internal/template/generator"
)

func recoverSymlinkTransitionJournal(_, _ string, _ ...string) error {
	return nil
}

func symlinkTransitionJournalPending(_ string) (bool, error) {
	return false, nil
}

type symlinkTransitionTransactions struct{}

func prepareSymlinkTransitionTransactions(_ string, transitions map[string]generator.SymlinkTransition, _ ...string) (*symlinkTransitionTransactions, error) {
	for _, transition := range transitions {
		if transition.Disposition == generator.SymlinkTransitionEligible {
			return nil, fmt.Errorf("managed directory-to-symlink transitions are unsupported on this platform")
		}
	}
	return &symlinkTransitionTransactions{}, nil
}

func (*symlinkTransitionTransactions) rollback() error { return nil }
func (*symlinkTransitionTransactions) commit() error   { return nil }
func (*symlinkTransitionTransactions) markArtifactsCommitted() error {
	return nil
}
func transitionPathIsSafe(_, _ string) bool { return false }

func fingerprintTransitionSource(_, _ string) (string, error) {
	return "", fmt.Errorf("managed directory-to-symlink transitions are unsupported on this platform")
}
