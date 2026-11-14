---
description: Fetch unresolved PR review comments and fix them in a review branch (user)
argument-hint: [pr-url]
---

## Context

- Current branch: !`git branch --show-current`
- Repository: !`git remote get-url origin 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' || echo "Unknown"`

## Arguments

This command accepts an optional PR URL argument:

**Format**: `/fix-unresolved-pr-comments [pr-url]`

**Examples**:
- `/fix-unresolved-pr-comments` - Use the current branch's PR
- `/fix-unresolved-pr-comments https://github.com/owner/repo/pull/123` - Use specified PR

## Your Task

Fetch all unresolved review comments from the PR, create a review branch (`{orig_branch}_review_{n}`), and implement fixes for each unresolved comment.

**No user confirmation is required** - fixes are implemented automatically.

## Workflow Overview

1. Identify the target PR (from argument or current branch)
2. Fetch all unresolved review threads from the PR using GraphQL
3. Check for uncommitted changes (exit if any)
4. Create a review branch with pattern `{original_branch}_review_{n}`
5. Delegate fixes to apply-pr-review-chunk agents grouped by package
6. Commit fixes and create PR from review branch to original branch
7. Display summary of resolved and remaining comments

---

## Process Steps

### Step 0: Check Prerequisites

**Check for uncommitted changes:**
```bash
git status --porcelain
```
- If any changes: Exit with error "Uncommitted changes detected. Please commit or stash changes before running this command."

### Step 1: Determine Target PR

**If PR URL provided as argument:**
- Parse the PR URL to extract owner, repo, and PR number
- Validate URL format: `https://github.com/{owner}/{repo}/pull/{number}`

**If no argument provided:**
- Get the PR associated with the current branch

```bash
# Get current branch
CURRENT_BRANCH=$(git branch --show-current)

# Find PR for current branch
gh pr view "$CURRENT_BRANCH" --json number,url,title,state,headRefName
```

**Error if no PR found:**
```
Error: Could not find a PR associated with the current branch.
Please provide a PR URL or ensure the current branch has an open PR.
```

**Store variables:**
- `ORIGINAL_BRANCH`: The branch the PR is from
- `PR_NUMBER`: The PR number
- `PR_URL`: The PR URL

### Step 2: Fetch Repository and PR Information

```bash
# Get repository info
REPO_INFO=$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')
OWNER=$(echo "$REPO_INFO" | cut -d'/' -f1)
REPO=$(echo "$REPO_INFO" | cut -d'/' -f2)

# Get PR details including head commit
gh pr view {pr_number} --json number,title,state,headRefName,baseRefName,headRefOid
```

### Step 3: Fetch All Unresolved Review Threads via GraphQL

Use GraphQL API to fetch review threads with their thread IDs:

```bash
gh api graphql -f query='
query {
  repository(owner: "{owner}", name: "{repo}") {
    pullRequest(number: {pr_number}) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          path
          line
          startLine
          diffSide
          originalLine
          originalStartLine
          comments(first: 10) {
            nodes {
              id
              databaseId
              body
              path
              line
              originalLine
              originalCommit {
                oid
              }
              commit {
                oid
              }
              createdAt
              author {
                login
              }
            }
          }
        }
      }
    }
  }
}'
```

**Filter to get unresolved threads:**

```bash
# Extract unresolved threads with details
jq '
  .data.repository.pullRequest.reviewThreads.nodes
  | map(select(.isResolved == false))
  | .[] | {
      thread_id: .id,
      path: .path,
      line: .line,
      original_line: .originalLine,
      first_comment: {
        comment_id: .comments.nodes[0].databaseId,
        body: .comments.nodes[0].body,
        original_commit: .comments.nodes[0].originalCommit.oid,
        current_commit: .comments.nodes[0].commit.oid,
        created_at: .comments.nodes[0].createdAt,
        author: .comments.nodes[0].author.login
      }
    }
'
```

**If no unresolved comments:**
```
No unresolved review comments found in PR #{number}.
Nothing to fix.
```

**Display summary of unresolved comments:**
```
## Unresolved Review Comments Found: {count}

{for each unresolved thread:}
- [{path}:{line}] @{author}: "{truncated_body}"
```

### Step 4: Create Review Branch

**Find available review branch name** using `.claude/scripts/find-available-branch.sh`:
```bash
./.claude/scripts/find-available-branch.sh "$ORIGINAL_BRANCH"
```
- Checks both local and remote branches
- Finds first available `{original_branch}_review_{n}` number
- Store as `REVIEW_BRANCH`

**Create and checkout review branch:**
```bash
git checkout -b "$REVIEW_BRANCH"
```

### Step 5: Build PR Comment URLs and Categorize

**5.1: Build PR comment URLs**

For each unresolved comment, build the PR comment URL:

```
https://github.com/{owner}/{repo}/pull/{pr_number}#discussion_r{comment_id}
```

**5.2: Categorize comments by actionability**

For each unresolved comment, assess actionability based on comment content:

**ACTIONABLE** - Clear, specific instruction that can be implemented:
- "Add error handling for this case"
- "This variable should be named 'userID' instead of 'userId'"
- "Missing validation for empty string"
- "This function should return error instead of panic"

**POSSIBLY NOT ACTIONABLE** - Vague or requires clarification:
- "This could be better"
- "Consider refactoring this"
- "Performance concern" (without specific fix)

**5.3: Group by package with context**

