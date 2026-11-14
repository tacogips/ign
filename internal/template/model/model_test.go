package model

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestVarType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		varType  VarType
		expected string
	}{
		{"string type", VarTypeString, "string"},
		{"int type", VarTypeInt, "int"},
		{"number type", VarTypeNumber, "number"},
		{"bool type", VarTypeBool, "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.varType) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.varType))
			}
		})
	}
}

func TestTemplateRef_Struct(t *testing.T) {
	ref := TemplateRef{
		Provider: "github",
		Owner:    "myorg",
		Repo:     "templates",
		Path:     "go/basic",
		Ref:      "v1.0.0",
	}

	if ref.Provider != "github" {
		t.Errorf("expected Provider 'github', got %s", ref.Provider)
	}
	if ref.Owner != "myorg" {
		t.Errorf("expected Owner 'myorg', got %s", ref.Owner)
	}
	if ref.Repo != "templates" {
		t.Errorf("expected Repo 'templates', got %s", ref.Repo)
	}
	if ref.Path != "go/basic" {
		t.Errorf("expected Path 'go/basic', got %s", ref.Path)
	}
	if ref.Ref != "v1.0.0" {
		t.Errorf("expected Ref 'v1.0.0', got %s", ref.Ref)
	}
}

func TestTemplateFile_Struct(t *testing.T) {
	file := TemplateFile{
		Path:     "main.go",
		Content:  []byte("package main"),
		Mode:     0644,
		IsBinary: false,
	}

	if file.Path != "main.go" {
		t.Errorf("expected Path 'main.go', got %s", file.Path)
	}
	if string(file.Content) != "package main" {
		t.Errorf("expected Content 'package main', got %s", string(file.Content))
	}
	if file.Mode != 0644 {
		t.Errorf("expected Mode 0644, got %o", file.Mode)
	}
	if file.IsBinary {
		t.Errorf("expected IsBinary false, got true")
	}
}

