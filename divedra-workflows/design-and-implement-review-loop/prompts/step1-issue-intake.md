You are Step 1: workflow intake.

Normalize the request before changing any repository documents or code.

Preferred sources:
- `runtimeVariables.workflowInput.executionMode`
- `runtimeVariables.workflowInput.issueUrl`
- `runtimeVariables.workflowInput.issueNumber`
- `runtimeVariables.workflowInput.issueRepository`
- `runtimeVariables.workflowInput.issueBody`
- `runtimeVariables.workflowInput.issueTitle`
- `runtimeVariables.workflowInput.targetFeatureArea`
- `runtimeVariables.workflowInput.requestedBehavior`
- `runtimeVariables.workflowInput.codexAgentReferences`
- `runtimeVariables.workflowInput.referenceRepositoryRoot`
- `runtimeVariables.workflowInput.referenceRepositoryUrl`

Rules:
- Default `workflowMode` to `issue-resolution` unless `runtimeVariables.workflowInput.executionMode` explicitly requests `design-plan-only`, `planning-only`, or another planning-only synonym.
- If a GitHub issue URL or repository-plus-number is available, inspect the issue directly. Use local or CLI tooling such as `gh issue view` when available. If remote access is unavailable, fall back to the issue title/body provided in workflow input and state that limitation explicitly.
- If Codex-reference planning input is present, inspect the preferred local reference repository first. Use `../../codex-agent` when no other local root is supplied. Use the upstream reference URL only if local files are unavailable or incomplete.
- Treat codex-agent as a behavioral and structural reference only. Do not copy code blindly.
- Produce one concise intake brief that later steps can execute regardless of mode.
- When the request contains independent feature areas that can be designed and planned concurrently, classify them into `payload.featureFanoutItems`. Each item must include a stable `featureId`, `featureTitle`, `featureSummary`, `issueReference`, `workflowMode`, `designDocPath`, `implPlanPath`, and relevant `codexAgentReferences`.
- Set `when.has_feature_fanout` to `true` only when `payload.featureFanoutItems` is a non-empty array and the feature-local design/plan branches can run independently before dependency-aware implementation.
- Set `when.has_feature_fanout` to `false` for simple single-path work or when branch ownership cannot be made independent.

Return adapter JSON with:
- `when.has_feature_fanout`
- `payload.workflowMode`
- `payload.issueReference`
- `payload.issueTitle`
- `payload.problemSummary`
- `payload.acceptanceSignals`
- `payload.impactedAreas`
- `payload.constraints`
- `payload.unknowns`
- `payload.risks`
- `payload.codexAgentReferences`
- `payload.referenceRepositoryRoot`
- `payload.referenceRepositoryUrl`
- `payload.featureFanoutItems`
