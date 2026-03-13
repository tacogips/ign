package model

// IgnManifest records files created by ign so they can be removed later.
type IgnManifest struct {
	// Files contains generated file paths as written during checkout/update.
	Files []string `json:"files"`
}
