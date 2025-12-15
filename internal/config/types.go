package config

// Config represents the global ign configuration.
type Config struct {
	// Cache configuration for template caching.
	Cache CacheConfig `json:"cache"`
	// GitHub configuration for repository access.
	GitHub GitHubConfig `json:"github"`
	// Templates configuration for template processing.
	Templates TemplateConfig `json:"templates"`
	// Output configuration for display and logging.
	Output OutputConfig `json:"output"`
	// Defaults configuration for default values.
	Defaults DefaultsConfig `json:"defaults"`
}

// CacheConfig represents cache settings.
type CacheConfig struct {
	// Enabled indicates whether template caching is enabled.
	Enabled bool `json:"enabled"`
	// Directory is the cache directory path.
	Directory string `json:"directory"`
	// TTL is the cache time-to-live in seconds (0 = no expiration).
	TTL int `json:"ttl"`
	// MaxSizeMB is the maximum cache size in megabytes.
	MaxSizeMB int `json:"max_size_mb"`
	// AutoClean indicates whether to automatically clean old cache entries.
	AutoClean bool `json:"auto_clean"`
}

// GitHubConfig represents GitHub-specific settings.
type GitHubConfig struct {
	// Token is the GitHub personal access token for private repositories.
	Token string `json:"token,omitempty"`
	// DefaultRef is the default branch/ref if not specified.
	DefaultRef string `json:"default_ref"`
	// APIURL is the GitHub API URL (for enterprise installations).
	APIURL string `json:"api_url"`
	// Timeout is the request timeout in seconds.
	Timeout int `json:"timeout"`
}

// TemplateConfig represents template processing settings.
type TemplateConfig struct {
	// MaxIncludeDepth is the maximum nested include depth.
	MaxIncludeDepth int `json:"max_include_depth"`
	// PreserveExecutable preserves executable permissions from templates.
	PreserveExecutable bool `json:"preserve_executable"`
	// IgnorePatterns are global ignore patterns (glob syntax).
	IgnorePatterns []string `json:"ignore_patterns"`
	// BinaryExtensions are file extensions to skip template processing.
	BinaryExtensions []string `json:"binary_extensions"`
}

// OutputConfig represents output and display settings.
type OutputConfig struct {
	// Color enables colored terminal output.
	Color bool `json:"color"`
	// Progress shows progress indicators during operations.
	Progress bool `json:"progress"`
	// Verbose enables verbose logging output.
	Verbose bool `json:"verbose"`
	// Quiet suppresses non-error output.
	Quiet bool `json:"quiet"`
}

// DefaultsConfig represents default values for various settings.
type DefaultsConfig struct {
	// BuildDir is the default build directory name.
	BuildDir string `json:"build_dir"`
	// OutputDir is the default output directory for ign init.
	OutputDir string `json:"output_dir"`
}
