---
id: "<feature-id>"
type: spec
status: draft # draft | implementing | verifying | archived
created: "{{date:YYYY-MM-DD}}"
issue: ""   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# {TITLE}

> **Naming**: file lives at `<repo>/specs/<feature-id>/proposal.md`. `<feature-id>` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

Single paragraph. The user or business problem this feature solves. Link to the vault roadmap or the bitácora board issue if applicable. If you cannot write this in 3 sentences, you do not understand the problem yet.

## What

Concrete behavior change. What does the system do after this PR that it did not do before? Observable, not implementation-focused.

## Out of scope

Things this PR explicitly does NOT include. Forces a sharp boundary and prevents scope creep.

-
-

## Risks / open questions

Failure modes, dependencies, and unknowns to clarify before implementation. If any item here is unresolved, do not move to `tasks.md` yet.

-
-

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] Outcome 1
- [ ] Outcome 2
- [ ] Outcome 3

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `<repo>/docs/adr/adr-XXX.md` (if any)
- Related patterns: `00_meta/patterns/<pattern>.md` (if any)
