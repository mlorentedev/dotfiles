---
spec: "CLI-127-lane-reconcile"
verdict: "FAIL"
reviewed_sha: "65d2486e4b96f9264ae63594758ff369284d65b0"
reviewer: "nan/deepseek-v4-flash"
date: "2026-10-16"
---

## Adversarial review

**Scope**: `CLI-127-lane-reconcile` — `dotf lane status | plan | apply`, over the whole change
`git diff 1ac92409f1e8af930e55ee20dcbfc2e55d96edaa...HEAD` (27 commits, 126 files, +6105/-523; the
lane work is 3407 added lines under `cli/internal/lane/` + `cli/internal/cmd/lane*.go`).
**Sources**: `specs/CLI-127-lane-reconcile/{proposal,tasks,verification,features}.md` +
`review-request.json`; `cli/internal/lane/{phase,status,plan,apply,forge,git,queue,handoff,lease}.go`;
`cli/internal/cmd/lane_{status,plan,apply}.go`, `cli/internal/cmd/lane.go`;
`cli/internal/worktree/{sweep,session,harness_session,process_export,done}.go`;
`cli/internal/prland/prland.go`. No PR reference was supplied; the diff and HEAD are the evidence.

### Spec and task alignment

- **Contract set is intact and this review is not stale.** `ContractDigests` (the repo's own
  normaliser, run from `cli/internal/spec`) returns `proposal.md 207c2d84…`, `tasks.md 91899629…`,
  `features.json 1337d5ff…` — byte-equal to the `contract_digests` the launcher recorded in
  `review-request.json`. No contract file was touched by this review.
- **AC1–AC4 are each traceable to a named test**, and the four `features.json` verification lines
  pass: `TestDerive`/`TestStatus`, `TestPlan`/`TestPlanMutatesNothing`,
  `TestApply`, `TestApplyNeverReapsALiveWorktree` — all `--- PASS`, exit 0, run fresh in this session.
- All `tasks.md` boxes are ticked and the artifacts behind them exist; no `[AGENT-DRAFT]` or
  `[AGENT-SUGGESTION]` tag remains; `features.json` carries no `passing`-with-empty-`evidence` entry
  (all four are `pending`), so the tasks.md pass-state gate is not violated.
- `proposal.md` declares neither `risk: high` nor `review: waived`, so no second signature applies.
  Its `wip_override` (12 active specs against a limit of 10, owner decision 2026-10-16) is declared
  and justified in the frontmatter rather than hidden — noted, not a finding.
- **Baseline suites run fresh in this session**: `go build ./... && go vet ./...` exit 0;
  `GOOS=windows go vet ./...` exit 0; `go test ./...` all packages `ok`;
  `golangci-lint run ./...` (2.12.2, the pin) → `0 issues`.
- **Verified sound (checked, no finding):** `Action.Due` uses `json:"omitzero"` and Go 1.26.0 is the
  toolchain, so an `auto` action emits no zero `due` and a `pending` one emits a real timestamp;
  `Row.Meta` is `json:"-"`, so `--json` exposes no lease internals; `Verbs` has no adopt/land member,
  so D2/D3 are structurally unreachable from `apply` rather than merely unasserted.

### Findings

| # | Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|---|----------|---------|------|---------|----------|---------------------------|--------------|
| F1 | Major | REAL | test coverage / safety wiring | **The `apply` wiring's oracle and its session stores can both be deleted with the whole suite green — and the session-store half reaps a worktree with a live, resumable session.** `realLaneVerbs` (`cli/internal/cmd/lane_apply.go:59-72`) is the only production construction of `lane.ForgeSweep`, and it is exercised by *nothing*: grep for `realLaneVerbs`/`realLaneSources` in `*_test.go` returns no caller. `TestApplyNeverReapsWhatPlanCalledAmbiguous` builds its **own** `ForgeSweep{…, Forge: forge}` inside the test (`apply_oracle_test.go:56`), so it pins the fixture, not the wiring. This is the exact gap the round-1 F1 fix was about (lesson 454). | Mutations I applied and reverted: (a) removing `Forge: src.Forge` from `lane_apply.go:66` → `go test ./internal/cmd ./internal/lane` → **both `ok`** (undetected; at runtime it would panic, `ForgeSweep` with a nil `Forge` dereferences on `PRFor`); (b) removing `Sessions: sessionStores()` from the sweep options → **both `ok`** (silent). Reverting mutation (a) **inside the test file** instead does fail it (`changed=2`), which is what `verification.md` reports — so that claim is true of the test's own line only. Reproduced consequence of (b): a merged, clean, lease-expired worktree with a pi session written seconds ago, `RecentHarnessSession → live=true`, swept `WITH sessionStores()` → `reaped=0 session_skips=1 worktree_present=true`; `WITHOUT` → **`reaped=1 session_skips=0 worktree_present=false`**. Gate f does not cover it: a resumed-but-not-running session has no process. | **UNTESTED** — no named test enters `realLaneVerbs`. | code + tests (a `cmd`-level test over the real wiring, or extract the runner/sweep-option construction and assert it carries the forge and the session stores; the two halves differ in blast radius — the forge half panics loudly, the session half deletes a live worktree silently) |
| F2 | Major | REAL | spec-vs-code / D1 contract | **The resolving command `plan` prints for a live-session worktree does not resolve it.** `proposal.md` D1: *"Each `ambiguous` row carries the one command that resolves it, such as `dotf worktree done --force <path>`"*. `abandon()` emits exactly `dotf worktree done --force <path>` for the liveness-`live`/`unknown` row (`planLanded`, `plan.go:117-121`). But `worktree.Done` refuses a worktree with a recent harness session *whatever `--force` says* — the escape is `--ignore-recent-session`, and `cli/internal/cmd/worktree.go:652` documents that flag as the one `--force` does not substitute for. So the class AC4 exists to protect is handed a command that exits non-zero and leaves the worktree. Safe direction (it refuses rather than destroys), but the "yes or no, not design work" promise fails for that row and the human gets no command that does the job. | Probe over a real repo + real `SaveMetadata` + a pi session store written now: `RecentHarnessSession` → `live=true`; `worktree.Done{DoneOptions{…, Force: true, Sessions: {PiSessions: pi}}}` → `err="a pi session for … wrote … 0s ago; removing the worktree would strand it. End or resume it elsewhere, or pass --ignore-recent-session"`, worktree still on disk. (`--force` maps 1:1 to `DoneOptions.Force`, `worktree.go:628`, so the probe exercises the printed command's code path.) | **UNTESTED** — `TestApplyNeverReapsALiveWorktree` proves `apply` will not reap, and `TestPlanQuotesTheResolvingCommand` proves the path is quoted; neither runs the printed string against `worktree.Done`. | code + tests (print the command that actually resolves the row, or add the escape flag and say in the reason that it strands the session) |
| F3 | Minor | REAL | consistency / contract fidelity | **`planUnadopted` emits `Kind: adopt` with an empty `Resolve` when the path cannot be quoted**, while `abandon()` in the same file deliberately downgrades to `KindFlag` for that exact case (lesson 449). `plan.go:159-166` sets `a.Kind = KindAdopt` and then assigns `a.Resolve, a.Reason = withPath(...)`, discarding the "no command" outcome; `abandon` (`plan.go:76-80`) keeps `KindFlag` when `Resolve` comes back empty. The degradation is contained — `planDetail` falls back to `a.Reason`, and `Class` is `ambiguous` either way, so **not** mislabelled `auto` — but the row now claims an adopt action with nothing to run. | Code read of `plan.go:76-80` vs `159-166`; `TestPlanQuotesTheResolvingCommand` pins the `abandon` half (`fix/c`, `$` in path → no command + reason) and has no adopt analogue. | **UNTESTED** (no adopt row with an unquotable path) | code (fall back to `KindFlag` when `withPath` returns `""`) + tests |
| F4 | Minor | THEORETICAL | destructive advice | The same `abandon()` command is attached to the **DIRTY** row (merged + uncommitted changes). `dotf worktree done --force` discards uncommitted work — its own help says so ("after --force it may hold the only copy") — and `plan` presents it as the row's resolver with no warning in the printed detail beyond the reason string. A human reading "the exact command that resolves it" for a dirty lane may not register that the command is lossy. Committed-but-unpushed work survives (the branch is kept), so the loss is confined to the working tree. | Code read of `planLanded` → `abandon` → `withPath("dotf worktree done --force ", …)`; `worktree/done.go:57-62` + `cli/internal/cmd/worktree.go:586-611`. No reproduction attempted (would require asserting intent on a human's judgement call). | **UNTESTED** | code (print a lossy warning in the reason for the DIRTY row) or spec (`proposal.md` D1 to say the command discards local changes) |
| F5 | Minor | REAL | review scope | The reviewed range aggregates 27 commits that are not this spec's work: the lane verbs are ~3407 added lines, but the same range carries `converge_runlog` barrier work, `deploy/record.go` (ADR-051), `env.DotfilesCheckout`, `doctor` catalog refresh, `prtriage` review-channel reads, `secrets` registry resolution, and many asset/version bumps. The launcher's base is the spec's start, so this is by construction rather than creep, but it means "diff matches proposal" can only be graded approximately and an unrelated regression would be attributed here. | `git diff --stat 1ac92409...HEAD` (126 files); the eight commits `8ff83b00 a55776ce 75c36c09 cf89a31b 65d2486e …` are lane, the rest are not. | n/a (scope observation) | spec (none needed; recorded so the coupling is not mistaken for a defect of this change) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC1–AC3 hold and the four feature commands pass, but AC4's safety property is guarded by no test that reaches the production wiring (F1), and the D1 resolver does not resolve the AC4 class (F2). |
| Verification       | C | Evidence is otherwise reproducible (I re-ran build/vet/both-OS/lint/the 4 features), but the round-1 F1 claim — "replacing the override with ancestry makes it fail" — is reproducible only against the test's own line, not the production one, so the headline fix's guard is overstated. |
| Scope              | B | The lane diff matches `proposal.md` §What; the surrounding range carries unrelated work by launcher construction (F5), documented rather than hidden. |
| Reliability        | B | Error paths handled (apply stops at the first verb error; a forge/herdr that cannot answer fails closed; `LiveUnknown` never feeds an auto action), idempotency proven (`changed=0` second run) — but a silently removable safety option (F1) is a reliability gap. |
| Maintainability    | B | Small functions, explanatory comments, no dead code; `Apply` CC=10 and `Handoff` CC=13 sit at/above the skill's ≤10 line but well under the repo's enforced gocyclo 22 (checked with `golangci-lint` at `min-complexity: 8`). |
| Handoff-readiness  | A | Spec triad complete, 9 lessons promoted and indexed, the ADR line answered with a reason, `verification.md` records two review rounds and the live-run defects with the ticket that closed them. |

