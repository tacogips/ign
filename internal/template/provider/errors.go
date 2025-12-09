package provider

import "fmt"

// ProviderErrorType represents the type of provider error.
type ProviderErrorType int

const (
	// ProviderFetchFailed indicates the template could not be fetched.
	ProviderFetchFailed ProviderErrorType = iota
	// ProviderNotFound indicates the template was not found at the source.
	ProviderNotFound
	// ProviderAuthFailed indicates authentication failed (e.g., private repo).
	ProviderAuthFailed
	// ProviderTimeout indicates the operation timed out.
	ProviderTimeout
	// ProviderInvalidURL indicates the URL format is invalid.
	ProviderInvalidURL
	// ProviderInvalidTemplate indicates the template structure is invalid.
	ProviderInvalidTemplate
)

// String returns the string representation of the error type.
func (t ProviderErrorType) String() string {
	switch t {
	case ProviderFetchFailed:
		return "FetchFailed"
	case ProviderNotFound:
		return "NotFound"
	case ProviderAuthFailed:
		return "AuthFailed"
	case ProviderTimeout:
		return "Timeout"
	case ProviderInvalidURL:
		return "InvalidURL"
	case ProviderInvalidTemplate:
		return "InvalidTemplate"
	default:
		return "Unknown"
	}
}

// ProviderError represents a provider-specific error.
type ProviderError struct {
	// Type is the error type classification.
	Type ProviderErrorType
	// Message is the human-readable error message.
	Message string
	// Provider is the provider name (e.g., "github", "local").
	Provider string
	// URL is the template URL that caused the error.
	URL string
	// Cause is the underlying error, if any.
	Cause error
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s provider error [%s] for URL '%s': %s (caused by: %v)",
			e.Provider, e.Type.String(), e.URL, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s provider error [%s] for URL '%s': %s",
		e.Provider, e.Type.String(), e.URL, e.Message)
}

// Unwrap returns the underlying cause for error wrapping.
func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// NewProviderError creates a new ProviderError.
func NewProviderError(typ ProviderErrorType, provider, url, message string, cause error) *ProviderError {
	return &ProviderError{
		Type:     typ,
		Message:  message,
		Provider: provider,
		URL:      url,
		Cause:    cause,
	}
}

// NewFetchError creates a fetch failed error.
func NewFetchError(provider, url string, cause error) *ProviderError {
	return NewProviderError(ProviderFetchFailed, provider, url, "failed to fetch template", cause)
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(provider, url string) *ProviderError {
	return NewProviderError(ProviderNotFound, provider, url, "template not found", nil)
}

// NewAuthError creates an authentication failed error.
func NewAuthError(provider, url string) *ProviderError {
	return NewProviderError(ProviderAuthFailed, provider, url, "authentication failed (private repository?)", nil)
}

// NewTimeoutError creates a timeout error.
func NewTimeoutError(provider, url string) *ProviderError {
	return NewProviderError(ProviderTimeout, provider, url, "operation timed out", nil)
}

// NewInvalidURLError creates an invalid URL error.
func NewInvalidURLError(provider, url string, cause error) *ProviderError {
	return NewProviderError(ProviderInvalidURL, provider, url, "invalid URL format", cause)
}

// NewInvalidTemplateError creates an invalid template error.
func NewInvalidTemplateError(provider, url, message string, cause error) *ProviderError {
	return NewProviderError(ProviderInvalidTemplate, provider, url, message, cause)
}
