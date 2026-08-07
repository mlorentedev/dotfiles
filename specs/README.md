# specs/

Spec-Driven Development (SDD) workspace. Each spec is a short-lived folder that lives
here during implementation and moves to `specs/archive/` once the PR merges.

## Active specs (`specs/<id>/`)

A spec folder contains three files created by `/spec init <id>`:

| File | Purpose |
|---|---|
| `proposal.md` | Why + What + acceptance criteria (filled via `/spec fill`) |
| `tasks.md` | TDD-ordered implementation checklist |
| `verification.md` | Evidence: commit hashes, test outputs, smoke results |

Naming convention: `<TICKET-ID>-<kebab-slug>/` (e.g. `SELF-001-init-project-self-contained/`).

## Archive (`specs/archive/`)

Completed or abandoned specs move here after `/spec archive`. **The archive is cold
storage** — do not open files here during normal work. The spec-gate CI explicitly
excludes `specs/archive/*` from production diff counting.

As of 2026-08, the archive holds ~117 specs and the active tree ~23. Those counts
are load-bearing, not trivia: an active spec that has actually shipped is an
**alibi** for the gate's `SPEC_FLOOR` heuristic — any large PR can touch ten lines
of one and satisfy the Discipline Gate. Archiving on merge is what keeps that
surface small, and is now enforced (see below). See COLD-001 (#251) for the
cold-store formalisation work.

## Discipline Gate

The spec-gate CI job (`scripts/check-spec-gate.sh`) fails any PR with ≥ 50 LOC of
production diff that lacks an active `specs/<id>/` folder. Escape hatch: `skip-sdd`
label + `## SDD skip rationale` section in the PR body.

It also enforces the lifecycle's terminal step: a PR that **closes** an issue must
archive the active spec tracking it (`dotf spec archive <id>`), matched via the
spec's `issue:` frontmatter. This runs regardless of diff size and is not waived by
`skip-sdd`. Reference an issue without a closing keyword (`Refs #N`) when the work
genuinely continues; the escape hatch is the `skip-archive` label + an
`## Archive skip rationale` section.

See `AGENTS.md` "Spec-Driven Development" for the full trigger criteria and workflow.
