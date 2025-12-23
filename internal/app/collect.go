package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
)

// CollectVarsOptions holds options for collecting variables from templates.
type CollectVarsOptions struct {
	// Path is the template directory path.
	Path string
	// Recursive indicates whether to scan subdirectories.
	Recursive bool
	// DryRun shows what would be updated without writing.
	DryRun bool
	// Merge preserves existing variable definitions and only adds new ones.
	Merge bool
}

// CollectedVar represents a variable found in template files.
type CollectedVar struct {
	// Name is the variable name.
	Name string
	// Type is the inferred variable type.
	Type model.VarType
	// HasDefault indicates if a default value was found.
	HasDefault bool
	// Default is the default value (if any).
	Default interface{}
	// Required is true if no default value was provided.
	Required bool
	// Sources lists the files where this variable was found.
	Sources []string
}

// CollectVarsResult holds the result of variable collection.
type CollectVarsResult struct {
	// Variables is the map of collected variables.
	Variables map[string]*CollectedVar
	// FilesScanned is the number of files scanned.
	FilesScanned int
	// IgnJsonPath is the path to the ign.json file.
	IgnJsonPath string
	// Updated indicates if ign.json was updated.
	Updated bool
	// NewVars lists newly added variable names.
	NewVars []string
	// UpdatedVars lists variable names that were updated.
	UpdatedVars []string
}

// Regex patterns for extracting variables
var (
	// Pattern for @ign-var:ARGS@
	varDirectivePattern = regexp.MustCompile(`@ign-var:([^@]+)@`)
	// Pattern for @ign-if:VAR@
	ifDirectivePattern = regexp.MustCompile(`@ign-if:([^@]+)@`)
)

// CollectVars scans template files and collects variable definitions.
func CollectVars(ctx context.Context, opts CollectVarsOptions) (*CollectVarsResult, error) {
	debug.DebugSection("[app] CollectVars workflow start")
	debug.DebugValue("[app] Path", opts.Path)
	debug.DebugValue("[app] Recursive", opts.Recursive)
	debug.DebugValue("[app] DryRun", opts.DryRun)
	debug.DebugValue("[app] Merge", opts.Merge)

	// Resolve absolute path
	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, NewValidationError("failed to resolve path", err)
	}

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, NewValidationError(fmt.Sprintf("path not found: %s", absPath), err)
	}
	if !info.IsDir() {
		return nil, NewValidationError(fmt.Sprintf("path is not a directory: %s", absPath), nil)
	}

	// Check for ign.json
	ignJsonPath := filepath.Join(absPath, "ign.json")

	result := &CollectVarsResult{
		Variables:   make(map[string]*CollectedVar),
		IgnJsonPath: ignJsonPath,
	}

	// Scan template files
	err = scanTemplateFiles(ctx, absPath, opts.Recursive, result)
	if err != nil {
		return nil, err
	}

	debug.DebugValue("[app] Files scanned", result.FilesScanned)
	debug.DebugValue("[app] Variables found", len(result.Variables))

	// Load existing ign.json if it exists and merge mode is enabled
	var existingIgnJson *model.IgnJson
	if _, err := os.Stat(ignJsonPath); err == nil {
		existingIgnJson, err = config.LoadIgnJson(ignJsonPath)
		if err != nil {
			return nil, NewValidationError("failed to load existing ign.json", err)
		}
	}

	// Determine what's new and what's updated
	result.NewVars, result.UpdatedVars = categorizeVars(result.Variables, existingIgnJson, opts.Merge)

	// Update ign.json if not dry run
	if !opts.DryRun {
		err = updateIgnJson(ignJsonPath, result, existingIgnJson, opts.Merge)
		if err != nil {
			return nil, err
		}
		result.Updated = true
	}

	debug.Debug("[app] CollectVars workflow completed")
	return result, nil
}

