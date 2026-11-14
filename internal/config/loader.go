package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

// Loader defines the interface for loading configuration files.
type Loader interface {
	// Load loads configuration from the specified file path.
	Load(path string) (*Config, error)
	// LoadOrDefault loads configuration or returns defaults if file doesn't exist.
	LoadOrDefault(path string) (*Config, error)
	// Validate validates the configuration.
	Validate(config *Config) error
}

// FileLoader implements the Loader interface for file-based configuration loading.
type FileLoader struct{}

// NewLoader creates a new FileLoader instance.
func NewLoader() Loader {
	return &FileLoader{}
}

// Load loads configuration from the specified file path.
func (l *FileLoader) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewConfigErrorWithCause(ConfigNotFound, path, "configuration file not found", err)
		}
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "failed to read configuration file", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "invalid JSON syntax", err)
	}

	// Merge with defaults for any missing fields
	defaultCfg := DefaultConfig()
	mergeConfig(&cfg, defaultCfg)

	return &cfg, nil
}

// LoadOrDefault loads configuration or returns defaults if file doesn't exist.
func (l *FileLoader) LoadOrDefault(path string) (*Config, error) {
	cfg, err := l.Load(path)
	if err != nil {
		// If file not found, return defaults
		if cfgErr, ok := err.(*ConfigError); ok && cfgErr.Type == ConfigNotFound {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// Validate validates the configuration.
func (l *FileLoader) Validate(config *Config) error {
	if config.Cache.TTL < 0 {
		return NewConfigErrorWithField(ConfigValidationFailed, "", "cache.ttl", "TTL cannot be negative")
	}
	if config.Cache.MaxSizeMB < 0 {
		return NewConfigErrorWithField(ConfigValidationFailed, "", "cache.max_size_mb", "max size cannot be negative")
	}
	if config.GitHub.Timeout < 0 {
		return NewConfigErrorWithField(ConfigValidationFailed, "", "github.timeout", "timeout cannot be negative")
	}
	if config.Templates.MaxIncludeDepth < 1 {
		return NewConfigErrorWithField(ConfigValidationFailed, "", "templates.max_include_depth", "max include depth must be at least 1")
	}
	return nil
}

// LoadIgnVarJson loads ign-var.json from the specified path.
func LoadIgnVarJson(path string) (*model.IgnVarJson, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewConfigErrorWithCause(ConfigNotFound, path, "ign-var.json not found", err)
		}
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "failed to read ign-var.json", err)
	}

	var ignVar model.IgnVarJson
	if err := json.Unmarshal(data, &ignVar); err != nil {
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "invalid JSON syntax in ign-var.json", err)
	}

	return &ignVar, nil
}

// LoadIgnJson loads ign.json template metadata from the specified path.
// This function reads the template's ign.json file which contains template information
// (name, version, variable definitions). This is DIFFERENT from LoadIgnConfig which
// loads the user's project configuration (.ign/ign.json).
//
// Use cases:
//   - Loading template metadata from a template repository
//   - Reading template variable definitions during template collection
//
// NOT for loading user project configuration - use LoadIgnConfig instead.
func LoadIgnJson(path string) (*model.IgnJson, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewConfigErrorWithCause(ConfigNotFound, path, "ign.json not found", err)
		}
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "failed to read ign.json", err)
	}

	var ign model.IgnJson
	if err := json.Unmarshal(data, &ign); err != nil {
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "invalid JSON syntax in ign.json", err)
	}

	return &ign, nil
}

// SaveIgnVarJson saves ign-var.json to the specified path.
// Security: The path is validated to prevent path traversal attacks.
func SaveIgnVarJson(path string, ignVar *model.IgnVarJson) error {
	// Security: Validate path doesn't contain path traversal sequences
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return NewConfigErrorWithCause(ConfigInvalid, path,
			"path contains '..' which is not allowed for security reasons", nil)
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath,
			fmt.Sprintf("failed to create directory %s", dir), err)
	}

	data, err := json.MarshalIndent(ignVar, "", "  ")
	if err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to marshal ign-var.json", err)
	}

	if err := os.WriteFile(cleanPath, data, 0644); err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to write ign-var.json", err)
	}

	return nil
}

