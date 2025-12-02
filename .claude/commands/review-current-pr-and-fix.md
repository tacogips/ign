---
description: Review the current directory's PR and fix identified issues (user)
---

## Context

- Current branch: !`git branch --show-current`
- Default branch: !`git remote show origin | grep 'HEAD branch' | cut -d' ' -f5`
- Existing PR for current branch: !`gh pr view --json number,title,isDraft,url 2>/dev/null || echo "No PR found"`

## Arguments

This command accepts optional instruction arguments that customize the review scope:

**Format**: `/review-current-pr-and-fix [instruction]`

**Examples**:
- `/review-current-pr-and-fix` - Review all changes in the PR (default behavior)
- `/review-current-pr-and-fix Review only files in pkg/document/` - Review only files in pkg/document/
- `/review-current-pr-and-fix Review only test files` - Review only test files

**Important**: Instructions apply to review phase only. Files excluded from review will not have issues identified.

## Your Task

Review the pull request for the current directory's branch, identify all review targets (diff sections and review comments), delegate each target to the review-single-target agent for analysis, then fix all identified issues in a separate review branch.

**This command only works when a PR exists for the current branch.**

**Context Conservation Strategy:**
- This command's role is to orchestrate the workflow by:
  1. Extracting diff file names from the PR
  2. Delegating review tasks to `review-single-target` agents
  3. Collecting PR comment URLs from agent responses
  4. Delegating fixes to `apply-pr-review-chunk` agents by package
- Do NOT keep detailed context about fixes in this command
- Response comments only need the review PR URL - no fix details
- Let subagents handle all detailed analysis and implementation

## Workflow Overview

This command supports two modes:

### Normal Mode (New Review)
- Current branch does NOT end with `_review_{n}`
- Performs full review phase (Steps 1-5)
- Creates new review branch with pattern `{original_branch}_review_{n}`
- Posts new review comments to the original PR
- Creates a new PR from review branch back to original branch

### Continuation Mode (Resume Incomplete Fixes)
- Current branch DOES end with `_review_{n}`
- Extracts original branch name and review comment URLs from existing PR
- Analyzes which comments are complete/incomplete/failed
- Continues with apply-pr-review-chunk to address remaining issues
- Updates existing PR with additional fixes

---

## Pull Request Review and Fix Process

### Step 0: Check Current Branch and Prepare Review Branch

**Check if on review branch:**

Use `.claude/scripts/check-review-branch.sh` to:
- Detect if current branch ends with `_review_{n}` pattern
- Extract original branch name if in continuation mode
- Set SKIP_BRANCH_PREP, CONTINUATION_MODE, REVIEW_BRANCH flags

**If CONTINUATION_MODE=true** (Continuation Mode Flow):

1. **Verify PR exists** for current review branch:
   ```bash
   gh pr view --json number,title,body,baseRefName,headRefName,isDraft,url,reviews
   ```
   - If no PR: Exit with error (review branch should have a PR)
   - If PR exists: Extract PR details

2. **Extract review comment URLs** from PR body:
   - Parse "Review Target Comments" section
   - Extract URLs with format `https://github.com/owner/repo/pull/{pr_number}#discussion_r{id}`
   - Store as REVIEW_COMMENT_URLS array
   - **Error handling**:
     - If "Review Target Comments" section missing: Exit with error "PR body does not contain review target comments section"
     - If section exists but no valid URLs found: Exit with error "No valid review comment URLs found in PR body"
     - If URL format is invalid: Skip malformed URLs and log warning, continue with valid URLs
     - URL validation: Must match pattern `https://github.com/[owner]/[repo]/pull/[number]#discussion_r[id]`

3. **Fetch review comment contents** using `mcp__github-insight-mcp__get_pull_request_details`

4. **Analyze current fix status**:
   - For each review comment, check if issue is fixed
   - Categorize: [OK] Complete, [!] Incomplete, [X] Failed
   - **Create status report** and display to user:
     ```
     ## [SYNC] Continuation Mode Status Report

     Review Branch: {current_review_branch}
     Original Branch: {extracted_original_branch}
     Original PR: {original_pr_url}

     ### Fix Status Summary
     Total review comments: {total_count}
     - [OK] Complete: {complete_count} ({percentage}%)
     - [!] Incomplete: {incomplete_count} ({percentage}%)
     - [X] Failed: {failed_count} ({percentage}%)

     ### Detailed Status
     {for each comment:}
     {status_emoji} {comment_url}
        File: {file_path}:{line}
        Status: {Complete|Incomplete|Failed}
        {if incomplete/failed: Reason: {reason}}
     ```

