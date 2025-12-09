package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tacogips/ign/internal/template/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test cache defaults
	if !cfg.Cache.Enabled {
		t.Error("Cache should be enabled by default")
	}
	if cfg.Cache.TTL != 3600 {
		t.Errorf("Expected TTL=3600, got %d", cfg.Cache.TTL)
	}
	if cfg.Cache.MaxSizeMB != 500 {
		t.Errorf("Expected MaxSizeMB=500, got %d", cfg.Cache.MaxSizeMB)
	}

	// Test GitHub defaults
	if cfg.GitHub.DefaultRef != "main" {
		t.Errorf("Expected DefaultRef=main, got %s", cfg.GitHub.DefaultRef)
	}
	if cfg.GitHub.APIURL != "https://api.github.com" {
		t.Errorf("Expected APIURL=https://api.github.com, got %s", cfg.GitHub.APIURL)
	}

	// Test templates defaults
	if cfg.Templates.MaxIncludeDepth != 10 {
		t.Errorf("Expected MaxIncludeDepth=10, got %d", cfg.Templates.MaxIncludeDepth)
	}
	if !cfg.Templates.PreserveExecutable {
		t.Error("PreserveExecutable should be true by default")
	}

	// Test output defaults
	if !cfg.Output.Color {
		t.Error("Color output should be enabled by default")
	}
	if !cfg.Output.Progress {
		t.Error("Progress should be enabled by default")
	}

	// Test defaults
	if cfg.Defaults.BuildDir != ".ign-build" {
		t.Errorf("Expected BuildDir=.ign-build, got %s", cfg.Defaults.BuildDir)
	}
}

func TestDefaultBinaryExtensions(t *testing.T) {
	exts := DefaultBinaryExtensions()

	if len(exts) == 0 {
		t.Fatal("DefaultBinaryExtensions returned empty list")
	}

	// Check for some common extensions
	expectedExts := []string{".png", ".jpg", ".pdf", ".zip", ".exe"}
	for _, ext := range expectedExts {
		found := false
		for _, e := range exts {
			if e == ext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected extension %s not found in defaults", ext)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	loader := NewLoader()

	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.json")

		cfg := DefaultConfig()
		cfg.Cache.TTL = 7200
		cfg.GitHub.DefaultRef = "develop"

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(cfgPath, data, 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		loadedCfg, err := loader.Load(cfgPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.Cache.TTL != 7200 {
			t.Errorf("Expected TTL=7200, got %d", loadedCfg.Cache.TTL)
		}
		if loadedCfg.GitHub.DefaultRef != "develop" {
			t.Errorf("Expected DefaultRef=develop, got %s", loadedCfg.GitHub.DefaultRef)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loader.Load("/nonexistent/config.json")
		if err == nil {
			t.Fatal("Expected error for missing file")
		}

		cfgErr, ok := err.(*ConfigError)
		if !ok {
			t.Fatalf("Expected ConfigError, got %T", err)
		}
		if cfgErr.Type != ConfigNotFound {
			t.Errorf("Expected ConfigNotFound, got %v", cfgErr.Type)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.json")

		if err := os.WriteFile(cfgPath, []byte("{ invalid json }"), 0644); err != nil {
			t.Fatalf("Failed to write invalid config: %v", err)
		}

		_, err := loader.Load(cfgPath)
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}

		cfgErr, ok := err.(*ConfigError)
		if !ok {
			t.Fatalf("Expected ConfigError, got %T", err)
		}
		if cfgErr.Type != ConfigInvalid {
			t.Errorf("Expected ConfigInvalid, got %v", cfgErr.Type)
		}
	})
}

func TestLoadOrDefault(t *testing.T) {
	loader := NewLoader()

	t.Run("returns defaults for missing file", func(t *testing.T) {
		cfg, err := loader.LoadOrDefault("/nonexistent/config.json")
		if err != nil {
			t.Fatalf("LoadOrDefault should not error on missing file: %v", err)
		}

		if cfg == nil {
			t.Fatal("Expected default config, got nil")
		}

		// Verify it's the default config
		if cfg.Cache.TTL != 3600 {
			t.Errorf("Expected default TTL=3600, got %d", cfg.Cache.TTL)
		}
	})

	t.Run("loads valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.json")

		cfg := DefaultConfig()
		cfg.Cache.TTL = 7200

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		if err := os.WriteFile(cfgPath, data, 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		loadedCfg, err := loader.LoadOrDefault(cfgPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedCfg.Cache.TTL != 7200 {
			t.Errorf("Expected TTL=7200, got %d", loadedCfg.Cache.TTL)
		}
	})
}

func TestValidateConfig(t *testing.T) {
	loader := NewLoader()

	t.Run("valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		if err := loader.Validate(cfg); err != nil {
			t.Errorf("Valid config should pass validation: %v", err)
		}
	})

	t.Run("negative TTL", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Cache.TTL = -1
		if err := loader.Validate(cfg); err == nil {
			t.Error("Expected validation error for negative TTL")
		}
	})

	t.Run("negative max size", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Cache.MaxSizeMB = -1
		if err := loader.Validate(cfg); err == nil {
			t.Error("Expected validation error for negative max size")
		}
	})

	t.Run("negative timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.GitHub.Timeout = -1
		if err := loader.Validate(cfg); err == nil {
			t.Error("Expected validation error for negative timeout")
		}
	})

	t.Run("invalid include depth", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Templates.MaxIncludeDepth = 0
		if err := loader.Validate(cfg); err == nil {
			t.Error("Expected validation error for MaxIncludeDepth=0")
		}
	})
}

