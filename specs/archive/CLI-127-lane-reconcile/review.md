---
spec: "CLI-127-lane-reconcile"
verdict: "PASS"
reviewed_sha: "7a49aa8e87da304519e7d7f0011b9fccbffb1b18"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-10-16"
---

## Adversarial review

**Scope**: CLI-127-lane-reconcile (round 3), whole change `1ac92409f1e8af930e55ee20dcbfc2e55d96edaa...HEAD` (158 files, +7316/−612, 44 commits)
**Sources**: `specs/CLI-127-lane-reconcile/{proposal.md,tasks.md,verification.md,features.json}`, `review-request.json` (base_sha matches the launcher's prompt), `git diff`/`git log` over the stated range, `cli/internal/lane/*`, `cli/internal/cmd/lane*.go`, `cli/internal/worktree/sweep.go`, live read-only runs of `dotf lane status` / `dotf lane plan` on this machine

### Spec and task alignment

- **Gate evidence re-run fresh in this session** (not read from claims): `go build ./...` exit 0; `go vet ./...` exit 0; `GOOS=windows go vet ./...` exit 0; `golangci-lint run` (2.12.2, the pin) **0 issues**; full `go test ./...` **0 failures**; all four `features.json` verification commands executed → **exit 0 each**.
- **Four reviewer mutations, each reverted** (tree left clean): (1) removed `Apply`'s `Class != ClassAuto` filter → `TestApply/never_runs_a_verb_for_pending,_adopt,_flag_or_any_ambiguous_action` failed (sweeps=1, changed=5); (2) moved `planLanded`'s liveness check after the reapable case (the pre-round-1 ordering) → `TestApplyNeverReapsALiveWorktree` (2 subtests) + `TestPlan/reapable_with_a_live_session_is_ambiguous` + `reapable_with_unknown_liveness_is_ambiguous` failed; (3) dropped `Sessions: sessionStores()` from `realLaneVerbs` → `TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions` failed (reaped=2, live worktree removed) — round-2's F1 mutation claim reproduced; (4) pointed `ForgeSweep.IsPRMerged` back at the default ancestry/gh answer → `TestApplyNeverReapsWhatPlanCalledAmbiguous` failed (`changed=2`, ambiguous worktree removed) — round-1's F1 mutation claim reproduced.
- **AC1** → `TestDerive` (18 rows covering all 9 phases, herdr-absent rows, `TestDerive_HerdrNeverMovesAPhase`) and `TestStatus` (6 subtests incl. `HerdrAbsentStillAnswers`, `LandingHeadIsPickedAmongReviewLanes`, `ADraftIsNeverTheLandingHead`, `AHandNamedLaneIsNotAlsoPending`); cmd-side `TestLaneStatus_JSONShape`, `TestLaneStatus_TableAndExitCode`. f1 command passes.
- **AC2** → `TestPlan` (classes incl. pending-with-due and ambiguous-with-resolver assertions), `TestPlanMutatesNothing` (real repository; SHA-256 of refs + worktree registry + statuses + lease files before/after; asserts the plan is non-empty so the check is not trivial), plus `TestPlanQuotesTheResolvingCommand`, `TestPlanResolversDoWhatTheyPromise`, `TestCheckHead`. f2 command passes.
- **AC3** → `TestApply` (6 subtests: auto-only, no verb for pending/adopt/flag/ambiguous, counts sweep's own result, printed-only start ≠ change, **second apply over post-state = changed=0**, first verb error stops with partial count) + `TestApplyNeverReapsWhatPlanCalledAmbiguous` (real repo + real sweep) + cmd `TestLaneApply_PrintsChangedAndRunsOnlyAutoVerbs`. Structurally, `lane.Verbs` exposes only `Sweep` and `Next` — adopt and land are unreachable from `Apply`, not merely untested. f3 command passes.
- **AC4** → `TestApplyNeverReapsALiveWorktree` (live, and unknown reads as live) + `TestRealLaneVerbsSweepHonoursTheForgeAndTheSessions` (real sweep, real pi session on disk). Oracle consistency verified by reading both sides: plan and sweep ask the **same** process oracle (`hostProcessInside`, via `ProcessInside`) and the **same** session oracle (`RecentHarnessSession` over `SessionWindow`, 6h), both fail closed (no scan → `Inside: true` / `LiveUnknown`). Live half reproduced here: `dotf lane status` (24 rows, exit 0) and `dotf lane plan` (exit 0, read-only, herdr already running, no server started, `apply` never run) — **no worktree with a live session appears as `auto` or `pending`**; the literal wording gap observed is filed below.
- **Tasks**: every implementation box is `[x]` and each has diff/test evidence in the named files; `verification.md` carries both prior FAIL rounds with their fixes and dispositions. No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec folder.
- **Contract staleness**: `git status` shows `proposal.md`, `tasks.md` and `features.json` byte-identical to HEAD (= `reviewed_sha`); the digests in `review-request.json` are the normalized digests from `contract_digest.go`, so raw `sha256` output differs by design — cleanliness against HEAD is the check used here.
- **Scope note**: the range aggregates other merged PRs (release-please, tools/secrets/spec/doctor/worktree) by the launcher's base construction — the same observation round 2 recorded as F5; the spec's own files map to the proposal exactly.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec / AC4 wording | AC4's live-evidence sentence says "every worktree with a live session is classed `ambiguous`", but a live `Running` lane gets no plan class at all: my live run shows 4 of the 6 live worktrees as `Running` rows with zero actions (only the two live `Unadopted` ones print `ambiguous`). The operative safety property — never `auto`, never reaped — holds and is proven | Live run this session: `lane status`/`lane plan` outputs (2026-10-16); plan acts only on Landed/Stalled/Orphan/Unadopted phases by construction | `TestApply/"a second apply over the post-state changes nothing"` (Running row → no action) + `TestApplyNeverReapsALiveWorktree` | spec wording — disposition in `verification.md` (outside the contract set); no `proposal.md` edit is required for archive and one would invalidate this verdict |
| Minor | SPECULATIVE | injection | `lane.REST.PRFor` interpolates the branch unencoded into the `gh api` query string (`head={owner}:"+branch`); `git check-ref-format` allows `&` and `#` in ref names (both verified exit 0), so a locally crafted branch name could skew query params (and with them the phase/merged oracle). Exec-argv, not shell — impact is a wrong forge answer, not command execution; requires an adversarial branch in the local repo | `cli/internal/lane/forge.go:35` + `git check-ref-format 'refs/heads/fix/a&b'` → allowed | UNTESTED (no test feeds a branch containing query metacharacters) | code — pass the branch as a typed `gh api` field (`-F`) or URL-encode; follow-up ticket, non-blocking |
| Minor | REAL | scope | The reviewed range contains ~150 files of unrelated merged PRs beyond this spec (release, tools, secrets, doctor, deploy, worktree, other specs) | `git log 1ac9240..HEAD` (44 commits; only ~12 are lane/spec commits); launcher-resolved base, same as round-2 F5 | n/a (observation) | none — launcher construction; already documented in `verification.md` |
| Question / assumption | SPECULATIVE | concurrency | Two `lane apply` runs started simultaneously: sweep is serialized by `TryLockFile` (second run errors loudly) and starts are guarded by `lane next`'s lease claim, but no test starts two applies at once | `cli/internal/worktree/sweep.go:251-259`; no reproduction attempted | UNTESTED | tests (follow-up) or decline with a reason — surfaced, does not gate |

No Blocker and no REAL Major was found. The four mutating checks above show the safety properties from rounds 1 and 2 (same oracle for plan and apply, live-session protection, auto-only application) are enforced by tests that actually fail when the property is broken, not by assertions about them.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs verified against code, named tests, four killing mutations and a reproduced live run; AC4's literal evidence sentence does not reproduce for Running lanes (Minor finding) keeps it from A |
| Verification       | B | Full gate re-ran green here (build, vet, windows-vet, lint 0 issues, full suite, all four features.json commands) and live status/plan reproduced; the no-process-scan (Windows) and concurrent-apply paths remain compile/prose-only as disclosed |
| Scope              | B | Spec's own files match the proposal exactly (status/plan/apply + spec triad); the 158-file window is the launcher's base spanning other merged specs (stated limit, same as round-2 F5 and the CLI-119 precedent) |
| Reliability        | B | Fail-closed everywhere: forge error aborts loudly, unknown liveness reads live, session-read error skips, no-backend start counts 0, first verb error stops with the partial count; untested simultaneous-apply path keeps it from A |
| Maintainability    | B | Clean seams (`Sources`/`PRSource`/`Verbs`/`LandCheck`), WHY-comments throughout, lint 0 issues at the repo pin (gocyclo min 22); strict CC≤10 not separately measured (no gocyclo/lizard binary installed) — disclosed |
| Handoff-readiness  | A | Triad complete, both prior FAIL rounds recorded with dispositions and reproductions, lessons 449–457 in `docs/lessons/`, promotion lines answered, archive checklist present, features.json contract-correct (state pending, evidence harness-owned) |

Aggregation: no C, no D → the rubric alone reaches PASS; the severity path finds no Blocker and no REAL Major, so both paths agree.

### Verdict
PASS

### Recommended next steps
- All items below are for the implementer to disposition in `verification.md` (outside the contract set) or carry into a follow-up ticket. **No edit to `proposal.md`, `tasks.md` or `features.json`** — the contract set is closed by this verdict and editing it would invalidate the review.
- AC4 wording row: record in `verification.md` that "classed ambiguous" applies to lanes plan acts on (live `Running` lanes produce no action by design), or leave it as-is; the safety property itself is proven.
- Follow-up ticket: typed/encoded branch parameter in `lane.REST.PRFor` (Minor, injection checklist) with a test feeding a branch containing `&`.
- Optional follow-up: a test that starts two `lane apply` runs concurrently (or a written decline noting the sweep lock and the `lane next` lease claim as sufficient).
