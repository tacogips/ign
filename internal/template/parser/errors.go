package parser

import "fmt"

// ParseErrorType represents the type of parsing error.
type ParseErrorType int

const (
	// UnknownDirective indicates an unrecognized @ign-* directive.
	UnknownDirective ParseErrorType = iota
	// MissingVariable indicates a variable reference that doesn't exist.
	MissingVariable
	// TypeMismatch indicates a type conversion error (e.g., non-bool in @ign-if:).
	TypeMismatch
	// UnclosedBlock indicates a missing @ign-endif@ for @ign-if:.
	UnclosedBlock
	// CircularInclude indicates a circular dependency in @ign-include: directives.
	CircularInclude
	// MaxIncludeDepth indicates the maximum include depth was exceeded.
	MaxIncludeDepth
	// IncludeNotFound indicates an include file doesn't exist.
	IncludeNotFound
	// InvalidDirectiveSyntax indicates malformed directive syntax.
	InvalidDirectiveSyntax
	// SecurityViolation indicates a security policy violation in filename variable values.
	SecurityViolation
)

// ParseError represents a template parsing error with detailed context.
type ParseError struct {
	// Type is the error type.
	Type ParseErrorType
	// Message is the error message.
	Message string
	// File is the file path where the error occurred.
	File string
	// Line is the line number where the error occurred (1-indexed, 0 if unknown).
	Line int
	// Directive is the problematic directive text.
	Directive string
	// Cause is the underlying error (optional).
	Cause error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s (directive: %s)", e.File, e.Line, e.Message, e.Directive)
	}
	if e.File != "" {
		return fmt.Sprintf("%s: %s (directive: %s)", e.File, e.Message, e.Directive)
	}
	if e.Directive != "" {
		return fmt.Sprintf("%s (directive: %s)", e.Message, e.Directive)
	}
	return e.Message
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *ParseError) Unwrap() error {
	return e.Cause
}

// newParseError creates a new ParseError with the given type and message.
func newParseError(typ ParseErrorType, message string) *ParseError {
	return &ParseError{
		Type:    typ,
		Message: message,
	}
}

// newParseErrorWithDirective creates a ParseError with directive context.
func newParseErrorWithDirective(typ ParseErrorType, message, directive string) *ParseError {
	return &ParseError{
		Type:      typ,
		Message:   message,
		Directive: directive,
	}
}

// newParseErrorWithFile creates a ParseError with file context.
func newParseErrorWithFile(typ ParseErrorType, message, file string, line int) *ParseError {
	return &ParseError{
		Type:    typ,
		Message: message,
		File:    file,
		Line:    line,
	}
}
