---
tags: [spec, verification, templates]
created: "2026-08-12"
---

# Verification - HARNESS-069

## Evidence

- **AC1** (records stamped with provenance) -> `inject_record_provenance()` +
  its two call sites in `do_refresh`, `scripts/compile-harness.sh`; bats
  `render: --refresh stamps the committed record with its own provenance
  (HARNESS-069), no $HOME write` and the agent equivalent; real
  `VAULT_PATH="$HOME/Projects/knowledge" ./scripts/compile-harness.sh --refresh`
  changed 38 committed records (37 skills + 1 agent), 3 lines each,
  `git diff harness/skills/handoff/SKILL.md` shown in this session's transcript.
- **AC2** (no field duplication) -> the strip rule added to `render_skill`'s
  awk; bats `render: --deploy renders records to $HOME with one set of
  provenance fields, not stacked (HARNESS-069)` plus the mirrored agent
  assertion; real `--deploy` to a scratch `$HOME` showed `generated:1
  generated_from:1 generated_sha:1` on a deployed skill file, not 2.
- **AC3** (`--check` unaffected) -> no change to `do_check`'s own code; real
  `--refresh` then `--check` against the actual vault returned `[check] OK: no
  harness drift` for all 38 records.
- **AC4** (bats coverage updated, not left contradicting reality) -> two
  renamed/rewritten tests (skills + agents "no provenance" tests, now
  asserting the new content) plus new single-set assertions in both deploy
  tests.

## Test status

- `bats tests/compile-harness.bats` -> 44/44, full suite, no regressions
- `shellcheck scripts/compile-harness.sh` -> clean
- `bash -n` / `zsh -n scripts/compile-harness.sh` -> both clean
- Manual smoke test: real `--refresh` against `$VAULT_PATH`, real `--check`,
  real `--deploy` to a scratch `$HOME` (not the fixture) — all three run in
  this session, not only the bats fixture harness
- No regressions in existing test suite: yes — same 44 tests that existed
  before this change all still pass, two of them updated to match new
  (intended) behavior rather than skipped or deleted

## Decisions made during implementation

- **Hit a real bash quoting bug while writing the fix, not a design decision
  but worth recording**: an apostrophe in an English comment
  ("`$HOME's relationship`") written *inside* an already-open single-quoted
  `awk '...'` bash string prematurely closed the quote, producing a bash
  syntax error that made every test needing `do_refresh` fail, including ones
  in completely unrelated sections (ENGINE-002, HARNESS-054) — because the
  parse error was in the same file bash sources as a whole. Root cause found
  by `bash -n` pointing at the exact broken line; fixed by rewording the
  comment to avoid the apostrophe rather than escaping it (`'\''` inside a
  script comment is worse to read than just not using a contraction).
- **`generated_sha` on the record hashes the vault source, not the record
  itself** — deliberate, mirrors the existing deploy-time semantic
  (`generated_sha` in a $HOME copy = sha of the record it came from) so the
  field consistently means "sha of what I was generated FROM", not "sha of
  myself".
- **Did not mark `features.json` entries `"passing"`** despite having run
  every verification command myself with observed exit 0, because this spec's
  own template says only "the harness" may set that terminal state, and the
  implementer self-certifying is exactly the failure mode this repo's whole
  adversarial-review discipline (DOCS-013, same day) exists to catch. Left as
  `"pending"` with empty evidence for an independent run to fill in.
- **Wrote the spec after implementing, not before** — the risk-verification
  step (does `--check` byte-compare the record, would provenance duplicate at
  deploy) was done first as due diligence per the advisor's brief, then the
  code, then this spec filled in to match what was actually built and
  verified. Acceptance criteria and tasks below are accurate to what happened,
  not aspirational.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? Yes — an apostrophe in a comment written inside an open single-quoted `awk '...'` block silently reopens bash's own parser, and the resulting syntax error can present as unrelated test failures far from the actual typo.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — mechanical extension of an existing mechanism, not an architectural decision.
- [ ] New pattern candidate for `00_meta/patterns/`? No — single-repo shell gotcha, not yet observed recurring elsewhere.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-069/` -> `specs/archive/HARNESS-069/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
