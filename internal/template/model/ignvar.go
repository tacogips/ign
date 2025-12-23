package model

import "time"

// IgnConfig represents the ign.json configuration file.
// This file contains template source information and the downloaded template hash.
type IgnConfig struct {
	// Template identifies the template source.
	Template TemplateSource `json:"template"`
	// Hash is the SHA256 hash of the downloaded template content.
	Hash string `json:"hash"`
	// Metadata contains configuration metadata (auto-generated, informational).
	Metadata *FileMetadata `json:"metadata,omitempty"`
}

// IgnVarJson represents the ign-var.json user variables file.
// This file stores user-provided variable values used during template generation.
// It is separate from ign.json to allow updating template source independently from variables.
type IgnVarJson struct {
	// Variables contains all user-provided variable values mapped by variable name.
	Variables map[string]interface{} `json:"variables"`
	// Metadata contains generation metadata (auto-generated, informational).
	Metadata *FileMetadata `json:"metadata,omitempty"`
}

// FileMetadata contains metadata about configuration file generation.
// This is used by both IgnConfig (ign.json) and IgnVarJson (ign-var.json) to track
// when and how the files were generated.
type FileMetadata struct {
	// GeneratedAt is when the file was generated.
	GeneratedAt time.Time `json:"generated_at,omitempty"`
	// GeneratedBy is the tool/command that generated the file.
	GeneratedBy string `json:"generated_by,omitempty"`
	// TemplateName is the name of the template.
	TemplateName string `json:"template_name,omitempty"`
	// TemplateVersion is the version of the template.
	TemplateVersion string `json:"template_version,omitempty"`
	// IgnVersion is the version of ign that generated the file.
	IgnVersion string `json:"ign_version,omitempty"`
}

// TemplateSource identifies the template location.
type TemplateSource struct {
	// URL is the template repository URL (required).
	URL string `json:"url"`
	// Path is the subdirectory path within the repository.
	Path string `json:"path,omitempty"`
	// Ref is the git branch, tag, or commit SHA.
	Ref string `json:"ref,omitempty"`
}