5. **Determine continuation action**:
   - If all complete: Display success message and exit
   - If incomplete/failed: Store as PENDING_COMMENT_URLS and proceed to Step 1 with message "Continuing with {count} pending fixes"

**If CONTINUATION_MODE=false** (Normal Mode):

1. **Check for uncommitted changes**:
   ```bash
   git status --porcelain
   ```
   - If any changes: Exit with error

2. **Find available review branch name** using `.claude/scripts/find-available-branch.sh`:
   - Checks both local and remote branches
   - Finds first available `{current_branch}_review_{n}` number
   - Store as REVIEW_BRANCH

3. **Store original branch**: `ORIGINAL_BRANCH="$CURRENT_BRANCH"`

### Step 1: Verify PR Exists

```bash
gh pr view --json number,title,body,baseRefName,headRefName,isDraft,url,reviews
```

Extract:
- PR number, base branch, head branch
- PR title, body, draft status, URL

**Important**: Review comments will be posted to:
- SKIP_BRANCH_PREP=true: PR for current review branch
- SKIP_BRANCH_PREP=false: PR for ORIGINAL_BRANCH

### Step 2: Collect Review Targets

**Parse optional instruction argument:**
- If provided: Store as REVIEW_INSTRUCTION
- Pass to review-single-target agent for each file
- Use for file filtering

**Collect review targets:**

1. **Get diff sections from GitHub PR**:
   ```bash
   gh pr diff <pr-number>
   ```
   - **IMPORTANT**: Fetch from GitHub, NOT local git diff
   - **Apply instruction-based filtering** if REVIEW_INSTRUCTION provided:
     - Filtering mechanism: Use glob pattern matching against file paths
     - Supported patterns:
       - Exact paths: `pkg/document/service.go`
       - Directory patterns: `pkg/document/` or `pkg/document/**`
       - Wildcard patterns: `**/*_test.go`, `**/*.md`
       - Multiple patterns: Combined with OR logic (matches any pattern)
     - Pattern matching is applied BEFORE default skip rules
     - Example: `pkg/document/` includes all files under that directory
     - Example: `**/*_test.go` includes all test files across all directories
   - Skip binary files, files > 10,000 lines, non-reviewable extensions (applied AFTER instruction filtering)
   - Track skipped files with reasons (instruction filter vs. default skip rules)

2. **Get review comments**:
   ```bash
   gh api repos/{owner}/{repo}/pulls/<pr-number>/comments
   ```
   - Extract: file path, line number, comment text, author, comment ID, URL

3. **Create consolidated list** with both diff sections and review comments

### Step 3: Display Review Targets Summary

Show summary including:
- Review instruction (if provided)
- Diff sections count
- Review comments count
- Skipped files with reasons
- List of files to review
- List of review comments

### Step 4: Review Each Target

**Phase 4.1: Single-File Review**

**For each review target**, launch `review-single-target` agent:

**If target type is "diff":**

```
Review the changes in file: <file-path>

Context:
- PR: #<number> - <title>
- Repository: <repository-url>
- Base/Head branches: <base>/<head>
- File: <file-path>

[If REVIEW_INSTRUCTION provided:]
Special Instruction: <REVIEW_INSTRUCTION>

Task:
Analyze the diff and identify issues. Post review comments to GitHub PR using gh api.

For each issue found, post a detailed review comment:
- Repository: <owner>/<repo>
- PR number: <pr-number>
- Commit SHA: <head-commit-sha>

The diff content from GitHub PR:
```diff
<diff-content-from-github>
```

Remember:
- Read entire file for context
- Read complete functions containing changes
- Find and read all caller functions
- Check test coverage
- Post review comment for each issue using gh api
- Include comment URL in your response
```

**If target type is "review-comment":**

