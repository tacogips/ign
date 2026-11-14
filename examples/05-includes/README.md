# @ign-var:project_name@

## About

This project demonstrates the `@ign-include:` directive for reusable template components.

## Structure

```
@ign-var:project_name@/
├── _includes/              # Shared include files (template source only)
│   ├── license-header.txt  # License header for source files
│   ├── generated-warning.txt
│   └── common-imports.txt
├── cmd/app/
│   └── main.go             # Uses variables in comments
├── internal/service/
│   └── service.go          # Also uses same pattern
├── config/
│   └── app.yaml            # Config file using includes
└── LICENSE
```

## Include Directive Usage

Include shared content with absolute paths (from template root):
```
@ign-raw:@ign-include:/_includes/license-header.txt@@
```

Or relative paths:
```
@ign-raw:@ign-include:../_includes/common-imports.txt@@
```

## License Header

The following is included from `_includes/license-header.txt`:

@ign-include:/_includes/license-header.txt@

## License

See LICENSE file for full text.
