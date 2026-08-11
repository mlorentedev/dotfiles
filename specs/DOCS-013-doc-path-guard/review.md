---
spec: "DOCS-013-doc-path-guard"
verdict: "FAIL"
reviewed_sha: "20c4807560a00610c08c35e36e44532c81f875ec"
reviewer: "claude-sonnet-5"
date: "2026-08-11"
---

## Adversarial review

**Scope**: DOCS-013-doc-path-guard, round 4. No PR open; reviewed via
`git -C /home/manu/Projects/dotfiles-wt-r2 diff main...HEAD` at `20c4807`
(branch `fix/docs-013-review-findings`, base `main` `6267bea`). Round 4's own
commit is `20c4807` alone (`test(docs): discover governed instruction files
instead of listing them`); `f91a08d`/`f5ee9a3` are rounds 2/3's already-reviewed
fixes, re-verified here only in passing.

**Sources**: `specs/DOCS-013-doc-path-guard/{proposal,tasks,verification}.md`,
`scripts/check-doc-paths.sh`, `tests/check-doc-paths.bats`, `cli/internal/spec/review.go`
(frontmatter contract), plus direct execution — bats, shellcheck, bash -n/zsh -n,
manual mutation of the discovery regex and of git-level failure conditions in a
disposable local clone (discarded before writing this file).

Per the brief: diff read and code run before `verification.md`; `verification.md`'s
round-4 claims were then independently reproduced rather than trusted.

### Spec and task alignment

- Round 3's Major (`ai/agy/AGY.md` ungoverned) is fixed, and fixed at the class
  level as `tasks.md`/`verification.md` claim: `instruction_files()` now
  discovers from `git ls-files` instead of a hand-written list. Verified by
  running the exact pipeline in the bats file — it returns exactly the 9 files
  named in the test's own comments (`AGENTS.md`, `ai/agy/AGY.md`,
  `ai/claude/CLAUDE.md`, `ai/copilot/copilot-instructions.md`,
  `ai/hermes/AGENTS.md`, `.claude/CLAUDE.md`, `cli/AGENTS.md`,
  `.github/copilot-instructions.md`, `README.md`), and no more.
- `verification.md`'s testable round-4 claims were reproduced, not just read:
  `bats tests/check-doc-paths.bats` → 13/13 (confirmed, 3 runs); discovery
  returns 9 files (confirmed by direct execution); `shellcheck` clean and
  `bash -n`/`zsh -n` clean (confirmed); the probe test leaves the worktree
  clean **on the success path** (confirmed, repeated 3x). No false claim
  found this round — a contrast with round 2's verification.md, which
  asserted a mutation turned a test red without having run it.
- The "~40 scripts" and "1200+ BATS tests" corrections in `README.md` check
  out against the actual tree (39 `scripts/*.sh`+`*.ps1`, 1211 `@test` cases).
- Acceptance criteria in `proposal.md` are otherwise unchanged from round 1
  and were not re-litigated here; rounds 1–3 already verified them.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| Major | REAL | tests/check-doc-paths.bats — probe hermeticity | The new probe test's cleanup (`git rm --cached` + `rm -rf`) only runs if every command *before* it succeeds. `git -C "$DOTFILES_DIR" add -N "$probe"` is not wrapped in `run`/`\|\| true`; under bats' errexit-on-bare-command-failure, a failure there aborts the test *before* cleanup and leaves `ai/probe-agent/AGENTS.md` — a fabricated instruction file claiming a dead path — sitting untracked in the real working tree the whole suite shares (not a scratch dir). | Reproduced directly: in a disposable local clone pinned to `20c4807`, `touch .git/index.lock` before running `bats --filter "newly added instruction file"` — the test correctly reports `not ok` (`git add -N` fails with status 128, "Unable to create '.git/index.lock': File exists"), but `ai/probe-agent/` is left on disk afterward. Confirmed the `discovered=` assignment is *not* the risk (it already has `\|\| true`) — the risk is specifically the unguarded `git add -N` line. Separately, a genuine discovery regression (regex forced to match nothing) was also reproduced and, in that case, cleanup *does* survive — so the gap is narrower than "cleanup is broken," but it is real for this one command. Independent corroboration that concurrent git activity against this exact worktree/branch is not a contrived scenario: a human commit (`094cbff`) landed on this same branch in this same worktree partway through this review session. | UNTESTED — no case exercises a git-level failure during the probe; `teardown()` (lines 17-19) only removes `$SCRATCH`, not `ai/probe-agent`, so there is no defense-in-depth if the inline ordering is ever disturbed by a future edit. | tests — wrap the mutating setup commands so a failure can't skip cleanup (e.g. `git -C "$DOTFILES_DIR" add -N "$probe" \|\| true`, or move the `rm -rf`/`git rm --cached` into `teardown()` unconditionally so bats' own guarantee — teardown always runs — covers this instead of manual line ordering) |
