---
generated: true
generated_from: 00_meta/skills/dispatching-parallel-agents/SKILL.md
generated_sha: 48910b6387cc1d47
id: dispatching-parallel-agents-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: dispatching-parallel-agents
description: Use when facing 2+ independent tasks that can run concurrently without
  shared state or sequential dependencies.
keywords: [parallel agents, dispatch agents, parallel dispatch, subagent fan out,
  concurrent tasks]
paths: []
---
# Dispatching Parallel Agents

Multiple independent problems → dispatch one agent per domain → work concurrently.

## When to Use

**Use when:**

- 3+ test files failing with different root causes
- Multiple subsystems broken independently
- Each problem can be understood without context from others
- No shared state between investigations

**Don't use when:**

- Failures are related (fix one might fix others)
- Need to understand full system state first
- Agents would interfere (editing same files)

## Pattern

### 1. Identify Independent Domains

Group failures by what's broken:

```
File A tests: Tool approval flow
File B tests: Batch completion behavior
File C tests: Abort functionality
```

### 2. Create Focused Agent Tasks

Each agent gets:

- **Specific scope:** One test file or subsystem
- **Clear goal:** Make these tests pass
- **Constraints:** Don't change other code
- **Expected output:** Summary of findings and fixes

### 3. Dispatch in Parallel

```typescript
Task("Fix agent-tool-abort.test.ts failures")
Task("Fix batch-completion-behavior.test.ts failures")
Task("Fix tool-approval-race-conditions.test.ts failures")
// All three run concurrently
```

### 4. Review and Integrate

- Read each summary
- Verify fixes don't conflict
- Run full test suite
- Integrate all changes

## Agent Prompt Template

```markdown
Fix the failing tests in [file]:

1. [test name] - [symptom]
2. [test name] - [symptom]

Your task:
1. Read the test file and understand what each test verifies
2. Identify root cause
3. Fix the issue
4. Do NOT change unrelated code

Return: Summary of root cause and changes made, ending with the mandatory reconciliation block:

```yaml:reconciliation
agent_verdict: SUCCESS # SUCCESS | FAILURE | PARTIAL
files_modified:
  - path/to/file1
tickets_created: []
unresolved_blockers: []
summary: "Brief 1-2 sentence human summary"
```

## Mandatory Structured Handoff

Every dispatched subagent MUST conclude with the `yaml:reconciliation` block so the orchestrator can mechanically verify changes against `git status` without ambiguous parsing.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Too broad scope | One file/subsystem per agent |
| No context | Include error messages and test names |
| No constraints | Specify what NOT to change |
| Vague output | Request specific summary format |
