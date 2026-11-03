---
spec: "PLAT-005-vm-session-verbs"
verdict: "PASS"
reviewed_sha: "1280b8ad4813acfc9f47ee08fd1f2785fb3db38c"
reviewer: "nan/qwen3.8-flash"
date: "2026-11-02"
---

## Adversarial review

**Scope**: PLAT-005-vm-session-verbs — round 2, re-review after the round-1 FAIL fix (#3107)
**Sources**: `specs/PLAT-005-vm-session-verbs/{proposal,tasks,verification,features}.md|json`, `specs/PLAT-005-vm-session-verbs/review-round1.md`, `docs/adr/adr-055-vm-guests-are-driven-through-session-verbs.md`, `cli/internal/vm/*`, `cli/internal/cmd/vm*.go`, diff `4a70ad7dab62d0c681697882ad9063358acba802...HEAD` (base stated by the launcher)

### Spec and task alignment

- **Freshness verified, not assumed.** `dotf spec digest PLAT-005-vm-session-verbs` reproduces all three `contract_digests` in `review-request.json` exactly (features.json `9fdad57f…`, proposal.md `3ddae75e…`, tasks.md `ae09e359…`). features.json's digest differs from the round-1 request (`83a3c869…`), which is the f5 edit the round-1 disposition records — so this round does read the current contract.
- **This round's real question was the fix, not the re-confirmation.** Round 1 was a FAIL on one REAL, UNTESTED Major: under `--json` a store that could not open wrote no record. I verified that claim independently rather than trusting it, and then attacked the fix three ways: mutation, a real-binary reproduction over the launcher's PATH, and the exit-code path the fix's second commit introduced.
- **The round-1 Major is fixed, and the fix is pinned.** `sessionGuest.openFor` (cli/internal/cmd/vm_session.go) routes every store-open error through `writeSessionJSON` under `--json`.
  - *Named test exists and is non-vacuous*: `TestVMVerbsWriteAFailedRecordWhenTheStoreCannotOpen` (6 subtests: up, down, status, exec, cp, windows status).
  - *Mutation proof, run this session*: replacing `writeSessionJSON(cmd, rec)` with a no-op turns **all six subtests red** with `--json stdout = "" (unexpected end of JSON input)`; restoring it returns the package to `ok`. The guarded property is shown, not assumed.
  - *Real path, not only the injected seam*: a build of this tree (`go -C cli build -o /tmp/dotf-review ./cmd/dotf`) run with `PATH=/usr/bin:/bin` and an empty `XDG_STATE_HOME` now emits exactly one record for `vm linux status|up|down|exec|cp --json` and `vm windows status|exec --json`: `{"version":1,"session":"proof","os":"linux","status":"failed","exit":null,…,"stderr":"limactl is not on PATH: run `dotf converge --only tools`"}`, exit 2. Stdout carries one line; the human error still goes to stderr, so AC5's "stdout is exactly one record" holds on the failure path too.
  - *Exit code survives the join*: `ExitCode` uses `errors.As`, which traverses the `errors.Join` tree, so `openFor`'s `errors.Join(err, werr)` keeps the open error's 2. Untested branch (finding 3 below), but argued from `cli/internal/cmd/exit.go`.
- **Round-1 dispositions are real, not prose.** `gh issue view` this session: #3105 "vm: concurrent up on one session name can destroy the winner's guest" OPEN; #3106 "vm: Windows exec refuses stdin before an unknown session, and the function is 70 lines" OPEN. f5's edit is applied and I verified its named tests carry no `GOOS` skip (`t.Skip` count in `vm_session_test.go` = 0; the darwin-only `TestVMProgressGoesToStderr` is no longer the only one named). The spec's issue #2842 is still OPEN — correct, the archive closes it.
- **Suite and gates, fresh, this session, on this tree**: `go -C cli test ./... -count=1` → **exit 0, 52 packages ok, zero FAIL**; `go -C cli vet ./internal/vm/ ./internal/cmd/` → clean; `golangci-lint run ./internal/vm/... ./internal/cmd/...` (from `cli/`) → **0 issues**; `go -C cli test ./internal/vm/ ./internal/cmd/ -count=1 -race` → ok.
- **Every `features.json` verification command run, and checked for vacuity.** All 11 exit 0; each ran real tests (26, 16, 4, 16, 12, 82, 6, 4, 21, 20, 22 RUN/PASS lines respectively), so none is a `-run` pattern that matched nothing. The `TestWindows*` names are not dead on this host: `windows_exec.go`/`windows_session.go` carry **no build tags** (host-side orchestration over QGA), so those tests execute on macOS as well.
- **AC7, the security criterion, re-probed rather than accepted.** `TestExecForwardsASecretAndMasksIt` asserts the value is absent from the record's stdout **and** stderr, from **plain** stdout and **plain** stderr, and from argv, with the guest writing the secret split across two `Write` calls (`fakeSecret[:7]` then the rest) — the case a naive boundary-flushing masker leaks. `linux_exec.go:105-106` wraps both streams in `redact.New` over a `MultiWriter`, so the record's own captured bytes are the redacted ones, which also means `Record.Summary()` (the non-JSON error line) cannot echo a forwarded value. The Windows verb refuses `--secret` outright (ADR-055 decision 12; `TestWindowsExecRefusesStdinAndSecrets`). No credential reaches the transcript; tests use a synthetic `fakeSecret` (lesson 261).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags in any spec artifact (grep over the spec folder excluding transcripts).
- All 11 acceptance criteria and all `tasks.md` boxes are ticked, and each maps to code plus a named test that exists and runs. AC6's rewrite ("composed *below* the verbs") is implemented as written on #3092 and its `run` assertions survived round 1's diff check.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | contract / stdout contract | Fixing the open-error class moved the record/no-record line but did not make it explicit. Under `--json`, a *usage* refusal still writes **no** record and exits 2: `vm linux exec --json --session ../bad -- true`, `vm linux exec --json --session proof` (no program), `vm linux cp --json proof:/a proof:/b`, `vm linux cp --json ./missing`. Decision 7 says "one record per call to stdout, **whatever the outcome**"; decision 8 files "bad flag" under exit 2. A strict reading of 7 covers these too; a JSON consumer must still special-case exit 2 with empty stdout. Defensible as drawn (intent is not yet known, so `os`/`session` would be guesses), but the ADR does not say so. | Reproduced this session against `/tmp/dotf-review` built from `1280b8ad`, all four exit 2, stdout empty | UNTESTED — `TestVMVerbsWriteAFailedRecordWhenTheStoreCannotOpen` covers the open-error seam only; no test pins the usage-refusal class either way | docs (ADR-055 decision 7/8 note — **outside** the contract set, so it can be written after this verdict), or code if the owner wants total uniformity |
| Minor | THEORETICAL | exit codes | A cobra **parse** error exits 1, where ADR-055 decision 8's parenthetical names "bad flag" for 2: `vm linux status --json --session proof` → `Error: unknown flag: --session`, exit 1. Pre-existing `exit.go` behaviour (untagged → 1, deliberately fail-closed) and CLI-wide, not introduced here; a semantic bad value does exit 2 (`--timeout -5s` → 2, reproduced). | Reproduced this session | UNTESTED (no named test pins an unknown-flag code for the verbs) | docs (one clause in ADR-055 decision 8: "bad flag *value*"; undefined flags stay 1) |
| Minor | SPECULATIVE | error path | `openFor`'s `errors.Join(err, werr)` branch — taken when the record itself cannot be written — has no named test; the code is read as correct (`errors.As` traverses a multi-error tree, so the open error's 2 survives) but the property is unpinned. If it regressed, a consumer would read 1 instead of 2 on a broken stdout. | Code read of `vm_session.go:133-142` + `exit.go:29-38`; not reproduced | UNTESTED | tests (optional: a subtest with a failing stdout writer) |
| Minor | THEORETICAL | concurrency | **Carried from round 1, tracked**: two concurrent `up --session <same name>` both pass `held()`/`holder()` before either claims it; the loser's cleanup can delete the winner's instance/overlay. | `LinuxSessions.Up` / `WindowsSessions.Up` code read (round 1); not reproduced | UNTESTED | code + tests — **ticketed #3105 (OPEN, verified this session)** |
| Minor | THEORETICAL | refusals ordering / quality | **Carried from round 1, tracked**: Windows `exec` applies the stdin/`--secret` refusal before `serving()`, so an unknown name with piped stdin answers `unsupported` (2) where AC11 promises `no-session` (3); `WindowsSessions.Exec` is 70 lines against the repo's <40 rule. | Code read of `WindowsSessions.Exec` (round 1) | UNTESTED (Windows ordering) | code — **ticketed #3106 (OPEN, verified this session)** |
| Question | — | review scope | The launcher's range `4a70ad7d…HEAD` spans ~110 commits, most of which belong to other specs (prreview, harness, secrets, sshtarget). Both rounds reviewed PLAT-005's own surface (the `vm` verbs, `cmd/vm*.go`, `redact`, the AC10 fixtures) plus their interaction with `run`; the rest of the range is outside this spec's contract and was not re-adjudicated here. | `git log --oneline` on the range | n/a | human: confirm the base is what the archive should be gated on, or narrow it to PLAT-005's PRs |