- Extract package name from file path (e.g., `internal/{package_name}/`)
- Group comments by package
- Include actionability assessment for each comment

### Step 6: Launch apply-pr-review-chunk Agents by Package

For each package with unresolved comments, use the Task tool with subagent_type='apply-pr-review-chunk':

**6.1: For each package**, launch `apply-pr-review-chunk` agent:

```
subagent_type: 'apply-pr-review-chunk'
prompt: |
  Implement fixes for unresolved PR review comments in package: {package_name}

  Context:
  - PR: #{pr_number} - {title}
  - Repository: {owner}/{repo}
  - Package: {package_name}
  - Number of comments: {comment_count}

  GitHub PR Comment URLs with fix instructions:
  {for each comment in this package:}
  ---
  - URL: {comment_url}
    File: {path}:{line}
    Comment: "{truncated_body}"
    Author: @{author}
    Actionability: {ACTIONABLE | POSSIBLY NOT ACTIONABLE}
  ---

  Task:
  1. Fetch and process each PR comment URL to extract full modification instructions
  2. For ACTIONABLE comments: Implement fixes
  3. For POSSIBLY NOT ACTIONABLE comments: Evaluate if actionable after reading full context, skip if unclear
  4. Run `go build` and `go vet` after changes
  5. Run `go test` to verify all tests pass
  6. Report completion status for each comment:
     - FIXED: Successfully implemented
     - SKIPPED: Not actionable or unclear instructions
     - FAILED: Could not implement due to error

  Remember:
  - Focus only on files within {package_path}/
  - Follow CLAUDE.md guidelines
  - Do not fix issues in other packages
  - Stop if unrelated errors block progress
```

**6.2: Collect agent responses:**
- Fixes applied, files modified, compilation/test status, blockers
- Track progress per package
- Categorize results: FIXED, SKIPPED, FAILED

### Step 7: Commit Fixes

After agent completes:

**Stage and commit all changes:**
```bash
git add -A

git commit -m "$(cat <<'EOF'
fix: address PR review comments for #{pr_number}

Review comments addressed:
{for each fixed comment:}
- {URL}: {brief description}

{for each failed comment:}
- {URL}: (NOT FIXED) {reason}

Files modified:
{list of files with +/- stats}
EOF
)"
```

### Step 8: Push and Create Pull Request

**Push review branch:**
```bash
git push -u origin "$REVIEW_BRANCH"
```

**Create PR from review branch to original branch:**
```bash
gh pr create \
  --base "$ORIGINAL_BRANCH" \
  --head "$REVIEW_BRANCH" \
  --title "fix: address review comments for PR #${PR_NUMBER}" \
  --body "$(cat <<'EOF'
## Summary

This PR addresses unresolved review comments from PR #{PR_NUMBER}.

## Review Target Comments

{for each unresolved comment URL:}
- {URL}

## Changes Made

{summary of changes grouped by file}

## Verification

- [ ] Compilation check passed
- [ ] Tests passed
EOF
)"
```

### Step 9: Display Summary

```
## Fix Unresolved PR Comments Summary

### PR Information
- Original PR: #{pr_number} - {title}
- Repository: {owner}/{repo}
- Original Branch: {original_branch}
- Review Branch: {review_branch}

### Review Fixes PR
**URL: {review_fixes_pr_url}**

### Fix Results
- Total unresolved comments: {total_count}
- Successfully fixed: {fixed_count}
- Failed to fix: {failed_count}

### Fixed Comments
{for each fixed:}
[OK] {path}:{line}
    Comment: "{truncated_body}"
    Fix: {brief description}

### Failed Fixes
{for each failed:}
[X] {path}:{line}
    Comment: "{truncated_body}"
    Reason: {why_failed}

### Next Steps
1. Review the changes in the review fixes PR
2. Merge the review fixes PR into the original branch
3. Use /resolve-pr-review-comments to mark addressed comments as resolved
```

---

## Important Notes

### Prerequisites

- GitHub CLI (`gh`) must be authenticated
- User must have write access to the repository
- Local repository must be up to date with remote
- No uncommitted changes allowed

### Review Branch Pattern

- Branch name: `{original_branch}_review_{n}`
- `n` starts at 1 and increments for each review round
- Checks both local and remote branches for availability

### Automatic Fixes

- This command does NOT ask for user confirmation before fixing
- Fixes are implemented automatically based on review comment content
- Only fixes issues with clear, actionable instructions

### Handling Unclear Comments

If a review comment is unclear or cannot be fixed automatically:
- Skip the comment and continue with others
- Report the comment as "failed to fix" with reason
- Include in summary for manual review

### Multi-Package Support

- Comments are grouped by package for efficient processing
- Each package is processed by a separate apply-pr-review-chunk agent
- Cross-package dependencies are noted but not fixed

---

## Error Handling

**No PR found:**
```
Error: Could not find a PR to process.
Please provide a PR URL or ensure the current branch has an open PR.
```

**No unresolved comments:**
```
No unresolved review comments found in PR #{number}.
Nothing to fix.
```

**Uncommitted changes:**
```
Error: Uncommitted changes detected.
Please commit or stash changes before running this command.
```

**Push failed:**
```
Error: Failed to push review branch to remote.
Possible causes: network issue, permission denied, remote hook rejection.
```

**PR creation failed:**
```
Error: Failed to create pull request.
Please check your GitHub permissions and try creating manually.
```
