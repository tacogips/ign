package defaults

import (
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/template/model"
)

const placeholderCurrentDir = "{current_dir}"

// ContainsPlaceholder reports whether value includes a supported runtime placeholder.
func ContainsPlaceholder(value interface{}) bool {
	str, ok := value.(string)
	return ok && strings.Contains(str, placeholderCurrentDir)
}

// CurrentDirName returns the base name of the resolved current directory.
func CurrentDirName(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}

	resolved := dir
	if !filepath.IsAbs(resolved) {
		if abs, err := filepath.Abs(resolved); err == nil {
			resolved = abs
		}
	}

	resolved = filepath.Clean(resolved)
	return filepath.Base(resolved)
}

// ResolveValue expands supported placeholders in default values.
// Only string defaults are transformed.
func ResolveValue(value interface{}, dir string) interface{} {
	str, ok := value.(string)
	if !ok {
		return value
	}

	currentDir := CurrentDirName(dir)
	if currentDir == "" {
		return str
	}

	return strings.ReplaceAll(str, placeholderCurrentDir, currentDir)
}

// ResolveVarDef returns a copy of varDef with placeholders in Default resolved.
func ResolveVarDef(varDef model.VarDef, dir string) model.VarDef {
	resolved := varDef
	resolved.Default = ResolveValue(varDef.Default, dir)
	return resolved
}

// ResolveVarDefs returns a copy of the variable definitions with placeholders resolved.
func ResolveVarDefs(varDefs map[string]model.VarDef, dir string) map[string]model.VarDef {
	if varDefs == nil {
		return nil
	}

	resolved := make(map[string]model.VarDef, len(varDefs))
	for name, varDef := range varDefs {
		resolved[name] = ResolveVarDef(varDef, dir)
	}
	return resolved
}

// ResolveIgnJSON returns a shallow copy of ignJson with resolved variable defaults.
func ResolveIgnJSON(ignJson *model.IgnJson, dir string) *model.IgnJson {
	if ignJson == nil {
		return nil
	}

	resolved := *ignJson
	resolved.Variables = ResolveVarDefs(ignJson.Variables, dir)
	return &resolved
}
