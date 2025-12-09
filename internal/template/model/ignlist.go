package model

// IgnListJson represents the ign-list.json template list file.
type IgnListJson struct {
	// Version is the list format version (currently "1.0").
	Version string `json:"version"`
	// Name is the list name.
	Name string `json:"name,omitempty"`
	// Description is a description of the template list.
	Description string `json:"description,omitempty"`
	// Templates is the array of template definitions.
	Templates []TemplateEntry `json:"templates"`
}

// TemplateEntry represents a single template in the list.
type TemplateEntry struct {
	// Name is the unique template identifier (required).
	Name string `json:"name"`
	// URL is the repository URL (required).
	URL string `json:"url"`
	// Path is the subdirectory path within the repository.
	Path string `json:"path,omitempty"`
	// Ref is the default git ref (branch/tag).
	Ref string `json:"ref,omitempty"`
	// Description is the template description.
	Description string `json:"description,omitempty"`
	// Tags are searchable tags for the template.
	Tags []string `json:"tags,omitempty"`
	// Category is the template category.
	Category string `json:"category,omitempty"`
	// Maintainer is the maintainer name/email.
	Maintainer string `json:"maintainer,omitempty"`
}
