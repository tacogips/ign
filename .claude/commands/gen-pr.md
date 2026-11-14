---
description: Create or update a pull request (user)
---

Use the Task tool to spawn the `generate-pr` subagent to create or update a pull request.

Pass all arguments from the user's command directly to the subagent's prompt.

**Example invocations**:
- `/gen-pr` -> Create draft PR against default branch
- `/gen-pr State:Open` -> Create open PR
- `/gen-pr develop` -> Create draft PR against develop branch
- `/gen-pr State:Open Desc: description text` -> Create open PR with description
- `/gen-pr https://github.com/owner/repo/issues/123` -> Update PR with related issue
- `/gen-pr State:Draft` -> Convert existing PR to draft
