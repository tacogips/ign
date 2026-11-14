//go:build !darwin && !linux

package app

import (
	"context"
	"testing"

	"github.com/tacogips/ign/internal/template/generator"
)

type noOpRollbackGenerator struct{}

func (noOpRollbackGenerator) Generate(context.Context, generator.GenerateOptions) (*generator.GenerateResult, error) {
	return &generator.GenerateResult{}, nil
}

func (noOpRollbackGenerator) DryRun(context.Context, generator.GenerateOptions) (*generator.GenerateResult, error) {
	return &generator.GenerateResult{}, nil
}

func TestPrepareCheckoutGenerationRollbackAllowsOrdinaryGenerationWithoutAtomicSupport(t *testing.T) {
	rollback, err := prepareCheckoutGenerationRollback(context.Background(), noOpRollbackGenerator{}, generator.GenerateOptions{})
	if err != nil {
		t.Fatalf("prepareCheckoutGenerationRollback error = %v", err)
	}
	if rollback == nil {
		t.Fatal("prepareCheckoutGenerationRollback returned nil rollback")
	}
}
