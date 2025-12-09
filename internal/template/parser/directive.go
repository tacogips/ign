package parser

import (
	"regexp"
)

// DirectiveType identifies the type of template directive.
type DirectiveType int

const (
	// DirectiveVar represents @ign-var:NAME@
	DirectiveVar DirectiveType = iota
	// DirectiveComment represents @ign-comment:NAME@
	DirectiveComment
	// DirectiveRaw represents @ign-raw:CONTENT@
	DirectiveRaw
	// DirectiveIf represents @ign-if:VAR@
	DirectiveIf
	// DirectiveElse represents @ign-else@
	DirectiveElse
	// DirectiveEndif represents @ign-endif@
	DirectiveEndif
	// DirectiveInclude represents @ign-include:PATH@
	DirectiveInclude
)

// String returns the string representation of the directive type.
func (dt DirectiveType) String() string {
	switch dt {
	case DirectiveVar:
		return "var"
	case DirectiveComment:
		return "comment"
	case DirectiveRaw:
		return "raw"
	case DirectiveIf:
		return "if"
	case DirectiveElse:
		return "else"
	case DirectiveEndif:
		return "endif"
	case DirectiveInclude:
		return "include"
	default:
		return "unknown"
	}
}

// DirectiveMatch represents a matched directive in text.
type DirectiveMatch struct {
	// Type is the directive type.
	Type DirectiveType
	// Start is the starting byte index in the input.
	Start int
	// End is the ending byte index in the input (exclusive).
	End int
	// Name is the directive name (e.g., "var", "if").
	Name string
	// Args is the argument string (content between : and @).
	Args string
	// RawText is the original matched text.
	RawText string
}

var (
	// Regex patterns for matching directives
	// Pattern: @ign-DIRECTIVE:ARGS@ or @ign-DIRECTIVE@
	// Note: For @ign-raw:, we need special handling since content can contain @
	directivePattern = regexp.MustCompile(`@ign-([a-z]+)(?::([^@]*))?@`)

	// Special pattern for @ign-raw: that allows @ in content
	// Matches @ign-raw:CONTENT@ where CONTENT can contain @ symbols
	// The trick: content should match up to and including embedded directives
	// Example: @ign-raw:@ign-var:name@@ has content "@ign-var:name@" and closes with final @
	// We use non-greedy (.*?) followed by @ to capture content, then match the closing @
	rawDirectivePattern = regexp.MustCompile(`@ign-raw:(.*@)@`)
)

// findDirectives scans input and returns all directive matches in order.
func findDirectives(input []byte) []DirectiveMatch {
	text := string(input)

	var matches []DirectiveMatch

	// First, find all raw directives using the special pattern
	rawMatches := rawDirectivePattern.FindAllStringSubmatchIndex(text, -1)
	rawMatchPositions := make(map[int]bool)

	for _, match := range rawMatches {
		// match[0], match[1]: full match start, end (@ign-raw:CONTENT@@)
		// match[2], match[3]: content group start, end (CONTENT)

		args := text[match[2]:match[3]]

		matches = append(matches, DirectiveMatch{
			Type:    DirectiveRaw,
			Start:   match[0],
			End:     match[1],
			Name:    "raw",
			Args:    args,
			RawText: text[match[0]:match[1]],
		})

		// Mark this range as a raw directive to avoid double-matching
		for i := match[0]; i < match[1]; i++ {
			rawMatchPositions[i] = true
		}
	}

	// Then find all other directives
	allMatches := directivePattern.FindAllStringSubmatchIndex(text, -1)

	for _, match := range allMatches {
		// Skip if this position overlaps with a raw directive
		if rawMatchPositions[match[0]] {
			continue
		}

		// match[0], match[1]: full match start, end
		// match[2], match[3]: directive name group start, end
		// match[4], match[5]: args group start, end (may be -1 if no args)

		directiveName := text[match[2]:match[3]]

		// Skip raw directives here since we already handled them
		if directiveName == "raw" {
			continue
		}

		var args string
		if match[4] != -1 && match[5] != -1 {
			args = text[match[4]:match[5]]
		}

		// Determine directive type
		dirType := parseDirectiveType(directiveName)

		matches = append(matches, DirectiveMatch{
			Type:    dirType,
			Start:   match[0],
			End:     match[1],
			Name:    directiveName,
			Args:    args,
			RawText: text[match[0]:match[1]],
		})
	}

	// Sort matches by start position to maintain order
	sortDirectiveMatches(matches)

	return matches
}

// sortDirectiveMatches sorts matches by start position.
func sortDirectiveMatches(matches []DirectiveMatch) {
	// Simple bubble sort (good enough for small arrays)
	n := len(matches)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if matches[j].Start > matches[j+1].Start {
				matches[j], matches[j+1] = matches[j+1], matches[j]
			}
		}
	}
}

// parseDirectiveType converts directive name string to DirectiveType.
func parseDirectiveType(name string) DirectiveType {
	switch name {
	case "var":
		return DirectiveVar
	case "comment":
		return DirectiveComment
	case "raw":
		return DirectiveRaw
	case "if":
		return DirectiveIf
	case "else":
		return DirectiveElse
	case "endif":
		return DirectiveEndif
	case "include":
		return DirectiveInclude
	default:
		return DirectiveType(-1) // Unknown directive
	}
}
