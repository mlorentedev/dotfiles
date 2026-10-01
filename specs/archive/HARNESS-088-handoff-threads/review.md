---
spec: "HARNESS-088-handoff-threads"
verdict: "PASS"
reviewed_sha: "500f59de82f8fb01c9bc3fa119c74babc3e2f37e"
reviewer: "nan/glm5.3-flash"
date: "2026-09-30"
---

## Adversarial review

**Scope**: HARNESS-088-handoff-threads — git diff 0df4bf8d59a012b226ec214ecbf3640e6ab47c7a...HEAD (round 3; this spec's surface is `cli/internal/mem/` handoff/thread/identity/session-end, `cli/internal/cmd/mem_handoff.go`, `cli/internal/filelock/`, the handoff skill, and the spec artifacts).
**Sources**: specs/HARNESS-088-handoff-threads/{proposal,tasks,verification}.md, features.json, review-round-1.md, review-round-2.md; code and tests in `cli/internal/{mem,cmd,filelock}`.

### Spec and task alignment
- AC1–AC7 map to named tests in `cli/internal/mem/handoff_test.go` / `thread_test.go`; AC9 to two tests in `cli/internal/cmd/mem_handoff_concurrent_test.go`; AC10 to `TestJournalWriterSkipsTheProjectWord`. All exist and pass.
- AC8 is declined with a ticket (#1881), not claimed — proposal, tasks and verification all record the same decline consistently.
- The round-1 REAL Major (lost update under concurrency) is fixed by `cli/internal/filelock` spanning read→compute→rename, with the lock key derived from the `EvalSymlinks`-canonicalized path. Both round-1 claims re-verified by mutation this round (below).
- The round-2 Major on cyclomatic complexity is fixed for this change: after `500f59de`, `gocyclo -over 9` reports no function in `internal/cmd/mem_handoff.go`; the `internal/cmd` ratchet (`.golangci.yml`, min-complexity 22) passes.
- The round-2 Major on `SessionEnd` hardcoding `claude` was declined with the claim that the `mem session-end` hook is bound only to Claude. Verified: `harness/manifest.json` registers `mem-session-end` only under `agents.bind[0]`, whose target is `claude` (`.claude/settings.json`). The decline stands.
- The diff range contains ~96k insertions across 1230 files — the branch base includes many other specs' landed and archived work, as round 2 also noted. This review scoped to the spec's own surface; the rest carries its own specs and reviews.
- No `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags in the spec artifacts. Contract files untouched since `2bba67e0`, before this review request.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | concurrency (Windows) | `filepath.EvalSymlinks` is assumed to resolve a Windows junction to the same canonical path as the vault path; if it does not, two Windows writers holding the two paths take different locks and the lost update returns on that platform. Disclosed in code and in `verification.md`. | Code read of `lockMemoryFileWithin` (`mem_handoff.go`); `TestTwoPathsToOneMemoryShareTheLock` skips on Windows with the reason stated. | `TestTwoPathsToOneMemoryShareTheLock` (Unix only; the Windows half is UNTESTED) | tests (measure on a real junction; extend or ticket) |
| Minor | THEORETICAL | concurrency | Lock directory is per-user (`XDG_RUNTIME_DIR`/`UserCacheDir`), so writers under different OS users on one vault do not share a lock. Round 2 declined this; one OS user owns a vault checkout and a shared dir would reintroduce a symlink-attack surface. Decline accepted. | Code read of `memoryLockDir()`. | UNTESTED | none (declined; record stands in verification.md) |
| Minor | SPECULATIVE | attribution | `journalWriter` skips only the journal name's first word, so a project with an agent word after its first segment (`my-pi-app`) would be misattributed and another agent's block forked. Disclosed in a code comment; no such project exists. | Code read of `journalWriter` (`handoff.go`). | `TestJournalWriterSkipsTheProjectWord` (covers the skip rule; the residual case is UNTESTED) | none (surface only; do not gate) |
| Question / assumption | THEORETICAL | quality | `SessionEnd` remains at CC 14, above the repo's 10 guideline. Round 2 declined it: unchanged by this spec and outside the `internal/cmd` gocyclo ratchet, whose config comment confirms other packages hold older outliers. Consistent, but it leaves a known outlier. | `.golangci.yml` exclusions scope; `gocyclo -over 10 ./internal/mem/session_end.go`. | UNTESTED | code (follow-up under the HARNESS-150 one-extraction-at-a-time ratchet) |

#### Verification performed this round (fresh commands, this session)
- `go build ./... && go vet ./...` — clean.
- `golangci-lint run ./internal/cmd/... ./internal/mem/... ./internal/filelock/...` — 0 issues.
- `go test ./internal/{mem,filelock,cmd}/ -count=1` — ok; `go test ./...` — every package ok (only `internal/review` has no test files).
- `go test ./internal/cmd/ -run 'TestConcurrentWritesDoNotLoseAThread|TestTwoPathsToOneMemoryShareTheLock' -race -count=3` — all PASS.
- **Mutation 1 (red)**: removed the `lockMemoryFile` acquisition from `handoffWrite.run` → `TestConcurrentWritesDoNotLoseAThread` FAILs ("thread feat-w7 is missing after 8 concurrent writes"). Restored → PASS. The AC9 regression test genuinely holds the lock.
- **Mutation 2 (red)**: removed the `filepath.EvalSymlinks` canonicalization → `TestTwoPathsToOneMemoryShareTheLock` FAILs. Restored → PASS. The symlink key-derivation is genuinely load-bearing.
- Every `features.json` verification command executed non-vacuously (exit 0, tests matched — the f5 failure mode this spec itself recorded does not recur).
- AC10: `TestJournalWriterSkipsTheProjectWord` — PASS.
- Working tree left clean apart from this review.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All ACs verified including negative paths; both concurrency claims proven by mutation red-green this round. |
| Verification       | A | Reproducible named-test commands per AC in verification.md; all re-ran green here, non-vacuously. |
| Scope              | A | The spec's own diff matches the proposal exactly; the branch base carries other specs' already-reviewed work. |
| Reliability        | B | Bounded lock wait, temp-file rename preserving file mode, loud errors on absent section/empty body; Windows junction path unmeasured. |
| Maintainability    | B | `newMemHandoffWriteCmd` split under CC 10; `writeThread` sits exactly at 10; `SessionEnd` outlier deliberately deferred. |
| Handoff-readiness  | A | Two review rounds retained, decisions and declines recorded with evidence, debts ticketed (#1881, #1882). |

### Verdict
PASS

### Recommended next steps
(The contract set is closed; these are for the implementer to disposition in `verification.md` — applied, ticketed, or declined with a reason — or to carry into a follow-up ticket.)

- Measure whether `EvalSymlinks` resolves a Windows junction to the vault path; either un-skip `TestTwoPathsToOneMemoryShareTheLock` on Windows with a junction fixture or record the limitation and the affected writers where the lock dir choice is documented.
- Carry the `SessionEnd` CC-14 extraction into the existing HARNESS-150 ratchet sequence rather than opening a new ticket.
- The `my-pi-app` attribution residual is documented in code; no action needed unless such a project name appears.

### Verdict completion
- (a) **Verdict: PASS** — no Blockers, no REAL Majors; the three open Majors from rounds 1–2 are fixed or validly declined; rubric all B or above.
- (b) `dotf spec archive` / `/spec archive` **is advisable** in the current state, subject to the archive gate's own checks (contract digests, review freshness).
- (c) N/A — no FAIL. The minimum follow-ups are the three recommendations above, none of which block archive.