// LoadIgnConfig loads user project configuration (ign.json) from the specified path.
// This function reads the user's project configuration file (.ign/ign.json) which contains
// template source information and template hash. This is DIFFERENT from LoadIgnJson which
// loads template metadata from the template repository.
//
// Use cases:
//   - Loading user project configuration during checkout operations
//   - Reading template source and hash information from .ign directory
//
// NOT for loading template metadata - use LoadIgnJson instead.
//
// The loaded configuration is validated to ensure required fields are present and hash format is valid.
func LoadIgnConfig(path string) (*model.IgnConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewConfigErrorWithCause(ConfigNotFound, path, "ign.json not found", err)
		}
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "failed to read ign.json", err)
	}

	var ignConfig model.IgnConfig
	if err := json.Unmarshal(data, &ignConfig); err != nil {
		return nil, NewConfigErrorWithCause(ConfigInvalid, path, "invalid JSON syntax in ign.json", err)
	}

	// Validate required fields and format
	if ignConfig.Template.URL == "" {
		return nil, NewConfigErrorWithField(ConfigValidationFailed, path, "template.url", "template URL is required")
	}

	// Validate hash format if present (64 hex characters for SHA256)
	if ignConfig.Hash != "" && !isValidSHA256Hash(ignConfig.Hash) {
		return nil, NewConfigErrorWithField(ConfigValidationFailed, path, "hash",
			"hash must be a valid SHA256 string (64 hexadecimal characters)")
	}

	return &ignConfig, nil
}

// isValidSHA256Hash validates that a string is a valid SHA256 hash (64 hex characters).
func isValidSHA256Hash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// SaveIgnConfig saves ign.json to the specified path.
// Security: The path is validated to prevent path traversal attacks.
func SaveIgnConfig(path string, ignConfig *model.IgnConfig) error {
	// Security: Validate path doesn't contain path traversal sequences
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return NewConfigErrorWithCause(ConfigInvalid, path,
			"path contains '..' which is not allowed for security reasons", nil)
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath,
			fmt.Sprintf("failed to create directory %s", dir), err)
	}

	data, err := json.MarshalIndent(ignConfig, "", "  ")
	if err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to marshal ign.json", err)
	}

	if err := os.WriteFile(cleanPath, data, 0644); err != nil {
		return NewConfigErrorWithCause(ConfigInvalid, cleanPath, "failed to write ign.json", err)
	}

	return nil
}

// mergeConfig merges missing fields from defaults into cfg.
func mergeConfig(cfg, defaults *Config) {
	// Cache
	if cfg.Cache.Directory == "" {
		cfg.Cache.Directory = defaults.Cache.Directory
	}
	if cfg.Cache.TTL == 0 {
		cfg.Cache.TTL = defaults.Cache.TTL
	}
	if cfg.Cache.MaxSizeMB == 0 {
		cfg.Cache.MaxSizeMB = defaults.Cache.MaxSizeMB
	}

	// GitHub
	if cfg.GitHub.DefaultRef == "" {
		cfg.GitHub.DefaultRef = defaults.GitHub.DefaultRef
	}
	if cfg.GitHub.APIURL == "" {
		cfg.GitHub.APIURL = defaults.GitHub.APIURL
	}
	if cfg.GitHub.Timeout == 0 {
		cfg.GitHub.Timeout = defaults.GitHub.Timeout
	}

	// Templates
	if cfg.Templates.MaxIncludeDepth == 0 {
		cfg.Templates.MaxIncludeDepth = defaults.Templates.MaxIncludeDepth
	}
	if len(cfg.Templates.IgnorePatterns) == 0 {
		cfg.Templates.IgnorePatterns = defaults.Templates.IgnorePatterns
	}
	if len(cfg.Templates.BinaryExtensions) == 0 {
		cfg.Templates.BinaryExtensions = defaults.Templates.BinaryExtensions
	}

	// Defaults
	if cfg.Defaults.BuildDir == "" {
		cfg.Defaults.BuildDir = defaults.Defaults.BuildDir
	}
	if cfg.Defaults.OutputDir == "" {
		cfg.Defaults.OutputDir = defaults.Defaults.OutputDir
	}
}

// ExpandPath expands ~ to home directory and evaluates relative paths.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if len(path) == 1 {
			return homeDir, nil
		}
		if path[1] == filepath.Separator {
			return filepath.Join(homeDir, path[2:]), nil
		}
	}

	// Make absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}
