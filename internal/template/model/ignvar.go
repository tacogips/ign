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
	Metadata *ConfigMetadata `json:"metadata,omitempty"`
}

// ConfigMetadata contains metadata about the configuration file generation.
type ConfigMetadata struct {
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

// IgnVarJson represents the ign-var.json user variables file.
// This file contains only user-provided variable values.
type IgnVarJson struct {
	// Variables contains all user-provided variable values.
	Variables map[string]interface{} `json:"variables"`
	// Metadata contains generation metadata (auto-generated, informational).
	Metadata *VarMetadata `json:"metadata,omitempty"`
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

// VarMetadata contains metadata about the variable file generation.
type VarMetadata struct {
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
