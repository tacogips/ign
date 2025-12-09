# App Workflows

**Status**: Completed

## Spec Reference
- docs/implementation/architecture.md Section 4 (Key Workflows)
- docs/reference/cli-commands.md Section 1-2 (Build Init, Project Init)

## Implemented
- [x] BuildInit workflow (`internal/app/build.go`)
- [x] Init workflow (`internal/app/init.go`)
- [x] Variable loading with @file: resolution (`internal/app/variables.go`)
- [x] Common workflow utilities (`internal/app/workflows.go`)
- [x] App-layer errors (`internal/app/errors.go`)
- [x] Unit tests (`internal/app/app_test.go`)
- [x] CLI integration - build.go updated
- [x] CLI integration - init.go updated

## Remaining
- (none - all items complete)

## Design Decisions
- Clean Architecture: CLI -> App -> Domain layer separation
- Validation at app layer for better testability
- @file: paths resolved relative to .ign-build/
- GitHub token resolution: GITHUB_TOKEN env > GH_TOKEN env > gh auth token
- Non-fatal errors accumulated for partial generation

## Notes
- 6 test suites with 31 test cases
- BuildInit creates ign-var.json with empty variables and metadata
- Init supports dry-run mode for preview
- Error messages include context for debugging
- All tests pass in under 0.01s
