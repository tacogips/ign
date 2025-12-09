package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// defaultMaxIncludeDepth is the default maximum include depth.
	defaultMaxIncludeDepth = 10
)

// processIncludes processes all @ign-include:PATH@ directives in input.
// Includes are processed recursively with circular dependency detection.
func processIncludes(ctx context.Context, input []byte, pctx *ParseContext) ([]byte, error) {
	text := string(input)
	matches := findDirectives(input)

	// Process includes from last to first (to maintain correct positions)
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if match.Type != DirectiveInclude {
			continue
		}

		// Check maximum include depth
		if pctx.IncludeDepth >= defaultMaxIncludeDepth {
			return nil, newParseErrorWithFile(MaxIncludeDepth,
				fmt.Sprintf("maximum include depth (%d) exceeded", defaultMaxIncludeDepth),
				pctx.CurrentFile, 0)
		}

		// Get include path
		includePath := strings.TrimSpace(match.Args)
		if includePath == "" {
			return nil, newParseErrorWithDirective(InvalidDirectiveSyntax,
				"include path is empty",
				match.RawText)
		}

		// Resolve include path
		resolvedPath, err := resolveIncludePath(includePath, pctx.TemplateRoot, pctx.CurrentFile)
		if err != nil {
			return nil, newParseErrorWithDirective(IncludeNotFound,
				fmt.Sprintf("failed to resolve include path: %v", err),
				match.RawText)
		}

		// Check for circular includes
		if containsString(pctx.IncludeStack, resolvedPath) {
			chain := append(pctx.IncludeStack, resolvedPath)
			return nil, newParseErrorWithFile(CircularInclude,
				fmt.Sprintf("circular include detected: %s", strings.Join(chain, " -> ")),
				resolvedPath, 0)
		}

		// Read include file
		content, err := os.ReadFile(resolvedPath)
		if err != nil {
			return nil, newParseErrorWithDirective(IncludeNotFound,
				fmt.Sprintf("failed to read include file: %v", err),
				match.RawText)
		}

		// Create new parse context for included file
		includeCtx := &ParseContext{
			Variables:    pctx.Variables,
			IncludeDepth: pctx.IncludeDepth + 1,
			IncludeStack: append(pctx.IncludeStack, resolvedPath),
			TemplateRoot: pctx.TemplateRoot,
			CurrentFile:  resolvedPath,
		}

		// Recursively process the included content
		processed, err := parseInternal(ctx, content, includeCtx)
		if err != nil {
			return nil, err
		}

		// Replace the include directive with processed content
		text = text[:match.Start] + string(processed) + text[match.End:]
	}

	return []byte(text), nil
}

// resolveIncludePath resolves an include path relative to template root or current file.
// Supports:
// - Relative paths: relative to current file's directory
// - Absolute paths (starting with /): relative to template root
func resolveIncludePath(includePath, templateRoot, currentFile string) (string, error) {
	// Validate path doesn't contain ..
	if strings.Contains(includePath, "..") {
		return "", fmt.Errorf("include path contains '..': %s", includePath)
	}

	var resolvedPath string

	if filepath.IsAbs(includePath) || strings.HasPrefix(includePath, "/") {
		// Absolute path within template (from template root)
		resolvedPath = filepath.Join(templateRoot, strings.TrimPrefix(includePath, "/"))
	} else {
		// Relative path (from current file's directory)
		currentDir := filepath.Dir(currentFile)
		resolvedPath = filepath.Join(currentDir, includePath)
	}

	// Clean the path
	resolvedPath = filepath.Clean(resolvedPath)

	// Verify the resolved path is within template root
	if !strings.HasPrefix(resolvedPath, templateRoot) {
		return "", fmt.Errorf("include path escapes template root: %s", includePath)
	}

	// Check if file exists
	if _, err := os.Stat(resolvedPath); err != nil {
		return "", fmt.Errorf("include file not found: %s", resolvedPath)
	}

	return resolvedPath, nil
}

// containsString checks if a slice contains a string.
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
