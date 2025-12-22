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
- `/resolve-pr-review-comments` - Use the current branch's merged PR (if any)
- `/resolve-pr-review-comments https://github.com/owner/repo/pull/123` - Use specified PR

## Your Task

Verify which PR review comments have been addressed by merged commits and resolve those comments on GitHub. This command is intended to be used after a review PR has been merged to clean up resolved review threads.

## Workflow Overview

1. Identify the PR to process (from argument or current branch)
2. Find the original PR that received review comments
3. Fetch all review comments from the original PR
4. Verify which comments have been addressed by the merged changes
5. Resolve verified comments on GitHub
6. Display summary of resolved and unresolved comments

---

## Process Steps

### Step 1: Determine Target PR

**If PR URL provided as argument:**
- Parse the PR URL to extract owner, repo, and PR number
- Use this as the review fixes PR

**If no argument provided:**
- Check if current branch is a review branch (ends with `_review_{n}`)
- If yes: Look for merged PR from this review branch
- If no: Exit with error "Please provide a PR URL or run from a review branch"

```bash
# Check if on review branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" =~ ^(.+)_review_[0-9]+$ ]]; then
  ORIGINAL_BRANCH="${BASH_REMATCH[1]}"
  REVIEW_BRANCH="$CURRENT_BRANCH"

  # Find merged PR for review branch
  gh pr view "$REVIEW_BRANCH" --json state,mergedAt,number,url,body
fi
```

### Step 2: Verify Review Fixes PR is Merged

```bash
gh pr view {pr_number} --json state,mergedAt,mergeCommit
```

**If PR is not merged:**
- Display message: "PR #{number} is not merged yet. Only merged PRs can have comments resolved."
- Exit

**If PR is merged:**
- Continue to next step
- Store merge commit SHA for verification

### Step 3: Extract Original PR and Review Comment URLs

**From the review fixes PR body, extract:**

1. **Original PR reference:**
   - Look for "Original PR:" or link to the original PR
   - Parse to get original PR number

2. **Review comment URLs:**
   - Look for "Review Target Comments" section
   - Extract all URLs matching pattern: `https://github.com/{owner}/{repo}/pull/{number}#discussion_r{id}`

```bash
# Get review fixes PR body
PR_BODY=$(gh pr view {pr_number} --json body -q '.body')

# Extract original PR URL (look for patterns like "Original PR: #123" or full URL)
ORIGINAL_PR=$(echo "$PR_BODY" | grep -oP 'https://github.com/[^/]+/[^/]+/pull/\d+' | head -1)

# Extract review comment URLs
REVIEW_COMMENTS=$(echo "$PR_BODY" | grep -oP 'https://github.com/[^/]+/[^/]+/pull/\d+#discussion_r\d+')
```

**Error handling:**
- If original PR not found: Exit with error "Could not find original PR reference in PR body"
- If no review comments found: Exit with message "No review comment URLs found in PR body"

### Step 3.5: Fetch Review Comments and Thread IDs via GraphQL API

**IMPORTANT**: Use GraphQL API to fetch review comments with their thread IDs, which are required for resolution.

**GraphQL Query to fetch all review threads with comments:**

```bash
# Fetch all unresolved review threads with their comment details
gh api graphql -f query='
query {
  repository(owner: "{owner}", name: "{repo}") {
    pullRequest(number: {pr_number}) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 1) {
            nodes {
              id
              databaseId
              body
              path
              line
            }
          }
        }
      }
    }
  }
}'
```

**Parse the response to get thread IDs:**

```bash
# Get unresolved threads with their comment IDs and thread IDs
gh api graphql -f query='...' | jq '
  .data.repository.pullRequest.reviewThreads.nodes
  | map(select(.isResolved == false))
  | .[] | {
      thread_id: .id,
      comment_id: .comments.nodes[0].databaseId,
      path: .comments.nodes[0].path,
      line: .comments.nodes[0].line,
      body: .comments.nodes[0].body[0:100]
    }
'
```

**Alternative: Fetch comments via REST API and filter:**

```bash
# List all review comments (REST API)
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments

# Filter to get specific comment details by ID
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments 2>/dev/null | \
  jq '.[] | select(.id == {comment_id})'
```

