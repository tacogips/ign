# CLI Workflow Refactoring Specification

## Overview

Refactor the CLI command structure from the current two-step workflow (`ign build init` → `ign init`) to a new workflow (`ign init` → `ign checkout`).

## Current Workflow (Before)

```bash
# Step 1: Create .ign/ign-var.json
ign build init github.com/owner/repo

# Step 2: Generate project
ign init --output ./my-project
```

## New Workflow (After)

```bash
# Step 1: Create .ign/ign.json and .ign/ign-var.json
ign init github.com/owner/repo

# Step 2: Generate project
ign checkout ./my-project
```

## Configuration File Split

The `.ign/` directory now contains two separate configuration files:

1. **`ign.json`** - Template source reference and content hash
   ```json
   {
     "template": { "url": "...", "path": "...", "ref": "..." },
     "hash": "sha256:...",
     "metadata": { ... }
   }
   ```

2. **`ign-var.json`** - User-provided variable values only
   ```json
   {
     "variables": { "key": "value", ... }
   }
   ```

## Changes Required

### 1. Delete `internal/cli/build.go`

- Remove the entire file
- This eliminates the `ign build` command and its `init` subcommand

### 2. Refactor `internal/cli/init.go`

**Current behavior**: Generates project files from `.ign/ign-var.json`
**New behavior**: Creates `.ign/ign-var.json` from template URL

**Changes**:
- Command signature: `init [URL]` (requires 1 argument)
- Flags:
  - `--ref, -r` (git branch/tag/commit, default: "main")
  - `--force, -f` (backup existing config and reinitialize)
  - Remove: `--output`, `--overwrite`, `--config`, `--dry-run`, `--verbose`
- Logic:
  - Call `app.Init()` instead of current logic
  - Check if `.ign/` exists:
    - If exists and no `--force`: print message and exit
    - If exists and `--force`: backup `ign-var.json` and reinitialize
    - If not exists: create `.ign/ign-var.json`
- Success message:
  ```
  Created: .ign/ign-var.json

  Next steps:
    1. Edit .ign/ign-var.json to set variable values
    2. Run: ign checkout ./my-project
  ```

### 3. Create `internal/cli/checkout.go`

**Purpose**: Generate project files using existing `.ign/ign-var.json`

**Command**: `checkout <path>` (requires 1 argument)
**Flags**:
- `--force, -f` (overwrite existing files)
- `--dry-run, -d` (show what would be generated)
- `--verbose, -v` (detailed output)

**Logic**:
- Verify `.ign/ign-var.json` exists
  - If not: error "Configuration not found. Run 'ign init <url>' first."
- Call `app.Checkout()` with path from argument
- Display generation summary (files created/skipped/overwritten)

**Success message**:
```
Project generated successfully

Summary:
  Created: X files
  Skipped: Y files (already exist)

Project ready at: ./my-project
```

### 4. Rename and Refactor `internal/app/build.go`

**Rename to**: `internal/app/config_init.go`

**Function changes**:
- `BuildInit()` → `Init()`
- `BuildInitOptions` → `InitOptions`
- Update struct fields:
  - Remove: `IgnVersion` (not needed for init)
  - `OutputDir` → always `.ign` (hardcoded)
  - Keep: `URL`, `Ref`, `Force`, `Config`, `GitHubToken`

**Force flag behavior**:
When `Force = true` and `.ign/` exists:
1. Check if `.ign/ign-var.json` exists
2. Find next available backup number (bk1, bk2, bk3, etc.)
3. Rename `ign-var.json` to `ign-var.json.bk{N}`
4. Proceed with initialization

**Backup numbering logic**:
```go
// Find next available backup number
func findNextBackupNumber(dir string) int {
    n := 1
    for {
        backupPath := filepath.Join(dir, fmt.Sprintf("ign-var.json.bk%d", n))
        if _, err := os.Stat(backupPath); os.IsNotExist(err) {
            return n
        }
        n++
    }
}
```

### 5. Rename and Refactor `internal/app/init.go`

**Rename to**: `internal/app/checkout.go`

**Function changes**:
- `Init()` → `Checkout()`
- `InitOptions` → `CheckoutOptions`
- `InitResult` → `CheckoutResult`

**Update struct fields**:
- Remove: `ConfigPath` (always `.ign/ign-var.json`)
- Keep: `OutputDir`, `Overwrite`, `DryRun`, `Verbose`, `GitHubToken`

**Logic changes**:
- Hardcode config path to `.ign/ign-var.json`
- Build directory is always `.ign`

### 6. Update `internal/config/defaults.go`

**Change**:
```go
// Line 40
BuildDir: ".ign",  // was: ".ign"
```

### 7. Update `internal/cli/root.go`

**Changes**:
- Remove: `rootCmd.AddCommand(buildCmd)` (line 62)
- Add: `rootCmd.AddCommand(checkoutCmd)` (after initCmd)
- Update Long description to reflect new workflow:
  ```
  It provides a two-step workflow:
    1. "ign init <url>" - Creates configuration from a template
    2. "ign checkout <path>" - Generates project files using the configuration
  ```

### 8. Replace All `.ign` References

**Files to update** (found via grep):
- `examples/README.md`
- `docs/spec.md`
- `docs/reference/template-syntax.md`
- `internal/template/generator/filter.go`
- `docs/progress/app-workflows.md`
- `docs/reference/configuration.md`
- `docs/reference/cli-commands.md`
- `test/integration/init_test.go`
- `test/integration/build_init_test.go`
- `test/integration/e2e_test.go`
- `docs/implementation/architecture.md`
- `docs/progress/project-generator.md`
- `internal/template/generator/generator_test.go`
- `internal/template/generator/IMPLEMENTATION_SPEC.md`
- `internal/config/config_test.go`

**Replacement**: `.ign` → `.ign`

### 9. Update Test Files

**Files requiring command updates**:
- `test/integration/build_init_test.go` → rename to `config_init_test.go`, update commands
- `test/integration/init_test.go` → rename to `checkout_test.go`, update commands
- `test/integration/e2e_test.go` → update command sequences

**Command mapping**:
- `ign build init <url>` → `ign init <url>`
- `ign init --output <path>` → `ign checkout <path>`

## Implementation Order

1. Create backup specification (this file)
2. Update `internal/config/defaults.go` (simple change)
3. Rename `internal/app/build.go` → `internal/app/config_init.go`, refactor
4. Rename `internal/app/init.go` → `internal/app/checkout.go`, refactor
5. Create `internal/cli/checkout.go`
6. Refactor `internal/cli/init.go`
7. Delete `internal/cli/build.go`
8. Update `internal/cli/root.go`
9. Replace all `.ign` → `.ign` in docs/examples
10. Update test files
11. Run `go mod tidy`, `go build`, `go test`

## Verification Checklist

- [ ] `ign init github.com/owner/repo` creates `.ign/ign-var.json`
- [ ] `ign init` without args shows error (requires URL)
- [ ] `ign init` with existing `.ign/` skips (no --force)
- [ ] `ign init --force` backs up existing config
- [ ] Backup creates `.ign/ign-var.json.bk1`, `.bk2`, etc.
- [ ] `ign checkout .` generates to current directory
- [ ] `ign checkout ./my-project` generates to specified path
- [ ] `ign checkout` without args shows error (requires path)
- [ ] `ign checkout` without `.ign/` shows error
- [ ] All tests pass
- [ ] `go build ./...` succeeds
- [ ] No references to `.ign` remain
- [ ] No references to `ign build` command remain