func TestIgnJson_MarshalUnmarshal(t *testing.T) {
	minVal := 1024.0
	maxVal := 65535.0

	original := IgnJson{
		Name:        "go-rest-api",
		Version:     "2.1.0",
		Description: "Production-ready Go REST API",
		Author:      "John Doe <john@example.com>",
		Repository:  "https://github.com/johndoe/templates",
		License:     "MIT",
		Tags:        []string{"go", "api", "rest"},
		Variables: map[string]VarDef{
			"project_name": {
				Type:        VarTypeString,
				Description: "Project name",
				Required:    true,
				Example:     "my-api",
				Pattern:     "^[a-z][a-z0-9-]*$",
			},
			"port": {
				Type:        VarTypeInt,
				Description: "Server port",
				Default:     8080,
				Min:         &minVal,
				Max:         &maxVal,
			},
			"enable_swagger": {
				Type:        VarTypeBool,
				Description: "Enable Swagger",
				Default:     true,
			},
		},
		Settings: &TemplateSettings{
			PreserveExecutable: true,
			IgnorePatterns:     []string{"*.log", "*.tmp"},
			BinaryExtensions:   []string{".png", ".jpg"},
			IncludeDotfiles:    true,
			MaxIncludeDepth:    10,
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal from JSON
	var decoded IgnJson
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.Name != original.Name {
		t.Errorf("Name: expected %s, got %s", original.Name, decoded.Name)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version: expected %s, got %s", original.Version, decoded.Version)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description: expected %s, got %s", original.Description, decoded.Description)
	}
	if len(decoded.Tags) != len(original.Tags) {
		t.Errorf("Tags length: expected %d, got %d", len(original.Tags), len(decoded.Tags))
	}
	if len(decoded.Variables) != len(original.Variables) {
		t.Errorf("Variables length: expected %d, got %d", len(original.Variables), len(decoded.Variables))
	}

	// Verify specific variable
	projectName, ok := decoded.Variables["project_name"]
	if !ok {
		t.Fatal("project_name variable not found")
	}
	if projectName.Type != VarTypeString {
		t.Errorf("project_name.Type: expected %s, got %s", VarTypeString, projectName.Type)
	}
	if !projectName.Required {
		t.Error("project_name.Required: expected true, got false")
	}

	// Verify int variable with min/max
	port, ok := decoded.Variables["port"]
	if !ok {
		t.Fatal("port variable not found")
	}
	if port.Min == nil || *port.Min != minVal {
		t.Errorf("port.Min: expected %v, got %v", minVal, port.Min)
	}
	if port.Max == nil || *port.Max != maxVal {
		t.Errorf("port.Max: expected %v, got %v", maxVal, port.Max)
	}

	// Verify settings
	if decoded.Settings == nil {
		t.Fatal("Settings is nil")
	}
	if decoded.Settings.MaxIncludeDepth != 10 {
		t.Errorf("MaxIncludeDepth: expected 10, got %d", decoded.Settings.MaxIncludeDepth)
	}
}

func TestIgnVarJson_MarshalUnmarshal(t *testing.T) {
	now := time.Now().UTC()

	original := IgnVarJson{
		Variables: map[string]interface{}{
			"project_name": "user-service",
			"port":         8080,
			"enable_tls":   true,
			"description":  "User management service",
		},
		Metadata: &FileMetadata{
			GeneratedAt:     now,
			GeneratedBy:     "ign build init v1.0.0",
			TemplateName:    "go-rest-api",
			TemplateVersion: "2.1.0",
			IgnVersion:      "1.0.0",
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal from JSON
	var decoded IgnVarJson
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify variables
	if len(decoded.Variables) != len(original.Variables) {
		t.Errorf("Variables length: expected %d, got %d", len(original.Variables), len(decoded.Variables))
	}

	// Verify string variable
	if projectName, ok := decoded.Variables["project_name"].(string); !ok || projectName != "user-service" {
		t.Errorf("project_name: expected 'user-service', got %v", decoded.Variables["project_name"])
	}

	// Verify int variable (JSON numbers unmarshal as float64)
	if port, ok := decoded.Variables["port"].(float64); !ok || int(port) != 8080 {
		t.Errorf("port: expected 8080, got %v", decoded.Variables["port"])
	}

	// Verify bool variable
	if enableTLS, ok := decoded.Variables["enable_tls"].(bool); !ok || !enableTLS {
		t.Errorf("enable_tls: expected true, got %v", decoded.Variables["enable_tls"])
	}

	// Verify metadata
	if decoded.Metadata == nil {
		t.Fatal("Metadata is nil")
	}
	if decoded.Metadata.TemplateName != "go-rest-api" {
		t.Errorf("Metadata.TemplateName: expected 'go-rest-api', got %s", decoded.Metadata.TemplateName)
	}
}

func TestIgnConfig_MarshalUnmarshal(t *testing.T) {
	now := time.Now().UTC()

	original := IgnConfig{
		Template: TemplateSource{
			URL:  "github.com/myorg/templates",
			Path: "go/rest-api",
			Ref:  "v2.1.0",
		},
		Hash: "abc123def456",
		Metadata: &FileMetadata{
			GeneratedAt:     now,
			GeneratedBy:     "ign checkout v1.0.0",
			TemplateName:    "go-rest-api",
			TemplateVersion: "2.1.0",
			IgnVersion:      "1.0.0",
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Unmarshal from JSON
	var decoded IgnConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify template source
	if decoded.Template.URL != original.Template.URL {
		t.Errorf("Template.URL: expected %s, got %s", original.Template.URL, decoded.Template.URL)
	}
	if decoded.Template.Path != original.Template.Path {
		t.Errorf("Template.Path: expected %s, got %s", original.Template.Path, decoded.Template.Path)
	}
	if decoded.Template.Ref != original.Template.Ref {
		t.Errorf("Template.Ref: expected %s, got %s", original.Template.Ref, decoded.Template.Ref)
	}

	// Verify hash
	if decoded.Hash != original.Hash {
		t.Errorf("Hash: expected %s, got %s", original.Hash, decoded.Hash)
	}

	// Verify metadata
	if decoded.Metadata == nil {
		t.Fatal("Metadata is nil")
	}
	if decoded.Metadata.TemplateName != "go-rest-api" {
		t.Errorf("Metadata.TemplateName: expected 'go-rest-api', got %s", decoded.Metadata.TemplateName)
	}
	if decoded.Metadata.TemplateVersion != "2.1.0" {
		t.Errorf("Metadata.TemplateVersion: expected '2.1.0', got %s", decoded.Metadata.TemplateVersion)
	}
}

func TestTemplate_Struct(t *testing.T) {
	template := Template{
		Ref: TemplateRef{
			Provider: "github",
			Owner:    "myorg",
			Repo:     "templates",
			Path:     "go/basic",
			Ref:      "v1.0.0",
		},
		Config: IgnJson{
			Name:    "go-basic",
			Version: "1.0.0",
			Variables: map[string]VarDef{
				"project_name": {
					Type:        VarTypeString,
					Description: "Project name",
					Required:    true,
				},
			},
		},
		Files: []TemplateFile{
			{
				Path:     "main.go",
				Content:  []byte("package main"),
				Mode:     0644,
				IsBinary: false,
			},
			{
				Path:     "logo.png",
				Content:  []byte{0x89, 0x50, 0x4e, 0x47}, // PNG header
				Mode:     0644,
				IsBinary: true,
			},
		},
		RootPath: "/tmp/templates/go-basic",
	}

	// Verify structure
	if template.Ref.Provider != "github" {
		t.Errorf("Ref.Provider: expected 'github', got %s", template.Ref.Provider)
	}
	if template.Config.Name != "go-basic" {
		t.Errorf("Config.Name: expected 'go-basic', got %s", template.Config.Name)
	}
	if len(template.Files) != 2 {
		t.Errorf("Files length: expected 2, got %d", len(template.Files))
	}
	if template.Files[0].IsBinary {
		t.Error("Files[0].IsBinary: expected false, got true")
	}
	if !template.Files[1].IsBinary {
		t.Error("Files[1].IsBinary: expected true, got false")
	}
	if template.RootPath != "/tmp/templates/go-basic" {
		t.Errorf("RootPath: expected '/tmp/templates/go-basic', got %s", template.RootPath)
	}
}

func TestIgnJson_RoundTrip(t *testing.T) {
	// Test with minimal required fields
	minimal := IgnJson{
		Name:      "minimal-template",
		Version:   "1.0.0",
		Variables: map[string]VarDef{},
	}

	data, err := json.Marshal(minimal)
	if err != nil {
		t.Fatalf("failed to marshal minimal: %v", err)
	}

	var decoded IgnJson
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal minimal: %v", err)
	}

	if decoded.Name != minimal.Name {
		t.Errorf("Name: expected %s, got %s", minimal.Name, decoded.Name)
	}
	if decoded.Version != minimal.Version {
		t.Errorf("Version: expected %s, got %s", minimal.Version, decoded.Version)
	}
}

func TestVarDef_WithFileMode(t *testing.T) {
	// Test that os.FileMode works correctly in TemplateFile
	file := TemplateFile{
		Path:    "script.sh",
		Content: []byte("#!/bin/bash\necho 'hello'"),
		Mode:    os.FileMode(0755), // Executable
	}

	// Check executable bit
	if file.Mode&0111 == 0 {
		t.Error("Expected file to have executable permissions")
	}

	// Non-executable file
	regularFile := TemplateFile{
		Path:    "readme.txt",
		Content: []byte("README"),
		Mode:    os.FileMode(0644),
	}

	if regularFile.Mode&0111 != 0 {
		t.Error("Expected file to NOT have executable permissions")
	}
}

func TestIgnVarJson_EmptyMetadata(t *testing.T) {
	// Test that metadata can be nil
	varConfig := IgnVarJson{
		Variables: map[string]interface{}{
			"name": "test",
		},
		Metadata: nil,
	}

	data, err := json.Marshal(varConfig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IgnVarJson
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Metadata should be nil (omitempty)
	if decoded.Metadata != nil {
		t.Errorf("Expected nil metadata, got %+v", decoded.Metadata)
	}
}

func TestIgnConfig_EmptyMetadata(t *testing.T) {
	// Test that metadata can be nil
	config := IgnConfig{
		Template: TemplateSource{
			URL: "github.com/test/repo",
		},
		Hash:     "abc123",
		Metadata: nil,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IgnConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Metadata should be nil (omitempty)
	if decoded.Metadata != nil {
		t.Errorf("Expected nil metadata, got %+v", decoded.Metadata)
	}
}

func TestTemplateSettings_Defaults(t *testing.T) {
	// Test with zero values
	settings := TemplateSettings{}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Should produce "{}" due to omitempty
	expected := "{}"
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}
