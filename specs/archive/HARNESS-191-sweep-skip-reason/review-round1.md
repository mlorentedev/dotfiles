---
spec: "HARNESS-191-sweep-skip-reason"
verdict: "FAIL"
reviewed_sha: "e02600e07d6b298403a99b2dfe89117b6034ccc6"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-11"
---

## Adversarial review

**Scope**: HARNESS-191-sweep-skip-reason (#3263 / PR #3355, head `e02600e0` = reviewed sha)
**Sources**: `specs/HARNESS-191-sweep-skip-reason/{proposal,tasks,verification}.md`, `features.json`,
`git diff c94246c87fdc7c1765c3970361c69f5fc2ecfb73...HEAD` (exactly the 6 spec commits; no merges,
no other work in range), CI runs 43137297456 / 43137297400 / 43137708222 / 43137708277 on PR #3355.

### Spec and task alignment

- **AC1** (dirty lane not parked, reason names the dirty tree) — **proven**. `TestParkSingleNamedLane_Outcomes/a lane with an untracked file is not parked` passes; mutation of the reason string to `"refused"` turns it red (`sweep_park_test.go:113`), so the assertion bites.
- **AC2** (forge error carried; landed heads park) — **proven**. Both subtests pass; blanking `laneLanded`'s `why` makes `a lane whose landing cannot be checked…` fail with `reason "its landing could not be decided"` (`sweep_park_test.go:92`), proving the test distinguishes the forge-error reason from the fallback.
- **AC3** (refused detach keeps state and branch, git's first line in the reason) — **proven**. Subtest asserts state, branch (`symbolic-ref` → `feat/lane-a`), the `detach` prefix and git's `index.lock` token; replacing `gitFailure(out, err)` with `err.Error()` turns it red (`sweep_park_test.go:137`, reason became `…exit status 128`).
- **AC4** (`LaneSkips` in the report; one printed line per entry) — **proven at report level** (`TestSweepReportsALaneItDidNotPark`, red under the same mutation at `sweep_park_test.go:176`) and **proven by the command test on this machine** (`TestWorktreeSweepPrintsWhyALaneWasNotParked`, green locally), but the **same named test fails on Windows in CI** — finding F1 below, and it takes `cli-gate` with it.
- **Behavioral equivalence of the `laneLanded` extraction**: checked case-by-case against the pre-fix inline logic (4 combinations of `isMerged` × `landedHeads` nil/non-nil) — verdicts identical in every reachable case; only the previously-dropped `why` is now surfaced. The parking side (dirty gate order, lease write, refused-detach restore, branch delete) is untouched. `git checkout` output was discarded before (`Run()` with nil stdio → `/dev/null`), so capturing it with `CombinedOutput` loses no previously-visible output.
- **Contract digests**: `dotf spec digest HARNESS-191-sweep-skip-reason` matches `review-request.json` exactly (features.json / proposal.md / tasks.md).
- **Tasks**: every ticked box carries diff evidence; "PR opened" is unticked while PR #3355 exists (cosmetic). Verification claims reproduced: `go test ./internal/worktree/ ./internal/cmd/ -count=1` → ok/ok; `go vet` darwin+windows clean; `golangci-lint run` 0 issues both GOOS on the pinned 2.12.2 (`versions.conf:79`). features.json commands f1–f4 run green and select 1/2/2/2 tests as their evidence claims.
- **Red-green**: production files reverted to `67e45595` → worktree package fails to build against the new tests and the command test fails at runtime (`sweep output lacks "skipped: … git refused to detach it: "`); restored → green. Tree left clean; probes only in-tree-and-reverted.
- **PR gates triaged** (disposition recorded here since commenting on the PR is out of scope for this run): `hygiene`, `knowledge-gate`, `cli-gate`+`test (windows-latest)` = real findings F1/F2; `spec-gate` = the expected archive-on-merge lifecycle gate (resolves when this verdict lands and the spec archives in this PR — **not** a finding); `review-attestation` = pending, satisfied by this review; test/lint/integration legs otherwise green.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | REAL | tests / CI | The AC4 command test is not Windows-portable: `want` is built from `filepath.EvalSymlinks(lane)` (backslashes `C:\Users\…`) while sweep prints git-style `C:/Users/…`, so `Contains` fails on the separator. Product output on Windows is correct — CI log shows the full line `- skipped: C:/…/lane-a: git refused to detach it: fatal: Unable to create 'C:/…/index.lock': File exists.` The test fails anyway, `cli-gate` fails as a consequence, PR #3355 cannot merge in this state. | CI run 43137297456, job `test (windows-latest)`, `worktree_sweep_lane_skip_test.go:68`; reproduced mechanism by code read (test uses `EvalSymlinks`, sweep prints `info.Path` from git). Not reproducible on this darwin host — Windows leg is CI-only (UNVERIFIED locally, REAL via CI log). | `TestWorktreeSweepPrintsWhyALaneWasNotParked` — **is itself the failing test** on Windows; green on darwin/ubuntu | tests (normalise both sides, e.g. compare `filepath.ToSlash` forms) — non-contract |
| Major | REAL | hygiene / handoff | The new lesson takes number 584, already owned on main (`lesson-584-a-guard-that-defaults-an-unreadable-measurement-to-zero-measures-nothing.md` exists at the base). Repo gate red, reproduced in-session: `go run ./cmd/dotf lessons fmt --check --dir ../docs/lessons` → exit 1, `lesson number 584 is used twice`. The PR body already names `lesson-585-the-push-gate-…`, so `knowledge-gate` separately fails on the file/body mismatch. Lesson number allocated without a live query. | CI `hygiene` run 43137297400 + local reproduction above; `knowledge-gate` run 43137708222 (`Lesson: docs/lessons/lesson-585-… is not changed by this PR`); `git cat-file -e c94246c:docs/lessons/lesson-584-a-guard-…` succeeds | UNTESTED (the `lessons fmt --check` gate is the red proof; no test asserts unique lesson ids — the gate does) | file rename 584→585 + frontmatter `id:` + `dotf lessons fmt` index regen; reference updates in `verification.md` (non-contract) and **`tasks.md` "Lesson 584" (contract set → re-review follows)**; PR body already correct |
| Minor | REAL | verification hygiene | `features.json` f4 anchors its evidence to `babfb894`, but `cli/internal/cmd/worktree_sweep_lane_skip_test.go` does not exist at that commit (added in `5a74c5c0`); at that sha the `-run` filter matches nothing and `ok` would be vacuous. The criterion itself is proven at the reviewed tip by this review's own run. | `git log --follow` on the file; `git show babfb894:…` fails; my run of f4's command at `e02600e0` → `--- PASS: TestWorktreeSweepPrintsWhyALaneWasNotParked` | `TestWorktreeSweepPrintsWhyALaneWasNotParked` (green at reviewed tip) | spec artifacts (features.json — contract set; disposition or accept, edit ⇒ re-review) |
| Minor | THEORETICAL | tests / coverage | Two reason strings enumerated in the proposal's "What" have no named test: `its lease could not be written: <err>` (needs an unwritable lease path) and the `laneLanded` fallback `its landing could not be decided` (reached only if `branchLanding` returns an empty why — detached-HEAD race between list and park, or a named lane on main); `branchLanding`'s `its tip could not be read` is likewise untested and now in the spec vocabulary. | grep over `cli/**/*_test.go`: no assertion on any of the three strings | UNTESTED | tests (force each path, or state in the proposal that they are defensive-only) |
| Minor | SPECULATIVE | tests / coverage | The wire contract `lane_skips` (proposal: "What") is pinned by no test — renaming the JSON tag ships green. Same is true of the pre-existing `session_skips`/`merge_unknown_skips`. | grep `lane_skips` → only `sweep.go` | UNTESTED | tests (marshal a report with one skip, or source-inspect the tag) |
| Minor | SPECULATIVE | code | `laneLanded` fails open when `landedHeads` is nil **and** the package var `mergedPRHeads` is nil: it returns `landed=true` where the pre-refactor code would have called the nil reader inside `landedInMergedPR` (panic). Unreachable in production (`mergedPRHeads = ghMergedPRHeads`; the cmd passes `forgeLandedHeads` non-nil); only reachable by nil-ing the package var in a test. | code read `sweep.go:456-466` vs `list.go:183` | UNTESTED | code (optional follow-up: nil reader ⇒ "its landing could not be decided", never landed) — ticket |
| Minor | REAL (measured) | quality | `parkSingleNamedLane` is 44 lines and `SweepWithRunner` 77 (rule: <40). The park family improved (54 → 44) and is declared out of scope, ticketed #3265 — CI's own adjacent-issue advisory names #3265 against `sweep.go`. `SweepWithRunner` (76 → 77) is named by no ticket. | `awk` line counts at base and HEAD; CI adjacent-issues output | n/a | code — #3265 already covers the park family; ticket `SweepWithRunner` |
| Minor | REAL | tests / coverage | `TestSweepReportsALaneItDidNotPark` never asserts `SkippedCount == 1`, so the count↔`LaneSkips` consistency (every non-parked named lane counted exactly once) is unproven; the cmd test prints totals but does not assert them either. | code read of the test | UNTESTED | tests |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All four ACs verified with red–green and string-level mutation evidence; `laneLanded` is verdict-equivalent to the code it replaced; residual gaps: two untested reason strings, one unreachable fail-open. |
| Verification       | C | Every command in verification.md/features.json reproduced green here, but the artifacts claim a green suite while the project's own Windows leg and hygiene/knowledge gates are red (never covered), and f4's evidence is anchored to a commit that lacks the test. |
| Scope              | A | Range is exactly the 6 spec commits: sweep reasons + tests + lesson + spec docs; out-of-scope list respected (no gate/park/reap semantics changed, no `worktree list --json` touch). |
| Reliability        | B | Every park-failure path now returns a reason; refused-detach still restores lease and branch (test-proven); no new error path can panic in production. |
| Maintainability    | B | vet + golangci-lint 0 issues on both GOOS (pin confirmed); helpers small and commented; functions over the 40-line rule are pre-existing or improved-and-ticketed (#3265). |
| Handoff-readiness  | B | Proposal/tasks/verification/features complete, lesson captured, promotions decided — but the lesson's number collides with main and the PR body names a different number, so the handoff is broken at the gate. |

### Verdict

**FAIL** — two **REAL Majors** (F1: the AC4 named test fails on Windows in CI, taking `cli-gate`; F2: duplicate lesson number 584, red `hygiene` + `knowledge-gate`, reproduced locally). Severity path rules: a REAL Major forces FAIL regardless of the rubric; independently, Verification is C (PASS-WITH-GAPS floor), so the more severe of the two paths is FAIL. No Blocker: the product's Windows output is correct — what is wrong is the test's path normalisation and the lesson's identity, both fixable without touching the spec's behaviour.

### Recommended next steps

Minimum set that flips this to PASS (in order; two touch the contract set, so a re-review follows by design):

1. **tests (non-contract)** — fix `TestWorktreeSweepPrintsWhyALaneWasNotParked`'s path comparison for Windows separators (`filepath.ToSlash` both sides, or assert on a separator-normalised `out`). Green proof is the Windows CI leg (cannot run on this darwin host).
2. **spec + docs** — renumber the lesson to 585 (file name, frontmatter `id:`, `dotf lessons fmt` to regen the index), update `verification.md`'s lesson paths (non-contract) and the **`tasks.md` "Lesson 584" line (contract)**; PR body already says 585 and becomes consistent. This clears `hygiene` and `knowledge-gate`.
3. **spec artifacts (contract)** — restate `features.json` f4's evidence against a commit that contains the command test (or the reviewed tip) — folded into the same re-review as (2).
4. **tracked, not blocking** — the UNTESTED minors (lease-reason and fallback tests, `lane_skips` tag pin, `SkippedCount` assertion, `SweepWithRunner` ticket, optional nil-reader guard) go to `verification.md` dispositions or a follow-up ticket; they do not gate.
5. Re-run `dotf spec review HARNESS-191-sweep-skip-reason` after the contract edits, then `dotf spec archive` **in this PR** (the `spec-gate` archive-on-merge failure resolves with the archive; it is lifecycle, not a defect).

**`dotf spec archive` is NOT advisable in the current state**: the verdict is FAIL, and two repo gates (`hygiene`, `cli-gate`) are red against this branch regardless of the review.