func TestLoadIgnJson(t *testing.T) {
	t.Run("valid ign.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		ignPath := filepath.Join(tmpDir, "ign.json")

		ign := &model.IgnJson{
			Name:        "test-template",
			Version:     "1.0.0",
			Description: "Test template",
			Variables: map[string]model.VarDef{
				"project_name": {
					Type:        model.VarTypeString,
					Description: "Project name",
					Required:    true,
				},
			},
		}

		data, err := json.MarshalIndent(ign, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal ign.json: %v", err)
		}

		if err := os.WriteFile(ignPath, data, 0644); err != nil {
			t.Fatalf("Failed to write ign.json: %v", err)
		}

		loaded, err := LoadIgnJson(ignPath)
		if err != nil {
			t.Fatalf("Failed to load ign.json: %v", err)
		}

		if loaded.Name != "test-template" {
			t.Errorf("Expected name=test-template, got %s", loaded.Name)
		}
		if loaded.Version != "1.0.0" {
			t.Errorf("Expected version=1.0.0, got %s", loaded.Version)
		}
	})

	t.Run("missing ign.json", func(t *testing.T) {
		_, err := LoadIgnJson("/nonexistent/ign.json")
		if err == nil {
			t.Fatal("Expected error for missing ign.json")
		}

		cfgErr, ok := err.(*ConfigError)
		if !ok {
			t.Fatalf("Expected ConfigError, got %T", err)
		}
		if cfgErr.Type != ConfigNotFound {
			t.Errorf("Expected ConfigNotFound, got %v", cfgErr.Type)
		}
	})
}

func TestLoadIgnVarJson(t *testing.T) {
	t.Run("valid ign-var.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		varPath := filepath.Join(tmpDir, "ign-var.json")

		ignVar := &model.IgnVarJson{
			Template: model.TemplateSource{
				URL: "github.com/owner/repo",
				Ref: "v1.0.0",
			},
			Variables: map[string]interface{}{
				"project_name": "my-project",
				"port":         8080,
				"enable_tls":   true,
			},
		}

		data, err := json.MarshalIndent(ignVar, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal ign-var.json: %v", err)
		}

		if err := os.WriteFile(varPath, data, 0644); err != nil {
			t.Fatalf("Failed to write ign-var.json: %v", err)
		}

		loaded, err := LoadIgnVarJson(varPath)
		if err != nil {
			t.Fatalf("Failed to load ign-var.json: %v", err)
		}

		if loaded.Template.URL != "github.com/owner/repo" {
			t.Errorf("Expected URL=github.com/owner/repo, got %s", loaded.Template.URL)
		}
		if loaded.Variables["project_name"] != "my-project" {
			t.Errorf("Expected project_name=my-project, got %v", loaded.Variables["project_name"])
		}
	})

	t.Run("missing ign-var.json", func(t *testing.T) {
		_, err := LoadIgnVarJson("/nonexistent/ign-var.json")
		if err == nil {
			t.Fatal("Expected error for missing ign-var.json")
		}

		cfgErr, ok := err.(*ConfigError)
		if !ok {
			t.Fatalf("Expected ConfigError, got %T", err)
		}
		if cfgErr.Type != ConfigNotFound {
			t.Errorf("Expected ConfigNotFound, got %v", cfgErr.Type)
		}
	})
}

func TestSaveIgnVarJson(t *testing.T) {
	tmpDir := t.TempDir()
	varPath := filepath.Join(tmpDir, "build", "ign-var.json")

	ignVar := &model.IgnVarJson{
		Template: model.TemplateSource{
			URL: "github.com/owner/repo",
		},
		Variables: map[string]interface{}{
			"name": "test",
		},
	}

	if err := SaveIgnVarJson(varPath, ignVar); err != nil {
		t.Fatalf("Failed to save ign-var.json: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(varPath); os.IsNotExist(err) {
		t.Fatal("ign-var.json was not created")
	}

	// Load and verify content
	loaded, err := LoadIgnVarJson(varPath)
	if err != nil {
		t.Fatalf("Failed to load saved ign-var.json: %v", err)
	}

	if loaded.Template.URL != ignVar.Template.URL {
		t.Errorf("URL mismatch after save/load")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"empty path", "", false},
		{"absolute path", "/tmp/test", false},
		{"relative path", "./test", false},
		{"home directory", "~", false},
		{"home subdirectory", "~/test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, err := ExpandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.path != "" && !tt.wantErr && expanded == "" {
				t.Errorf("ExpandPath() returned empty string for non-empty path")
			}
		})
	}
}
