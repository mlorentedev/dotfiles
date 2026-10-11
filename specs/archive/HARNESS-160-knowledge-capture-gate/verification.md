---
tags: [spec, verification, templates]
created: "2026-09-25"
---

# Verification - HARNESS-160-knowledge-capture-gate

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] **AC1:** the checker's exit codes and its defects. `tests/check-knowledge-gate.bats` has 33 cases, each run under bash and zsh with identical output, and 16 mutations are each killed. Merged in #1732 (`6ceebd4`).
- [x] **AC2:** the dependency-bot skip, and the bot list held equal to spec-gate's. Same suite, the `dependencies:` cases.
- [x] **AC3:**
  - `spec-gate-pr.sh --gate`: `tests/spec-gate-pr.bats`, 18 cases, including two end to end through the real checker.
  - The workflow's triggers and concurrency: `tests/knowledge-gate-workflow.bats`, 5 cases.
  - 8 mutations of the wiring, each killed.
- [x] **AC4:** the release-please footer passes the checker: the `release-please footer` case.
- [x] **AC5:** the promotion pre-flight merged in #1761 (`030ade1e`), with 12 cases, 10 mutations, and the command's vault wiring. `go test ./internal/spec -run Promotion` passes at `3187ea8d`.
- [x] **AC6:** the PR template and DoD §2 merged in #1759 (`2e73eab3`), the `verification.md` template in #1761. DoD §2 is in the vault's `pattern-change-lifecycle.md` and in every compiled harness instruction file. Feature f6.
- [x] **AC7:** #1759's own section names `docs/lessons/lesson-301-knowledge-goes-where-a-mechanism-asks-for-it.md` and `docs/adr/adr-039-knowledge-asked-for-by-the-pr.md`, and `knowledge-gate` passed on its head. Feature f7.

## Test status

- `bats tests/check-knowledge-gate.bats tests/spec-gate-pr.bats tests/knowledge-gate-workflow.bats tests/stub-real-pairing.bats`: 62 run, 0 failures, on the wiring branch rebased onto main after #1732.
- Re-run at `18029443` for the archive (2026-10-10): 63 run (34 + 19 + 5 + 5), 0 failures. `actionlint .github/workflows/knowledge-gate.yml` exits 0.
- `cd cli && go test ./...` passes. `compile-harness.sh --check` reports no drift, and `actionlint` passes on `knowledge-gate.yml`.
- No regressions: the adapter's existing cases, and spec-gate's, are unchanged and pass.

## Decisions made during implementation

- **The id became HARNESS-160.** HARNESS-024 is held by an archived spec (#446), and `guard-spec-ids-unique` refused the collision. #387 was retitled with it.
- **The checker runs under zsh as well as bash,** because the repository's scripts must (pr-agent's finding on #1732). Every case runs under both shells and fails if they disagree.
- **HTML comments are stripped and fenced code is ignored,** because GitHub renders neither as the author's answer. An unclosed fence is named in the refusal.
- **A reason copied verbatim from the template (`<reason>`) is refused.**
- **The vault edits are held.** The DoD line (`659b33c8`) and the promotion template (`aec3300b`) are reverted in vault `99530771` until their PRs merge, because the template made every local `go test` on main fail `TestEmbeddedTemplatesMatchVault`.

## Required check (slice 4)

`knowledge-gate` is declared required in `forge/branch-protection.json`. `dotf forge protection apply --dry-run` on 2026-10-10: `changed=1 (0 applied, 1 planned) … 0 refused`. Applying it is the owner's step; `dotf forge protection check` reports the one drift until then.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-301-knowledge-goes-where-a-mechanism-asks-for-it.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-039-knowledge-asked-for-by-the-pr.md
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. yes: 00_meta/patterns/pattern-change-lifecycle.md

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-160-knowledge-capture-gate/` -> `specs/archive/HARNESS-160-knowledge-capture-gate/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)

## Archive review dispositions (2026-10-10)

The archive review (`PASS`, reviewed `18029443`) is committed unchanged with this PR. Its four Minor findings:

| Finding | Disposition |
|---------|-------------|
| A code fence inside an HTML comment opens a fence the stripper never closes (a false red, fail-closed) | Ticketed: #2310 (BUG-119), with the reproducer and the single-pass fix. |
| The "Test status" counts no longer reproduce | Applied: the counts are re-measured above (63 run, 0 failures). |
| `proposal.md` lists "declaring the check required" as out of scope, but Slice 4 did it | Accepted as a scope widening, recorded here rather than in the contract set: the owner applied the protection on 2026-10-10 (#2298), so the proposal bullet describes the plan before that decision. |
| `actionlint` was not run by the reviewer | Applied: it is installed now and passes on `knowledge-gate.yml`. |
