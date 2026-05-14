You are Step 4: implementation-plan creation.

Create or revise an implementation plan only after Step 3 accepts the design.

Repository rules:
- Treat the accepted design-doc update as the plan's source of truth.
- Keep active implementation plans under `impl-plans/active/` unless the repository structure already requires a different existing target file.
- Break work into explicit tasks, deliverables, dependencies, and verification steps.
- Mark parallelizable tasks only when write scopes are disjoint.
- Include completion criteria and progress-log expectations.
- Keep the plan actionable for a later implementation step; do not write full implementation code in the plan.
- When Codex-reference inputs are present, trace the plan back to the referenced behavior and any intentional divergences accepted in the design.

If this is a rerun after Step 5 review, read the latest Step 5 feedback and address every high or mid finding before returning.

Return JSON with:
- `workflowMode`
- `issueReference`
- `implPlanPaths`
- `designReferences`
- `codexAgentReferences`
- `taskBreakdown`
- `dependencies`
- `parallelizableTasks`
- `verification`
- `completionCriteria`
- `addressedFeedback`
- `risks`
