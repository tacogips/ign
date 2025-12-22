---
description: Verify and resolve PR review comments that have been addressed and merged (user)
argument-hint: [pr-url]
---

## Context

- Current branch: !`git branch --show-current`
- Repository: !`git remote get-url origin 2>/dev/null | sed 's/.*github.com[:/]\(.*\)\.git/\1/' || echo "Unknown"`

## Arguments

This command accepts an optional PR URL argument:

**Format**: `/resolve-pr-review-comments [pr-url]`

**Examples**:
- `/resolve-pr-review-comments` - Use the current branch's PR
- `/resolve-pr-review-comments https://github.com/owner/repo/pull/123` - Use specified PR

## Your Task

Verify which PR review comments have been addressed by commits and automatically resolve those comments on GitHub. This command compares the current source code with the source at the time of review to determine if issues have been fixed.

**No user confirmation is required** - resolved comments are automatically updated.

## Workflow Overview

1. Identify the target PR (from argument or current branch)
2. Fetch all unresolved review comments from the PR
3. For each comment, compare the source at review time vs. current source
4. Determine if the issue has been fixed by analyzing code changes
5. Automatically resolve verified comments on GitHub
6. Display summary of resolved and remaining comments

---

## Process Steps

### Step 1: Determine Target PR

**If PR URL provided as argument:**
- Parse the PR URL to extract owner, repo, and PR number

**If no argument provided:**
- Get the PR associated with the current branch

```bash
# Get current branch
CURRENT_BRANCH=$(git branch --show-current)

# Find PR for current branch
gh pr view "$CURRENT_BRANCH" --json number,url,title,state
```

**Error if no PR found:**
```
Error: Could not find a PR associated with the current branch.
Please provide a PR URL or ensure the current branch has an open PR.
```

### Step 2: Fetch Repository and PR Information

```bash
# Get repository info
REPO_INFO=$(git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')
OWNER=$(echo "$REPO_INFO" | cut -d'/' -f1)
REPO=$(echo "$REPO_INFO" | cut -d'/' -f2)

# Get PR details
gh pr view {pr_number} --json number,title,state,headRefName,baseRefName,commits
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
Nothing to resolve.
```

### Step 4: Identify Commits That May Have Fixed Issues

Find commits that could have addressed review comments:

**4.1: Commits merged from review branches**

```bash
# Find merged review branch commits
git log --oneline --merges {base_branch}..HEAD | grep -E '_review_[0-9]+' || true

# Get commits from review branches that were merged
git log --oneline --ancestry-path {merge_commit}^..{merge_commit} 2>/dev/null
```

**4.2: Direct commits to the PR branch**

```bash
# Get all commits on the PR branch after the review comments were made
git log --oneline {base_branch}..HEAD
```

**4.3: Find commits that modified files mentioned in review comments**

```bash
# For each file with review comments, find modifying commits
git log --oneline -- {file_path}
```

### Step 5: Compare Source at Review Time vs. Current Source

For each unresolved review comment:

**5.1: Get the source code at the time of review**

The `originalCommit` field contains the commit SHA when the review was made:

```bash
# Get file content at review time
git show {original_commit}:{file_path}
```

**5.2: Get the current source code**

```bash
# Get current file content
git show HEAD:{file_path}
```

**5.3: Compare the specific lines mentioned in the review**

```bash
# Extract lines around the review comment (e.g., line 42 with context)
# At review time:
git show {original_commit}:{file_path} | sed -n '{start_line},{end_line}p'

# Current:
git show HEAD:{file_path} | sed -n '{start_line},{end_line}p'
```

### Step 6: Launch Verification Agent

Use the `verify-pr-comment-resolution` agent to verify each comment:

**Task tool invocation:**
```
subagent_type: 'verify-pr-comment-resolution'
prompt: |
  Verify the following unresolved PR review comments by comparing source code at review time vs. current source.

  PR: #{pr_number} ({owner}/{repo})
  PR URL: {pr_url}

  For each comment below:
  1. Compare the source at original_commit vs. HEAD
  2. Analyze if the issue raised in the comment has been fixed
  3. Check for related commits that may have addressed the issue
  4. Return verification status for each comment

  Unresolved Comments:
  {for each unresolved thread:}
  ---
  Thread ID: {thread_id}
  Comment ID: {comment_id}
  File: {path}:{line}
  Original Line: {original_line}
  Original Commit: {original_commit}
  Comment: "{body}"
  ---

  Verification Criteria:
  - RESOLVED: The code at the commented location has been changed AND the change addresses the review feedback
  - UNRESOLVED: No relevant changes OR changes do not address the feedback
  - PARTIAL: Some changes made but not fully addressing the concern

  For each comment that should be resolved, provide the thread_id for resolution.
```

