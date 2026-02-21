---
name: prd
description: Use when gathering requirements, creating a PRD, or defining product requirements. Guides interactive requirements elicitation through structured dialogue before any implementation begins.
---

# PRD Generation

Create a Product Requirements Document through interactive dialogue. Ask questions one at a time, validate assumptions, and produce a structured PRD that feeds into planning and issue tracking.

## Process

### 1. Understand the Problem

- Ask ONE question at a time (prefer multiple choice when possible)
- Clarify: What problem are we solving? Who experiences it? What does success look like?
- Identify existing constraints (tech stack, timeline, dependencies)
- Do NOT assume requirements — extract them through dialogue

### 2. Define Personas

- Propose 2-3 personas based on the problem space
- Each persona: name, role, goal, pain point
- Confirm with user before proceeding
- Keep it lean — skip personas if the project is internal tooling

### 3. Elicit Requirements

**Functional Requirements (FRs):**

| ID | Requirement | Priority | Epic |
|----|-------------|----------|------|
| FR-001 | Description | Must/Should/Could/Won't | EPIC-001 |

- Assign unique IDs (FR-001, FR-002, ...)
- Use MoSCoW prioritization (Must, Should, Could, Won't)
- Group related FRs into Epics

**Non-Functional Requirements (NFRs):**

| ID | Category | Requirement | Priority |
|----|----------|-------------|----------|
| NFR-001 | Performance | Description | Must/Should |

- Categories: Performance, Security, Scalability, Reliability, Usability, Compliance

### 4. Structure Epics

- Group FRs into logical Epics (3-7 epics typical)
- Each Epic gets:
  - Unique ID (EPIC-001)
  - Title and description
  - Acceptance criteria as checklist
  - List of contained FRs

### 5. Finalize and Save

- Use the PRD template from `references/prd-template.md`
- Save to `docs/prd/<project-name>-prd.md` in the project repo
- Suggest: "Should I also save a copy to the knowledge vault under `10_projects/<project>/`?"

## Traceability

Maintain a mapping from requirements to implementation:

```
FR-001 → EPIC-001 → Story/Issue #N
FR-002 → EPIC-001 → Story/Issue #M
FR-003 → EPIC-002 → Story/Issue #P
```

This matrix is included at the bottom of the PRD and updated when issues are created via the `prd-to-issues` skill.

## Rules

1. **One question at a time** — never dump a questionnaire
2. **Multiple choice preferred** — reduce cognitive load on the user
3. **Validate before proceeding** — confirm each section before moving to the next
4. **MoSCoW is mandatory** — every FR and NFR must have a priority
5. **Unique IDs required** — every requirement gets a traceable ID
6. **Keep it lean** — skip sections that add no value (e.g., personas for CLI tools)
7. **No implementation details** — PRD describes WHAT, not HOW
8. **Template required** — always use `references/prd-template.md` for output format
