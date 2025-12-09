package model

// IgnJson represents the ign.json template configuration file.
type IgnJson struct {
	// Name is the template identifier (required).
	Name string `json:"name"`
	// Version is the template version using semantic versioning (required).
	Version string `json:"version"`
	// Description is a human-readable description of the template.
	Description string `json:"description,omitempty"`
	// Author is the author name and email.
	Author string `json:"author,omitempty"`
	// Repository is the source repository URL.
	Repository string `json:"repository,omitempty"`
	// License is the license identifier (e.g., "MIT", "Apache-2.0").
	License string `json:"license,omitempty"`
	// Tags are searchable tags for the template.
	Tags []string `json:"tags,omitempty"`
	// Variables defines all template variables.
	Variables map[string]VarDef `json:"variables"`
	// Settings contains template-specific settings.
	Settings *TemplateSettings `json:"settings,omitempty"`
}

// VarDef defines a template variable with validation rules.
type VarDef struct {
	// Type is the variable type (string, int, or bool).
	Type VarType `json:"type"`
	// Description is a human-readable description of the variable.
	Description string `json:"description"`
	// Required indicates if the variable must have a value.
	Required bool `json:"required,omitempty"`
	// Default is the default value if not provided (type must match).
	Default interface{} `json:"default,omitempty"`
	// Example is an example value for documentation.
	Example interface{} `json:"example,omitempty"`
	// Pattern is a regex validation pattern (for string variables only).
	Pattern string `json:"pattern,omitempty"`
	// Min is the minimum value (for integer variables only).
	Min *int `json:"min,omitempty"`
	// Max is the maximum value (for integer variables only).
	Max *int `json:"max,omitempty"`
}

// TemplateSettings contains template-specific settings for generation.
type TemplateSettings struct {
	// PreserveExecutable preserves the executable bit from template files.
	PreserveExecutable bool `json:"preserve_executable,omitempty"`
	// IgnorePatterns are glob patterns for files to ignore during generation.
	IgnorePatterns []string `json:"ignore_patterns,omitempty"`
	// BinaryExtensions are file extensions to copy without template processing.
	BinaryExtensions []string `json:"binary_extensions,omitempty"`
	// IncludeDotfiles includes hidden files (starting with '.') in generation.
	IncludeDotfiles bool `json:"include_dotfiles,omitempty"`
	// MaxIncludeDepth is the maximum nested include depth for @ign-include directives.
	MaxIncludeDepth int `json:"max_include_depth,omitempty"`
}