**Note**: The REST API endpoint `/repos/{owner}/{repo}/pulls/comments/{comment_id}` may return 404 for some comments. Use the list endpoint and filter with jq instead:

```bash
# Correct approach - list and filter
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments | \
  jq '.[] | select(.id == 2639259350) | {id, node_id, path, line, body: .body[0:200]}'
```

**Map comment IDs to thread IDs:**

The `discussion_r{id}` in the URL corresponds to the comment's `databaseId` in GraphQL. To resolve a thread, you need the thread's `id` (not the comment's `id`). Use the GraphQL query above to create a mapping:

```bash
# Create mapping of comment_id -> thread_id
THREAD_MAP=$(gh api graphql -f query='...' | jq '
  .data.repository.pullRequest.reviewThreads.nodes
  | map({
      comment_id: .comments.nodes[0].databaseId,
      thread_id: .id
    })
  | from_entries
')
```

### Step 4: Verify Original PR State

```bash
gh pr view {original_pr_number} --json state,mergedAt
```

**Check if original PR is still open or merged:**
- If merged: The review was for a merged PR
- If open: The review branch was merged back to the feature branch

### Step 5: Launch Verification Agent

Use the `verify-pr-comment-resolution` agent to verify each comment:

**Task tool invocation:**
```
subagent_type: 'general-purpose'
prompt: |
  Use the verify-pr-comment-resolution agent guidelines from .claude/agents/verify-pr-comment-resolution.md.

  Verify the following PR review comments:

  PR URL: {original_pr_url}
  Review Fixes PR: {review_fixes_pr_url} (merged)

  Review Comment URLs to verify:
  {list of comment URLs}

  For each comment:
  1. Fetch the comment content and context
  2. Analyze if the comment was addressed by commits in the review fixes PR
  3. Check that the review fixes PR is merged
  4. Determine if the comment can be resolved

  Return the verification report with:
  - List of resolvable comments (addressed and merged)
  - List of non-resolvable comments (with reasons)
```

### Step 6: Display Verification Results

Show summary of verification results:

```
## Review Comment Verification Results

### PR Information
- Review Fixes PR: #{pr_number} - {title} (MERGED)
- Original PR: #{original_pr_number} - {original_title}
- Merge Commit: {sha}

### Verification Summary
- Total comments analyzed: {count}
- Resolvable: {count}
- Not resolvable: {count}

### Resolvable Comments
{for each resolvable comment:}
[OK] {comment_url}
    File: {file_path}:{line}
    Issue: "{truncated_comment}"
    Resolved by: Commit {sha}

### Non-Resolvable Comments
{for each non-resolvable comment:}
[!] {comment_url}
    File: {file_path}:{line}
    Issue: "{truncated_comment}"
    Reason: {reason}
```

### Step 7: Confirm Resolution Action

Before resolving comments, ask for confirmation:

```
Found {count} comments that can be resolved.

Do you want to resolve these comments on GitHub?
```

**Options**:
- Yes, resolve all - Proceed to resolve all resolvable comments
- Review individually - Show each comment for individual confirmation
- No, cancel - Exit without resolving

### Step 8: Resolve Comments on GitHub

For each confirmed resolvable comment:

**Step 8.1: Get Thread ID from Comment ID**

First, map the comment ID (from `discussion_r{id}`) to the thread ID using GraphQL:

```bash
# Full GraphQL query to get all threads with their comment IDs
gh api graphql -f query='
query {
  repository(owner: "tacogips", name: "ign") {
    pullRequest(number: 9) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 1) {
            nodes {
              id
              databaseId
              body
              path
              line
            }
          }
        }
      }
    }
  }
}'
```

**Step 8.2: Filter to find specific thread by comment ID**

```bash
# Filter unresolved threads and extract thread_id, comment_id mapping
gh api graphql -f query='...' | jq '
  .data.repository.pullRequest.reviewThreads.nodes
  | map(select(.isResolved == false))
  | .[] | {
      thread_id: .id,
      comment_id: .comments.nodes[0].databaseId,
      body: .comments.nodes[0].body[0:100]
    }'
```

Example output:
```json
{
  "thread_id": "PRRT_kwDONMmJGs5m-mWt",
  "comment_id": 2639259350,
  "body": "## [REVIEW] Critical: Missing path separator validation..."
}
```

**Step 8.3: Resolve the thread using GraphQL mutation**

```bash
# Resolve a single review thread
gh api graphql -f query='
mutation {
  resolveReviewThread(input: {threadId: "PRRT_kwDONMmJGs5m-mWt"}) {
    thread {
      id
      isResolved
    }
  }
}'
```

Expected success response:
```json
{
  "data": {
    "resolveReviewThread": {
      "thread": {
        "id": "PRRT_kwDONMmJGs5m-mWt",
        "isResolved": true
      }
    }
  }
}
```

**Complete example workflow:**

```bash
# 1. Get all unresolved threads for PR #9
THREADS=$(gh api graphql -f query='
query {
  repository(owner: "tacogips", name: "ign") {
    pullRequest(number: 9) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 1) {
            nodes {
              databaseId
            }
          }
        }
      }
    }
  }
}')

# 2. Find thread ID for comment ID 2639259350
THREAD_ID=$(echo "$THREADS" | jq -r '
  .data.repository.pullRequest.reviewThreads.nodes[]
  | select(.comments.nodes[0].databaseId == 2639259350)
  | .id
')

# 3. Resolve the thread
gh api graphql -f query="
mutation {
  resolveReviewThread(input: {threadId: \"$THREAD_ID\"}) {
    thread {
      id
      isResolved
    }
  }
}"
```

**Alternative approach using REST API (not recommended - no direct resolve):**
```bash
# Get the review comment details (use list and filter, not direct endpoint)
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments | \
  jq '.[] | select(.id == {comment_id})'

# REST API does not support resolving threads directly
# You can only post a reply to the thread
gh api repos/{owner}/{repo}/pulls/comments/{comment_id}/replies \
  -f body="Resolved by merge of #{review_fixes_pr_number}"
```

**Error handling for each comment:**
- If resolution fails: Log error but continue with other comments
- Track success/failure count
- Common errors:
  - `NOT_FOUND`: Thread ID is invalid or thread was already deleted
  - `FORBIDDEN`: No permission to resolve threads in this repository

### Step 9: Display Final Summary

```
## Resolution Complete

### Summary
- Comments resolved: {success_count}
- Failed to resolve: {failure_count}
- Skipped: {skipped_count}

### Resolved Comments
{for each successfully resolved:}
[RESOLVED] {comment_url}

### Failed Resolutions
{for each failed:}
[FAILED] {comment_url}
    Error: {error_message}

### Remaining Unresolved
{for each non-resolvable:}
[PENDING] {comment_url}
    Reason: {reason}
```

---

## Important Notes

### Prerequisites

- GitHub CLI (`gh`) must be authenticated
- User must have write access to the repository
- The review fixes PR must be merged

### Comment Resolution

- Only resolves comments that have clear evidence of being addressed
- Conservative approach: Does not resolve uncertain cases
- Posts a resolution note linking to the merge commit

### Workflow Integration

This command is designed to be used after:
1. `/review-current-pr-and-fix` creates a review fixes PR
2. The review fixes PR is reviewed and merged
3. Run this command to clean up resolved comments

### URL Format

Review comment URLs must match:
`https://github.com/{owner}/{repo}/pull/{number}#discussion_r{id}`

Issue comments (`#issuecomment-{id}`) are not supported as they cannot be resolved.

### GraphQL vs REST API

- GitHub's GraphQL API provides `resolveReviewThread` mutation
- Thread ID is needed, which can be obtained from the comment's thread context
- REST API alternative: Reply to comment with resolution message

---

## Error Handling

**No PR found:**
```
Error: Could not find a PR to process.
Please provide a PR URL or run from a review branch with a merged PR.
```

**PR not merged:**
```
Error: PR #{number} is not merged.
Comments can only be resolved after the PR is merged.
```

**No review comments found:**
```
No review comment URLs found in PR #{number} body.
Nothing to resolve.
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
