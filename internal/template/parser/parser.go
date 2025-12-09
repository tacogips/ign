package parser

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// Parser processes template files and substitutes variables.
type Parser interface {
	// Parse processes a template file with variable substitution.
	Parse(ctx context.Context, input []byte, vars Variables) ([]byte, error)

	// ParseWithContext processes a template with full parse context.
	ParseWithContext(ctx context.Context, input []byte, pctx *ParseContext) ([]byte, error)

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
	return parseInternal(ctx, input, pctx)
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
	result, rawContent, err := extractRawDirectives(result)
	if err != nil {
		return nil, err
	}

	// Step 2: Process @ign-include: directives (recursive)
	result, err = processIncludes(ctx, result, pctx)
	if err != nil {
		return nil, err
	}

	// Step 3: Process @ign-if:/@ign-else@/@ign-endif@ blocks
	result, err = processConditionals(result, pctx.Variables)
	if err != nil {
		return nil, err
	}

	// Step 4: Process @ign-comment: directives (line-by-line to preserve context)
	result, err = processCommentDirectivesInText(result, pctx.Variables)
	if err != nil {
		return nil, err
	}

	// Step 5: Process @ign-var: directives
	result, err = processVarDirectivesInText(result, pctx.Variables)
	if err != nil {
		return nil, err
	}

	// Step 6: Restore raw content from placeholders
	result = restoreRawContent(result, rawContent)

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

// processCommentDirectivesInText processes all @ign-comment:NAME@ directives.
// This must be line-by-line to correctly detect and remove comment markers.
func processCommentDirectivesInText(input []byte, vars Variables) ([]byte, error) {
	lines := bytes.Split(input, []byte("\n"))
	var result [][]byte

	for _, line := range lines {
		lineStr := string(line)
		matches := findDirectives(line)

		// Check if this line has @ign-comment: directives
		hasComment := false
		for _, match := range matches {
			if match.Type == DirectiveComment {
				hasComment = true
				break
			}
		}

		if hasComment {
			// Process comment directives in this line
			processed, err := processCommentDirectivesInLine(lineStr, vars, matches)
			if err != nil {
				return nil, err
			}
			result = append(result, []byte(processed))
		} else {
			result = append(result, line)
		}
	}

	return bytes.Join(result, []byte("\n")), nil
}

// processCommentDirectivesInLine processes @ign-comment: directives in a single line.
func processCommentDirectivesInLine(line string, vars Variables, matches []DirectiveMatch) (string, error) {
	// Process directives from last to first to maintain positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if match.Type != DirectiveComment {
			continue
		}

		// Process the comment directive with the entire line for context
		processed, err := processCommentDirective(match.Args, vars, line)
		if err != nil {
			return "", err
		}

		return processed, nil
	}

	return line, nil
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
		case DirectiveVar, DirectiveComment:
			if strings.TrimSpace(match.Args) == "" {
				return newParseErrorWithDirective(InvalidDirectiveSyntax,
					"variable name is empty",
					match.RawText)
			}
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
func (p *DefaultParser) ExtractVariables(input []byte) ([]string, error) {
	matches := findDirectives(input)
	varNames := make(map[string]struct{})

	for _, match := range matches {
		switch match.Type {
		case DirectiveVar, DirectiveComment, DirectiveIf:
			name := strings.TrimSpace(match.Args)
			if name != "" {
				varNames[name] = struct{}{}
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(varNames))
	for name := range varNames {
		result = append(result, name)
	}

	return result, nil
}