```
Verify review comment status for file: <file-path>

Context:
- PR: #<number> - <title>
- File: <file-path>:<line>
- Comment: "<comment-text>" (@<author>)

[If REVIEW_INSTRUCTION provided:]
Special Instruction: <REVIEW_INSTRUCTION>

Task:
Check if this review comment has been addressed. If not, include as an issue and post review comment.
```

**Collect agent responses:**
- Issues found with: file, line, severity, problem, direction, PR comment URL
- Track progress: `[v] Reviewed [X/N]: <file> - Found Y issues, posted Y PR comments`

**Phase 4.2: Cross-File Review**

1. **Collect related file chunks** using `collect-relative-files-in-pr` agent:

```
Analyze PR changes to identify groups of related files for cross-file consistency review.

Context:
- PR: #<number> - <title>
- Repository: <repository-url>
- Base/Head branches: <base>/<head>

Task:
Identify groups of related files (caller/callee, interface/implementation, etc.) that should be reviewed together. Return chunks of 2-5 related files each.
```

**Expected output**: List of file chunks with relationship types

2. **For each file chunk**, launch `review-multiple-target` agent:

```
Review cross-file consistency for related files.

Context:
- PR: #<number> - <title>
- Repository: <repository-url>
- Base/Head branches: <base>/<head>
- Chunk: <chunk-id>
- Relationship type: <relationship-type>

Files to review:
- <file1-path> (<role1>)
- <file2-path> (<role2>)
- <file3-path> (<role3>)

Relationship description:
<description>

Review focus:
<focus-areas>

[If REVIEW_INSTRUCTION provided:]
Special Instruction: <REVIEW_INSTRUCTION>

Task:
Analyze these files for cross-file consistency issues. Check for:
- Interface/implementation mismatches
- Type inconsistencies across layers
- Contract violations
- Incomplete change propagation
- Architectural layer violations

Post review comments to GitHub PR for each cross-file issue found.

For each issue found, post a detailed review comment:
- Repository: <owner>/<repo>
- PR number: <pr-number>
- Commit SHA: <head-commit-sha>
- Post on the primary file where fix should be applied
- Mention all affected files in comment body

Remember:
- Focus only on cross-file issues (single-file issues handled by review-single-target)
- Read all files in the chunk completely
- Post review comment for each cross-file issue using gh api
- Include comment URL in your response
```

**Collect agent responses:**
- Cross-file issues found with: affected files, severity, problem, direction, PR comment URL
- Track progress: `[v] Cross-file review [X/N chunks]: Chunk Y - Found Z issues, posted Z PR comments`

**Combine results from both phases:**
- Merge single-file issues and cross-file issues
- Deduplicate if same issue identified in both phases
- Track total PR comment URLs from both phases

### Step 5: Consolidate Review Findings

1. **Categorize by severity** using standardized criteria:
   - **Critical**: Issues causing crashes, data loss, security vulnerabilities, or blocking deployment
     - Examples: Null pointer dereferences, SQL injection risks, authentication bypasses
   - **High**: Issues breaking core functionality, significant performance problems, or incorrect behavior
     - Examples: API returning wrong results, memory leaks, broken error handling
   - **Medium**: Issues causing degraded UX, maintainability concerns, or minor bugs
     - Examples: Poor error messages, code duplication, missing edge case handling
   - **Low**: Style issues, documentation gaps, minor inconsistencies
     - Examples: Naming inconsistencies, missing comments, formatting issues

2. **Generate review report** using template `.claude/templates/review-report-format.md`:
   - Populate with PR details, issue counts, categorized issues
   - Include review comments status (fixed/not fixed/partial)
   - Apply severity criteria consistently across all issues

3. **Display the report**

### Step 6: Create Review Branch

**Only if SKIP_BRANCH_PREP=false:**

```bash
git checkout -b "$REVIEW_BRANCH"
```

**If SKIP_BRANCH_PREP=true:** Already on review branch

**Store branch information:**
- REVIEW_BRANCH: Branch where fixes will be committed
- ORIGINAL_BRANCH: Base branch for PR

### Step 7: Group Issues and Delegate to Modify-Chunk Agents

1. **Determine repository URL** from git remote

2. **Determine which PR comment URLs to process:**
   - CONTINUATION_MODE=true: Use PENDING_COMMENT_URLS (incomplete/failed only)
   - CONTINUATION_MODE=false: Use all PR comment URLs from Step 4

