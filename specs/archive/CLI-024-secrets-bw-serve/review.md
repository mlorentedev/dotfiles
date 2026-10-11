---
spec: "CLI-024-secrets-bw-serve"
verdict: "PASS"
reviewed_sha: "eb7b86444a00b931080575b21abc1b30abd06223"
reviewer: "agy/gemini-3.1-pro-high"
date: "2026-10-10"
---

## Adversarial review

**Scope**: `CLI-024-secrets-bw-serve` (Round 2 review) — the read-path `bw serve` backend: `BWServeDaemon`, `BWServeClient`, `BWServeReader`, `dotf secrets unlock`/`lock`, backend selection (`SelectBWBackend`), doctor daemon check (`checkBWServeDaemon`), and the newly added per-secret cost and benchmark suite (`TestBWServeReader_ReadCostIsBoundedPerSecret`, `BenchmarkBWServeReader_Field`).
**Sources**: `specs/CLI-024-secrets-bw-serve/{proposal,tasks,verification,features}.md`; `specs/CLI-024-secrets-bw-serve/review-request.json`; `git diff 271e4ec9cb9fac0a426524e1e4f53df7427aeb0c...HEAD` (`git rev-parse HEAD` = `eb7b86444a00b931080575b21abc1b30abd06223`); `cli/internal/secrets/bwserve.go`, `bwserve_cost_test.go`, `bwserve_test.go`, `bwbackend.go`, `bwbackend_test.go`, `cli/internal/cmd/secrets_unlock.go`, `cli/internal/cmd/secrets_unlock_test.go`, `cli/internal/doctor/checks_bw_serve.go`, `cli/internal/doctor/checks_bw_serve_test.go`.