Walked the code-level checklist over the fix and its neighbours: no injection surface (argv never reaches a shell; the Windows program travels in `request.json`, not the task line; `ValidSessionName` fences path elements — `../bad` is refused, reproduced); no hardcoded credentials (registry-resolved, refused before decryption when short or multi-line, `linux_exec.go:132`); the failed-open record carries only the host error string, no secret; no unbounded buffer (1 MiB cap per stream, QGA chunked); no new blocking or goroutine paths in the fix; `openFor` returns early so a call can never emit two records.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | The round-1 REAL defect is closed and pinned by a named test; every AC maps to a test that runs here; residual gaps are wording ambiguity in the record-vs-usage-refusal line, not observed misbehaviour |
| Verification | B | All 11 features.json commands re-run green and proven non-vacuous, plus a mutation red→green on the fix; the real-guest runs (AC8/AC10) and `evidence` fields are prose summaries I cannot reproduce on this host, so they stay credible-but-unreplicated |
| Scope | B | PLAT-005's own commits match the proposal; the launcher's range carries ~110 unrelated merged PRs, and no in-scope commit is unexplained |
| Reliability | B | Timeout/cleanup/partial-boot paths on both guests; the concurrent-`up` TOCTOU is real, narrow and ticketed (#3105) |
| Maintainability | B | vet clean, golangci-lint 0 issues, -race clean, tests table-driven; `WindowsSessions.Exec` still exceeds the 40-line rule (ticketed #3106) |
| Handoff-readiness | A | Round-1 review retained as `review-round1.md`, every finding disposed with a real ticket or an applied edit, dispositions recorded in `verification.md`, ADR-055 and lesson 546 in the tree, promotion lines answered with existing paths |

Aggregation: all six dimensions B or above, no C, no D → **PASS** by the rubric. Severity axis independently: no Blocker; the only REAL Major (round 1) is fixed and re-verified; the open Majors are zero; remaining findings are Minor, and the two carried code Minors are ticketed. The two paths agree.

### Verdict

**PASS (adversarial)**

`dotf spec archive PLAT-005-vm-session-verbs` **is advisable** in the current state: the contract digests are fresh against `1280b8ad`, the verdict is passing, no `[AGENT-DRAFT]` tag remains, and the promotion answers point at files that exist (`docs/adr/adr-055-…`, `docs/lessons/lesson-546-…`).

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed** by this verdict — an edit to any of them would stale this review. These are for the implementer to disposition in `verification.md` (applied / ticketed / declined with reason) or a follow-up ticket, and all of them are outside the contract set:

1. **docs** — one clause in ADR-055 decision 7 or 8 naming where the `--json` record is owed and where it is not: the store-open failure now writes one, a usage refusal does not, and an undefined flag exits 1 rather than 2. Writing that sentence closes findings 1 and 2 without touching code.
2. **tests (optional)** — a subtest for `openFor`'s `errors.Join` branch (record write fails → exit code stays the open error's), so finding 3 stops being UNTESTED.
3. **code, already tracked** — #3105 (concurrent `up` TOCTOU: claim the name atomically, e.g. an exclusive `Mkdir`) and #3106 (Windows refusal precedence; split `WindowsSessions.Exec`'s settle/timeout block as `launch`/`collect` already are). Neither gates this archive; both stay open with their root cause recorded.
4. **human** — archive, then close #2842 so the board reflects the landed work.

### Definition of Done (this review session)

- **Debt**: all findings recorded above; none left in conversation only. The two carried code Minors have live tickets, verified OPEN this session.
- **Knowledge**: a review pass owes no new lesson; the durable knowledge this change produced (ADR-055, lesson 546) is already in the tree and was not amended by the fix.
- **Board**: skipped by mandate — this run's instructions forbid issue and PR actions; the archive and #2842's closure are the implementer's step, named above.
- **Review**: N/A — no open PR in this window; the spec's work landed through merged PRs (#2919…#3107), each of which carried its own review.
- **Evidence**: fresh this session — full suite `go -C cli test ./... -count=1` exit 0 (52 pkgs ok), 11 features.json commands re-run with per-command RUN/PASS counts, mutation of `openFor` red→green, real-binary reproduction on seven verb/OS combinations, `go vet` clean, `golangci-lint` 0 issues, `-race` on `internal/vm` and `internal/cmd` ok, digests via `dotf spec digest`, `gh issue view` for #3105/#3106/#2842. Working tree left clean apart from `specs/PLAT-005-vm-session-verbs/review.md` (the mutation revert restored `vm_session.go`; `git status` shows no source-file changes).
