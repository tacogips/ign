package parser

import (
	"regexp"
	"strings"

	"github.com/tacogips/ign/internal/debug"
)

// commentDirectivePattern matches @ign-comment:XXX@ directive
var commentDirectivePattern = regexp.MustCompile(`@ign-comment:[^@]*@`)

// validateCommentDirectiveLine validates that a line containing @ign-comment:XXX@
// has only whitespace before and after the directive.
// Returns an error if there are non-whitespace characters around the directive.
func validateCommentDirectiveLine(line string) error {
	match := commentDirectivePattern.FindStringIndex(line)
	if match == nil {
		return nil // No directive found
	}

	// Check text before the directive
	before := line[:match[0]]
	if strings.TrimSpace(before) != "" {
		return newParseErrorWithDirective(InvalidDirectiveSyntax,
			"@ign-comment directive must be on its own line (non-whitespace found before directive)",
			line)
	}

	// Check text after the directive
	after := line[match[1]:]
	if strings.TrimSpace(after) != "" {
		return newParseErrorWithDirective(InvalidDirectiveSyntax,
			"@ign-comment directive must be on its own line (non-whitespace found after directive)",
			line)
	}

	debug.Debug("[parser] processCommentDirectivesInText: removing comment line=%s", strings.TrimSpace(line))
	return nil
}

// lineContainsCommentDirective checks if a line contains @ign-comment:XXX@ directive.
func lineContainsCommentDirective(line string) bool {
	return commentDirectivePattern.MatchString(line)
}
