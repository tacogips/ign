# ign-list Example

This example demonstrates how to use `ign-list.json` for batch template operations.

## What is ign-list?

`ign-list.json` allows you to define multiple templates that should be generated together.
This is useful for:

- Microservices architectures (multiple services from templates)
- Monorepo setups (multiple packages)
- Infrastructure-as-code (multiple environments)

## Usage

```bash
# Initialize all templates from the list
ign build init --from-list .ign-build/ign-list.json

# Build all templates
ign build
```

## Directory Structure After Generation

```
project/
├── .ign-build/
│   └── ign-list.json
├── services/
│   ├── api-gateway/
│   ├── user-service/
│   └── order-service/
└── shared/
    └── common/
```