### Step 7: Automatically Resolve Verified Comments

**No user confirmation required** - proceed directly to resolution.

For each comment marked as resolvable:

**7.1: Resolve the thread using GraphQL mutation**

```bash
gh api graphql -f query='
mutation {
  resolveReviewThread(input: {threadId: "{thread_id}"}) {
    thread {
      id
      isResolved
    }
  }
}'
```

**7.2: Track resolution results**

- Count successful resolutions
- Log any failures with error details
- Continue with remaining comments even if some fail

### Step 8: Display Final Summary

```
## Review Comment Resolution Summary

### PR Information
- PR: #{pr_number} - {title}
- Repository: {owner}/{repo}
- Branch: {head_branch}

### Resolution Results
- Total unresolved comments analyzed: {total_count}
- Automatically resolved: {resolved_count}
- Failed to resolve: {failed_count}
- Remaining unresolved: {unresolved_count}

### Resolved Comments
{for each resolved comment:}
[RESOLVED] {path}:{line}
    Comment: "{truncated_body}"
    Reason: {resolution_reason}

### Failed Resolutions
{for each failed:}
[FAILED] {path}:{line}
    Error: {error_message}

### Remaining Unresolved
{for each still unresolved:}
[PENDING] {path}:{line}
    Comment: "{truncated_body}"
    Reason: {why_not_resolved}
```

---

## Verification Logic

### How to Determine if a Comment is Addressed

1. **Direct Code Change Detection**:
   - Get the code at `original_commit` (when review was made)
   - Get the code at `HEAD` (current)
   - If the specific lines mentioned in the review have changed, analyze the change

2. **Semantic Analysis**:
   - Parse the review comment to understand what issue was raised
   - Check if the code change addresses that specific issue
   - Common patterns:
     - "Add error handling" -> Check if error handling was added
     - "This is a security issue" -> Check if the vulnerability was fixed
     - "Rename this variable" -> Check if variable was renamed
     - "Add validation" -> Check if validation was added

3. **Commit Message Analysis**:
   - Look for commits that reference the file
   - Check commit messages for keywords like "fix", "address review", "resolve"
   - Consider commits from merged `_review_{n}` branches

4. **Conservative Approach**:
   - If unclear whether the fix is complete, mark as UNRESOLVED
   - Only resolve when there is clear evidence of the fix

### Resolution Decision Matrix

| Code Changed? | Addresses Feedback? | Resolution |
|---------------|---------------------|------------|
| Yes           | Yes                 | RESOLVED   |
| Yes           | Partially           | Check context, may resolve if intent is clear |
| Yes           | No                  | UNRESOLVED |
| No            | N/A                 | UNRESOLVED |

---

## Important Notes

### Prerequisites

- GitHub CLI (`gh`) must be authenticated
- User must have write access to the repository
- Local repository must be up to date with remote

### Automatic Resolution

- This command does NOT ask for user confirmation
- Comments are resolved automatically when verification passes
- Only resolves comments with clear evidence of being addressed

### Source Comparison Method

- Uses `git show {commit}:{path}` to retrieve source at specific commits
- Compares `originalCommit` (review time) vs. `HEAD` (current)
- Analyzes the diff to determine if the issue was fixed

### Finding Fix Commits

The command identifies potential fix commits from:
1. Merged commits from `_review_{n}` branches
2. Direct commits to the PR branch after review comments were made
3. Any commit that modifies files mentioned in review comments

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
Nothing to resolve.
```

**File not found at commit:**
```
Warning: Could not retrieve {file_path} at commit {commit_sha}.
File may have been deleted or renamed. Skipping this comment.
```

**API rate limit:**
```
Warning: GitHub API rate limit reached.
Resolved {count} of {total} comments before limit.
Please wait and try again.
```

**Permission denied:**
```
Error: Permission denied when resolving comment.
Ensure you have write access to the repository.
```

**GraphQL mutation error:**
```
Warning: Failed to resolve thread {thread_id}: {error}
Continuing with remaining comments.
```