**Scope note:** As noted in Round 1, the launcher-supplied base commit `271e4ec9` is 719 commits behind `origin/main` because `CLI-024` was implemented in August 2026 (#975) and its spec is now being closed and archived. The review scope evaluated here covers the entire `CLI-024` surface and its subsequent fixes up to `HEAD` (`eb7b8644`), with specific focus on commit `ebb2458f` where Round 1 findings were addressed.

### Spec and task alignment

- **Round 1 Major 1 (missing benchmark test for AC2):** Resolved. Commit `ebb2458f` introduced `cli/internal/secrets/bwserve_cost_test.go` containing both `TestBWServeReader_ReadCostIsBoundedPerSecret` (which gates that 41 reads take exactly 82 requests, 0 forced syncs, and 0 backoff sleeps) and `BenchmarkBWServeReader_Field` (which measures per-read decode/HTTP latency at ~0.13 ms/op against a local fake). Verified fresh in this session: the gate passes, and a mutation injecting a forced sync turns it red (123 requests vs 82).
- **Round 1 Major 2 (`features.json` f2 contract honesty):** Resolved. `features.json` f2 was updated to reference the reproducible cost gate `TestBWServeReader_ReadCostIsBoundedPerSecret`, the benchmark, and the operator's live 1.361 s timing. Its verification command now executes the cost gate test and asserts exit 0.
- **Round 1 Major 3 (stale citations of removed `BWFallbackReader`):** Resolved. `tasks.md` and `verification.md` were updated to reflect the architectural convergence in #1611 (`SelectBWBackend`, pinning reader, writer, and syncer per command to prevent BUG-084 split-brain), citing current tests `TestSelectBWBackend_ReadAndWriteAlwaysAgree` and `TestSelectBWBackend_ProbesOnce`.
- **Round 1 Minor 1 (f4 evidence citing obsolete test names):** Resolved. `features.json` f4 now cites `TestCheckBWServeDaemon_States` and `TestCheckBWServeDaemon_StatusUnreadable`.
- **Round 1 Limit (`golangci-lint` clean claim):** Resolved. Fresh execution of `cd cli && golangci-lint run ./...` with clean cache returned `0 issues.` across all packages.
- **Acceptance criteria coverage:**
  - **AC1** (hidden prompt, daemon start, POST /unlock, no leaks): Fully verified. Passwords are typed via hidden prompt, zeroed with `scrubBytes`, scrubbed from error messages (`scrubPassword`), and never appear in stdout or logs (`TestSecretsUnlock_Succeeds_PasswordNeverInOutput`, `TestSecretsUnlock_WrongPassword_ErrorsWithoutLeakingIt`).
  - **AC2** (serve-backed reader under 2 s): Fully verified. Bounded cost gate passes (`TestBWServeReader_ReadCostIsBoundedPerSecret`), benchmark reports ~0.13 ms/op (`BenchmarkBWServeReader_Field`), and live operator timing recorded at 1.361 s for 41 entries.
  - **AC3** (CLI fallback with no daemon): Fully verified. `SelectBWBackend` falls back to `lockHintReader`/`lockHintWriter` with CLI shellout when daemon is locked, absent, or unauthenticated (`TestSelectBWBackend_ReadAndWriteAlwaysAgree`). All 36 packages in `cli/` pass unmodified (`go test ./... -count=1`).
  - **AC4** (doctor reports daemon state distinctly): Fully verified. `checkBWServeDaemon` distinguishes absent (never started vs pid dead vs alive but unanswering), locked, and unlocked, plus cache staleness (`TestCheckBWServeDaemon_States`, `_StatusUnreadable`, `_CacheAge`).
  - **AC5** (localhost bind only): Fully verified. Hardcoded `bwServeHostname = "127.0.0.1"` in `bwserve.go`. Mutation testing confirmed: changing to `0.0.0.0` immediately causes `TestBWServeCommand_BindsLocalhostOnly` to fail.
  - **AC6** (lock command, idempotent unlock): Fully verified. `dotf secrets lock` calls POST /lock (`TestSecretsLock`), and re-running unlock while already unlocked succeeds without reprompting or spawning a second daemon (`TestSecretsUnlock_Idempotent`).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | SPECULATIVE | security (pre-existing, bounded) | While the daemon is unlocked, any local process on `127.0.0.1:8087` can query Bitwarden CLI's unauthenticated REST endpoints. | Documented in `proposal.md` "Risks / open questions" §2; invariant strictly bounded to loopback by `TestBWServeCommand_BindsLocalhostOnly`. | `TestBWServeCommand_BindsLocalhostOnly` | — (surface only; pre-existing protocol property) |
| Minor | THEORETICAL | contract hygiene (cosmetic) | `proposal.md` lines 99–115 retain unchecked `- [ ] AC1:` through `- [ ] AC6:` markdown boxes, whereas `tasks.md` and `verification.md` confirm all ACs are satisfied. | `proposal.md` lines 99–115 vs `tasks.md` line 43. | UNTESTED (record hygiene) | — (do NOT edit `proposal.md` now; modifying contract set invalidates review; `dotf spec archive` rewrites status without requiring AC ticks) |
| Minor | REAL | operator boundary | Unattended live smoke verification cannot be reproduced in-agent because typing the Bitwarden master password is an operator-only action under ADR-028 secret safety doctrine. | `verification.md` "Live evidence" and ADR-028 doctrine. | `TestSecretsUnlock_Succeeds_PasswordNeverInOutput`, `TestBWServeReader_ReadCostIsBoundedPerSecret` | — (standing doctrinal boundary; verified by mock test battery and operator log) |

### Verified strengths (only where they mitigate a named risk)

- **Loopback isolation is mutation-proven:** `const bwServeHostname = "127.0.0.1"` in `bwserve.go` was mutated to `"0.0.0.0"`. `TestBWServeCommand_BindsLocalhostOnly` immediately failed (`expected --hostname 127.0.0.1 in args, got: "bw serve --hostname 0.0.0.0 --port 8087"`), proving the localhost bind guard is real.
- **Per-secret cost gate is mutation-proven:** In `cli/internal/secrets/bwserve_cost_test.go`, injecting a forced `r.Client.Sync()` per read turned `TestBWServeReader_ReadCostIsBoundedPerSecret` red (`41 reads made 123 requests, want 82`), proving the test actively prevents latency regressions.
- **Credential safety and zero-leak guarantees:** `TestSecretsUnlock_Succeeds_PasswordNeverInOutput` and `TestSecretsUnlock_WrongPassword_ErrorsWithoutLeakingIt` assert the password never leaks into stdout or errors. Passwords in memory are zeroed via `scrubBytes`.
- **Poisoning and split-brain defenses:** `probeReadable` uses `/list/object/folders` rather than `/status` to prevent BUG-082 HTTP 500 poisoning, and `SelectBWBackend` pins reader, writer, and syncer together to prevent BUG-084 split-brain.
- **Clean build and linter across platforms:** `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`, and `golangci-lint run ./...` all passed with 0 errors/issues. Full test suite across 36 packages passed (`go test ./... -count=1` exit 0).

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 6 acceptance criteria verified with automated tests, negative/leak cases covered, and cost/benchmark guards in place. |
| Verification       | A | Machine-verifiable tests and benchmarks back each feature in `features.json`; cost gate and bind invariants verified by mutation. |
| Scope              | A | Implementation strictly satisfies proposal; Round 1 fixes added targeted cost tests and updated spec records without scope creep. |
| Reliability        | A | Robust error paths: fallback on daemon lock/unreachability, backoff for sync windows (BUG-113), and backend pinning (BUG-084). |
| Maintainability    | A | Code is cleanly structured, <= 40 lines per function, gocyclo compliant, and doc comments comprehensively explain failure modes. |
| Handoff-readiness  | A | Round 1 review findings fully resolved; `features.json` hardened; promotion candidates answered; archive checklist ready. |

### Verdict
PASS

All findings from Round 1 have been fully resolved with committed code, benchmark tests, and synchronized spec records. No Blockers or open Major issues exist. Evaluator rubric scores are all A.

### Recommended next steps

1. **Archive the spec:** Run `dotf spec archive CLI-024-secrets-bw-serve` to transition the spec folder to `specs/archive/CLI-024-secrets-bw-serve/` and rewrite its status to `archived`.
2. **Close tracking issue:** Open a PR referencing the archived spec folder and close bitácora issue #622.
3. **Do NOT edit contract files:** Do not edit `proposal.md`, `tasks.md`, or `features.json` to tick cosmetic boxes, as doing so would invalidate this review's contract digests.
