---
spec: "BUG-3253-gittest-quiet-appends-inherited-config"
verdict: "FAIL"
reviewed_sha: "f87a87315a3b125fe1b1034f1d036dfeef2436ce"
reviewer: "nan/glm5.3-flash"
date: "2026-11-08"
---

## Adversarial review

**Scope**: BUG-3253-gittest-quiet-appends-inherited-config — `gittest.Quiet` merges an inherited `GIT_CONFIG_COUNT` environment instead of panicking (`cli/internal/gittest/gittest.go`, `gittest_test.go`, lesson 517).

**Sources**: `specs/BUG-3253-gittest-quiet-appends-inherited-config/{proposal,tasks,verification,features.json}` + `git diff 0b1eb901...f87a87315a3b125fe1b1034f1d036dfeef2436ce` (two commits: `8881f845` fix, `f87a8731` tests). Note: `review-request.json` recorded `base_sha: 8881f845` — the fix commit itself — which would have hidden the fix from the diff; this review used the branch point `0b1eb901` (merge-base with `origin/main`) instead. Launcher-side defect, not a finding against the change.

### Spec and task alignment

- All four acceptance criteria map to named tests; all tasks `[x]` carry diff evidence. The one unticked task (`PR opened`) is honestly open.
- `features.json` f1–f3 predicates re-ran green under a foreign 3-pair environment this session (9/9 `TestQuiet*` PASS, 39.9s). f4 (`internal/cmd -run '^$'` under the harness env) re-ran green, exit 0. `go vet ./internal/gittest` clean.
- Reviewer also reproduced AC1's "runs its tests" beyond binary-start: `TestDeclaredCapacity` (2.0s) and `TestHarnessRefresh_AFailedRenderIsAnErrorAndWritesNothing` (67.7s) each ran to completion under `GIT_CONFIG_COUNT=3` + three inherited pairs. The 67.7s vs the 46.6s cleared-env figure is consistent with verification.md's measured hook-chain cost, not a regression from this change.
- Old-code behavior re-confirmed from the diff pre-image (`0b1eb901:cli/internal/gittest/gittest.go`): any non-own `GIT_CONFIG_COUNT` panicked. A stash-based A/B re-execution was attempted and aborted — see the process note below.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | spec-vs-tests | `proposal.md` Risks states the duplicate-key precedence ("If the caller already pins `gc.auto`, both pairs are present and the later one wins") is *"asserted rather than assumed"*. No test sets a caller pair that conflicts with `gc.auto`/`maintenance.auto`, so AC2's "Quiet's own pairs apply on top" is only proven on **non-conflicting** keys, where "on top" has no observable effect. | Full `gittest_test.go` diff read; `grep -n gc.auto` shows it only as Quiet's own assertion (lines 60, 79, 117) — never as a caller pair with a different value. Absence verified this session. | UNTESTED | tests (add a named regression test where the caller pins `gc.auto=1` and assert the child reads `gc.auto=0`). Once that test exists the proposal sentence becomes true and **no contract-set edit is needed**; the REAL Major still requires a re-review after the fix. |
| Minor | THEORETICAL | resilience | `Quiet` runs `quietSystem()` **before** validating the count, so a malformed-count refusal is no longer side-effect-free: the system-config file is written and `GIT_CONFIG_SYSTEM` set before the panic. Old code refused first. Bounded consequence (stray file in the 0700 user cache dir; the process aborts anyway), and the refusal itself demonstrably survives (`TestQuietRefusesAMalformedCount` PASS). | Code order in `Quiet()` (gittest.go:56-60). | `TestQuietRefusesAMalformedCount` covers the refusal; the side-effect-free property is UNTESTED. | code (parse `callerPairs()` before `quietSystem()`) |
| Minor | THEORETICAL | memory | `callerPairs` allocates `n × 32B` from a caller-controlled count with no plausibility bound: `GIT_CONFIG_COUNT=1000000000` → ~32 GB allocation → OOM crash instead of the old clean panic. No legitimate harness sets such a count. | Code read of `callerPairs`; no repro executed. | UNTESTED | code (cap `n` or refuse a count with no `GIT_CONFIG_KEY_0` present) |
| Minor | REAL | verification | The full `go test ./internal/cmd -count=1` never completed on this host (10m timeout; measured #3199 hook-chain cost, predates this change). AC1 is evidenced by binary-start (f4) plus — now — this reviewer's single-test reproductions, not by a completed package run. verification.md documents this honestly and names the cause. | verification.md "Suite note"; reviewer reproductions above. | `TestDeclaredCapacity`, `TestHarnessRefresh_AFailedRenderIsAnErrorAndWritesNothing` (partial, named) | disposition in verification.md; CI is the authoritative full-suite check |
| Question | SPECULATIVE | correctness | `strconv.Atoi` accepts forms git's own parser may treat differently (`+3`, `03`). Quiet could accept a count git refuses. Neither direction demonstrated. | Code read only. | UNTESTED | surface only — do not gate on this alone |
| Process | REAL | hygiene (not the change) | A foreign stash entry (`stash@{0}: On feat/mise-windows: copilot-mise-windows-pre-origin-main-2026-10-12`) was left in this shared review worktree. A reviewer's `git stash push`/`pop` cycle (this reviewer's own A/B attempt) popped it into a 4-file merge conflict; restored to HEAD cleanly and the stash entry is intact — but the near-miss is real. | `git stash list`; the conflict output and restoration in this session's transcript. | n/a | Dispose of the stash in its own branch context; never leave stashes in shared worktrees (`runbook-worktree-safety`) |

