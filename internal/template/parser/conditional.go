package parser

import (
	"fmt"
	"strings"
)

// conditionalBlock represents an if/else/endif block structure.
type conditionalBlock struct {
	condition   string // Variable name for the condition
	ifStart     int    // Start position of @ign-if:VAR@
	ifEnd       int    // End position of @ign-if:VAR@
	elseStart   int    // Start position of @ign-else@ (or -1 if no else)
	elseEnd     int    // End position of @ign-else@ (or -1 if no else)
	endifStart  int    // Start position of @ign-endif@
	endifEnd    int    // End position of @ign-endif@
	ifContent   string // Content between if and else (or endif if no else)
	elseContent string // Content between else and endif (empty if no else)
	outerStart  int    // Start of entire block (start of @ign-if:)
	outerEnd    int    // End of entire block (end of @ign-endif@)
}

// processConditionals processes all @ign-if:/@ign-else@/@ign-endif@ blocks in input.
// Returns the processed content with conditional blocks evaluated.
func processConditionals(input []byte, vars Variables) ([]byte, error) {
	text := string(input)
	matches := findDirectives(input)

	// Find and process conditional blocks (innermost first for nested conditionals)
	for {
		block, err := findInnermostConditionalBlock(matches, text)
		if err != nil {
			return nil, err
		}
		if block == nil {
			// No more conditional blocks
			break
		}

		// Evaluate the condition
		conditionValue, err := vars.GetBool(block.condition)
		if err != nil {
			return nil, newParseErrorWithDirective(TypeMismatch,
				fmt.Sprintf("condition variable must be boolean: %s (%v)", block.condition, err),
				"@ign-if:"+block.condition+"@")
		}

		// Choose content based on condition
		var replacement string
		if conditionValue {
			replacement = block.ifContent
		} else {
			replacement = block.elseContent
		}

		// Replace the entire block with the chosen content
		text = text[:block.outerStart] + replacement + text[block.outerEnd:]

		// Re-scan directives since we modified the text
		matches = findDirectives([]byte(text))
	}

	return []byte(text), nil
}

// findInnermostConditionalBlock finds the innermost (most nested) complete conditional block.
// Returns nil if no complete block found, error if unclosed block detected.
func findInnermostConditionalBlock(matches []DirectiveMatch, text string) (*conditionalBlock, error) {
	// Stack-based approach to find innermost matching block
	type stackEntry struct {
		ifMatch   DirectiveMatch
		elseMatch *DirectiveMatch
	}

	var stack []stackEntry
	var innermostBlock *conditionalBlock
	maxDepth := 0

	for _, match := range matches {
		switch match.Type {
		case DirectiveIf:
			depth := len(stack)
			if depth > maxDepth {
				maxDepth = depth
			}
			stack = append(stack, stackEntry{
				ifMatch: match,
			})

		case DirectiveElse:
			if len(stack) == 0 {
				return nil, newParseErrorWithDirective(InvalidDirectiveSyntax,
					"@ign-else@ without matching @ign-if:",
					match.RawText)
			}
			// Record else for the current if
			stack[len(stack)-1].elseMatch = &match

		case DirectiveEndif:
			if len(stack) == 0 {
				return nil, newParseErrorWithDirective(InvalidDirectiveSyntax,
					"@ign-endif@ without matching @ign-if:",
					match.RawText)
			}

			// Pop the stack and create a complete block
			entry := stack[len(stack)-1]
			currentDepth := len(stack) - 1
			stack = stack[:len(stack)-1]

			// Calculate content ranges
			ifContentStart := entry.ifMatch.End
			var ifContentEnd int
			var elseContentStart int
			var elseContentEnd int

			if entry.elseMatch != nil {
				ifContentEnd = entry.elseMatch.Start
				elseContentStart = entry.elseMatch.End
				elseContentEnd = match.Start
			} else {
				ifContentEnd = match.Start
			}

			block := &conditionalBlock{
				condition:  strings.TrimSpace(entry.ifMatch.Args),
				ifStart:    entry.ifMatch.Start,
				ifEnd:      entry.ifMatch.End,
				endifStart: match.Start,
				endifEnd:   match.End,
				ifContent:  text[ifContentStart:ifContentEnd],
				outerStart: entry.ifMatch.Start,
				outerEnd:   match.End,
			}

			if entry.elseMatch != nil {
				block.elseStart = entry.elseMatch.Start
				block.elseEnd = entry.elseMatch.End
				block.elseContent = text[elseContentStart:elseContentEnd]
			}

			// Keep track of the innermost block (deepest nesting)
			if innermostBlock == nil || currentDepth >= maxDepth {
				innermostBlock = block
			}
		}
	}

	// Check for unclosed blocks
	if len(stack) > 0 {
		unclosed := stack[len(stack)-1]
		return nil, newParseErrorWithDirective(UnclosedBlock,
			fmt.Sprintf("unclosed @ign-if:%s@ block (missing @ign-endif@)", unclosed.ifMatch.Args),
			unclosed.ifMatch.RawText)
	}

	return innermostBlock, nil
}
