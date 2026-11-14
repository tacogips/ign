# Template Parser

**Status**: Completed

## Spec Reference
- docs/implementation/architecture.md Section 3.2 (Template Parser)
- docs/reference/template-syntax.md (Complete directive reference)

## Implemented
- [x] Parser interface (`internal/template/parser/parser.go`)
- [x] ParseContext struct for state management (`internal/template/parser/parser.go`)
- [x] Directive types and matching (`internal/template/parser/directive.go`)
- [x] Variables interface and MapVariables (`internal/template/parser/variable.go`)
- [x] @ign-var:NAME@ substitution (`internal/template/parser/variable.go`)
- [x] @ign-if:/@ign-else@/@ign-endif@ conditionals (`internal/template/parser/conditional.go`)
- [x] @ign-include:PATH@ with circular detection (`internal/template/parser/include.go`)
- [x] @ign-comment:TEXT@ template comment with line removal (`internal/template/parser/comment.go`)
- [x] @ign-raw:CONTENT@ escaping (`internal/template/parser/raw.go`)
- [x] Parser-specific errors (`internal/template/parser/errors.go`)
- [x] Comprehensive unit tests (`internal/template/parser/parser_test.go`)

## Remaining
- (none - all items complete)

## Design Decisions
- Placeholder substitution technique for raw directives to prevent nested processing
- Stack-based matching for nested conditionals
- Maximum include depth of 10 levels (configurable)
- @ign-comment: removes entire line from output (template-only comment)
- Error types include file, line, and directive context

## Notes
- 36 unit tests covering all directive types
- Supports arbitrary nesting of conditional blocks
- Variables interface provides type-safe access (string, int, bool)
- Total implementation: 1,628 lines across 9 files
- All tests pass in under 0.01s
