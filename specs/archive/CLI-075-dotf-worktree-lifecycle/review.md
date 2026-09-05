---
spec: "CLI-075-dotf-worktree-lifecycle"
verdict: "PASS"
reviewed_sha: "5899c64b00fd11efcc7f69fa3c944ea7b8bd9d65"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-05"
---

## Adversarial review

**Scope**: CLI-075-dotf-worktree-lifecycle
**Sources**: `specs/CLI-075-dotf-worktree-lifecycle/{proposal,tasks,verification}.md`, `features.json`; `cli/internal/worktree/{worktree,list,add,sweep,done,clean,lock,lock_unix,lock_windows}.go` + `_test.go`; `cli/internal/cmd/worktree.go`, `cli/internal/cmd/root.go`; diff `7b00e84...HEAD` (merge-base, ~3270 insertions, 6 commits). Reviewed the unmerged `feat/cli-075-dotf-worktree-lifecycle` branch at HEAD `5899c64`; no open PR exists.

This is a re-review of the change after the **Round-1 review** at HEAD `47c138c` returned **FAIL** on a REAL Blocker: `dotf worktree done` (default) silently deleted gitignored non-disposable content (`.env`). The implementer then landed 5 more commits (fixes for the `done` gitignored guard, in-worktree root resolution, PR-query caching, and gate-f enforcement under lock). I re-reviewed the changed state, re-ran the full battery independently, and re-reproduced the round-1 Blocker through the actual CLI at the new HEAD to confirm it is closed.

### Spec and task alignment

- Acceptance criteria AC1–AC7 are present; `tasks.md` implementation boxes are all `[x]`; the only open box is the closing gate "Adversarial review passes before archive" (this review).
- `features.json` has 7 feature entries (f1–f7), all `state: "passing"` with non-vacuous `verification` commands. Note: the repo doctrine reserves the `passing` terminal state for the harness after verification; I could not confirm from here whether the harness or an agent set it. It is not read by the archive gate, so it does not block archive — surface only.
- `verification.md` maps every AC to named tests. The AC7 mapping now includes `TestDonePreservesGitignoredLocalFiles` — the test the round-1 Blocker demanded. That test exists and passes; the round-1 Blocker is fixed (verified live below).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in `proposal.md`, `tasks.md`, `features.json`, or `verification.md` (grep returned nothing).

**Independent reproduction of the verification claims** (all re-run in this session, not taken from `verification.md`):

| Claim | Result |
|---|---|
| `go -C cli build ./...` | exit 0 |
| `go -C cli vet ./internal/worktree/...` | exit 0 |
| `go -C cli test -v ./internal/worktree/...` | 26/26 top-level funcs PASS, exit 0 |
| `go -C cli test ./...` | all packages PASS, exit 0 |
| `GOOS=windows GOARCH=amd64` + `GOOS=darwin GOARCH=arm64` build | exit 0 |
| `golangci-lint run ./internal/worktree/...` (2.12.2) | 0 issues |
| `gocyclo cli/internal/worktree/*.go cli/internal/cmd/worktree.go` | max cyclomatic = 9 (threshold < 10) |
| `dotf worktree done` with a gitignored `.env` present (live repro, below) | refuses, `.env` survives, worktree intact |
| `dotf worktree done` clean / disposable-only (live repro) | succeeds, worktree removed |

