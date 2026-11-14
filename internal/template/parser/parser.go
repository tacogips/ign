package parser

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/tacogips/ign/internal/debug"
)

// Parser processes template files and substitutes variables.
type Parser interface {
	// Parse processes a template file with variable substitution.
	Parse(ctx context.Context, input []byte, vars Variables) ([]byte, error)

	// ParseWithContext processes a template with full parse context.
	ParseWithContext(ctx context.Context, input []byte, pctx *ParseContext) ([]byte, error)

	// ParseFilename processes a filename with only @ign-var: and @ign-raw: directives.
	// Other directives (@ign-if:, @ign-comment:, @ign-include:) are NOT processed.
	ParseFilename(ctx context.Context, input []byte, vars Variables) ([]byte, error)

	// Validate validates template syntax without processing.
	Validate(ctx context.Context, input []byte) error

	// ExtractVariables finds all variable references in a template.
	ExtractVariables(input []byte) ([]string, error)
}

// ParseContext holds state during parsing.
type ParseContext struct {
	// Variables holds variable values.
	Variables Variables
	// IncludeDepth tracks the current include nesting depth.
	IncludeDepth int
	// IncludeStack tracks the chain of included files (for circular detection).
	IncludeStack []string
	// TemplateRoot is the root directory of the template.
	TemplateRoot string
	// CurrentFile is the current file being parsed.
	CurrentFile string
}

// DefaultParser implements Parser interface.
type DefaultParser struct{}

// NewParser creates a new DefaultParser.
func NewParser() Parser {
	return &DefaultParser{}
}

// Parse processes a template file with variable substitution.
func (p *DefaultParser) Parse(ctx context.Context, input []byte, vars Variables) ([]byte, error) {
	debug.Debug("[parser] Parse: starting with input size=%d bytes", len(input))
	pctx := &ParseContext{
		Variables:    vars,
		IncludeDepth: 0,
		IncludeStack: []string{},
		TemplateRoot: "",
		CurrentFile:  "",
	}
	return p.ParseWithContext(ctx, input, pctx)
}

// ParseWithContext processes a template with full parse context.
func (p *DefaultParser) ParseWithContext(ctx context.Context, input []byte, pctx *ParseContext) ([]byte, error) {
	debug.Debug("[parser] ParseWithContext: depth=%d, file=%s, root=%s",
		pctx.IncludeDepth, pctx.CurrentFile, pctx.TemplateRoot)
	return parseInternal(ctx, input, pctx)
}

// ParseFilename processes a filename with only @ign-var: and @ign-raw: directives.
// This is a simplified parser that skips @ign-if:, @ign-comment:, and @ign-include: directives.
func (p *DefaultParser) ParseFilename(ctx context.Context, input []byte, vars Variables) ([]byte, error) {
	debug.Debug("[parser] ParseFilename: starting with input size=%d bytes", len(input))
	return parseFilenameInternal(ctx, input, vars)
}

// parseFilenameInternal processes filenames with only @ign-var: and @ign-raw: directives.
// Processing order:
// 1. Process @ign-raw: directives (replace with placeholders)
// 2. Process @ign-var: directives
// 3. Restore raw content from placeholders
func parseFilenameInternal(ctx context.Context, input []byte, vars Variables) ([]byte, error) {
	var err error
	result := input

	// Step 1: Process @ign-raw: directives first (replace with placeholders to prevent further processing)
	debug.Debug("[parser] Filename Step 1: Extracting raw directives")
	result, rawContent, err := extractRawDirectives(result)
	if err != nil {
		return nil, err
	}
	debug.Debug("[parser] Filename Step 1: Extracted %d raw directive(s)", len(rawContent))

	// Step 2: Process @ign-var: directives
	debug.Debug("[parser] Filename Step 2: Processing variables")
	result, err = processVarDirectivesInText(result, vars)
	if err != nil {
		return nil, err
	}

	// Step 3: Restore raw content from placeholders
	debug.Debug("[parser] Filename Step 3: Restoring raw content")
	result = restoreRawContent(result, rawContent)

	debug.Debug("[parser] Filename parsing complete, output size=%d bytes", len(result))
	return result, nil
}

// parseInternal is the internal parsing implementation.
// Processing order:
// 1. Process @ign-raw: directives (replace with placeholders)
// 2. Process @ign-include: directives (recursively)
// 3. Process @ign-if:/@ign-else@/@ign-endif@ blocks
// 4. Process @ign-comment: directives (line by line)
// 5. Process @ign-var: directives
// 6. Restore raw content from placeholders
func parseInternal(ctx context.Context, input []byte, pctx *ParseContext) ([]byte, error) {
	var err error
	result := input

	// Step 1: Process @ign-raw: directives first (replace with placeholders to prevent further processing)
	debug.Debug("[parser] Step 1: Extracting raw directives")
	result, rawContent, err := extractRawDirectives(result)
	if err != nil {
		return nil, err
	}
	debug.Debug("[parser] Step 1: Extracted %d raw directive(s)", len(rawContent))

	// Step 2: Process @ign-include: directives (recursive)
	debug.Debug("[parser] Step 2: Processing includes")
	result, err = processIncludes(ctx, result, pctx)
	if err != nil {
		return nil, err
	}

	// Step 3: Process @ign-if:/@ign-else@/@ign-endif@ blocks
	debug.Debug("[parser] Step 3: Processing conditionals")
	result, err = processConditionals(result, pctx.Variables)
	if err != nil {
		return nil, err
	}

	// Step 4: Process @ign-comment: directives (line-by-line to preserve context)
	debug.Debug("[parser] Step 4: Processing comments")
	result, err = processCommentDirectivesInText(result, pctx.Variables)
	if err != nil {
		return nil, err
	}

	// Step 5: Process @ign-var: directives
	debug.Debug("[parser] Step 5: Processing variables")
	result, err = processVarDirectivesInText(result, pctx.Variables)
	if err != nil {
		return nil, err
	}

	// Step 6: Restore raw content from placeholders
	debug.Debug("[parser] Step 6: Restoring raw content")
	result = restoreRawContent(result, rawContent)

	debug.Debug("[parser] Parsing complete, output size=%d bytes", len(result))
	return result, nil
}

