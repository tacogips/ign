---
allowed-tools: Bash(gh:*), Bash(git:*), Read, Grep, Task, Edit, Write, TodoWrite
description: Fix GitHub Actions errors from the current PR (user)
---

## Context

- Current branch: !`git branch --show-current`
- Default branch: !`git remote show origin | grep 'HEAD branch' | cut -d' ' -f5`
- Existing PR for current branch: !`gh pr view --json number,title,url 2>/dev/null || echo "No PR found"`

## Your task

Fetch the current branch's pull request, retrieve GitHub Actions check failures and error messages, analyze the errors, and fix them.

**This command only works when a PR exists for the current branch.**

## GitHub Actions Error Fix Process

### Step 1: Verify PR exists for current branch

1. **Check if PR exists**:

   ```bash
   gh pr view --json number,title,url,baseRefName,headRefName
   ```

   - If no PR exists: Show error message and exit
   - If PR exists: Extract PR details (number, title, URL, base branch, head branch)

### Step 2: Fetch GitHub Actions check status

1. **Get all checks for the PR**:

   ```bash
   gh pr checks
   ```

   This command shows the status of all checks (GitHub Actions workflows) for the current PR.

2. **Parse check output**:
   - Identify failed checks (status: ✗ or × or fail)
   - Extract check names and their status
   - If all checks are passing: Report success and exit
   - If checks are pending: Wait or notify user
   - If checks failed: Proceed to error analysis

### Step 3: Fetch detailed error logs from failed checks

For each failed check identified in Step 2:

1. **Get detailed check run logs**:

   ```bash
   # List all check runs for the PR
   gh api repos/{owner}/{repo}/commits/{head_sha}/check-runs \
     --jq '.check_runs[] | select(.conclusion == "failure") | {name: .name, id: .id, html_url: .html_url}'
   ```

   - Extract check run IDs for failed checks
   - Get the head SHA from the PR info

2. **Fetch logs for each failed check**:

   ```bash
   # Get logs for a specific check run
   gh api repos/{owner}/{repo}/actions/runs/{run_id}/jobs --jq '.jobs[]'
   ```

   Or use the simpler command:

   ```bash
   gh run view {run_id} --log-failed
   ```

   This retrieves only the failed job logs.

3. **Extract error messages**:
   - Parse the log output to find error messages
   - Look for compilation errors, test failures, lint errors, etc.
   - Common patterns:
     - `error:` or `ERROR:` for compilation/lint errors
     - `FAILED` for test failures
     - `panic:` messages in Go tests for panic handling
     - Stack traces and file:line references

### Step 4: Categorize and analyze errors

1. **Group errors by type**:
   - **Compilation Errors**: Go compiler errors (type errors, undefined variables, etc.)
   - **Test Failures**: Failed unit tests, integration tests
   - **Lint Errors**: Clippy warnings/errors, formatting issues
   - **Build Errors**: Dependency resolution, feature flag issues
   - **Other Errors**: Unexpected failures

2. **For each error, extract**:
   - Error type/category
   - File path and line number
   - Error message
   - Code snippet (if available in logs)
   - Suggested fix direction (from compiler/tool output if available)

3. **Create error summary**:

   ```markdown
   ## GitHub Actions Error Summary

   **PR**: #{number} - {title}
   **URL**: {pr_url}
   **Failed Checks**: {count}

   ### Failed Checks:
   1. {check_name_1} - {failure_reason}
   2. {check_name_2} - {failure_reason}
   ...

   ### Error Details:

   #### Compilation Errors ({count}):
   1. **{file_path}:{line}** - {error_message}
   2. **{file_path}:{line}** - {error_message}
   ...

   #### Test Failures ({count}):
   1. **{test_name}** in {file_path} - {failure_reason}
   2. **{test_name}** in {file_path} - {failure_reason}
   ...

   #### Lint Errors ({count}):
   1. **{file_path}:{line}** - {lint_message}
   ...
   ```

### Step 5: Display error summary and plan fixes

1. **Show the error summary to the user**:

   ```
   🚨 GitHub Actions Errors Found
   ────────────────────────────────────────────────────

   [Display the complete error summary from Step 4]

   ────────────────────────────────────────────────────
   ```

2. **Use TodoWrite to create fix tasks**:
   Create todo items for each error that needs fixing:

   ```
   - Fix compilation error in {file_path}:{line}: {brief_description}
   - Fix test failure: {test_name} - {brief_reason}
   - Fix lint error in {file_path}:{line}: {brief_description}
   ```

   Order by priority:
   - Compilation errors (highest priority - blocking)
   - Test failures (high priority)
   - Lint errors (medium priority)

### Step 6: Fix errors systematically

For each error in priority order:

1. **Mark todo as in_progress**

2. **Read the relevant file**:

   ```
   Use Read tool to view the file containing the error
   ```

3. **Analyze the error context**:
   - Read the surrounding code
   - Understand the function/module structure
   - Check for related files (imports, dependencies)
   - Read error message details from GitHub Actions logs

