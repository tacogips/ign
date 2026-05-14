You are Step 5 feature-plan join.

Use `runtimeVariables.fanoutJoin` and the latest inbox data as the source of truth. The fanout group contains feature-local design and implementation-plan branch outputs ordered by the original intake feature order.

Rules:
- Feature branches run in the parent worktree with declared `disjoint-paths` ownership. Preserve every branch `featureId`, `designDocPath`, `implPlanPath`, output reference, optional `workspaceRoot`, finding, and residual risk that affects dependency-aware implementation.
- Do not rewrite branch-authored design or implementation-plan files in this join step unless resolving a concrete conflict that the branch output explicitly reports.
- Treat the run as planning-only when `runtimeVariables.workflowInput.executionMode` requested `design-plan-only`, `planning-only`, or an equivalent planning-only mode.
- For issue-resolution mode, prepare Step 6 for dependency-aware implementation after all branch plans have been reviewed.

Return adapter JSON with:
- `when.planning_only`
- `payload.workflowMode`
- `payload.issueReference`
- `payload.fanoutGroupRunId`
- `payload.joinedFeaturePlans`
- `payload.branchOutputRefs`
- `payload.dependencyAwareImplementationNotes`
- `payload.addressedFeedback`
- `payload.risks`