**Test deletion record.** `TestQuietRefusesAnEnvironmentThatAlreadyCarriesConfig` (deleted): it enforced the old contract — panic on any non-own count — which this change deliberately inverts (#3253 is that panic, so the failure it detected is the bug). Stronger proof remains: `TestQuietKeepsTheCallersConfig` + `TestQuietAppendsAfterAForeignEnvironment` cover the kept-pairs contract, and the two refusals that must survive are asserted by `TestQuietRefusesAMalformedCount` and the pre-existing `TestQuietRefusesNoSystem`. `alreadyQuiet()` was unexported with no callers outside the package. Focused validation: `go test ./internal/gittest -run '^TestQuiet' -count=1` — green this session. Deletion is sound.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs hold on the observed paths; the conflicting-key precedence path — the only case where "apply on top" is observable — is untested. |
| Verification       | B | Reproducible commands, honest suite note; reviewer re-ran the predicates and two real `internal/cmd` tests green; full package run absent (host limitation, measured cause). |
| Scope              | A | Two commits, diff matches the proposal exactly; lesson update in scope; no creep. |
| Reliability        | B | Refusals preserved and covered; two bounded robustness minors (side-effect ordering, unbounded allocation). |
| Maintainability    | A | Small named functions, comments explain why, `alreadyQuiet` deduplicated rather than kept in sync; `go vet` clean. |
| Handoff-readiness  | A | Lesson 517 rewritten in-session citing #3253; features.json emitted; verification.md names its own limits and the out-of-scope #3199. |

### Verdict

**FAIL** — one **REAL, UNTESTED Major** (the proposal's duplicate-key precedence is claimed as asserted but no test asserts it; it is also the only case where AC2's "apply on top" is observable). Rubric has no D; everything else is Minor or Question and does not move the verdict.

### Recommended next steps

- **tests** (outside the contract set — do this first): add a named regression test, e.g. `TestQuietOutranksAConflictingCallerPair`: `cleanEnv`, caller pins `gc.auto=1` (and/or `maintenance.auto=true`) via `GIT_CONFIG_KEY_0/VALUE_0`, call `Quiet`, assert a child `git config` reads `gc.auto=0` / `maintenance.auto=false` and `GIT_CONFIG_COUNT == 1+len(quietPairs)`. This makes the proposal's "asserted rather than assumed" sentence true without touching `proposal.md`, `tasks.md` or `features.json`.
- **re-review** after the fix: a REAL Major requires it; this review is then superseded by a fresh round against the new HEAD.
- **verification.md** (free to edit): add the reviewer reproductions above to the Criterion 1 evidence line, and disposition the side-effect-ordering and allocation-cap minors (apply or ticket).
- **housekeeping** (not this change): remove the foreign `feat/mise-windows` stash from this worktree; the review launcher should record the merge-base, not the first work commit, as `base_sha`.
