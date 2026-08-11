---
name: divedra-impl-workflow
description: Use when implementation work in this repository changes behavior, adds functionality, or fixes bugs and the user has not explicitly asked to avoid workflows. Routes the work through the project-local divedra workflow `divedra-workflows/design-and-implement-review-loop`, including design/plan alignment, implementation, review, user-facing documentation refresh, and final handoff.
---

# Divedra Implementation Workflow

Use this skill as the default path for implementation work in this repository.

## Apply This Skill When

- fixing a bug
- adding or changing runtime behavior
- implementing a feature from a design or implementation plan
- making a non-trivial refactor that changes implementation behavior

## Do Not Apply This Skill When

- the user explicitly says not to use a workflow
- the task is documentation-only or planning-only with no implementation
- the task is specifically to debug or repair `divedra` itself; use `divedra-fix`

## Default Workflow

Use the project-local workflow bundle:

- Workflow id: `design-and-implement-review-loop`
- Workflow definition path: `divedra-workflows/design-and-implement-review-loop`

Run from the repository root:

```bash
divedra workflow run design-and-implement-review-loop --workflow-definition-dir ./divedra-workflows
```

## Runtime Inputs

For normal implementation work, run the workflow in issue-resolution mode.

Pass structured workflow input through `--variables` when the task needs
explicit issue/reference context. Typical fields:

- `workflowInput.issueUrl`
- `workflowInput.issueNumber`
- `workflowInput.issueRepository`
- `workflowInput.issueTitle`
- `workflowInput.issueBody`
- `workflowInput.targetFeatureArea`
- `workflowInput.requestedBehavior`
- `workflowInput.codexAgentReferences`
- `workflowInput.referenceRepositoryRoot`
- `workflowInput.referenceRepositoryUrl`

When the intake step splits accepted scope into feature fanout items, the local
workflow expects `featureFanoutItems` at the root of the step output payload for
the planning fanout transition.

Planning-only mode is available via:

- `workflowInput.executionMode: "design-plan-only"`

## Expected Behavior

The workflow is responsible for:

1. issue or task intake
2. design-document updates
3. design self-review
4. design review
5. implementation-plan creation or revision
6. implementation-plan self-review
7. implementation-plan review
8. implementation work
9. implementation self-review
10. implementation review
11. user-facing documentation refresh (`README.md` and exposed workflow skill docs)
12. final workflow output for the accepted planning handoff or issue-resolution handoff

For release or installation work, keep the supported Homebrew surface in scope:

- `README.md` must show the user install path:
  `brew tap tacogips/tap`, `brew install ign`, and `ign version`.
- `packaging/homebrew/README.md` is the release runbook for archive creation,
  GitHub Release asset upload, tap formula rendering, tap commit/push, and
  smoke testing.
- `tacogips/homebrew-tap` owns the published `Formula/ign.rb`; verify the tap
  formula with `brew audit --formula --strict tacogips/tap/ign` and
  `brew test tacogips/tap/ign` when Homebrew behavior changes.

The workflow does not commit or push changes automatically. If the user asks to
commit afterward, follow the repository git commit policy.

## Documentation Refresh Gate

In full `issue-resolution` mode, after implementation review accepts the code,
refresh user-facing documentation before final workflow output. Always review
`README.md` and this workflow skill, and update any other README or workflow
skill whose documented behavior changed.

For defect-remediation work, the docs refresh should reflect shipped behavior
such as CLI exit/error semantics, supported flags, safety guarantees, and
verification commands. Keep this gate documentation-only: do not reopen design
or implementation scope after Step 7 acceptance.

For CLI template-variable prompting changes, ensure user-facing docs describe
non-interactive stdin behavior for `init`, `checkout`, and `switch`: unresolved
variables require a TTY for interactive prompts, while scripted runs should pass
all required values with repeatable `--var key=value` or `-V key=value`.

## Reporting

After the workflow finishes, report:

- workflow mode
- changed files
- verification commands
- documentation refresh summary
- residual risks
- for release work, GitHub Release URL and Homebrew tap/formula status

If the workflow fails because `divedra` appears incorrect, switch to the
`divedra-fix` skill.
