package model

import "time"

const (
	// IgnProjectMetadataVersionV1 is the legacy single-template project metadata version.
	IgnProjectMetadataVersionV1 = 1
	// IgnProjectMetadataVersionV2 is the multi-template project metadata version.
	IgnProjectMetadataVersionV2 = 2
)

// IgnConfig represents the ign.json configuration file.
// This file contains template source information and the downloaded template hash.
type IgnConfig struct {
	// Version is the project metadata schema version. Missing means version 1.
	Version int `json:"version,omitempty"`
	// OutputRoot is the project output root for version 2 metadata.
	OutputRoot string `json:"output_root,omitempty"`
	// Template identifies the template source.
	Template TemplateSource `json:"template,omitempty"`
	// Hash is the SHA256 hash of the downloaded template content.
	Hash string `json:"hash,omitempty"`
	// Metadata contains configuration metadata (auto-generated, informational).
	Metadata *FileMetadata `json:"metadata,omitempty"`
	// Templates contains ordered template checkout records for version 2 metadata.
	Templates []TemplateCheckoutRecord `json:"templates,omitempty"`
}

// IgnVarJson represents the ign-var.json user variables file.
// This file stores user-provided variable values used during template generation.
// It is separate from ign.json to allow updating template source independently from variables.
// Note: Metadata is intentionally not included here as it duplicates ign.json metadata.
type IgnVarJson struct {
	// Version is the project metadata schema version. Missing means version 1.
	Version int `json:"version,omitempty"`
	// OutputRoot is the project output root for version 2 metadata.
	OutputRoot string `json:"output_root,omitempty"`
	// Variables contains all user-provided variable values mapped by variable name.
	Variables map[string]interface{} `json:"variables,omitempty"`
	// Templates contains per-template variable records for version 2 metadata.
	Templates []IgnVarTemplateRecord `json:"templates,omitempty"`
}

// TemplateCheckoutRecord identifies one checked-out template in version 2 metadata.
type TemplateCheckoutRecord struct {
	// ID is the stable user-facing checkout record identifier.
	ID string `json:"id"`
	// Order is the monotonic checkout order. Later orders take precedence.
	Order int `json:"order"`
	// Template identifies the template source.
	Template TemplateSource `json:"template"`
	// Hash is the SHA256 hash of the downloaded template content.
	Hash string `json:"hash"`
	// Metadata contains configuration metadata for this checkout record.
	Metadata *FileMetadata `json:"metadata,omitempty"`
}

// IgnVarTemplateRecord stores variables for one template checkout record.
type IgnVarTemplateRecord struct {
	// ID references a TemplateCheckoutRecord.
	ID string `json:"id"`
	// Variables contains user-provided variable values for this template.
	Variables map[string]interface{} `json:"variables"`
}

// FileMetadata contains metadata about configuration file generation.
// This is used by IgnConfig (ign.json) to track when and how the file was generated.
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
