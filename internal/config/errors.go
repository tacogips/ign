package config

import "fmt"

// ConfigErrorType represents the type of configuration error.
type ConfigErrorType int

const (
	// ConfigNotFound indicates the configuration file was not found.
	ConfigNotFound ConfigErrorType = iota
	// ConfigInvalid indicates the configuration file has invalid syntax or structure.
	ConfigInvalid
	// ConfigValidationFailed indicates configuration validation failed.
	ConfigValidationFailed
)

// ConfigError represents a configuration-related error.
type ConfigError struct {
	// Type is the error type.
	Type ConfigErrorType
	// Message is the error message.
	Message string
	// File is the configuration file path.
	File string
	// Field is the configuration field that caused the error.
	Field string
	// Cause is the underlying error if any.
	Cause error
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Field != "" {
		if e.Cause != nil {
			return fmt.Sprintf("configuration error in %s [field: %s]: %s: %v", e.File, e.Field, e.Message, e.Cause)
		}
		return fmt.Sprintf("configuration error in %s [field: %s]: %s", e.File, e.Field, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("configuration error in %s: %s: %v", e.File, e.Message, e.Cause)
	}
	return fmt.Sprintf("configuration error in %s: %s", e.File, e.Message)
}

// Unwrap returns the underlying cause error.
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates a new ConfigError.
func NewConfigError(typ ConfigErrorType, file, message string) *ConfigError {
	return &ConfigError{
		Type:    typ,
		File:    file,
		Message: message,
	}
}

// NewConfigErrorWithField creates a new ConfigError with a field name.
func NewConfigErrorWithField(typ ConfigErrorType, file, field, message string) *ConfigError {
	return &ConfigError{
		Type:    typ,
		File:    file,
		Field:   field,
		Message: message,
	}
}

// NewConfigErrorWithCause creates a new ConfigError with a cause.
func NewConfigErrorWithCause(typ ConfigErrorType, file, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:    typ,
		File:    file,
		Message: message,
		Cause:   cause,
	}
}
