package app

import "fmt"

// AppErrorType represents the type of application error.
type AppErrorType int

const (
	// BuildInitFailed indicates build initialization failed.
	BuildInitFailed AppErrorType = iota
	// InitFailed indicates project initialization failed.
	InitFailed
	// VariableLoadFailed indicates variable loading failed.
	VariableLoadFailed
	// TemplateFetchFailed indicates template fetching failed.
	TemplateFetchFailed
	// ValidationFailed indicates validation failed.
	ValidationFailed
)

// AppError represents an application-layer error.
type AppError struct {
	// Type is the error type.
	Type AppErrorType
	// Message is the error message.
	Message string
	// Cause is the underlying error.
	Cause error
}

// Error returns the error message.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewAppError creates a new AppError.
func NewAppError(errType AppErrorType, message string, cause error) *AppError {
	return &AppError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// NewBuildInitError creates a build init error.
func NewBuildInitError(message string, cause error) *AppError {
	return NewAppError(BuildInitFailed, message, cause)
}

// NewInitError creates an init error.
func NewInitError(message string, cause error) *AppError {
	return NewAppError(InitFailed, message, cause)
}

// NewVariableLoadError creates a variable load error.
func NewVariableLoadError(message string, cause error) *AppError {
	return NewAppError(VariableLoadFailed, message, cause)
}

// NewTemplateFetchError creates a template fetch error.
func NewTemplateFetchError(message string, cause error) *AppError {
	return NewAppError(TemplateFetchFailed, message, cause)
}

// NewValidationError creates a validation error.
func NewValidationError(message string, cause error) *AppError {
	return NewAppError(ValidationFailed, message, cause)
}