// extractRawDirectives replaces @ign-raw:CONTENT@ with placeholders and returns the content map.
// Returns: (modified input, raw content map, error)
func extractRawDirectives(input []byte) ([]byte, map[string]string, error) {
	text := string(input)
	matches := findDirectives(input)
	rawContent := make(map[string]string)
	placeholderIndex := 0

	// Process from last to first to maintain positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if match.Type != DirectiveRaw {
			continue
		}

		content, err := processRawDirective(match.Args)
		if err != nil {
			return nil, nil, err
		}

		// Create unique placeholder
		placeholder := fmt.Sprintf("\x00IGN_RAW_%d\x00", placeholderIndex)
		rawContent[placeholder] = content
		placeholderIndex++

		text = text[:match.Start] + placeholder + text[match.End:]
	}

	return []byte(text), rawContent, nil
}

// restoreRawContent replaces placeholders with their raw content.
func restoreRawContent(input []byte, rawContent map[string]string) []byte {
	text := string(input)
	for placeholder, content := range rawContent {
		text = strings.ReplaceAll(text, placeholder, content)
	}
	return []byte(text)
}

// processVarDirectivesInText processes all @ign-var:NAME@ directives in text.
func processVarDirectivesInText(input []byte, vars Variables) ([]byte, error) {
	text := string(input)
	matches := findDirectives(input)

	// Process from last to first to maintain positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if match.Type != DirectiveVar {
			continue
		}

		replacement, err := processVarDirective(match.Args, vars)
		if err != nil {
			return nil, err
		}

		text = text[:match.Start] + replacement + text[match.End:]
	}

	return []byte(text), nil
}

// processCommentDirectivesInText removes all lines containing @ign-comment:XXX@ directives.
// Lines with this directive are template comments and should not appear in output.
// Returns an error if a line has non-whitespace characters before or after the directive.
func processCommentDirectivesInText(input []byte, vars Variables) ([]byte, error) {
	lines := bytes.Split(input, []byte("\n"))
	var result [][]byte

	for _, line := range lines {
		lineStr := string(line)

		// Check if this line contains @ign-comment: directive
		if lineContainsCommentDirective(lineStr) {
			// Validate that the directive is on its own line
			if err := validateCommentDirectiveLine(lineStr); err != nil {
				return nil, err
			}
			// Skip this line (remove from output)
			continue
		}

		result = append(result, line)
	}

	return bytes.Join(result, []byte("\n")), nil
}

// Validate validates template syntax without processing.
func (p *DefaultParser) Validate(ctx context.Context, input []byte) error {
	// Try to find all directives
	matches := findDirectives(input)

	for _, match := range matches {
		// Check for unknown directives
		if match.Type == DirectiveType(-1) {
			return newParseErrorWithDirective(UnknownDirective,
				fmt.Sprintf("unknown directive: %s", match.Name),
				match.RawText)
		}

		// Validate directive arguments
		switch match.Type {
		case DirectiveVar:
			if strings.TrimSpace(match.Args) == "" {
				return newParseErrorWithDirective(InvalidDirectiveSyntax,
					"variable name is empty",
					match.RawText)
			}
		case DirectiveComment:
			// @ign-comment: can have any content (including empty) - it's a template comment
			// Validation of line positioning is done during processing
		case DirectiveInclude:
			if strings.TrimSpace(match.Args) == "" {
				return newParseErrorWithDirective(InvalidDirectiveSyntax,
					"include path is empty",
					match.RawText)
			}
		case DirectiveIf:
			if strings.TrimSpace(match.Args) == "" {
				return newParseErrorWithDirective(InvalidDirectiveSyntax,
					"condition variable is empty",
					match.RawText)
			}
		}
	}

	// Check for unclosed blocks
	_, err := findInnermostConditionalBlock(matches, string(input))
	if err != nil {
		return err
	}

	return nil
}

// ExtractVariables finds all variable references in a template.
// Note: @ign-comment: is NOT included as it's a template comment, not a variable reference.
func (p *DefaultParser) ExtractVariables(input []byte) ([]string, error) {
	matches := findDirectives(input)
	varNames := make(map[string]struct{})

	for _, match := range matches {
		switch match.Type {
		case DirectiveVar, DirectiveIf:
			name := strings.TrimSpace(match.Args)
			if name != "" {
				varNames[name] = struct{}{}
			}
			// DirectiveComment is intentionally excluded - it's a template comment, not a variable
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(varNames))
	for name := range varNames {
		result = append(result, name)
	}

	debug.Debug("[parser] ExtractVariables: found %d unique variable(s): %v", len(result), result)
	return result, nil
}