| Minor | THEORETICAL | tests/check-doc-paths.bats — exclusion coverage | The `grep -vE '^harness/\|^specs/\|^docs/'` exclusion in `instruction_files()` is not exercised by any real file today (the discovered set is byte-identical before and after applying the exclusion — verified by running both halves of the pipeline separately) and no test stages a synthetic file under those prefixes to prove the exclusion actually fires. I confirmed by hand (synthetic `AGENTS.md` under `harness/`, `specs/…`, and `docs/adr/` in a disposable clone) that it does work, but that check lives only in this review, not in the suite. | UNTESTED | tests — a fourth probe case (or an extension of case 8) that stages a file under one excluded prefix and asserts it is *not* discovered would close this the same way case 8 closed the enumeration gap |
| Minor | informational | scripts/check-doc-paths.sh — CI wiring | The guard is still never invoked as a standalone CI step; its only enforcement path is the "every instruction file's repo paths resolve" bats case, which does run in the required `test` job (`actions/checkout@v7` keeps `.git`, so `git ls-files` works there) — so the proposal's "fails CI" claim holds, just indirectly. Pre-existing since round 1, not introduced or changed by round 4; noted for completeness, not scored against this round. | n/a | n/a (not this round's scope) |

No Blockers found. The traversal-ordering fix and the flush-left zsh-regex fix
(rounds 2/3) were re-run as part of the full `check-doc-paths.bats` pass and
still hold; I did not re-derive them from scratch since they were not touched
by `20c4807` and were independently verified in rounds 2 and 3.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|---|---|---|
| Correctness | B | Discovery mechanism, regex, and exclusions all verified correct against real and synthetic data; the one gap is in a test's own robustness, not in the guard's behavior |
| Verification | B | Every round-4 claim in `verification.md` was independently reproduced and held up (unlike round 2's false claim) — but verification stopped at the happy path for the new probe test and missed the failure-path hermeticity gap this review found |
| Scope | B | Broader than the one-line fix round 3 asked for, but the broadening is deliberately justified against this chain's own 3-round record of the same defect class recurring — a documented, reasoned deviation, not creep |
| Reliability | C | The probe test's cleanup is not guaranteed under a reproduced git-level failure; a real, if narrow, error path is unhandled |
| Maintainability | B | Clear naming, extensive rationale comments consistent with repo convention, no dead code; `shellcheck` and both shell parsers clean |
| Handoff-readiness | B | `verification.md` and the round-by-round record are thorough and accurate; this round's own gap (cleanup hermeticity) had not yet been captured anywhere before this review |

### Verdict
FAIL

One REAL Major (UNTESTED, reproduced): the probe test can leave fabricated
instruction-file content behind in the real working tree if `git add -N` fails
partway through — a plausible condition for this project's own documented
workflow (parallel sessions sharing a worktree), and one that materialized
literally during this review as an unrelated concurrent commit on the same
branch.

### Recommended next steps (before archive)

- Fix the probe test (`tests/check-doc-paths.bats`, the `git -C "$DOTFILES_DIR"
  add -N "$probe"` line): make the failure of that command unable to skip the
  cleanup two lines later — either neutralize it (`\|\| true`, matching how the
  `discovered=` line already handles this) and let the subsequent assertions
  report the real failure, or move `git rm --cached`/`rm -rf` into `teardown()`
  so bats' own always-runs guarantee replaces reliance on manual statement
  order.
- Add a case that stages a synthetic file under one of the `harness/`, `specs/`,
  or `docs/` prefixes and asserts `instruction_files()` excludes it — the
  exclusion is currently correct but unpinned by any committed test.
- Both are small, targeted, and consistent with this round's own principle
  (discovery over enumeration, mechanism over instance) — this is not a case
  for reverting the discovery rewrite, only for finishing its error handling.
