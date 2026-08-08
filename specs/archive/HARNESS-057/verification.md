---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - HARNESS-057

## Evidence

- [x] AC1 (every skill carries the seven keys) -> test `HERMES-018: every committed skill record satisfies the contract`, and a store-side sweep reporting `all 36 skills carry the full 7-key contract`
- [x] AC1 (`created` matches first commit) -> derived, not stamped: each value came from `git log --diff-filter=A --format=%ad --date=short -1 -- <file>` in the **store**, which is where those files were authored. Deliberately *not* encoded as a `features.json` check — the equivalent command run against this repository compares the records' copy dates and would be a check that passes for the wrong reason.
- [x] AC2 (schema requires all seven, and the guard bites) -> tests `HERMES-018: the schema requires the store's frontmatter law` and `HERMES-018: --check rejects a record that drops a required key`. Proved red first: removing `owner:` from the `audit` record made `--check` exit 1 with `required frontmatter key "owner" missing or empty`; restored, it passes.
- [x] AC3 (vendored-provenance rule recorded in the pattern) -> `pattern-cross-agent-skill-pipeline.md` § One frontmatter contract
- [x] AC4 (a dialect cannot come back) -> test `HERMES-018: a vendored skill carries provenance and an attribution row`, plus the record-satisfaction test above

## Test status

- `bats tests/skills-pipeline.bats` -> 22/22 (18 before, 4 added)
- `bats tests/compile-harness.bats` -> 44/44, unchanged
- `shellcheck scripts/compile-harness.sh` -> clean
- `compile-harness.sh --check` -> no drift with the tightened schema
- Store-side: `vault-validate.py` incomplete-frontmatter count 383 -> 354; no skill appears in the frontmatter findings

## Decisions made during implementation

- **Proved the guard before trusting it.** `validate_skill_frontmatter` greps required keys rather than evaluating the schema, so a tightened `required[]` could have been decorative. Removing a key and watching `--check` fail is the only thing that distinguishes an enforced contract from a documented one.
- **Dates derived, never stamped.** Backfilling `created: 2026-08-08` onto a file written in May would read as knowledge while being false.
- **Provenance left out of the schema.** The 17 skills migrated from the old `ai/skills/` tree have no recorded upstream; a conditional rule in the pattern states what a vendored skill owes, and no `source` was invented to satisfy a validator.
- **`agent-lifecycle` now satisfies the contract but keeps its runtime keys**, admitted by `additionalProperties: true`. Whether it belongs in an overlay instead stays open in the umbrella — this PR does not settle it by silently normalising the file.
- **Corrected a stale comment in the validator** that still described the two-key minimum, and stated explicitly that type clauses are not evaluated so a passing check is not read as more than it proves.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? no — the transferable half (a law nothing enforces is a preference) is recorded in the pattern where the contract lives.
- [ ] ADR-worthy decision? no.
- [x] New pattern candidate for `00_meta/patterns/`? **update, not new** — `pattern-cross-agent-skill-pipeline.md` gained the contract and the provenance rule.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-057/` -> `specs/archive/HARNESS-057/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
