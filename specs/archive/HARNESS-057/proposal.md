---
id: "HARNESS-057"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-08"
issue: "mlorentedev/dotfiles#823"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-057

> **Naming**: file lives at `<repo>/specs/HARNESS-057/proposal.md`. `HARNESS-057` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #823: HARNESS-057: one frontmatter contract for the skill library, enforced by the engine -->

The 36-skill library carried three frontmatter dialects — 9 skills with the store's full Frontmatter Law, 7 with `name`/`description`/`targets`, 20 with only `name`/`description`, and `owner` missing from 33 of 36. They coexisted because the store's law and the engine's contract are different statements and only the second is checked: the schema required `name` and `description`, so every dialect validated and rendered cleanly. A law nothing enforces is a preference, and a library of preferences drifts a little further with each import.

## What

Every skill carries the same seven keys, and the engine refuses to render one that does not. `compile-harness.sh --check` fails on a record missing any of `name`, `description`, `id`, `type`, `status`, `created`, `owner` — verified by removing a key and watching it fail, not by assuming the validator reads the schema it is handed.

## Out of scope

- The rest of the agentskills.io scope in the umbrella (progressive disclosure, a `references/` convention, description-length convergence). Those are content decisions; this is the contract they will be written against.
- Fabricating provenance for the 17 skills migrated from the old `ai/skills/` tree. Their upstream is unknown, and inventing a `source` would launder a gap into a claim.
- Relocating `agent-lifecycle`, whose runtime-specific keys the contract now admits via `additionalProperties`. Whether it belongs in an overlay instead stays open in the umbrella.

## Risks / open questions

- **A decorative guard.** The validator awk-greps required keys rather than evaluating the schema, so tightening `required[]` could have changed nothing. Resolved empirically: a record with `owner` removed makes `--check` exit non-zero naming the key.
- **Backfilled dates.** Stamping today's date on a file written in May would read as knowledge while being false; each `created` is derived from the file's first commit instead.
- **Type clauses are documentation.** The schema's `type`/`const`/`minLength` are not evaluated by this validator; a passing `--check` is presence, not type validation. Stated in the script comment so nobody infers more than it proves.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [x] Every skill in the store carries the seven required keys, with `created` matching its first commit.
- [x] The schema requires all seven and `--check` fails when a record drops one, demonstrated red before green.
- [x] The vendored-provenance rule (`source` + `license` + attribution row, only for vendored skills) is recorded in the pattern rather than in the schema.
- [x] A test asserts every committed record satisfies the contract, so the next import cannot reintroduce a dialect.

## References

- Bitácora board: mlorentedev/dotfiles#823; umbrella mlorentedev/knowledge#29 with the audit evidence in its comments
- Related patterns: `00_meta/patterns/pattern-cross-agent-skill-pipeline.md` (contract + provenance rule)
- Encountered and ticketed on the way: mlorentedev/dotfiles#825 — the work-gate cannot reference an issue in another repository, which is why this ticket exists separately from the umbrella