3. **Group PR comment URLs by package:**
   - Extract package name from file path (e.g., `internal/{package_name}/`)
   - Group comment URLs by package

4. **For each package with issues**, launch `apply-pr-review-chunk` agent:

```
Implement fixes for issues identified in package: {package_name}

Context:
- PR: #{pr_number} - {title}
- Repository: {repository_url}
- Package: {package_name}
- Number of issues: {issue_count}

GitHub PR Comment URLs with detailed fix instructions:
{list of comment URLs for this package}

Task:
1. Fetch and process each PR comment URL to extract modification instructions
2. Implement fixes for all issues in this package
3. Run `make check` or verification commands after changes
4. Run `make test` to verify all tests pass
5. Report completion status and any blockers

Remember:
- Focus only on files within the assigned package
- Follow CLAUDE.md guidelines
- Do not fix issues in other packages
- Stop if unrelated errors block progress
```

5. **Collect agent responses:**
   - Fixes applied, files modified, compilation/test status, blockers
   - Track progress per package

6. **Handle failures:**
   - Note blockers and continue with other packages
   - Mark issues needing manual intervention

### Step 8: Pre-Commit Verification and Commit Fixes

1. **Re-check review branch availability** using `.claude/scripts/verify-and-rename-branch.sh`:
   - Handles case where another team member created same branch during review
   - Increments suffix number if collision detected
   - Creates new branch and checks it out
   - Updates FINAL_REVIEW_BRANCH
   - **Race condition handling**:
     - Script checks both local and remote branches atomically
     - If collision detected: increment suffix and retry (max 10 attempts)
     - If max retries exceeded: Exit with error "Unable to find available review branch name"
     - Store final branch name in FINAL_REVIEW_BRANCH for use in Step 9

2. **Prepare commit message** with conditional sections:
   - **Validate all required placeholders** are available:
     - PR number (required)
     - At least one modified comment URL (required)
     - Files modified list (required)
   - **Use conditional sections** based on data availability:
     - Compilation status:
       - If checked and passed: `Compilation status: [OK] Passed`
       - If checked and failed: `Compilation status: [X] Failed - <error summary>`
       - If not checked (blocked): `Compilation status: [!] Not checked (blocked by out-of-scope errors)`
     - Test status:
       - If run and passed: `Test status: [OK] Passed (X tests)`
       - If run and failed: `Test status: [X] Failed (X passed, Y failed)`
       - If not run (blocked): `Test status: [!] Not run (blocked by compilation errors)`
     - Issues section:
       - Only include if there are issues requiring manual intervention
       - Omit section entirely if no manual intervention needed
   - **Format comment modifications**:
     - Complete: `- {URL}: {description}`
     - Incomplete: `- {URL}: (INCOMPLETE) {reason}`
     - Failed: `- {URL}: (FAILED) {reason}`

3. **Commit all fixes:**
   ```bash
   git add -A

   git commit -m "$(cat <<'EOF'
   fix: address code review findings for PR #<pr_number>

   Modifications based on review comments:
   - <Comment URL 1>: <Brief description>
   - <Comment URL 2>: <Brief description>
   - <Comment URL 3>: (INCOMPLETE) <Reason>

   Files modified:
   - <file_path> (+X, -Y)

   Compilation status: <conditional status>
   Test status: <conditional status>

   [Optional section - only if manual intervention needed:]
   Issues requiring manual intervention:
   - <Issue description>
   EOF
   )"
   ```

4. **Store commit information** for PR body

### Step 9: Push and Create/Update Pull Request

1. **Push review branch with error handling:**
   ```bash
   if ! git push origin "$FINAL_REVIEW_BRANCH"; then
     echo "Error: Failed to push branch to remote"
     echo "Possible causes: network issue, permission denied, remote hook rejection"
     exit 1
   fi
   ```
   - **Error handling**:
     - Network failures: Display error and exit (user should retry after network recovery)
     - Permission issues: Display error and verify GitHub authentication
     - Push rejection: Check for hook failures or branch protection rules
   - **Verification**: Confirm push succeeded before proceeding to PR creation

