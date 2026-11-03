---
spec: "PLAT-005-vm-session-verbs"
verdict: "FAIL"
reviewed_sha: "f478e1b1a59acb6cfbabe671b1b85d64852beda5"
reviewer: "nan/mimo-v2.6-flash"
date: "2026-11-02"
---

## Adversarial review

**Scope**: PLAT-005-vm-session-verbs (diff `4a70ad7dab62d0c681697882ad9063358acba802...HEAD`, the base the launcher resolved)
**Sources**: `specs/PLAT-005-vm-session-verbs/{proposal,tasks,verification,features}.json…` (proposal.md, tasks.md, verification.md, features.json), `docs/adr/adr-055-vm-guests-are-driven-through-session-verbs.md`, `cli/internal/vm/*`, `cli/internal/cmd/vm*.go`, `review-request.json`

### Spec and task alignment

- Contract digests verified: `dotf spec digest PLAT-005-vm-session-verbs` matches `review-request.json` for all three contract files (features.json `83a3c869…`, proposal.md `3ddae75e…`, tasks.md `ae09e359…`).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in any spec file (the only grep hits are this review's own transcript).
- All 11 acceptance criteria are ticked and each has a named test that exists in the tree; I ran the features.json verification commands directly (`f4`, `f10` green; `f7` green with all three named tests executing) and the whole suite fresh: `go -C cli test ./... -count=1` → **exit 0, 52 packages ok, zero FAIL** (2026-11-02, this session). `go vet ./internal/vm/ ./internal/cmd/` → 0; `golangci-lint run ./internal/vm/... ./internal/cmd/...` → 0 issues; `go test -race ./internal/vm/ -count=1` → ok.
- Every `[x]` task maps to a commit in the range (PR 1a `3ccbc265`, 1b `08a4ef7a`/`dfaf2867`/`d3e4852a`/`df550f48`, 1c `cd72ea42`/`1cd1ec58`/`90bbfd81`, 1d `85d5dfa2`, PR 2 `b49c5bad`, 2b `a11b7114`, PR 3 `c57851b1`, progress split `32a24d8d`). No `[x]` without diff evidence.
- AC6's honest rewrite (run composed *below* the verbs) is implemented as written: `Verify` literally calls `LinuxSessions{L: l}.guestRun(...)` with `forwardable` + `redact.New` (vm.go), and Windows `Run` shares `bootGuest`/`startTask`/`pollExit` with the session verbs (run.go). Existing `run` assertions were kept: the deleted lines in run_test.go/vm_test.go are signature/stream-plumbing changes only (`Out` → `Progress`, `Verify` gained a ctx), not assertion changes.
- Test deletion from PR 1a passes the retention bar: all 7 `TestRedactWriter_*` tests moved intact to `cli/internal/redact` as `TestWriter_*` (7 for 7, renamed not weakened), commit `3ccbc265`. I mutation-tested the guarded property: deleting the redaction keys in `linux_exec.go` (`redact.New(..., nil)`) turns `TestExecForwardsASecretAndMasksIt` red with the raw value leaking (`linux_exec_test.go:257`), and reverting restores green — shown, not assumed.
- features.json: all 11 entries carry non-empty `evidence`; none is agent-written `passing` with an empty evidence field.
- verification.md's claim that `TestEmbeddedTemplatesMatchVault` fails on every branch did **not** reproduce here (vault and initrepo both `ok` in the fresh run) — its claim is environment-dependent and stale at worst; the suite is greener than documented.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|--------------|
| Major | REAL | contract / JSON record | With `--json`, **no record is written at all** when the session store cannot open (`limactl` or `qemu` missing, StateDir error): `g.open()`'s error returns a plain error before any `emitSession`. ADR-055 decision 7 — the contract this spec implements — says "`--json` writes one record per call to stdout, whatever the outcome", `failed` is defined in `session.go` as "dotf could not run the verb", and `status`'s own help says "a failure writes one failed record instead". A JSON consumer (the spec's stated audience) gets empty stdout and must fall back to the process code the ADR says never to read. | Reproduced this session: `PATH=/usr/bin:/bin <dotf> vm linux exec --session ghost --json -- true` → exit 2, **stdout empty**; same for `vm linux status --json`, `vm linux up --session proof --json`, and `vm windows status --json` (`qemu-system-aarch64 is not on PATH`). No design note in proposal/tasks/ADR carves this class out; the test table's no-record cases are all usage errors, a different class. | UNTESTED — `TestVMVerbRefusals` covers session-state and usage refusals only; no test exercises the store-open failure path (grep for `LookPath` in `cmd/*_test.go` finds nothing for the session verbs) | code + tests (emit a version-1 `failed` record under `--json` on open failure for all five verbs; add a named regression case) |
| Minor | THEORETICAL | concurrency | Two concurrent `up --session <same name>` both pass the `held()`/`holder()` check before either claims it (TOCTOU; both `os.MkdirAll` on an existing dir succeed). On Linux the loser's failure cleanup runs `limactl delete --force` + `os.RemoveAll` on the shared instance/state the winner just created; on Windows the loser's boot-failure path kills the process and removes the overlay disk. The proposal only resolved the *serialized* case ("two callers that pick one name do not share a guest"). | Code read of `LinuxSessions.Up` (held → MkdirAll, non-exclusive) and `WindowsSessions.Up` (refusal → bootGuest → MkdirAll); not reproduced | UNTESTED | code + tests (atomic claim, e.g. `mkdir` O_EXCL / `os.Mkdir`) |
| Minor | THEORETICAL | refusals ordering | On Windows, the stdin/`--secret` refusals run *before* `serving()`, so `dotf vm windows exec --session ghost` with piped stdin answers `unsupported` exit 2 where AC11 promises `no-session` exit 3 for an unknown name. Narrow: only when stdin is a pipe (or `--secret` given) *and* the name is unknown. | Code read of `WindowsSessions.Exec` refusal order; `TestVMVerbRefusals` exercises this only through the Linux verb | UNTESTED (Windows ordering) | code or spec note (fix in code to prefer AC11; otherwise document the precedence) |
| Minor | SPECULATIVE | verification portability | f5's sole named test `TestVMProgressGoesToStderr` skips outside macOS Arm64, so on a Linux CI host its verification command exits 0 having run **zero** tests; the portable AC5 evidence (`TestVMUpWritesProgressToStderrAndTheNameToStdout`) exists but f5 does not name it. | Test guard `if runtime.GOOS != "darwin" … t.Skip`; f5 command in features.json | `TestVMUpWritesProgressToStderrAndTheNameToStdout` (runs everywhere) | spec (features.json — contract set; valid to edit now, before the re-review) |
| Minor | REAL | quality | `WindowsSessions.Exec` is 70 lines against the repo's <40-line function rule (golangci passes only because the repo config does not enforce it). | `awk` line count on `cli/internal/vm/windows_exec.go` = 70 | n/a (structure, not behavior) | code (extract the settle/timeout block, as `launch`/`collect` already are) |

Walked the code-level checklist: no injection surface (argv never passes a shell — Linux `limactl shell -- argv`, Windows program travels in `request.json`, not on the task's line; `ValidSessionName` fences path elements); no hardcoded secrets (tests use synthetic `fakeSecret`, lesson 261 respected); secrets resolve through the registry and are masked at both writers (mutation-proven); host deadlines and `WaitDelay`/`taskkill` paths handle the timeout cases on both guests; no unbounded buffers (1 MiB `capped` per stream, QGA chunked).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | C | AC1–AC11 behaviors verified with strong negative-path coverage, but one documented outcome of AC2/AC5 (store cannot open) produces no record — criteria partially met, reproduced above |
| Verification       | B | Every feature names a real, passing command and real-guest evidence is detailed; the store-open outcome is untested and the guest runs are not independently reproducible here |
| Scope              | B | PLAT-005's own commits match the proposal; the launcher-set range also carries ~110 commits from other specs' PRs, and the in-vm clock sync (#2963) is a separate PR with its own tests — documented side-changes, no silent creep |
| Reliability        | B | Error/timeout/partial-boot paths handled on both guests (grace polls, settle table, cleanup on half-boot); residual TOCTOU on concurrent same-name `up` |
| Maintainability    | B | lint 0 issues, tests extensive and table-driven; one 70-line function exceeds the 40-line standing rule |
| Handoff-readiness  | A | proposal/tasks/verification/features complete and consistent, ADR-055 and lesson 546 recorded, review-request.json well-formed |

Aggregation: one **C** (no D) → PASS WITH GAPS minimum; the severity axis independently requires more: one **REAL Major** → **FAIL**.

### Verdict

**FAIL** — one REAL, UNTESTED Major: under `--json` a store-open failure emits no record, contradicting ADR-055 decision 7 ("one record per call to stdout, whatever the outcome") and the `status` verb's own documented behavior. Reproduced on this session against a build of the reviewed tree (four invocations, all empty stdout). The Minor findings are tracked, not gating; the SPECULATIVE portability note cannot move the verdict by itself.

### Recommended next steps

Minimum set that flips this to PASS (round 2 re-reviews):

1. **Code**: in `cli/internal/cmd/vm_session.go` / `vm_exec.go` / `vm_cp.go`, route a `g.open()` failure through `emitSession` as a version-1 `failed` record when `--json` is set (all five verbs), keeping exit 2. If the owner instead reads decision 7 as excluding host-setup refusals, that reading must be written into the proposal/ADR explicitly — today the code contradicts both the ADR and the verb's own `--help` text.
2. **Tests**: add a named regression case (e.g. a `TestVMVerbRefusals` entry whose store fails to open) asserting stdout carries exactly one `failed` record under `--json` and empty stdout without it.
3. **Re-review**: `dotf spec review PLAT-005-vm-session-verbs` after the fix lands.

For the implementer to disposition in `verification.md` (applied / ticketed / declined with reason) or a follow-up ticket — these do **not** gate the archive once the Major is fixed: the concurrent-`up` TOCTOU, the Windows refusal-precedence ordering, f5's skip-portable verification command (a `features.json`/contract-set edit, so make it before the re-review if you want it), and the 70-line `WindowsSessions.Exec`.

### Definition of Done (this review session)

- **Debt**: all findings are recorded in this review.md (the tracked artifact the archive gate reads); nothing noticed and left unsaid. A fix ticket is the implementer's next step with the FAIL verdict in hand.
- **Knowledge**: nothing beyond the findings that the diff should carry; no promotion owed by a review pass.
- **Board**: skipped by mandate — the launcher forbids PR comments and issue actions for this run; board movement belongs to the fix round.
- **Review**: N/A — the spec's work landed via merged PRs (#2930–#3104); no open PR to triage, and commenting is forbidden by this run's instructions.
- **Evidence**: fresh this session — full `go test ./... -count=1` exit 0 (52 pkgs), `go vet` 0, `golangci-lint` 0 issues, `-race` on `internal/vm` ok, redaction mutation red→green, digest check via `dotf spec digest`, four reproduction runs for the Major.
