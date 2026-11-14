════════════════════════════════════════════════════════════════
✅ Review Fix Workflow Complete
════════════════════════════════════════════════════════════════

## 📊 Execution Summary

**Mode**: {{MODE}}
**Original Branch**: {{ORIGINAL_BRANCH}}
**Fix Branch**: {{FINAL_REVIEW_BRANCH}}

────────────────────────────────────────────────────────────────

## 🔗 Created/Updated PR

**Fix PR**: {{REVIEW_PR_URL}}
**Base Branch**: {{ORIGINAL_BRANCH}}
**Original PR**: {{ORIGINAL_PR_URL}}

────────────────────────────────────────────────────────────────

## 📝 Review Comment Status

**Total Comments**: {{TOTAL_COMMENTS}}
- ✅ **Completed**: {{COMPLETED_COUNT}}
- ⚠️ **Incomplete**: {{INCOMPLETE_COUNT}}
- ❌ **Manual Action Required**: {{MANUAL_COUNT}}

### ✅ Completed Comments ({{COMPLETED_COUNT}})

{{COMPLETED_COMMENTS_LIST}}

### ⚠️ Incomplete Comments ({{INCOMPLETE_COUNT}})

{{INCOMPLETE_COMMENTS_LIST}}

### ❌ Comments Requiring Manual Action ({{MANUAL_COUNT}})

{{MANUAL_COMMENTS_LIST}}

────────────────────────────────────────────────────────────────

## 💬 Posted Response Comments

**Posted to Original PR**: {{RESPONSE_COUNT}} (out of {{TOTAL_COMMENTS}} total)

{{RESPONSE_COMMENTS_LIST}}

────────────────────────────────────────────────────────────────

## 📦 Changed Files

**Files Changed**: {{FILE_COUNT}}
**Lines Added**: +{{ADDITIONS}}
**Lines Deleted**: -{{DELETIONS}}

### Changes by Package:

{{FILE_CHANGES_BY_PACKAGE}}

────────────────────────────────────────────────────────────────

## 🧪 Verification Results

**Compilation**: {{COMPILATION_STATUS}}
**Tests**: {{TEST_STATUS}}
**Coverage**: {{TEST_COVERAGE}}

────────────────────────────────────────────────────────────────

## 📌 Next Actions

### Immediate Actions:
1. Review fix PR: {{REVIEW_PR_URL}}
2. Check fix status in original PR: {{ORIGINAL_PR_URL}}
3. If there are incomplete fixes:
   - Switch to fix branch with `git checkout {{FINAL_REVIEW_BRANCH}}`
   - Run `/review-current-pr-and-fix` again to continue

### Items Requiring Manual Intervention:
{{MANUAL_INTERVENTION_ITEMS}}

════════════════════════════════════════════════════════════════
🎉 Workflow Complete
════════════════════════════════════════════════════════════════