// scanTemplateFiles recursively scans files for variable directives.
func scanTemplateFiles(ctx context.Context, dirPath string, recursive bool, result *CollectVarsResult) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return NewValidationError(fmt.Sprintf("failed to read directory: %s", dirPath), err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		// Skip hidden files and directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip ign.json itself
		if entry.Name() == "ign.json" {
			continue
		}

		if entry.IsDir() {
			if recursive {
				if err := scanTemplateFiles(ctx, fullPath, recursive, result); err != nil {
					return err
				}
			}
			continue
		}

		// Skip binary files
		if isBinaryFile(fullPath) {
			continue
		}

		// Scan the file
		if err := scanFile(ctx, fullPath, result); err != nil {
			debug.Debug("[app] Error scanning file %s: %v", fullPath, err)
			// Continue scanning other files
		}
	}

	return nil
}

// scanFile extracts variables from a single file.
func scanFile(ctx context.Context, filePath string, result *CollectVarsResult) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	result.FilesScanned++

	text := string(content)

	// Find @ign-var: directives
	varMatches := varDirectivePattern.FindAllStringSubmatch(text, -1)
	for _, match := range varMatches {
		if len(match) < 2 {
			continue
		}
		args := match[1]
		addVarFromDirective(args, filePath, result)
	}

	// Find @ign-if: directives (these are bool variables)
	ifMatches := ifDirectivePattern.FindAllStringSubmatch(text, -1)
	for _, match := range ifMatches {
		if len(match) < 2 {
			continue
		}
		varName := strings.TrimSpace(match[1])
		addConditionalVar(varName, filePath, result)
	}

	return nil
}

// addVarFromDirective parses a var directive and adds it to the result.
func addVarFromDirective(args string, filePath string, result *CollectVarsResult) {
	args = strings.TrimSpace(args)
	if args == "" {
		return
	}

	// Parse variable syntax: NAME, NAME:TYPE, NAME=DEFAULT, NAME:TYPE=DEFAULT
	varName, varType, defaultValue, hasDefault := parseVarArgs(args)
	if varName == "" {
		return
	}

	// Check if variable already exists
	existing, exists := result.Variables[varName]
	if exists {
		// Add source file
		if !containsSource(existing.Sources, filePath) {
			existing.Sources = append(existing.Sources, filePath)
		}
		// Update type if more specific
		if existing.Type == "" && varType != "" {
			existing.Type = varType
		}
		// Update default if not set
		if !existing.HasDefault && hasDefault {
			existing.HasDefault = true
			existing.Default = defaultValue
			existing.Required = false
		}
	} else {
		result.Variables[varName] = &CollectedVar{
			Name:       varName,
			Type:       varType,
			HasDefault: hasDefault,
			Default:    defaultValue,
			Required:   !hasDefault,
			Sources:    []string{filePath},
		}
	}
}

// addConditionalVar adds a boolean variable from @ign-if directive.
func addConditionalVar(varName string, filePath string, result *CollectVarsResult) {
	if varName == "" {
		return
	}

	existing, exists := result.Variables[varName]
	if exists {
		// Update type to bool if not set (conditional variables must be bool)
		if existing.Type == "" {
			existing.Type = model.VarTypeBool
		}
		if !containsSource(existing.Sources, filePath) {
			existing.Sources = append(existing.Sources, filePath)
		}
	} else {
		result.Variables[varName] = &CollectedVar{
			Name:     varName,
			Type:     model.VarTypeBool,
			Required: true,
			Sources:  []string{filePath},
		}
	}
}

// parseVarArgs parses variable directive arguments.
// Returns (name, type, default, hasDefault)
func parseVarArgs(args string) (string, model.VarType, interface{}, bool) {
	var varName string
	var varType model.VarType
	var defaultValue interface{}
	var hasDefault bool

	// Check for default value (split on '=')
	if idx := strings.Index(args, "="); idx != -1 {
		hasDefault = true
		defaultStr := args[idx+1:]
		args = args[:idx]
		defaultValue = parseDefaultValueStr(defaultStr)
	}

	// Check for type annotation (split on ':')
	if idx := strings.Index(args, ":"); idx != -1 {
		varName = strings.TrimSpace(args[:idx])
		typeStr := strings.TrimSpace(args[idx+1:])
		switch typeStr {
		case "string":
			varType = model.VarTypeString
		case "int":
			varType = model.VarTypeInt
		case "bool":
			varType = model.VarTypeBool
		}
	} else {
		varName = strings.TrimSpace(args)
		// Infer type from default value if available
		if hasDefault {
			varType = inferVarType(defaultValue)
		}
	}

	return varName, varType, defaultValue, hasDefault
}

