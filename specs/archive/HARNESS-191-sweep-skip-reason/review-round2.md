---
spec: "HARNESS-191-sweep-skip-reason"
verdict: "FAIL"
reviewed_sha: "94cb2f6f2e9258cc115bc1597cad26b52ed9b297"
reviewer: "nan/glm5.3-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-191-sweep-skip-reason (#3263 / PR #3355), whole range
`git diff c94246c87fdc7c1765c3970361c69f5fc2ecfb73...HEAD` at `94cb2f6f` — the 8 spec commits plus
nothing else (the range contains no merges; `--first-parent` lists exactly the 9 commits above).
Round-1 fixes (`ffd5c951`, `4ced9f9f`) are in this range on purpose and judged against the early
findings they answer.

**Sources**: `specs/HARNESS-191-sweep-skip-reason/{proposal,tasks,verification}.md`, `features.json`,
round-1 `review-round1.md` (FAIL on `e02600e0`, 8 findings, dispositions in `verification.md`),
`gh` state of PR #3355 and its check runs.

### Spec and task alignment

- **AC1** (dirty lane not parked, reason names the dirty tree) — **proven**. `TestParkSingleNamedLane_Outcomes/a lane with an untracked file is not parked` green; mutation replacing `gitFailure` with a fixed string turns the detach subtests red, and deleting the `LaneSkips` append turns `TestSweepReportsALaneItDidNotPark` red (probes run and reverted this session, tree clean).
- **AC2** (forge error carried; landed heads park; undecided landing named) — **proven**. The exact `features.json` f2 command selects 4 subtests as its evidence claims and all pass.
- **AC3** (refused detach keeps state and branch, git's first line in the reason) — **proven** locally, both through `parkSingleNamedLane` and through `SweepWithRunner` with a held `index.lock`. **Not proven on Windows** — see F1.
- **AC4** (`LaneSkips` in the report, one printed line per entry) — **proven** locally: report test, wire-name test (`lane_skips` present, omitted when empty), and the real-command test `TestWorktreeSweepPrintsWhyALaneWasNotParked`. **The command test has never executed on Windows** — see F1.
- **Contract digests**: `dotf spec digest HARNESS-191-sweep-skip-reason` matches `review-request.json` exactly (features.json / proposal.md / tasks.md; `spec.md` empty, full spec).
- **Round-1 dispositions verified against the diff**: F1 (`filepath.ToSlash` on both sides — present, `worktree_sweep_lane_skip_test.go:66-69`), F2 (lesson 585 exists; 583/584/585 numbering consistent, no collision), F4 (two new subtests present), F5 (`TestSweepReportNamesLaneSkipsOnTheWire` present), F6 (nil-reader subtest present), F8 (`SkippedCount == 2` asserted). F7 ticketed (#3360) as recorded.
- **Tasks**: every ticked box carries diff evidence; "PR opened" honestly unticked. Reproduced fresh this session: full `go test ./internal/worktree/ ./internal/cmd/ -count=1` → ok/ok; `go vet` darwin + windows clean; `golangci-lint run` (pinned 2.12.2) 0 issues on both GOOS.
- **PR checks triaged (not commented)**: every failing check (`test (windows-latest)`, `cli-gate`, `hygiene`, `knowledge-gate`, `spec-gate`) ran on `e02600e0` — the **pre-fix** head. The branch's remote tip is still `e02600e0` (`git ls-remote`); `ffd5c951`, `4ced9f9f`, `94cb2f6f` are unpushed. The failures are the round-1 findings, expected to clear on push — but they have not cleared yet, which is finding F1.
- **No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags** in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL (cited: `git ls-remote origin` → branch tip `e02600e0`; `gh run view 43137297456` log shows the exact F1 failure on that head) | verification | The round-1 REAL Major (F1, Windows path separators) is dispositioned "applied" with "proof is the Windows CI leg of #3355" — but that CI leg ran on `e02600e0`, **before** the fix commit `4ced9f9f`, and failed there. The three fix commits are unpushed, so the fixed test has never executed on Windows (or anywhere outside this macOS worktree). The disposition's proof does not exist yet. | `gh run list --branch fix/3263-sweep-skip-reason` shows no run on `4ced9f9f`/`94cb2f6f`; failed run 43137297456 `test (windows-latest)`: `sweep output lacks "skipped: C:\\Users\\...: git refused to detach it: "` vs git's slash path | UNTESTED on Windows (local POSIX runs green; the fix is correct by inspection and by the failure log's own output, but unexecuted on the platform that caught the bug) | tests/evidence: push `ffd5c951`+`4ced9f9f`+`94cb2f6f`; confirm `test (windows-latest)`, `hygiene`, `knowledge-gate`, `cli-gate` green; then re-review. `verification.md` (outside the contract set) may then amend the F1 proof line. |
| Minor | THEORETICAL | landing decision | The `laneLanded` extraction is not verdict-identical to the pre-fix logic: old code, when `isMerged` was true **and** `landedHeads != nil`, still required `branchLanding(landedHeads)` to agree; the new code parks on `isMerged` alone. Divergence is reachable only when `branchLanding` disagrees with `isMerged`: a detached-HEAD worktree classified PARKABLE, a lane on `main`/`master`, or a transient forge error between the two calls. The spec declares "any change to what sweep parks" out of scope, so this is a spec-vs-code mismatch in edge states. | Code read of `sweep.go` `laneLanded` vs `git show 5d1988e1` original block; no reproduction | UNTESTED (no test pins the old re-check; existing tests pin the new behavior) | code (restore the re-check) or ticket into #3265 with a decision; say which |
| Minor | THEORETICAL | error reasons | `parkSingleNamedLane`'s dirty-tree reason conflates "could not read status" with "dirty": `isDirty` returns true on `runner.Dirt` error, so an unreadable state is reported as `has uncommitted changes or untracked files`. The skip behavior is pre-existing; the new reason asserts a cause it does not know. | `sweep.go` `isDirty` (`err != nil || dirt.Any()`) feeding the fixed reason string | UNTESTED | code (a distinct "its state could not be read" reason) or ticket; declined-with-reason also acceptable |
| Minor | SPECULATIVE | error reasons | `gitFailure` reads the first line of `CombinedOutput`, so a stdout line emitted before git's stderr fatal line would be reported as the cause. On the observed failure (`index.lock`) git prints only the stderr fatal line first. | code read; CI log shows the first line is the `fatal:` line | UNTESTED | surface only; do not gate on this |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All four ACs verified with named tests; three mutation probes each turned a named test red and were reverted clean. |
| Verification       | B | Everything locally checkable was reproduced fresh (tests, vet, lint both GOOS, digests, features.json commands), but one recorded disposition cites CI proof that does not exist (F1). |
| Scope              | A | Diff is exactly the proposal's surface (sweep.go, cmd print, tests, lesson 585, spec folder); no creep. |
| Reliability        | A | Lease-write failure, refused detach (state+branch restored), undecided landing, nil reader — all handled and named-tested. |
| Maintainability    | B | `parkSingleNamedLane` is 44 lines and `SweepWithRunner` 77 (both above the repo's 40-line law), with #3265/#3360 as the declared extractions. |
| Handoff-readiness  | A | Round-1 dispositions, lesson 585, features.json evidence, archive checklist all recorded in-spec. |

### Verdict
FAIL

One **REAL** Major (F1): the round-1 Windows fix is unpushed and its claimed CI proof does not exist, so the platform-specific half of AC3/AC4 is UNTESTED. The code itself is substantively correct — every other finding is Minor/THEORETICAL and the rubric has no C or D.

### Recommended next steps
- **Flip to PASS (minimum set):** push `ffd5c951`, `4ced9f9f`, `94cb2f6f` to `fix/3263-sweep-skip-reason`; confirm `test (windows-latest)` (specifically `TestWorktreeSweepPrintsWhyALaneWasNotParked`), `hygiene`, `knowledge-gate`, and `cli-gate` green on the new head; run a round-3 review. The archive gate refuses this review's successor otherwise anyway, since the PR cannot merge with red required gates.
- After green CI, amend `verification.md`'s F1 disposition line (it is outside the contract set, so the amendment does not invalidate the next review).
- Disposition F2 and F3 in `verification.md` (apply, ticket into #3265/#3360, or decline with a reason); F4 needs no action.

**Archive advisable?** No — not until the branch is pushed, the Windows leg is green, and a fresh passing review covers it.
