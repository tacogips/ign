# Template Provider

**Status**: Completed

## Spec Reference
- docs/implementation/architecture.md Section 3.1 (Template Provider interface)
- docs/spec.md Section 2.3 (Template Sources)
- docs/reference/cli-commands.md Section 1.2 (URL Formats)

## Implemented
- [x] Provider interface (`internal/template/provider/provider.go`)
- [x] GitHubProvider with tarball download (`internal/template/provider/github.go`)
- [x] LocalProvider with security checks (`internal/template/provider/local.go`)
- [x] Provider factory with auto-detection (`internal/template/provider/factory.go`)
- [x] URL parsing utilities (`internal/template/provider/url.go`)
- [x] Provider-specific errors (`internal/template/provider/errors.go`)
- [x] Unit tests (`internal/template/provider/provider_test.go`)

## Remaining
- (none - all items complete)

## Design Decisions
- Used standard library only for HTTP and archive extraction
- Context support for cancellation/timeout
- Factory pattern for automatic provider selection
- Six distinct error types for specific failure scenarios
- Binary file detection using null byte heuristic

## Notes
- Supports all GitHub URL formats (https, git@, github.com/, owner/repo)
- Local provider rejects absolute paths and ".." for security
- GitHub token support via environment or configuration
- 32 unit tests covering URL parsing and path validation
- All tests pass in under 0.01s