// parseDefaultValueStr parses a default value string.
func parseDefaultValueStr(value string) interface{} {
	value = strings.TrimSpace(value)

	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try to parse as integer
	var intVal int
	if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
		return intVal
	}

	return value
}

// inferVarType infers VarType from a value.
func inferVarType(val interface{}) model.VarType {
	switch val.(type) {
	case bool:
		return model.VarTypeBool
	case int, int64, float64:
		return model.VarTypeInt
	default:
		return model.VarTypeString
	}
}

// containsSource checks if a file path is already in the sources list.
func containsSource(sources []string, filePath string) bool {
	for _, s := range sources {
		if s == filePath {
			return true
		}
	}
	return false
}

// categorizeVars determines which variables are new and which are updated.
func categorizeVars(collected map[string]*CollectedVar, existing *model.IgnJson, merge bool) (newVars, updatedVars []string) {
	existingVars := make(map[string]bool)
	if existing != nil && existing.Variables != nil {
		for name := range existing.Variables {
			existingVars[name] = true
		}
	}

	for name := range collected {
		if existingVars[name] {
			if !merge {
				updatedVars = append(updatedVars, name)
			}
		} else {
			newVars = append(newVars, name)
		}
	}

	sort.Strings(newVars)
	sort.Strings(updatedVars)
	return
}

// updateIgnJson updates or creates ign.json with collected variables.
func updateIgnJson(path string, result *CollectVarsResult, existing *model.IgnJson, merge bool) error {
	var ignJson *model.IgnJson

	if existing != nil {
		ignJson = existing
	} else {
		// Create new ign.json with defaults
		ignJson = &model.IgnJson{
			Name:        filepath.Base(filepath.Dir(path)),
			Version:     "0.1.0",
			Description: "Template description",
			Variables:   make(map[string]model.VarDef),
		}
	}

	// Ensure Variables map is initialized
	if ignJson.Variables == nil {
		ignJson.Variables = make(map[string]model.VarDef)
	}

	// Update variables
	for name, collected := range result.Variables {
		// Skip if merge mode and variable exists
		if merge {
			if _, exists := ignJson.Variables[name]; exists {
				continue
			}
		}

		varDef := model.VarDef{
			Type:        collected.Type,
			Description: fmt.Sprintf("Variable %s", name),
			Required:    collected.Required,
		}

		// Set default type if not specified
		if varDef.Type == "" {
			varDef.Type = model.VarTypeString
		}

		if collected.HasDefault {
			varDef.Default = collected.Default
		}

		ignJson.Variables[name] = varDef
	}

	// Calculate and update template hash
	templateDir := filepath.Dir(path)
	newHash, err := CalculateTemplateHashFromDir(templateDir)
	if err != nil {
		debug.Debug("[app] Failed to calculate template hash: %v", err)
		// Continue without hash if calculation fails
	} else {
		ignJson.Hash = newHash
		debug.DebugValue("[app] Template hash calculated", newHash)
	}

	// Write ign.json
	data, err := json.MarshalIndent(ignJson, "", "  ")
	if err != nil {
		return NewValidationError("failed to marshal ign.json", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return NewValidationError("failed to write ign.json", err)
	}

	return nil
}

// CalculateTemplateHashFromDir calculates SHA256 hash of all template files in a directory.
// Files are sorted by path to ensure deterministic hash generation.
// Excludes ign.json itself from the hash calculation.
func CalculateTemplateHashFromDir(dirPath string) (string, error) {
	h := sha256.New()
	var files []string

	// Collect all file paths
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Skip ign.json (we're calculating hash for everything else)
		if info.Name() == "ign.json" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return "", err
	}

	// Sort files for deterministic hash
	sort.Strings(files)

	// Hash each file's path and content
	for _, relPath := range files {
		fullPath := filepath.Join(dirPath, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return "", err
		}

		h.Write([]byte(relPath))
		h.Write(content)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
