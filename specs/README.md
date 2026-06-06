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

As of 2026-06, the archive contains ~44 specs. This is the main driver of "too much
content" perception in the repo — not OS mixing or script sprawl. The active tree has
~26 specs. See COLD-001 (#251) for the cold-store formalisation work.

## Discipline Gate

The spec-gate CI job (`scripts/check-spec-gate.sh`) fails any PR with ≥ 50 LOC of
production diff that lacks an active `specs/<id>/` folder. Escape hatch: `skip-sdd`
label + `## SDD skip rationale` section in the PR body.

See `AGENTS.md` "Spec-Driven Development" for the full trigger criteria and workflow.