4. **Apply the fix**:
   - Use Edit tool to make necessary changes
   - Follow Go best practices and CLAUDE.md guidelines
   - Ensure line length stays under 130 characters
   - Handle errors properly with Result<T, E>

5. **Verify the fix locally**:

   ```bash
   # For compilation errors:
   go build ./...

   # For specific package:
   go build ./internal/{package_name}

   # For lint errors:
   golangci-lint run ./...

   # For test failures:
   go test -run TestName -v

   # For formatting issues:
   gofmt -l .
   ```

6. **Mark todo as completed**

7. **Progress update**:

   ```
   ✅ Fixed [{X}/{N}]: {error_description}
   - File: {file_path}:{line}
   - Change: {brief_description_of_fix}
   ```

8. **Handle verification failures**:
   - If local check still fails: Re-analyze the error and try alternative approach
   - If fix causes new errors: Revert and try different solution
   - If fix cannot be automated: Mark as "needs manual intervention" and continue

### Step 7: Final verification

After all fixes have been applied:

1. **Run comprehensive local checks**:

   ```bash
   # Full workspace compilation check
   go build ./...

   # Clippy check
   golangci-lint run ./...

   # Format check
   gofmt -l .

   # Run all tests
   go test ./... -v
   ```

2. **Generate fix summary**:

   ```markdown
   🔧 Fix Summary
   ────────────────────────────────────────────────────

   **PR**: #{number} - {title}
   **URL**: {pr_url}

   ✅ Successfully Fixed:
   - [{count}] Compilation errors
   - [{count}] Test failures
   - [{count}] Lint errors

   📝 Changes Made:
   1. {file_path} - {description_of_fix}
   2. {file_path} - {description_of_fix}
   ...

   ⚠️ Needs Manual Intervention:
   - [{count}] Errors that could not be automatically fixed
   - [List of remaining issues if any]

   📊 Verification Results:
   - Compilation: ✅ Pass / ❌ Fail
   - Clippy: ✅ Pass / ❌ Fail
   - Format: ✅ Pass / ❌ Fail
   - Tests: ✅ Pass / ❌ Fail

   Next Steps:
   1. Review the changes made
   2. Commit and push the fixes to update the PR
   3. Wait for GitHub Actions to re-run checks
   4. Address any remaining manual fixes if needed
   ```

3. **Display the summary**:

   ```
   🎉 Error Fixing Complete
   ────────────────────────────────────────────────────
   [Display the complete fix summary]
   ────────────────────────────────────────────────────
   ```

### Step 8: Commit and push fixes (optional)

**IMPORTANT**: Only perform this step if the user explicitly requests to commit and push.

1. **Stage all changes**:

   ```bash
   git add -A
   ```

2. **Create commit with descriptive message**:

   ```bash
   git commit -m "fix: resolve GitHub Actions errors from PR #{pr_number}

   - Fix compilation errors in {files}
   - Fix test failures: {test_names}
   - Fix lint errors in {files}
   "
   ```

3. **Push to remote**:

   ```bash
   git push origin HEAD
   ```

4. **Confirm push**:

   ```
   ✅ Changes committed and pushed successfully!

   GitHub Actions will automatically re-run checks for this PR.
   You can monitor the status with: gh pr checks
   ```

## Important Notes

- **Current Branch Only**: This command only works for PRs associated with the current branch
  - It does NOT accept PR number arguments
  - Execute in the repository directory with an active PR

- **Error Detection**: The command detects errors from:
  - GitHub Actions workflow failures
  - Compilation errors (go build, go vet)
  - Test failures (go test)
  - Lint errors (golangci-lint, gofmt)

- **Automated Fixing**: The command attempts to automatically fix:
  - Common compilation errors
  - Simple lint violations
  - Formatting issues
  - Some test failures (if the fix is clear from error messages)

- **Manual Intervention**: Some errors may require manual intervention:
  - Complex logic errors
  - Architectural issues
  - Unclear error messages
  - Test failures requiring domain knowledge

- **Progress Tracking**: Use TodoWrite to track fix progress
  - One todo item per error to fix
  - Mark items as in_progress and completed appropriately

- **Verification**: Always verify fixes locally before pushing
  - Run go build, golangci-lint, gofmt, and go test
  - Ensure no new errors are introduced

- **CLAUDE.md Compliance**: All fixes must comply with:
  - Go style guidelines (following gofmt standards)
  - Error handling patterns (Go error interface)
  - Project structure conventions
  - Testing requirements

- **Commit Policy**:
  - **CRITICAL**: Only commit and push if the user explicitly instructs to do so
  - Default behavior: Show the fixes and let the user review
  - If user says "commit" or "push": Then perform Step 8

- **GitHub API Usage**: The command uses:
  - `gh pr view` - Get PR information
  - `gh pr checks` - List check statuses
  - `gh api` - Fetch detailed check run logs and error messages
  - `gh run view` - View workflow run logs

- **Language**: Error analysis and fix summaries should be in English, but follow existing code style
