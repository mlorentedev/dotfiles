---
spec: "HARNESS-041-ci-path-filtering"
verdict: "FAIL"
reviewed_sha: "b776e9da2b18dce2587adce65ad5f18bf4218978"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: HARNESS-041-ci-path-filtering (base `76d875159d3ac4644357b04fb3d328b7d2c58464` → `b776e9da2b18dce2587adce65ad5f18bf4218978`, as resolved by the launcher; the spec's own change is commit `3dadead`, PR #1105)

**Sources**: `specs/HARNESS-041-ci-path-filtering/{proposal,tasks,verification}.md`,
`git diff 76d875...HEAD` (374 commits / 1351 files — the spec's own change is 5 files at
`3dadead`; the rest of the range is later work, reviewed here only for how it interacts with
this spec's mechanism), `.github/workflows/ci.yml`, `tests/ci-path-filtering.bats`,
`forge/branch-protection.json`, `scripts/check-doc-paths.sh`.

**Verification commands run this session** (all at `b776e9d`):

- `bats tests/ci-path-filtering.bats tests/workflow-timeouts.bats tests/workflow-job-names.bats` → 9/9 ok, exit 0
- `actionlint .github/workflows/ci.yml` → exit 0
- PyYAML parse of `ci.yml` enumerating every step's `if:` → all heavy steps in `lint`,
  `lint-powershell`, `test`, `test-windows`, `integration` carry a guard; all five declare `needs: [changes]`
- Mutation 1 (remove `needs: [changes]` from `test-windows`) → bats 4/4 still green; actionlint errors (but see finding 1: nothing in this repo runs actionlint)
- Mutation 2 (remove the `if:` guard from the heavy "Run bats test suite" step) → bats 4/4 green **and** actionlint exit 0
- Mutation 3 (drop `setup-windows.ps1` from the `code` filter) → bats 4/4 green
- `git diff-tree` scan of all 374 commits in range against the `code` filter globs → 40 commits that would skip every test while touching files the suite reads

### Spec and task alignment

| AC | Claimed | Actual at `b776e9d` | Named test | Verdict |
|----|---------|----------------------|------------|---------|
| 1 | `changes` job using `dorny/paths-filter@v3` | job exists; now SHA-pinned **v4** (dependabot #1220) | `@test "HARNESS-041: ci.yml defines a changes job with dorny/paths-filter"` (greps the action name only) | intent met, letter stale |
| 2 | five jobs depend on `changes` | verified in YAML: all five declare `needs: [changes]` (`integration` also needs `lint`) | `@test "HARNESS-041: matrix jobs depend on changes job"` — **survives removing `needs`** | met in code, UNTESTED |
| 3 | heavy steps carry conditional guards | verified in YAML: every non-checkout step in the five jobs has an `if:` referencing `needs.changes.outputs.*` (one later-added lint step is event-guarded only, finding 7) | `@test "HARNESS-041: heavy steps carry changes conditional filter"` — **survives removing a guard** | met in code, UNTESTED |
| 4 | regression suite passes | 4/4 (verification.md says "3/3"; file has always had 4 tests) | itself | met, evidence miscounted |

Non-goals (AI reviewer integration, branch protection changes) respected: no branch-protection
files were altered. Out-of-scope creep: none in `3dadead`.

**Named risk mitigated (strength):** the proposal's "required check blockage" risk does not
materialize — all five jobs stay active at job level and only steps skip, so `lint`,
`lint-powershell`, `test`, `test-windows` (all in `forge/branch-protection.json`) report on
every PR; `cli.yml` deliberately carries no `pull_request` paths filter for the same reason
(documented in its header), and `spec-gate`/`review-attestation` have none either.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | REAL | test-traceability | The named tests for AC2/AC3 do not detect violations of them: deleting `needs: [changes]` from `test-windows`, or the `if:` guard from the heavy "Run bats test suite" step, leaves all four tests green. Mutation 2 (unguarded heavy step) is caught by nothing at all — actionlint exits 0, and actionlint is not wired into CI, pre-commit or any script (repo-wide grep: only a comment in `pr-agent.yml` and one other spec's verification command). So the only always-on gate over this feature is a grep suite that a one-line regression passes. | mutations run this session (output above); `grep -rn actionlint` across workflows/pre-commit/scripts | **UNTESTED** — `@test "HARNESS-041: matrix jobs depend on changes job"` and `@test "HARNESS-041: heavy steps carry changes conditional filter"` both survive their own mutation | tests (assert `needs: [changes]` per job block; assert every non-checkout step in each matrix job carries a `needs.changes.outputs.*` guard; show red run) |
| Major | REAL | filter coverage | The `code` filter omits non-docs paths the suite itself reads, so PRs touching only them are classified as docs-only and skip **every** test and lint step. Omissions include `secrets/**` (note: `sensitive/**` is listed, `secrets/` is not), `install.sh`, `ssh/**`, `systemd/**`, `forge/**`, `env-contract.json`, `machine.json.example`, `session-start-config.json`, `.gitattributes`, `.gitignore`, `.claude/**`. Observed occurrences in range: 6 merged PRs touched only `secrets/registry.yaml` (#1775, #1768, #1760, #1758, #1640, #1561) while `tests/setup-linux.bats:69`, `tests/verify-setup.bats:36`, `tests/pr-agent-config.bats:187`, `tests/secrets-show-callsites.bats:30` read that exact file; #1211 touched only `ssh/config` (read by 3 test files); #1465 touched only `.claude/CLAUDE.md`, which is precisely what `scripts/check-doc-paths.sh` (BUG-088) scans — and that lint step is behind the same `code` guard, so the check written to catch dead paths in instruction files did not run on the PR changing one. Breakage is not silent forever (the `push` guard runs the full matrix on main post-merge), but a pre-merge gate has become a red-main gate for these classes. | `git diff-tree` scan: 40 of 374 in-range commits would skip CI while touching files tests read | **UNTESTED** — no test compares filter entries against the paths the suite reads | code (add to `code` filter or record an explicit exclusion) + tests (coverage test over top-level entries) |
| Minor | REAL | spec drift | AC1 says `dorny/paths-filter@v3`; HEAD runs SHA-pinned `# v4`. Test 1 greps only the action name, so version drift is undetectable by design. | `git show 3dadead:.github/workflows/ci.yml` → `@v3`; HEAD → `@ceb8a2b8… # v4` (#1220) | `@test "HARNESS-041: ci.yml defines a changes job with dorny/paths-filter"` does not pin the version | spec artifacts (`proposal.md` — contract set) |
| Minor | REAL | verification evidence | `verification.md` claims "`bats tests/ci-path-filtering.bats` (3/3 pass)"; the file has always had 4 tests (4/4 this session). More importantly the proposal's core promise — docs-only PRs "run and report green in ~3-5 seconds" — has no measurement anywhere, and a `windows-latest` runner cannot provision that fast, so the number is aspirational. | bats output `1..4`; `git show 3dadead:tests/ci-path-filtering.bats` = 4 `@test` | n/a (evidence file) | `verification.md` (outside contract set) |
| Minor | REAL | spec artifacts | `proposal.md` leaves all four AC boxes `[ ]` and `status: implementing` while `tasks.md` marks every box `[x]` (including "PR opened … merged `3dadead`"). Inconsistent state for the archive gate to reconcile. | read of both files | n/a | spec artifacts (`proposal.md` — contract set) |
| Minor | REAL | spec artifacts | `tasks.md` documents "This spec emits a sibling `features.json`" with a skeleton; no `features.json` exists (the launcher records an empty digest). `cli/internal/spec/archive.go` never references it, so this looks non-blocking, but the contract text promises an artifact that is not there. | `ls specs/HARNESS-041-ci-path-filtering/`; grep of `cli/internal/spec/` | UNTESTED | spec artifacts (`tasks.md` — contract set) |
| Minor | THEORETICAL | range interaction | lint's "Verify a retired twin took its bats/Pester tests with it" step (added later, #1460) carries only an event guard, so it runs — including `git fetch --unshallow` — on every docs-only PR, eroding this spec's stated ~3-5 s target for that job. | `ci.yml` step `if: github.event_name == 'pull_request'` | n/a | code (guard on `code`, or accept and document) |
| Minor | THEORETICAL | range interaction | `pi-nan-package` (AI-046) skips at **job level** — the exact pattern this proposal's risks section rejects. It is not in the required-check list today, so nothing blocks, and `@test "required rule: a required job with a skipping if: is flagged, one under always() is not"` would flag it if it ever became required. | `forge/branch-protection.json` (no `pi-nan-package`); `tests/workflow-job-names.bats` | `@test "required rule: a required job with a skipping if: is flagged, one under always() is not"` | code (tracked; do not gate) |
| Minor | REAL | repo debt noticed in range | `diff.patch` was committed at the repo root by #1404 (`d4ea0f5`) and is inside the review range. Not this spec's change, but it is untracked-in-spirit junk in the tree. | `git log -1 -- diff.patch` | UNTESTED | code (follow-up ticket) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | All four criteria are satisfied in the current YAML (verified by parse), but AC1's letter drifted to v4 and two criteria are unenforced (mutations survive); whole classes of non-docs changes are misclassified as docs-only. |
| Verification       | C | Evidence is test-name mapping that does not survive a one-line mutation, a 3/3-vs-4 count mismatch, and no measurement of the "~3-5 seconds" promise. |
| Scope              | A | The spec's own commit `3dadead` touches exactly `ci.yml`, the bats suite and the three spec files; no creep, non-goals respected. |
| Reliability        | B | Required-check mitigation holds end-to-end (verified against `forge/branch-protection.json` and the other workflows); a failed `changes` job fails closed rather than green-skipping. |
| Maintainability    | B | Actions SHA-pinned, comments explain *why*, actionlint-clean; but the filter is hand-maintained with no coverage test (CI-002's own comment admits the analogous set "is not proven closed"). |
| Handoff-readiness  | C | Proposal ACs unticked + `status: implementing`, features.json promised-but-absent, verification miscounted; later specs patched the filter without a lesson recording the hand-maintenance hazard. |

Aggregation: C present, no D → PASS WITH GAPS floor; severity path overrides (below).

### Verdict

**FAIL** — two **REAL Majors**: (1) the named tests for AC2/AC3 do not detect violations of
those criteria (reproduced by mutation, and nothing else in the repo catches the guard
mutation), an UNTESTED Major by the traceability gate; (2) the `code` filter omits non-docs
paths the suite reads, with 40 in-range commits — including six `secrets/registry.yaml`-only
PRs — actually taking the skip path. A Blocker was not claimed: AC1's v3→v4 drift and the
stale contract text are Minor spec-artifact findings. The SPECULATIVE/THEORETICAL findings
(7, 8) are surfaced and tracked, and do not drive this verdict.

### Recommended next steps

Contract set (`proposal.md`, `tasks.md`, `features.json`) is closed to edits under a passing
verdict — here the verdict is FAIL, so a re-review follows and the contract fixes belong to
that round. Minimum set that flips this to PASS:

1. **tests** — make the AC2/AC3 tests kill their mutations: assert `needs: [changes]` in each
   of the five job blocks, and assert every non-checkout step in those jobs carries a
   `needs.changes.outputs.*` guard; attach a red run (mutation reverted → green) as evidence.
2. **code + tests** — close the filter: add the non-docs paths the suite reads
   (`secrets/**`, `install.sh`, `ssh/**`, `systemd/**`, `forge/**`, `env-contract.json`,
   `machine.json.example`, `session-start-config.json`, `.gitattributes`, `.gitignore`,
   `.claude/**`) or record each exclusion as a deliberate decision; add a test that compares
   top-level repo entries against the filter so a future directory cannot silently fall into
   "docs-only".
3. **spec artifacts** (re-review round) — reconcile AC1's wording with the pinned v4 action,
   tick the AC boxes, flip `status:`, and either emit the promised `features.json` or delete
   the promise from `tasks.md`.
4. **verification.md** — correct 3/3 → 4/4, and either measure a docs-only PR's wall time or
   drop the "~3-5 seconds" number.
5. **Disposition, not fixes** (implementer's call, record in `verification.md`): findings 7,
   8 and 9 are out-of-spec interactions — apply, ticket, or decline with a reason.

**`dotf spec archive` is NOT advisable in the current state**: the verdict is FAIL, and the
archive gate requires a fresh, passing review. The two Majors must land in code/tests (not in
the contract set) and be re-reviewed before archive.
