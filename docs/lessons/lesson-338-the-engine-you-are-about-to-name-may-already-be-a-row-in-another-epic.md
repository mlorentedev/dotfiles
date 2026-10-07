---
id: "lesson-338-the-engine-you-are-about-to-name-may-already-be-a-row-in-another-epic"
type: lesson
status: active
title: "The engine you are about to name may already be a row in another epic"
created: "2026-10-06"
---

# The engine you are about to name may already be a row in another epic

## Context
The macOS bring-up (PLAT-001, #2013) needed one cross-OS entrypoint for a factory-fresh machine.
The design converged quickly in conversation: a thin shell installer that fetches `dotf`, then a
declarative Go engine with ordered, idempotent steps, a plan mode and per-step post-conditions.
The working name was `dotf bootstrap`, and it was proposed to the owner under that name.

## The Trap
The engine already existed as a plan. Epic #1843 rows B6 and B7 define `dotf converge` (a
reconciler registry with `--plan` and an apply mode), and ADR-041 decision 4 fixes the order of a
convergence run. Neither was in the PLAT-001 handoff, and neither had code or an issue, so a
search of the CLI and the specs found nothing. Only reading ADR-041 while drafting the new ADR
surfaced it. Shipping `dotf bootstrap` would have created two engines for one job, which is the
exact defect the work set out to remove from the twin setup scripts. #2013's own constraints
already said "where a #1843 row covers the work, depend on it instead of copying it". Nobody
checked that rule against a name that had not been written down yet.

## The Solution
Before naming a new command, subsystem or engine, search the open epics and the ADRs, not only
the code: `gh issue list --label epic --state open` and a grep of `docs/adr/` for the concept
(here: "converge", "reconciler", "bootstrap"). A planned row with no implementation is still a
design decision someone already took. When one is found, deliver it under its own name and leave
a comment on the owning epic saying which spec delivers the row (here: #1843, spec
`PLAT-001b-one-entrypoint`, ADR-045), so the row is not built twice.
