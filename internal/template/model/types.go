package model

import "os"

// VarType represents the type of a template variable.
type VarType string

const (
	// VarTypeString represents a string variable type.
	VarTypeString VarType = "string"
	// VarTypeInt represents an integer variable type.
	VarTypeInt VarType = "int"
	// VarTypeNumber represents a floating-point number variable type (parsed as float64).
	VarTypeNumber VarType = "number"
	// VarTypeBool represents a boolean variable type.
	VarTypeBool VarType = "bool"
)

// TemplateRef represents a reference to a template source.
type TemplateRef struct {
	// Provider is the provider name (e.g., "github").
	Provider string
	// Owner is the repository owner.
	Owner string
	// Repo is the repository name.
	Repo string
	// Path is the subdirectory path within the repository (optional).
	Path string
	// Ref is the branch, tag, or commit SHA.
	Ref string
}

// TemplateFile represents a single file in the template.
type TemplateFile struct {
	// Path is the relative path from template root.
	Path string
	// Content is the file content.
	Content []byte
	// Mode is the file permission mode.
	Mode os.FileMode
	// IsBinary indicates whether the file is binary (should not be template-processed).
	IsBinary bool
}
