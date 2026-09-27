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
- [ ] **AC5:** the promotion pre-flight lands in the archive PR (branch `feat/archive-promotion-check`), with 12 cases, 10 mutations, and the command's vault wiring.
- [ ] **AC6:** the PR template and DoD §2 land in the wiring PR; the `verification.md` template lands in the archive PR.
- [ ] **AC7:** the wiring PR's own section names ADR-039 and lesson 301. It is checked when the PR opens.

## Test status

- `bats tests/check-knowledge-gate.bats tests/spec-gate-pr.bats tests/knowledge-gate-workflow.bats tests/stub-real-pairing.bats`: 62 run, 0 failures, on the wiring branch rebased onto main after #1732.
- `cd cli && go test ./...` passes. `compile-harness.sh --check` reports no drift, and `actionlint` passes on `knowledge-gate.yml`.
- No regressions: the adapter's existing cases, and spec-gate's, are unchanged and pass.

## Decisions made during implementation

- **The id became HARNESS-160.** HARNESS-024 is held by an archived spec (#446), and `guard-spec-ids-unique` refused the collision. #387 was retitled with it.
- **The checker runs under zsh as well as bash,** because the repository's scripts must (pr-agent's finding on #1732). Every case runs under both shells and fails if they disagree.
- **HTML comments are stripped and fenced code is ignored,** because GitHub renders neither as the author's answer. An unclosed fence is named in the refusal.
- **A reason copied verbatim from the template (`<reason>`) is refused.**
- **The vault edits are held.** The DoD line (`659b33c8`) and the promotion template (`aec3300b`) are reverted in vault `99530771` until their PRs merge, because the template made every local `go test` on main fail `TestEmbeddedTemplatesMatchVault`.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-160-knowledge-capture-gate/` -> `specs/archive/HARNESS-160-knowledge-capture-gate/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
