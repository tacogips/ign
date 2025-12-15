# Core Domain Models

**Status**: Completed

## Spec Reference
- docs/implementation/architecture.md Section 5 (Data Models)
- docs/reference/configuration.md (Schema definitions in Appendix A)

## Implemented
- [x] VarType constants (`internal/template/model/types.go:6-14`)
- [x] TemplateRef struct (`internal/template/model/types.go:17-29`)
- [x] TemplateFile struct (`internal/template/model/types.go:32-40`)
- [x] IgnJson struct with all fields (`internal/template/model/ignconfig.go:4-23`)
- [x] VarDef struct for variable definitions (`internal/template/model/ignconfig.go:26-43`)
- [x] TemplateSettings struct (`internal/template/model/ignconfig.go:46-57`)
- [x] IgnVarJson struct (`internal/template/model/ignvar.go:6-13`)
- [x] TemplateSource struct (`internal/template/model/ignvar.go:16-23`)
- [x] VarMetadata struct (`internal/template/model/ignvar.go:26-37`)
- [x] Template runtime struct (`internal/template/model/template.go:4-13`)
- [x] Unit tests for all structs (`internal/template/model/model_test.go`)

## Remaining
- (none - all items complete)

## Design Decisions
- Used `interface{}` for VarDef.Default and VarDef.Example to support string, int, and bool values
- Used pointer types (`*int`) for VarDef.Min and VarDef.Max to distinguish "not set" from "zero value"
- Used `os.FileMode` for TemplateFile.Mode to properly represent file permissions
- All JSON tags use `omitempty` for optional fields to produce clean JSON output
- All models in single package `internal/template/model` for cohesion

## Notes
- 10 unit tests pass covering all major data structures
- JSON marshaling/unmarshaling verified with round-trip tests
- Tests include examples from reference documentation
- Package follows Standard Go Project Layout with models in /internal/
