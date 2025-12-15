package config

import (
	"os"
	"path/filepath"
)

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "ign")

	return &Config{
		Cache: CacheConfig{
			Enabled:   true,
			Directory: cacheDir,
			TTL:       3600, // 1 hour
			MaxSizeMB: 500,
			AutoClean: true,
		},
		GitHub: GitHubConfig{
			Token:      "",
			DefaultRef: "main",
			APIURL:     "https://api.github.com",
			Timeout:    30,
		},
		Templates: TemplateConfig{
			MaxIncludeDepth:    10,
			PreserveExecutable: true,
			IgnorePatterns:     DefaultIgnorePatterns(),
			BinaryExtensions:   DefaultBinaryExtensions(),
		},
		Output: OutputConfig{
			Color:    true,
			Progress: true,
			Verbose:  false,
			Quiet:    false,
		},
		Defaults: DefaultsConfig{
			BuildDir:  ".ign-config",
			OutputDir: ".",
		},
	}
}

// DefaultIgnorePatterns returns the default ignore patterns.
func DefaultIgnorePatterns() []string {
	return []string{
		".DS_Store",
		"Thumbs.db",
		"*.swp",
		"*.swo",
		"*~",
	}
}

// DefaultBinaryExtensions returns the default binary file extensions.
func DefaultBinaryExtensions() []string {
	return []string{
		// Images
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".svg",
		// Documents
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		// Archives
		".zip", ".tar", ".gz", ".bz2", ".7z", ".rar",
		// Executables and libraries
		".exe", ".dll", ".so", ".dylib", ".a",
		// Fonts
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		// Media
		".mp3", ".mp4", ".avi", ".mov", ".wav",
		// Databases
		".db", ".sqlite", ".sqlite3",
	}
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "ign", "config.json")
}
