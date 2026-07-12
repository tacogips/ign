package app

import (
	"context"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
	"github.com/tacogips/ign/internal/template/provider"
)

type trackedTemplateFetchOptions struct {
	Source      model.TemplateSource
	GitHubToken string
}

type trackedTemplateFetchResult struct {
	Template      *model.Template
	TemplateRef   model.TemplateRef
	NormalizedURL string
}

func fetchTrackedTemplate(ctx context.Context, opts trackedTemplateFetchOptions) (*trackedTemplateFetchResult, error) {
	normalizedURL := NormalizeTemplateURL(opts.Source.URL)
	debug.DebugValue("[app] Normalized template URL", normalizedURL)

	prov, err := provider.NewProviderWithToken(normalizedURL, opts.GitHubToken)
	if err != nil {
		debug.Debug("[app] Failed to create provider: %v", err)
		return nil, NewCheckoutError("failed to create provider", err)
	}

	templateRef, err := prov.Resolve(normalizedURL)
	if err != nil {
		debug.Debug("[app] Failed to resolve template URL: %v", err)
		return nil, NewCheckoutError("failed to resolve template URL", err)
	}

	if opts.Source.Ref != "" {
		templateRef.Ref = opts.Source.Ref
	}
	if opts.Source.Path != "" {
		templateRef.Path = opts.Source.Path
	}

	template, err := prov.Fetch(ctx, templateRef)
	if err != nil {
		debug.Debug("[app] Failed to fetch template: %v", err)
		return nil, NewTemplateFetchError("failed to fetch template", err)
	}

	return &trackedTemplateFetchResult{
		Template:      template,
		TemplateRef:   templateRef,
		NormalizedURL: normalizedURL,
	}, nil
}
