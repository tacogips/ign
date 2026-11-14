package provider

import (
	"context"

	"github.com/tacogips/ign/internal/template/model"
)

// Provider abstracts template source locations (GitHub, local filesystem, etc.).
type Provider interface {
	// Fetch downloads a template from the provider.
	// Returns a Template with parsed ign.json and all template files.
	Fetch(ctx context.Context, ref model.TemplateRef) (*model.Template, error)

	// Validate checks if a template reference is valid and accessible.
	// Returns an error if the template cannot be accessed.
	Validate(ctx context.Context, ref model.TemplateRef) error

	// Resolve converts a URL string to a TemplateRef.
	// The URL format depends on the provider (e.g., GitHub URL, local path).
	Resolve(url string) (model.TemplateRef, error)

	// Name returns the provider name (e.g., "github", "local").
	Name() string
}
