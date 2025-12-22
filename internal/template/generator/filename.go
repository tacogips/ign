package generator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tacogips/ign/internal/debug"
	"github.com/tacogips/ign/internal/template/parser"
)

// ProcessFilename processes a template filename/path for variable substitution.
// It handles @ign-var:NAME@ directives in file and directory names.
// Returns the processed path or an error if:
// - Variable substitution fails
// - The resulting path contains path traversal (..)
// - The resulting path is absolute
// - The resulting path has empty components
func ProcessFilename(ctx context.Context, filePath string, vars parser.Variables, p parser.Parser) (string, error) {
	debug.Debug("[generator] ProcessFilename: input=%s", filePath)

	// Process each path component separately to handle directories with variables
	components := strings.Split(filePath, "/")
	processedComponents := make([]string, 0, len(components))

	for i, component := range components {
		// Skip empty components (e.g., from leading/trailing slashes)
		if component == "" {
			continue
		}

		debug.Debug("[generator] ProcessFilename: processing component[%d]=%s", i, component)

		// Process the component for variable substitution
		// We use ParseFilename to handle only @ign-var: and @ign-raw: directives
		// Other directives (@ign-if:, @ign-comment:, @ign-include:) are NOT processed in filenames
		processed, err := p.ParseFilename(ctx, []byte(component), vars)
		if err != nil {
			return "", fmt.Errorf("failed to process filename component %q: %w", component, err)
		}

		processedComponent := string(processed)
		debug.Debug("[generator] ProcessFilename: component[%d] processed: %s -> %s", i, component, processedComponent)

		// Validate the processed component
		if err := validateFilenameComponent(processedComponent, component); err != nil {
			return "", err
		}

		processedComponents = append(processedComponents, processedComponent)
	}

	// Reconstruct the path
	result := strings.Join(processedComponents, "/")
	debug.Debug("[generator] ProcessFilename: result=%s", result)

	// Final validation
	if err := validateProcessedPath(result, filePath); err != nil {
		return "", err
	}

	return result, nil
}

// validateFilenameComponent validates a single processed filename component.
// Note: Security validation for dangerous characters (null bytes, colons) in variable values
// is performed during variable substitution in the parser layer.
// This function validates the overall component structure after substitution.
func validateFilenameComponent(processed, original string) error {
	// Check for path traversal in component (must check before trimming).
	// Note: While the parser already rejects ".." as a complete variable value,
	// this check catches cases like "data..backup" where ".." appears within the value.
	// This provides defense-in-depth validation at the component level.
	if strings.Contains(processed, "..") {
		return fmt.Errorf("invalid filename component: %q contains path traversal (..) after variable substitution (original: %q)", processed, original)
	}

	// Check for path separators in component (security check)
	if strings.Contains(processed, "/") || strings.Contains(processed, "\\") {
		return fmt.Errorf("invalid filename component: %q contains path separator after variable substitution (original: %q)", processed, original)
	}

	// Check for empty result after trimming whitespace
	trimmed := strings.TrimSpace(processed)
	if trimmed == "" {
		return fmt.Errorf("filename component %q resulted in empty value after variable substitution (original: %q)", processed, original)
	}

	return nil
}

// validateProcessedPath validates the complete processed path.
func validateProcessedPath(processed, original string) error {
	// Check for absolute path
	if filepath.IsAbs(processed) {
		return fmt.Errorf("invalid filename: %q is absolute path after variable substitution (original: %q)", processed, original)
	}

	// Use filepath.Clean to normalize and check for path traversal.
	// Note: While validateFilenameComponent already checks for ".." in individual components,
	// this path-level validation serves as additional defense-in-depth by checking the
	// complete normalized path. filepath.Clean resolves sequences like "a/../b" which
	// wouldn't exist after component validation, but this check ensures no edge cases
	// slip through in path construction logic.
	cleaned := filepath.Clean(processed)

	// After cleaning, if the path starts with "..", it's trying to escape
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("invalid filename: %q attempts path traversal after variable substitution (original: %q)", processed, original)
	}

	// Check if the path is "." (current directory)
	if cleaned == "." {
		return fmt.Errorf("invalid filename: %q resolves to current directory after variable substitution (original: %q)", processed, original)
	}

	return nil
}
