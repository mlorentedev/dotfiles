---
name: qa-plan
description: Use when creating test plans, QA plans, or acceptance testing strategies. Generates structured test scenarios from PRDs with traceability back to requirements.
---

# QA Plan Generation

Generate a structured test plan from an existing PRD. Every test scenario traces back to a functional requirement and acceptance criterion.

## Process

### 1. Read PRD

- Look for PRD in `docs/prd/` in the current project
- If not found, ask the user for the PRD location
- Extract: Epics, FRs, NFRs, acceptance criteria

### 2. Generate Test Scenarios

For each acceptance criterion, generate test scenarios across these categories:

| Category | Description | Example |
|----------|-------------|---------|
| Happy Path | Expected behavior with valid inputs | User logs in with correct credentials |
| Edge Cases | Boundary conditions and limits | Empty form submission, max-length input |
| Error Handling | Invalid inputs and failure modes | Network timeout, invalid token |
| Security | Auth, injection, access control | SQL injection in search field |
| Performance | Load, response time, resource usage | API response under 200ms at 100 RPS |

### 3. Classify Each Scenario

Mark each test scenario as:
- **Manual** — requires human judgment (UX, visual, exploratory)
- **Automated** — deterministic, repeatable (unit, integration, API)
- **Deferred** — not testable yet (infrastructure not ready, out of current scope)

### 4. Save Test Plan

Save to `docs/qa/<project-name>-test-plan.md` in the project repo.

## Output Format

```markdown
# Test Plan: [Project Name]

> **PRD:** docs/prd/<project-name>-prd.md
> **Date:** YYYY-MM-DD
> **Coverage:** X scenarios across Y epics

## EPIC-001: [Title]

### FR-001: [Requirement]

| # | Scenario | Category | Type | Pass Criteria | Status |
|---|----------|----------|------|---------------|--------|
| TC-001 | [Description] | Happy Path | Automated | [Expected result] | [ ] |
| TC-002 | [Description] | Edge Case | Automated | [Expected result] | [ ] |
| TC-003 | [Description] | Error | Manual | [Expected result] | [ ] |

### FR-002: [Requirement]

| # | Scenario | Category | Type | Pass Criteria | Status |
|---|----------|----------|------|---------------|--------|
| TC-004 | [Description] | Happy Path | Automated | [Expected result] | [ ] |

## Coverage Summary

| Epic | FRs | Scenarios | Automated | Manual | Deferred |
|------|-----|-----------|-----------|--------|----------|
| EPIC-001 | 2 | 4 | 3 | 1 | 0 |
| **Total** | | | | | |
```

## Traceability

Every test scenario MUST reference:
- **FR-ID** — which functional requirement it validates
- **EPIC-ID** — which epic it belongs to
- **TC-ID** — unique test case identifier (TC-001, TC-002, ...)

## Rules

1. **PRD required** — do not generate a test plan without a PRD
2. **Full coverage** — every acceptance criterion must have at least one test scenario
3. **Unique IDs** — every test case gets a TC-NNN identifier
4. **Category balance** — do not generate only happy path tests; include edge cases and error handling
5. **No implementation** — this skill produces test PLANS, not test code (use `test` skill for code)
6. **Security scenarios** — always include at least one security test per epic that handles user input
7. **Measurable pass criteria** — every scenario must have a concrete expected result, not "works correctly"
