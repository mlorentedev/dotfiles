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

## Round 2 — after the first adversarial review

Round 1 (`reviewed_sha 508ca10`) returned **FAIL** on one REAL Blocker and one
REAL Major. Both independently verified by reading the actual code, not taken
on the reviewer's word.

**The Blocker: `setup-windows.ps1`'s PowerShell twin never got the fix.**
`Convert-SkillRecord` — the PS equivalent of `render_skill` — never received
the strip-rule that removes a record's own pre-existing `generated_*` lines
before injecting deploy's fresh set. Since this spec makes every committed
record carry its own provenance, every Windows `skill`/`command`-kind deploy
now produces duplicate `generated`/`generated_from`/`generated_sha` YAML
keys. Confirmed by reading `Convert-SkillRecord` directly (no equivalent
strip logic existed) and by porting its loop to Python against a real
on-disk record: 6 `generated*` lines instead of 3. This shipped live on
`main` with #927 — no test in `test-windows` CI asserts on deployed content,
only source-text `grep` on the `.ps1` file.

Two sweeps the reviewer did NOT do, done before writing the fix (floor, not
ceiling): confirmed there is no `Convert-AgentRecord` twin (agent records
have no PowerShell render path with this risk — `render_agent`'s bash side
was already allowlist-safe and has no PS equivalent to duplicate), and
confirmed `Deploy-SkillRecord`'s `$SrcPath` matches bash's vault-relative
convention exactly (`"$vsub/$($rec.Name)/SKILL.md"`).

**Fix**: mirrored bash's strip rule in `Convert-SkillRecord`, using
**`-cmatch`** rather than `-match` — PowerShell's `-match` is
case-insensitive by default, and bash's `grep`/`awk` are not; a byte-for-byte
port using `-match` would not actually mirror the bash behavior.

**The Major: `proposal.md`'s AC2 wording was wrong, not the code.** AC2 said
deployed provenance describes "the $HOME copy's relation to the record" —
but `generated_from` in a deployed copy names the **vault** path (where to
edit), while `generated_sha` hashes the **record** (what the copy was built
from). Two referents in one field pair, verified pre-existing (unrelated to
this spec's diff — `srcpath` at the deploy call sites was never touched) and
deliberate: `generated_from` is a "where do I fix this" pointer for a human,
`generated_sha` is a "is this still fresh" drift check. Fixed by rewording
AC2 and the "What" section in `proposal.md`, plus a doc comment in both
`render_skill` (bash) and `Convert-SkillRecord` (PS) stating the dual-referent
semantics explicitly, so the next reader doesn't re-discover this as a bug.

### Fix

- `setup-windows.ps1`: added the `-cmatch '^generated(_from|_sha)?:'` strip
  rule to `Convert-SkillRecord`, plus a doc comment above the function.
- `scripts/compile-harness.sh`: expanded `render_skill`'s doc comment to
  state the dual-referent semantics explicitly (no code change — the bash
  side was already correct, only the Blocker's explanation was missing).
- `specs/HARNESS-069/proposal.md`: reworded AC2 and the "What" section.
- `tests/setup-windows.bats`: one new **runtime** test —
  `Convert-SkillRecord does not stack a second set of generated_* fields on
  a record that already carries its own` — extracts the real function body
  (CRLF-stripped) via awk, feeds a fixture record (already carrying its own
  `generated_*` fields, as a HARNESS-069-refreshed record would) through
  `pwsh -NonInteractive -Command`, and asserts exactly one set of each field
  survives, with the fresh set's `generated_from` (not the stripped stale
  one) present in the output. Gated on `pwsh` availability like every other
  PS-invoking test in this suite (`# skip pwsh not available` locally; runs
  for real in the `test-windows` CI job). This is deliberately NOT a
  source-text grep — round 1's own Python port of the buggy logic was
  evidence of reproduction, not a regression test (the #891 corpus lesson:
  a port tests the port, not the original).

### Round-2 evidence

- `bats tests/setup-windows.bats` → 110/110 (new test present, `# skip pwsh
  not available` locally on this Linux dev machine — this repo has no local
  pwsh; the test executes for real only in `test-windows` CI, same as every
  other PS-runtime test in this suite)
- `./scripts/check-bats-names.sh tests/setup-windows.bats` → clean
- `shellcheck scripts/compile-harness.sh`, `bash -n`/`zsh -n` → clean
- `./scripts/check-md-escapes.sh specs/HARNESS-069/proposal.md` → clean
- Manual review: `setup-windows.ps1` line-ending integrity confirmed —
  2120/2120 lines still CRLF after the edit (`.gitattributes` declares
  `*.ps1 text eol=crlf`), no mixed endings introduced
- Could not run `pwsh`/PSScriptAnalyzer locally (not installed on this
  machine) — the new PS code is unverified by a live interpreter on THIS
  machine; CI's `test-windows` job is the first real execution. Disclosed,
  not hidden.

### Round 2 review — PASS

`reviewed_sha 8e0d891` (this branch's HEAD at review time). Both round 1
findings independently re-verified closed with fresh reproduction evidence
(not taken on the prior round's word) — see `review.md` for the full
methodology, including a byte-for-byte Python cross-validation of the
Windows strip rule against the real bash `awk` block, and primary-source
confirmation (GitHub's own `actions/runner-images` docs) that the new pwsh
regression test executes in the unconditional `test` job on `ubuntu-latest`,
not only the PR-gated `test-windows` job as this file had claimed.

Five Minor/informational findings, all explicitly scoped by the reviewer as
optional and not required for PASS: an `inject_record_provenance` strip-rule
asymmetry with its sibling `render_skill`, a stale test comment
(`tests/compile-harness.bats:228-230`) still describing the pre-round-1-fix
single-referent framing, two coverage-asymmetry gaps carried/found (bash AC2
opencode/copilot field counts, PowerShell `command`/`prompt` kinds), and one
informational note (no `Convert-AgentRecord` twin exists, confirmed
pre-existing and out of scope). Filed as #933 rather than extending this
branch further, since fixing them here would move HEAD past the reviewed sha
and force a third review round for polish-only changes.

Per the reviewer's own note: this is the **third** documented instance of
the bash/PowerShell twin drifting (same class as #776 and others) —
commented on #909 (CLI-035, the port that deletes both twins) citing this
incident as motivation, rather than filing a fourth mechanism-level ticket
for a class that already has one.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons.md`? Yes — an apostrophe in a comment written inside an open single-quoted `awk '...'` block silently reopens bash's own parser, and the resulting syntax error can present as unrelated test failures far from the actual typo.
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? No — mechanical extension of an existing mechanism, not an architectural decision.
- [ ] New pattern candidate for `00_meta/patterns/`? No — single-repo shell gotcha, not yet observed recurring elsewhere.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/HARNESS-069/` -> `specs/archive/HARNESS-069/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018) — pending PR merge
- [x] Promotions above executed (lesson written to `docs/lessons.md`; no ADR/pattern warranted)
