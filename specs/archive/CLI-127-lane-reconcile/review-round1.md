---
spec: "CLI-127-lane-reconcile"
verdict: "FAIL"
reviewed_sha: "79cf7e5d30f34216a715bc5ba269f3a929479048"
reviewer: "nan/qwen3.8-flash"
date: "2026-10-16"
---

## Adversarial review

**Scope**: `CLI-127-lane-reconcile` — `dotf lane status` / `plan` / `apply` (ADR-050 decision 4), issue mlorentedev/dotfiles#2550.

**Sources**: `specs/CLI-127-lane-reconcile/{proposal,tasks,verification,features}.md` + `git diff 1ac92409f1e8af930e55ee20dcbfc2e55d96edaa...HEAD` (the launcher's stated base; HEAD `79cf7e5d`).

**Note on the range.** The stated base is the pre-spec commit on a shared `main`, so the range carries 20 commits, of which 17 belong to other lanes (#2574, #2575, #2579, #2585, #2588, #2590, #2593, …). CLI-127's own diff is `26a2ba07` (spec), `dd1f5c78` (status, #2577), `8ff83b00` (plan, #2594), `a55776ce` (apply, #2595): `cli/internal/lane/{phase,status,plan,apply,forge,git}.go`, `cli/internal/cmd/lane{,_status,_plan,_apply}.go` and their tests, plus the `worktree.ProcessInside` / `RecentHarnessSession` exports it consumes. Findings are attributed to that set. The whole range was verified green (below) because the verbs run against it.

### Fresh verification run by this reviewer (not read from `verification.md`)

| Command | Result |
|---|---|
| `cd cli && go build ./...` | exit 0 |
| `go vet ./...` | clean |
| `GOOS=windows go vet ./...` | clean |
| `go test ./...` | exit 0, 42 packages ok, no FAIL |
| `golangci-lint run` (2.12.2 = the pin) | `0 issues.` |
| `features.json` f1–f4 verification lines, run verbatim | `f1 PASS / f2 PASS / f3 PASS / f4 PASS` (each named test `--- PASS`) |
| `dotf lane status` + `dotf lane plan` built from `79cf7e5d`, read-only, against the running herdr | 16 rows, exit 0; 8 `live` rows, none classed `auto`; the only `pending` reap (`feat/trust-worktree-root`) is `live idle`. AC4's live half reproduces. |
| Mutation: `planLanded`'s `case r.Live != LiveIdle` → `== LiveYes` | `TestPlan/reapable with unknown liveness is ambiguous` and `TestApplyNeverReapsALiveWorktree/liveness that is unknown` go FAIL — the guard has teeth |
| Mutation: `Apply`'s `if a.Class != ClassAuto { continue }` → never skip | `TestApply/never runs a verb for pending, adopt, flag or any ambiguous action` FAILs (sweeps=1, changed=5) |

Both probes were separate files, run, then deleted; `git status` is clean apart from this `review.md` and the launcher's pre-existing `review-request.json`. `proposal.md`, `tasks.md`, `features.json` untouched. No commit, no push, no PR comment.

### Spec and task alignment

- **AC1** — met. `TestDerive` covers all nine phases plus precedence rows (`no lease outranks an open PR`, `merged with a live session is still landed`), `TestDerive_HerdrNeverMovesAPhase` makes herdr display-only (ADR-032), `TestStatus` joins the nine through the real seam shape and asserts one forge ask per branch. herdr-absent rows answered (`statusHerdrAbsentStillAnswers`).
- **AC2** — met at the verb level. `TestPlan` has one row per D1 case, both D2 rows (adopt offered vs. no proposal), the D3 landing row; `TestPlanMutatesNothing` runs the whole read path over a real git fixture, asserts the plan is non-empty *before* comparing the sha256 fingerprint of refs, worktree registry, statuses and leases (lesson 451's own trap avoided). `TestMinAgeMatchesClassify` pins the unexported `minAge` literal to `worktree.Classify` — a drift guard this reviewer would not have expected and is glad of.
- **AC3** — met for the verb set, **not met for the "only the auto actions" promise**: see F1. `Verbs` structurally cannot adopt or land (two fields), and `TestApply` pins call counts, `changed` accounting, the printed-only start, and the stop-at-first-error.
- **AC4** — met. `TestApplyNeverReapsALiveWorktree` (live + unknown), `Combine`'s three-state probe merge (table-tested), the liveness case placed *before* the time-only cases in `planLanded`, and the reproducing live run above. The proposal's "default to alive when unsure" holds.
- **`What` §2** promises the head PR's **four-condition check** in `plan`; the verb prints the landing order and the head's command only. Deferred in `verification.md` as ticketed rather than copied (F6).
- `features.json` is all `state: "pending"` with empty `evidence` — correct under the pass-state rule (only the harness stamps `passing`); no finding.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in any spec file.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|---|---|---|---|---|---|
| **Major** | **REAL** | plan↔apply fidelity (`apply`) | `apply` reaps under a **different merge oracle than `plan`**, and removes worktrees the plan handed to a human as `ambiguous`. `realLaneSources` lists worktrees through `lane.ForgeGit` (REST-only `IsPRMerged`), but `realLaneVerbs.Sweep` calls `worktree.Sweep` → `&worktree.RealSweepRunner{}` → `IsPRMerged` = **ancestry** fast path (`worktree/list.go:83`). `Apply` calls sweep repo-wide whenever *any* auto reap exists, so a clean, leased, expired, ≥15 min, no-PR worktree at the base commit (ancestry = merged) is reaped and `git branch -D`'d while `Plan` classed it `flag/ambiguous` with `dotf worktree done --force <path>` — a human decision D1 reserved for the human. | Reproduced in a throwaway test on a real git fixture (two worktrees, both leased/expired/`reap_ok`/clean; forge says merged for one, no PR for the other): `row branch=fix/stalled phase=Stalled state=UNMERGED`; `action kind=flag class=ambiguous branch=fix/stalled resolve="dotf worktree done --force …"`; sweep → `reaped=2 … REAPED fix/stalled`. Probe deleted afterwards. Lesson 450 (this spec's own) documents the ancestry hazard and justifies the status-side fix with "sweep re-checks every gate itself" — that same property is what lets `apply` out-reach the plan. No data-loss path found: ancestry true ⟹ the tip is already in the base, the tree is clean, and gate f + the harness-session skip both held; the violation is of the preview/decision contract, not of work preservation. | **UNTESTED** — `TestApplyNeverReapsALiveWorktree` is the nearest and drives a *fake* recorder, so the two oracles never meet; `TestPlanMutatesNothing` proves `plan` does not mutate, not that `apply` reapplies only what `plan` listed. Needs a named fixture test: *a worktree the plan classed ambiguous survives `apply`*. | code (`cli/internal/cmd/lane_apply.go` + a seam so sweep answers with the lane's forge oracle, or a path-scoped gated reap verb) + tests |
| **Major** | **REAL** | `plan` output executability (`Resolve`) | The `ambiguous` row's "exact command that resolves it" carries an **unquoted path**, so it is not executable on any path with a space — the class this repo already twice lost incidents over. `plan.go:91,127` build it by concatenation: `"dotf worktree done --force " + r.Path`, `"dotf worktree adopt " + r.Path`. Contradicts lesson 449 (*in this same diff*): "The commands printed for a person … paths are **double-quoted** there, and a character live inside double quotes is refused." | Reproduced in a throwaway test: `Plan` on rows with `Path = "/wt/My Worktrees/fix/x"` → `"dotf worktree done --force /wt/My Worktrees/fix/x"` and `"dotf worktree adopt /wt/My Worktrees/fix/2501-a"` (both unquoted). Past incident for the same path class: #2537 / lesson 449 (`C:\Users\Jane Doe\…`). Harm ceiling is a refused command, not a wrong deletion. | **UNTESTED** — every `TestPlan` row uses `/wt/…` with no space; `TestLanePlan_PrintsClassesAndResolvingCommands` likewise. `TestLaneNext_ARepositoryUnderAPathWithASpaceHandsTheLaneOver` proves the repo *does* test this class elsewhere. | code (`cli/internal/lane/plan.go`, quote like the printed lane commands already do) + tests (a spaced-path row) |
| Minor | THEORETICAL | queue↔worktree dedup (`Status`) | `queuedLanes` ties a queue item to a worktree by *branch name* only (`BranchPrefix + "/" + Slug(title)`). A hand-named lane (this very spec runs on `feat/2550-lane-reconcile` for #2550; `chore/autopilot-cli-127-run`) never matches, so the same ticket reads as both a worktree row and a phantom `Pending` row, and `planStart` — which blocks only on `Running`/`Review`/`Landing` — can offer `start/auto` for an issue whose lane is already `Landed`. Not reproduced: the live queue [2501, 2434, 2279] matched its derived names, and `dotf lane next` itself names branches with the same formula (`handoff.go:84`), so only hand-named lanes fall in it. | Code read of `status.go:queuedLanes` + `plan.go:planStart`; live `dotf lane status` above shows no phantom row today. | UNTESTED — no case where a queue item's ticket already has a row under a different branch. Cheap fix: dedup by `Row.Ticket`, not by name. | code + tests (or a spec note that queue↔lane matching is name-derived by construction) |
| Minor | REAL | spec vs code (`What` §2) | `proposal.md` promises `plan` prints "the landing order **and the head PR's four-condition check**". `LandingOrder` prints the order and the head's `dotf pr triage-queue --pr N && dotf pr land N`; the four conditions are not evaluated (deliberate — `prland.Decide` keeps one implementation). The deferral is recorded in `verification.md` as "ticketed rather than copied", but **no ticket number is named**, so the claim cannot be checked. | Live `dotf lane plan` output above (no condition block); `plan.go` has no condition evaluation. | n/a — absent feature; UNTESTED | spec artifacts *or* the tracked ticket: either name the issue in `verification.md`, or accept the wording in `proposal.md` (a contract-set edit → re-review) |
| Minor | REAL | handoff hygiene | `verification.md` "Promotion candidates" is three unanswered `<yes: path / no: reason>` lines and the `Archive checklist` boxes are all unticked, while lessons 449–452 and ADR-051 already exist in the diff; `tasks.md`'s final box (`Verify (same suite). Lesson. Open the PR.`) is unticked although all three PRs merged. `dotf spec archive` refuses an unanswered promotion line, so this blocks the archive on its own. | Read of `specs/CLI-127-lane-reconcile/verification.md`, `tasks.md`; `docs/lessons/lesson-449…452*.md` present; `gh issue view 2550` → `OPEN`, assigned to the owner. | n/a (artifact, not code) | `verification.md` (outside the contract set — fix freely) |
| Minor | SPECULATIVE | performance | `status` walks N worktrees sequentially: one `gh api` **and** one `herdr worktree list` per branch. 16 rows ran in seconds here; `verification.md` measured ~10 s for 12 lanes. The proposal explicitly allows one request per branch, so this is compliant — surfaced only because a fleet on a slow link will feel it. | Live run above; `CachedForge` pins one ask per branch (`TestCachedForge_IsSharedNotStacked`). | covered | — (surface only; do not gate) |
| Question | — | `BranchPrefix` default | `realLaneSources` defaults `BranchPrefix: "fix"` when `.dotf/lane.json` is absent, while the repo's real lanes use `feat/`, `chore/`. Harmless (it only feeds the queued-branch derivation, F3 above), but confirm it is the intended default and not a leftover. | `lane_status.go:104`, `queue.go:LoadConfig` default `"fix"`. | `TestLoadConfig_DefaultsAndRefusesAnUnknownKey` | — |

### Why the guards are worth crediting (each mitigates a documented risk)

Three-state liveness (`Combine`) with `unknown` read as alive, the liveness case placed *before* any time-only reason in `planLanded` (the exact defect `verification.md` caught on its own live run), a forge that cannot answer returning an error rather than "no PR" (`TestREST_AFailedRequestIsAnErrorNotNoPR`), and the two-pass `Derive` + `markHead` restricted to `Review` rows — both mutation edits above went red on named tests, so those are real guards, not decoration.

### Evaluator rubric

| Dimension | Grade | Rationale |
|---|---|---|
| Correctness | **C** | AC1/AC2/AC4 hold under this reviewer's own runs; AC3's "only the auto actions" does not — `apply` deletes a worktree `plan` classed `ambiguous` (F1, reproduced), and a promised `plan` section is unshipped (F6). |
| Verification | **B** | Every verification line reproduces from a clean tree (f1–f4, full suite, lint at the pin, live read-only run); a couple of claims (the four-condition ticket) name no artifact. |
| Scope | **B** | CLI-127's own three commits stay inside `lane/` + `cmd/lane*` and consume `worktree`/`herdr` through exported API only, as the proposal requires; the launcher's base range additionally carries 17 other lanes' commits, which is a review-range fact, not scope creep by this change. |
| Reliability | **C** | Error paths and idempotence are genuinely fail-closed, but the one irreversible verb out-of-reaches its own preview (F1) and its printed resolving command is unexecutable on a spaced path (F2). |
| Maintainability | **B** | Longest production function 41 lines (`newLanePlanCmd`, declarative cobra glue), everything else ≤38; lint 0 issues at the pin; phases in one table with precedence written down. No dead code found. |
| Handoff-readiness | **C** | Lessons and an ADR were captured in-diff, but the promotion lines are unanswered, the archive checklist and the final task box are unticked, and #2550 is still open — the archive will refuse on the first two alone. |

Aggregation: three C, no D → PASS WITH GAPS at minimum; the two **REAL Major** findings (F1, F2) take the more severe path.

### Verdict

**FAIL**

### Recommended next steps (minimum set to flip to PASS)

Contract-set changes are on the table under a FAIL, and the next review round is the mechanism, not an inconvenience.

1. **F1 — one merge oracle for the reconcile.** Either drive `apply`'s sweep through the lane's forge-backed runner (export a read-only seam on `worktree.Sweep` the way `ProcessInside` was exported for `status`), or make the reap path-scoped so `apply` removes only the branches `Plan` classed `auto`. Then add the named test the suite does not have: *a fixture worktree that `Plan` classes `ambiguous` is still on disk after `Apply` with the real sweep runner.* If instead the owner accepts that a plan is verb-granular and not a path-level preview, that is a `proposal.md` (AC3 / D1) amendment — a contract edit, so it needs the re-review too. Say which.
2. **F2 — quote the path in `Resolve`.** Double-quote it and refuse what cannot live inside quotes, the way lesson 449's printed commands already do; add a `TestPlan` row whose `Path` contains a space.
3. **F6/F5 — close the artifacts.** Name the ticket holding the four-condition check in `verification.md`; answer the three promotion lines (`docs/lessons/lesson-449…452*.md` are in the diff, so they are real answers, not placeholders); tick or disposition the final `tasks.md` box; move #2550 with the archive.
4. Re-run `dotf spec review CLI-127-lane-reconcile` on the new head; `dotf spec archive` stays refused until a fresh passing review over the same contract digests exists.

`dotf spec archive` is **not advisable** in the current state: it would be refused twice over — by this FAIL verdict, and independently by the unanswered promotion lines in `verification.md`.
