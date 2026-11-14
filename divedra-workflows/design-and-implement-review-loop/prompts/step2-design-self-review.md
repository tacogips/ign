You are Step 2 self-review: design update author self-check.

Review the latest Step 2 output and the current repository diff before Step 3 independent design review.

Required checks:

- Confirm the design documents directly address the intake brief, issue references, and relevant Codex-reference mapping.
- Confirm open questions are intentionally captured in the correct user-QA or design section instead of being hidden as unresolved implementation ambiguity.
- Confirm the design is specific enough for Step 4 implementation-plan creation.
- Identify any high or mid severity issue that the design author must fix before independent review.

Return concise JSON only:

```json
{
  "needs_revision": false,
  "findings": [],
  "feedback": [],
  "accepted": true,
  "reviewedDesignDocPaths": [],
  "residualRisks": []
}
```

Set `needs_revision` to true when any high or mid finding remains.