**Live reproduction of the round-1 Blocker fix** (built the binary at HEAD `5899c64`, ran it): created a repo with `.gitignore` containing `.env`, `git worktree add -b feat/test`, wrote `SECRET=precious_credentials` into `repo-wt-feat/.env` (gitignored; `git status --porcelain` empty), then ran `dotf worktree done --repo <repo> --path <wt>`. Result: `Error: worktree has uncommitted changes or local scratchpad files (...); commit, stash, or pass --force`, exit 1; the `.env` survived and the worktree remained. The round-1 Blocker is **closed**. I also confirmed the happy paths still work: a clean worktree with no non-disposable gitignored content is removed (exit 0), and a worktree with only disposable `node_modules/` is removed (exit 0).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | test-traceability / verification accuracy | `verification.md` names `TestSweepTOCTOUDirtyErrorFailsClosed` and `TestSweepTOCTOUUnmergedFailsClosed` as proof that the dirty/merged gates are "re-checked under lock immediately prior to deletion" (proposal Gate c/d). **Those tests do not reach the re-check path.** `ListWithRunner` calls `IsDirty`/`IsPRMerged` for the main-repo entry first, so the mocks' call counters hit their "second call" during *discovery*, classifying the worktree `StateDirty`/`StateUnmerged` before `reapSingleWorktree` ever runs. | Mutation proof (reverted): removing the `isDirty` re-check from `reapSingleWorktree` did **not** fail `TestSweepTOCTOUDirtyErrorFailsClosed`; removing the `isMerged` re-check did **not** fail `TestSweepTOCTOUUnmergedFailsClosed`. The overall fail-closed property (dirty/unmerged → not reaped) is still proven at the discovery stage; the age gate *is* non-vacuously tested (removing it fails `TestSweepFailClosed` and `TestClassifyActiveStates`). So this is a coverage-accuracy gap, not a code defect — the re-check itself is real and correct. | The discovery-stage negative tests exist and pass; the *re-check* sub-claim has no reaching test → effectively UNTESTED for that layer. | tests — rewrite the two TOCTOU mocks so the re-check path is actually reached (e.g. account for the main-repo call, or make the runner clean/merged at discovery and dirty/unmerged at re-check), OR correct the `verification.md` wording to state the re-check is covered by code inspection rather than a named test. |
| Minor | THEORETICAL | rate-limit | `IsPRMerged` calls `queryGHPRMerged` (i.e. `gh pr view`) for every branch that fails the local `merge-base --is-ancestor` fast-path, not only branches marked `gone` or reaping candidates as the proposal's risk section explicitly mitigates ("Only query GitHub API for branches marked `gone` or candidate for reaping"). The `prCache` dedupes within a single invocation, but `Sweep()`/`List()` allocate a fresh `RealGitRunner` each run, so repeated sweeps re-query every non-merged branch. | Code read of `list.go` `IsPRMerged`. Low practical impact for an authenticated `gh` and a handful of worktrees, but it is a documented-mitigation-not-implemented spec mismatch. Carried over from Round 1; not addressed. | `TestPRQueryCache` covers cache dedup only, not the "only query gone/candidate" restriction. | code or spec — gate the `gh` call on the `gone`/candidate signal, or relax the proposal wording to state the per-run cache is the accepted mitigation. |
| Minor | THEORETICAL | TOCTOU residual | A non-disposable gitignored file (`.env`) created in the window between the under-lock `isDirty` re-check in `reapSingleWorktree` and the `git worktree remove` call can be deleted, because `git worktree remove` does not block on gitignored files. Window is sub-millisecond and mitigated by the immediate re-check, but not eliminated. Carried over from Round 1; the `done`-side analogue is now fixed, this is the residual `sweep`-side window. | Code read of `sweep.go` `reapSingleWorktree`. | UNTESTED. | code or spec — document as accepted residual, or re-check non-disposable gitignored content immediately before removal. |
| Minor | THEORETICAL | lifecycle / lease | The lease is written once at creation (`add`) and **cannot be renewed** — there is no `extend`/`renew` command. An active session that outlives the TTL (default 24h) and is containerized (invisible to host `/proc`) on a clean+merged branch would be reaped on a later sweep. Committed work is preserved (the branch is an ancestor of base / PR merged), and gitignored scratch is protected, so this is a liveness/disruption gap, not committed-data loss; `reap_ok: false` remains a manual hold. | Code read: cmd/worktree.go exposes only `list`, `add`, `sweep`, `done`; no lease-refresh path. | UNTESTED (no lease-renewal test; none exists). | code or spec — add a lease-refresh command, or document the fixed-TTL + `reap_ok: false` hold as the mechanism. |
| Minor | SPECULATIVE | availability | The lock file is a single global path `<temp>/dotf-worktree.lock` shared across **all** repos. Concurrent `sweep`/`done` on unrelated repos contend, and one fails with `ErrLocked` rather than running. Fail-closed (no corruption), but a false-contention/availability cost. Carried over from Round 1; not addressed. | Code read of `DefaultLockPath()` + `TryLockFile`. | `TestSweepFileLock` covers same-file contention, not cross-repo. | code — namespace the lock path by repo or user. |
| Minor | THEORETICAL | hygiene | `dotf worktree done` removes the worktree and prunes git metadata but does not delete the branch (local or remote). Consistent with the proposal wording ("removes worktree and prunes metadata cleanly"), so not a spec violation; it leaves a stale branch after a completed task. | Code read of `done.go` `removeWorktree` (no `git branch -D`). | `TestDoneTeardown` covers removal, not branch cleanup. | code or spec — optionally delete the branch in `done`, or document as intended. |