2. **Determine PR action:**
   - CONTINUATION_MODE=true: UPDATE existing PR
   - CONTINUATION_MODE=false: CREATE new PR

3. **Prepare review comment URLs list:**
   - Collect all PR comment URLs from Step 4 (both single-file and cross-file reviews)
   - Format as newline-separated list

4. **Get original PR URL:**
   - Extract from Step 1 (PR URL field)

5. **Create or Update PR using generate-pr agent:**

   **For CREATE mode:**

   Use Task tool with subagent_type='Plan' to delegate to generate-pr agent:

   ```
   Create a PR for review fixes using the generate-pr agent.

   Context:
   - Base branch: {ORIGINAL_BRANCH}
   - Head branch: {FINAL_REVIEW_BRANCH} (already pushed to remote)
   - Original PR: {original_pr_url}
   - Review comment URLs to include in PR body:
   {review_comment_urls_list}

   Task:
   Call the generate-pr agent (.claude/agents/generate-pr.md) with the following information:
   - Pass base branch as: {ORIGINAL_BRANCH}
   - Pass original PR URL in issue URLs list
   - Pass all review comment URLs in the description section
   - Use State:Draft for initial creation

   The agent will:
   1. Analyze the commits and changes
   2. Generate PR title and body in Japanese
   3. Include review comment URLs in a special section
   4. Create the PR and return the URL

   Return the created PR URL.
   ```

   **Agent invocation parameters:**
   - subagent_type: 'Plan'
   - Provide all necessary context for the agent to work independently
   - Expect PR URL as output

   **For UPDATE mode:**

   Use Task tool with subagent_type='Plan' to delegate to generate-pr agent:

   ```
   Update existing PR for review fixes using the generate-pr agent.

   Context:
   - Current review branch: {FINAL_REVIEW_BRANCH}
   - Original PR: {original_pr_url}
   - Review comment URLs to add to PR body:
   {additional_review_comment_urls_list}

   Task:
   Call the generate-pr agent (.claude/agents/generate-pr.md) with:
   - Original PR URL in issue URLs list
   - Additional review comment URLs in the description section
   - No state change (preserve current state)

   The agent will:
   1. Preserve existing PR content
   2. Add new review comment URLs
   3. Update file statistics table
   4. Return updated PR URL

   Return the updated PR URL.
   ```

   **Error recovery procedures**:
   - **Agent fails**: Fall back to manual PR creation with basic body
   - **Rate limit exceeded**: Display wait time and suggest retry
   - **API errors**: Log error details and suggest checking GitHub status