### Verdict

**FAIL** — two **REAL Major** findings (F1, F2), each `UNTESTED`, so both paths of the aggregation rule
agree (`severity × reality`, and a rubric containing C).

`dotf spec archive` is **not advisable** in this state. The minimum set of actions that flips this to
PASS: **(1)** add a test that exercises `realLaneVerbs` — or extracts and asserts its construction —
and fails when `Forge: src.Forge` or `Sessions: sessionStores()` is removed, then re-run mutation
`(b)` above and show the sweep skipping a live-session worktree; **(2)** make the `ambiguous` resolver
for a liveness-`live`/`unknown` row a command that actually resolves it (or state the escape flag and
the stranding cost in the reason), with a test that runs the printed string against
`worktree.Done`. F3 and F4 are tracked, not blocking.

### Recommended next steps

Written for the implementer to disposition in `verification.md` (applied / ticketed / declined with a
reason) or to carry into a follow-up ticket. **These are outside the contract set**, so acting on
them does not invalidate this verdict — but F1 and F2 need **code** changes and a fresh review, so
they are the FAIL's own next round rather than post-archive work.

- **F1 (code + tests)**: cover the wiring. A `cmd`-level test is the cheap form — build the real
  `realLaneVerbs` with a stubbed `lane.Sources` whose `Forge` answers `merged` for one branch and
  point `Sessions` at a temp store with a fresh transcript, then assert the sweep skipped it. If a
  full wiring test is judged too heavy, extract `newSweepOptions(src, root)` and assert the returned
  values; a construction assertion is weaker than the over-one-real-repo form lesson 454 already
  argues for.
- **F2 (code + tests)**: change the live/unknown row's resolver. Either emit
  `dotf worktree done --force --ignore-recent-session <path>` with the reason naming that it strands
  the session, or emit no command and say why. Add the case to `TestPlanQuotesTheResolvingCommand`
  or a sibling test, and ideally prove it by invoking `worktree.Done` with the parsed flags.
- **F3 (code + tests)**: in `planUnadopted`, when `withPath` returns an empty command, set
  `a.Kind = KindFlag` as `abandon` does; add the unquotable-path adopt row to the table.
- **F4 (code or spec)**: name the loss in the DIRTY row's reason, or record in `proposal.md` D1 that
  `done --force` discards local changes. **If you choose the spec route, that edit is a contract
  change and needs a re-review** — prefer the code route.
- **F5 (none)**: no action; recorded so the aggregated range is not read as this spec's scope.
- **Knowledge**: `docs/lessons/lesson-454-…` states that a preview and the verb that executes it must
  ask the same oracle. F1 shows the class has a sibling: *a shared oracle is only shared if the
  production wiring that supplies it is exercised* — the fix was made in the layer the test built for
  itself, so both halves agreed in the fixture while the wiring stayed unguarded. A lesson there
  would be in-session knowledge hygiene for this repo's `docs/lessons/`; I did not write it, since a
  reviewer writing an implementer's lesson weakens the independence this run exists for.
