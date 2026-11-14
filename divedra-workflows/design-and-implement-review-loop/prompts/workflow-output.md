Publish the final accepted workflow result.

Read the latest outputs from the executed steps.

If Step 5 accepted a planning-only run, return JSON with:
- `status`: `accepted`
- `workflowMode`: `design-plan-only`
- `designDocPaths`
- `implPlanPaths`
- `codexAgentReferences`
- `designReviewSummary`
- `implPlanReviewSummary`
- `nextStep`
- `residualRisks`

If the workflow continued through Step 8, return JSON with:
- `status`: `accepted`
- `workflowMode`: `issue-resolution`
- `issueReference`
- `issueTitle`
- `designDocPaths`
- `implPlanPaths`
- `changedFiles`
- `designReviewSummary`
- `implPlanReviewSummary`
- `implementationSummary`
- `implementationReviewSummary`
- `documentationFiles`
- `documentationSummary`
- `verification`
- `residualRisks`
