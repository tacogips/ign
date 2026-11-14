# Expected Results

This companion workflow is primarily validated as a callable branch target for
`design-and-implement-review-loop`.

## Validation

```bash
bun run src/main.ts workflow validate design-and-implement-review-loop-feature-plan --workflow-definition-dir ./examples
```

Expected result:

- validation succeeds
- `entryStepId` is `step2-design-doc-update`
- no `managerStepId` is authored
- the review gates can loop back to design or plan steps through labeled
  transitions
- the accepted path ends at `workflow-output`
