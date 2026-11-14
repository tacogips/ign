package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tacogips/ign/internal/config"
	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/model"
)

// VarsOptions contains options for inspecting project template variables.
type VarsOptions struct {
	// GitHubToken is the GitHub personal access token (optional).
	GitHubToken string
}

// VarsResult contains merged template declarations and current variable values.
type VarsResult struct {
	// DeclarationsAvailable indicates whether template declarations were fetched.
	DeclarationsAvailable bool `json:"declarations_available"`
	// Rows contains sorted variable inspection rows.
	Rows []VarsRow `json:"rows"`
	// UnsetCount is the number of rows without a current value.
	UnsetCount int `json:"unset_count"`
	// RequiredUnsetCount is the number of declared required rows without a current value.
	RequiredUnsetCount int `json:"required_unset_count"`
	// DeclarationError is a non-fatal error from fetching template declarations.
	DeclarationError error `json:"-"`
}

// VarsRow contains one merged template variable declaration and current value.
type VarsRow struct {
	Name        string      `json:"name"`
	Type        string      `json:"type,omitempty"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Current     interface{} `json:"current,omitempty"`
	HasCurrent  bool        `json:"has_current"`
	Description string      `json:"description,omitempty"`
	Unset       bool        `json:"unset"`
	Declared    bool        `json:"declared"`
}

// InspectVars loads project variables and merges them with template declarations.
func InspectVars(ctx context.Context, opts VarsOptions) (*VarsResult, error) {
	configDir := model.IgnConfigDir
	ignConfigPath := filepath.Join(configDir, model.IgnProjectConfigFile)
	ignVarPath := filepath.Join(configDir, model.IgnVarFile)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, NewValidationError(
			"vars requires prior checkout: .ign directory not found.\n"+
				"Run 'ign checkout <template-url>' first.",
			nil,
		)
	}

	ignConfig, err := config.LoadIgnConfig(ignConfigPath)
	if err != nil {
		return nil, NewCheckoutError(
			"failed to load .ign/ign.json: run 'ign checkout <template-url>' first",
			err,
		)
	}

	ignVar, err := config.LoadIgnVarJson(ignVarPath)
	if err != nil {
		return nil, NewCheckoutError(
			"failed to load .ign/ign-var.json: run 'ign checkout <template-url>' first",
			err,
		)
	}
	current := ignVar.Variables
	if current == nil {
		current = map[string]interface{}{}
	}

	fetched, fetchErr := fetchTrackedTemplate(ctx, trackedTemplateFetchOptions{
		Source:      ignConfig.Template,
		GitHubToken: opts.GitHubToken,
	})
	if fetchErr != nil {
		debug.Debug("[app] Template declaration fetch failed for vars: %v", fetchErr)
		return buildVarsResult(nil, current, fetchErr), nil
	}

	return buildVarsResult(fetched.Template.Config.Variables, current, nil), nil
}

func buildVarsResult(varDefs map[string]model.VarDef, current map[string]interface{}, declarationErr error) *VarsResult {
	rowsByName := make(map[string]VarsRow)

	for name, varDef := range varDefs {
		value, hasCurrent := current[name]
		unset := !hasCurrent
		row := VarsRow{
			Name:        name,
			Type:        string(varDef.Type),
			Required:    varDef.Required,
			Default:     varDef.Default,
			Current:     value,
			HasCurrent:  hasCurrent,
			Description: varDef.Description,
			Unset:       unset,
			Declared:    true,
		}
		rowsByName[name] = row
	}

	for name, value := range current {
		if _, ok := rowsByName[name]; ok {
			continue
		}
		rowsByName[name] = VarsRow{
			Name:       name,
			Current:    value,
			HasCurrent: true,
			Declared:   false,
		}
	}

	names := make([]string, 0, len(rowsByName))
	for name := range rowsByName {
		names = append(names, name)
	}
	sort.Strings(names)

	result := &VarsResult{
		DeclarationsAvailable: declarationErr == nil,
		DeclarationError:      declarationErr,
		Rows:                  make([]VarsRow, 0, len(names)),
	}
	for _, name := range names {
		row := rowsByName[name]
		result.Rows = append(result.Rows, row)
		if row.Unset {
			result.UnsetCount++
			if row.Declared && row.Required {
				result.RequiredUnsetCount++
			}
		}
	}

	return result
}

// FilterUnsetVars returns a copy of result containing only unset rows.
func FilterUnsetVars(result *VarsResult) *VarsResult {
	if result == nil {
		return nil
	}

	filtered := *result
	filtered.Rows = make([]VarsRow, 0, len(result.Rows))
	filtered.UnsetCount = 0
	filtered.RequiredUnsetCount = 0

	for _, row := range result.Rows {
		if !row.Unset {
			continue
		}
		filtered.Rows = append(filtered.Rows, row)
		filtered.UnsetCount++
		if row.Declared && row.Required {
			filtered.RequiredUnsetCount++
		}
	}

	return &filtered
}

func formatVarsValue(value interface{}, hasValue bool) string {
	if !hasValue {
		return ""
	}
	if value == nil {
		return "null"
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
