package generator

import "fmt"

// GeneratorErrorType categorizes generator errors.
type GeneratorErrorType int

const (
	// GeneratorWriteFailed indicates a file write operation failed.
	GeneratorWriteFailed GeneratorErrorType = iota
	// GeneratorProcessFailed indicates template processing failed.
	GeneratorProcessFailed
	// GeneratorPathError indicates an invalid or unsafe path was encountered.
	GeneratorPathError
)

// GeneratorError represents generator-specific errors.
type GeneratorError struct {
	// Type categorizes the error.
	Type GeneratorErrorType
	// Message is the error message.
	Message string
	// File is the file path related to the error (if applicable).
	File string
	// Cause is the underlying error (if any).
	Cause error
}

// Error implements the error interface.
func (e *GeneratorError) Error() string {
	if e.File != "" {
		if e.Cause != nil {
			return fmt.Sprintf("%s (file: %s): %v", e.Message, e.File, e.Cause)
		}
		return fmt.Sprintf("%s (file: %s)", e.Message, e.File)
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}

	return e.Message
}

// Unwrap returns the underlying cause error for error unwrapping.
func (e *GeneratorError) Unwrap() error {
	return e.Cause
}

// newGeneratorError creates a new GeneratorError.
func newGeneratorError(typ GeneratorErrorType, message, file string, cause error) *GeneratorError {
	return &GeneratorError{
		Type:    typ,
		Message: message,
		File:    file,
		Cause:   cause,
	}
}