6. **Post response comments** to completed fixes on **original PR**:

   For each successfully fixed review comment:

   a. **Identify completed fixes** using these criteria:
      - **Fully completed** means ALL of the following are true:
        1. Code changes address all points mentioned in the review comment
        2. Compilation passed for the modified package (no in-scope compilation errors)
        3. All new/modified tests passed (pre-existing test failures don't block completion)
        4. No out-of-scope errors blocking verification of the fix
      - **NOT completed** if any of these are true:
        - Only some points in multi-part review comment were addressed
        - Compilation fails due to errors in modified code (in-scope errors)
        - New/modified tests fail
        - Fix blocked by out-of-scope compilation errors (cannot verify)
      - **Edge case handling**:
        - Pre-existing test failures (not caused by this fix): Still considered complete if new tests pass
        - Out-of-scope compilation errors: Mark as incomplete with reason "Blocked by errors in {other_package}"
        - Partial multi-point fixes: Mark as incomplete with reason "Addressed {X} of {Y} points"

   b. **Post simple reply** using `.claude/scripts/post-response-comment.sh`:
      - Uses template `.claude/templates/response-comment-format.md`
      - Posts as reply to review comment thread
      - Only includes review PR URL (no fix details needed)

   c. **Track posted responses** for final report

   **Important**:
   - Only post for **fully completed** fixes (all criteria met)
   - Post as REPLIES to review comments (inline code comments)
   - Post to **original PR**, not review fixes PR
   - Use Japanese for all content
   - Do NOT gather fix details - only PR URL is needed
   - Let apply-pr-review-chunk agents handle all fix context

7. **Display comprehensive result summary** using template `.claude/templates/review-summary.md`:
   - Populate all placeholders with actual data
   - Show: mode, branches, PR URLs, comment status, file changes, verification results, next actions
   - Make review fixes PR URL prominent at top

### Step 10: Complete Workflow

1. **Check out back to original branch:**
   ```bash
   git checkout "$ORIGINAL_BRANCH"
   ```
   - Returns working directory to original branch
   - Leaves repository in clean state

---

## Important Notes

### Review Branch Workflow Modes

**Normal Mode** creates new review branch:
- Current branch does NOT end with `_review_{n}`
- Creates `{original_branch}_review_{n}` branch
- Full review phase (Steps 1-5)
- Posts new review comments to original PR
- Creates new PR from review branch to original

**Continuation Mode** resumes on existing branch:
- Current branch DOES end with `_review_{n}`
- Extracts original branch and review comment URLs from PR body
- Analyzes completion status of comments
- Continues with incomplete/failed fixes only
- Updates existing PR

### Key Features

- **Branch Collision Handling**: Pre-commit verification prevents conflicts if another team member creates same branch
- **GitHub PR Comments**: All issues posted as inline review comments with URLs for traceability
- **Response Comments**: Completed fixes get response comments posted to original PR review threads
- **Comprehensive Summary**: Detailed Japanese summary at end showing all results and next actions
- **File Skipping**: Binary files, large files (>10k lines), non-reviewable extensions automatically skipped
- **Package-Based Delegation**: Fixes grouped by package for independent processing
- **Continuation Support**: Can resume incomplete reviews by re-running command on review branch

### Critical Requirements

- **Always fetch diff from GitHub PR** (`gh pr diff`), NOT local git diff
- **Post review comments to PR** for every issue found (using gh api)
- **Extract comment URLs** from agent responses for fix delegation
- **Check uncommitted changes** before starting (exit if any found)
- **Use external templates/scripts** to keep command file concise
- **Follow CLAUDE.md guidelines** for all fixes
- **Use standard go command flags** for Go commands (no environment variables needed)

### Review Comment URL Format

- Format: `https://github.com/owner/repo/pull/{pr_number}#discussion_r{review_comment_id}`
- Note: Uses `#discussion_r{id}` (review comments), not `#issuecomment-{id}` (issue comments)
- Passed to apply-pr-review-chunk agents
- apply-pr-review-chunk uses github-insight-mcp to fetch comment content

### Multi-Agent Architecture

1. **review-single-target**: Analyzes individual files and posts PR comments for single-file issues
2. **collect-relative-files-in-pr**: Identifies groups of related files that should be reviewed together
3. **review-multiple-target**: Analyzes cross-file consistency and posts PR comments for integration issues
4. **apply-pr-review-chunk**: Implements fixes per package using PR comment URLs
5. **review-current-pr-and-fix** (this): Orchestrates entire workflow

### Multi-Phase Process

1. **Review Phase** (Two-Stage):
   - **Stage 1 - Single-File Review**: Collect issues from individual files using review-single-target
   - **Stage 2 - Cross-File Review**:
     - Use collect-relative-files-in-pr to identify related file chunks
     - Use review-multiple-target to check cross-file consistency for each chunk
   - Post all PR comments (both single-file and cross-file issues)
2. **Grouping Phase**: Group all PR comment URLs (from both stages) by package
3. **Fix Phase**: Delegate to apply-pr-review-chunk agents per package

### Instruction Argument Support

- Optional instruction can customize review scope
- Instructions passed to review-single-target agent
- Can specify files/directories to include/exclude
- Applies to review phase only (fixes only address reviewed issues)

---

## Templates and Scripts

**Templates** (in `.claude/templates/`):
- `review-pr-body.md` - PR body format for review fixes PR
- `review-summary.md` - Comprehensive result summary (end of workflow)
- `response-comment-format.md` - Response comment format for original PR
- `review-report-format.md` - Review report format (Step 5)

**Scripts** (in `.claude/scripts/`):
- `check-review-branch.sh` - Check if on review branch, extract original branch
- `find-available-branch.sh` - Find available review branch number
- `verify-and-rename-branch.sh` - Pre-commit branch verification and rename
- `post-response-comment.sh` - Post response comment to review thread (naming follows verb-object pattern for clarity)

All scripts are executable and can be called directly from bash.
