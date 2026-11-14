package model

// Template represents a runtime template with all its data.
type Template struct {
	// Ref is the template reference (source location).
	Ref TemplateRef
	// Config is the parsed ign.json configuration.
	Config IgnJson
	// Files are all template files to be processed.
	Files []TemplateFile
	// RootPath is the local path to the template root directory.
	RootPath string
}
