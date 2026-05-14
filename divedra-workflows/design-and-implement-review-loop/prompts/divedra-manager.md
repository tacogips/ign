Start the shared design, plan, and optional implementation workflow.

Output contract:
- Return one JSON object only. Do not include Markdown fences, prose before the JSON, or prose after the JSON.
- This manager step is a routing gate. On the initial run, return a small JSON object that records the decision to start Step 1; do not perform Step 1's work yourself.
- A valid initial response is:

```json
{
  "status": "continue",
  "nextStep": "step1-issue-intake",
  "reason": "Start intake before design, planning, or implementation."
}
```

Rules:
- Run Step 1 intake before any design, plan, or implementation work.
- Step 1 may normalize either an issue-resolution request or a Codex-reference planning-only request.
- Run Step 2 design-doc update only after Step 1 has identified the workflow mode and scope.
- If Step 3 returns `when.needs_revision: true`, route back to Step 2.
- Run Step 4 implementation-plan creation only after Step 3 accepts the design.
- If Step 5 returns `when.needs_design_revision: true`, route back to Step 2.
- If Step 5 returns `when.needs_revision: true`, route back to Step 4.
- If Step 5 returns `when.planning_only: true` and no revision flag is set, finish with workflow output.
- Run Step 6 implementation only after Step 5 accepts the plan for a full issue-resolution run.
- If Step 7 returns `when.needs_revision: true`, route back to Step 6.
- After Step 7 accepts the implementation, run Step 8 user-facing documentation refresh, then finish with workflow output.
- Finish only after the required review gates for the selected mode report no high or mid findings.
