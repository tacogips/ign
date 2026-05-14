---
name: impl-exec-auto
description: Analyze implementation plans and return executable tasks list. Main conversation handles orchestration.
tools: Read, Glob, Grep
model: sonnet
skills: exec-impl-plan-ref, ts-coding-standards
---

# Auto Task Selection Analysis Subagent

## Overview

This subagent **analyzes** implementation plans and returns a list of executable tasks. It does NOT execute tasks - the main conversation handles orchestration.

**Key Design**: This agent is analysis-only because Claude Code does not support nested subagent spawning (subagents cannot use Task tool).

## Workflow

```
1. Read PROGRESS.json (task status overview)
2. Identify executable tasks (deps satisfied, status "Not Started")
3. For each executable task, read plan file to get details
4. Return structured task list to main conversation
5. Main conversation uses impl-exec-specific to execute tasks
```

## CRITICAL: Use PROGRESS.json to Prevent Context Overflow

**NEVER read all plan files at once.** This causes context overflow (>200K tokens).

**Workflow**:
1. Read `impl-plans/PROGRESS.json` (~2K tokens) to find executable tasks
2. Read ONLY the specific plan files for executable tasks
3. Return structured analysis

---

## Execution Steps

### Step 1: Read PROGRESS.json

```bash
Read impl-plans/PROGRESS.json
```

Structure:
```json
{
  "lastUpdated": "2026-01-06T16:00:00Z",
  "phases": {
    "1": { "status": "COMPLETED" },
    "2": { "status": "READY" },
    "3": { "status": "BLOCKED" }
  },
  "plans": {
    "session-groups-types": {
      "phase": 2,
      "status": "Ready",
      "tasks": {
        "TASK-001": { "status": "Not Started", "parallelizable": true, "deps": [] },
        "TASK-002": { "status": "Completed", "parallelizable": true, "deps": [] }
      }
    }
  }
}
```

### Step 2: Identify Executable Tasks

A task is executable when:
1. **Phase is READY** (not BLOCKED or COMPLETED)
2. **Task status = "Not Started"**
3. **All dependencies are "Completed"**

```python
executable_tasks = []
for plan_name, plan in progress["plans"].items():
    phase = progress["phases"][str(plan["phase"])]
    if phase["status"] != "READY":
        continue  # Skip blocked phases

    for task_id, task in plan["tasks"].items():
        if task["status"] != "Not Started":
            continue

        # Check dependencies
        all_deps_complete = True
        for dep in task["deps"]:
            if ":" in dep:  # Cross-plan dep: "plan-name:TASK-xxx"
                dep_plan, dep_task = dep.split(":")
                if progress["plans"][dep_plan]["tasks"][dep_task]["status"] != "Completed":
                    all_deps_complete = False
            else:  # Same-plan dep: "TASK-xxx"
                if plan["tasks"][dep]["status"] != "Completed":
                    all_deps_complete = False

        if all_deps_complete:
            executable_tasks.append((plan_name, task_id))
```

### Step 3: Read Plan Files for Task Details

For each executable task, read the plan file and extract:
- Description (Purpose)
- Deliverables (Implementation Target)
- Completion Criteria

### Step 4: Return Structured Output

Return the analysis in this exact format:

---

## Required Output Format

```markdown
## Executable Tasks Analysis

### Phase Status
| Phase | Status |
|-------|--------|
| 1 | COMPLETED |
| 2 | READY |
| 3 | BLOCKED |

### Executable Tasks

Total: N tasks ready for execution

#### Task 1: [plan-name]:TASK-XXX
- **Plan File**: impl-plans/[plan-name].md
- **Purpose**: [task description]
- **Deliverables**:
  - [file path 1]
  - [file path 2]
- **Completion Criteria**:
  - [ ] [criterion 1]
  - [ ] [criterion 2]
- **Design Reference**: [design doc path if available]

#### Task 2: [plan-name]:TASK-YYY
...

### Blocked Tasks (for reference)
- [plan-name]:TASK-ZZZ - waiting on TASK-XXX
- ...

### Recommended Execution Order
1. [plan-name]:TASK-XXX (foundation task)
2. [plan-name]:TASK-YYY (no dependencies)
...
```

---

## No Executable Tasks Response

If no tasks are executable:

```markdown
## No Executable Tasks

### Phase Status
| Phase | Status |
|-------|--------|
| 1 | COMPLETED |
| 2 | READY |
| 3 | BLOCKED |

### Analysis
All tasks in READY phases have unmet dependencies or are already completed.

### Blocking Tasks
- [plan-name]:TASK-XXX (In Progress) - blocking TASK-YYY, TASK-ZZZ
- [plan-name]:TASK-AAA - waiting on TASK-BBB

### Recommended Actions
1. Wait for in-progress tasks to complete
2. Use `/impl-exec-specific` to manually run specific tasks
3. Check if any tasks are incorrectly marked
```

---

## Important Notes

1. **Analysis Only**: This agent does NOT spawn subagents or update files
2. **Main Orchestrates**: Main conversation uses impl-exec-specific to execute tasks (NOT direct ts-coding spawning)
3. **PROGRESS.json**: Main conversation updates PROGRESS.json after task completion
4. **Context Efficient**: Only reads necessary plan files, not all plans

## Orchestration Protocol: Main Conversation Actions

After receiving this agent's output, the **main conversation MUST use impl-exec-specific** to execute tasks.

**DO NOT spawn ts-coding agents directly from the main conversation.**

### Main Conversation Workflow

```
1. Receive executable tasks list from impl-exec-auto
2. For each plan with executable tasks:
   a. Invoke impl-exec-specific with task IDs
      Example: /impl-exec-specific session-groups-runner TASK-008
   b. impl-exec-specific handles:
      - ts-coding spawning
      - check-and-test-after-modify
      - ts-review cycle (up to 3 iterations)
      - Plan file status updates
3. After impl-exec-specific completes:
   a. Update PROGRESS.json status
   b. Report completion and newly unblocked tasks
4. Repeat for remaining executable tasks
```

### Why impl-exec-specific?

| Approach | Problem |
|----------|---------|
| Main spawns ts-coding directly | Main lacks review cycle logic, plan update format |
| impl-exec-auto spawns ts-coding | Claude Code prohibits nested subagent spawning |
| **Main invokes impl-exec-specific** | Correct - full implementation cycle handled |

### Example Main Conversation Response

```markdown
The impl-exec-auto analysis found 9 executable tasks.

Executing via impl-exec-specific:

1. /impl-exec-specific session-groups-runner TASK-008
2. /impl-exec-specific command-queue-core TASK-005 TASK-006
3. /impl-exec-specific markdown-parser-core TASK-004
...
```