No Blocker or Major findings. All remaining findings are Minor and either THEORETICAL/SPECULATIVE (carried from Round 1 and still unaddressed) or the test-traceability accuracy gap above. The single destructive-path issue from Round 1 (the `done` gitignored Blocker) is **fixed and verified live**.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | AC1–AC7 all met and fail-closed; round-1 Blocker fixed and reproduced live; minor negative-path evidence gap (the under-lock re-check is defended by code inspection, not a reaching regression test). |
| Verification | B | All build/vet/test/lint/gocyclo/cross-compile/live claims independently reproducible; `verification.md` overstates the TOCTOU re-check coverage (the named tests trigger at discovery). |
| Scope | A | Diff tracks the proposal; added decisions F1–F12 documented in-session; the round-1 fix is in scope; no unrelated changes. |
| Reliability | B | `sweep`/`done` error paths are fail-closed and SHA-recovery is logged; residual gitignored-TOCTOU window, cross-repo lock contention, and the fixed-TTL lease are Minor. |
| Maintainability | A | Max cyclomatic complexity 9 (under the <10 rule), clear naming, no dead code, named tests for the new logic. |
| Handoff-readiness | B | `verification.md` complete with decisions and the lesson-268 promotion ticked; proposal AC boxes still unchecked and `status: draft` is pre-archive-expected. |

### Verdict
**PASS** — no Blocker or Major findings, and no dimension grades below B. The one REAL Blocker from Round 1 (data-loss / ADR-028 secret leak in `dotf worktree done`) is fixed and verified through the actual CLI at this HEAD. The remaining findings are Minor and either THEORETICAL/SPECULATIVE or a verification-accuracy gap that does not affect the safety outcome.

### Recommended next steps (before archive)
1. **Verify the archive gate is satisfied:** `dotf spec archive CLI-075-dotf-worktree-lifecycle` is **advisable** in the current state — `reviewed_sha` matches HEAD, `reviewer` is `nan/deepseek-v4-flash` (in the pool), verdict is PASS, and no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.
2. **Close the test-traceability gap (optional, before or after archive):** rewrite `TestSweepTOCTOUDirtyErrorFailsClosed` / `TestSweepTOCTOUUnmergedFailsClosed` so they actually reach the `reapSingleWorktree` re-check (the current mocks hit their failure during discovery because the main-repo entry consumes the first call), or amend `verification.md` to state the re-check is verified by code inspection. This makes the "re-checked under lock" claim robust rather than cosmetic.
3. **Resolve the rate-limit spec drift (code or spec):** gate `queryGHPRMerged` on the `gone`/candidate signal per the proposal mitigation, or update the proposal's risk section to state the per-run cache is the accepted mitigation.
4. Optional minors (not gating): add a lease-refresh path or document the fixed-TTL + `reap_ok: false` hold; note the residual gitignored TOCTOU window; optionally delete the branch in `done`; namespace the global lock path.
