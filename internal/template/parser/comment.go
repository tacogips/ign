package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Regex patterns for detecting comment markers
	// Single-line comments: //, #, --
	singleLineCommentPattern = regexp.MustCompile(`^\s*(//|#|--)\s*`)

	// Block comments: /* */, <!-- -->
	blockCommentStartPattern = regexp.MustCompile(`^\s*/\*\s*`)
	blockCommentEndPattern   = regexp.MustCompile(`\s*\*/$`)
	htmlCommentStartPattern  = regexp.MustCompile(`^\s*<!--\s*`)
	htmlCommentEndPattern    = regexp.MustCompile(`\s*-->$`)
)

// processCommentDirective substitutes @ign-comment:NAME@ and removes comment markers.
func processCommentDirective(args string, vars Variables, lineContent string) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", newParseError(InvalidDirectiveSyntax, "variable name is empty in @ign-comment:")
	}

	val, ok := vars.Get(args)
	if !ok {
		return "", newParseErrorWithDirective(MissingVariable,
			fmt.Sprintf("variable not found: %s", args),
			"@ign-comment:"+args+"@")
	}

	// Convert value to string
	valueStr := valueToString(val)

	// Remove comment markers from the line
	cleaned := removeCommentMarkers(lineContent)

	// Replace the directive with the value in the cleaned line
	result := strings.Replace(cleaned, "@ign-comment:"+args+"@", valueStr, 1)

	return result, nil
}

// removeCommentMarkers removes comment markers from a line of text.
// Supported markers: //, #, --, /* */, <!-- -->
func removeCommentMarkers(line string) string {
	// Get leading whitespace
	leadingSpace := ""
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) < len(line) {
		leadingSpace = line[:len(line)-len(trimmed)]
	}

	content := trimmed

	// Try single-line comment markers: //, #, --
	if match := singleLineCommentPattern.FindStringSubmatch(content); match != nil {
		// Remove the comment marker and one following space (if present)
		content = strings.TrimPrefix(content, match[0])
		return leadingSpace + content
	}

	// Try block comment: /* ... */
	if blockCommentStartPattern.MatchString(content) && blockCommentEndPattern.MatchString(content) {
		content = blockCommentStartPattern.ReplaceAllString(content, "")
		content = blockCommentEndPattern.ReplaceAllString(content, "")
		return leadingSpace + strings.TrimSpace(content)
	}

	// Try HTML comment: <!-- ... -->
	if htmlCommentStartPattern.MatchString(content) && htmlCommentEndPattern.MatchString(content) {
		content = htmlCommentStartPattern.ReplaceAllString(content, "")
		content = htmlCommentEndPattern.ReplaceAllString(content, "")
		return leadingSpace + strings.TrimSpace(content)
	}

	// No comment marker found, return original
	return line
}
