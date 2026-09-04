---
spec: "CLI-072-dotf-hooks-install"
verdict: "PASS"
reviewed_sha: "eaf8a91fae83ed0e5343e510eb15f9f7b4dd8015"
reviewer: "nan/mimo-v2.5"
date: "2026-09-03"
---

## Adversarial review

**Scope**: CLI-072-dotf-hooks-install (PR #1464, merged as `c375102`)
**Sources**: `specs/CLI-072-dotf-hooks-install/{proposal,tasks,verification,features}.md`, diff `fcef14a..c375102`, source `cli/internal/hooks/{hooks,hooks_test}.go`, `cli/internal/cmd/{hooks,hooks_test}.go`, `cli/internal/doctor/checks_guard.go`, `setup-{linux.sh,windows.ps1}`, `scripts/check-twin-test-retirement.sh`, `tests/{check-twin-test-retirement,guard-setup-hooks-order}.bats`

### Spec and task alignment

All 8 acceptance criteria (AC1–AC8) are addressed in the implementation and have named test coverage. The one open task (`[ ] PR opened with Refs #1460`) is correctly left unchecked — the spec-gate refuses a closing keyword on a non-archiving PR. The deletion of `scripts/install-git-hooks.{sh,ps1}` and `tests/install-git-hooks.{bats,Tests.ps1}` is confirmed: `ls` returns exit 2. The prose sweep updated `checks_guard.go`'s FAIL remedy and `checks_tools.go`'s comment; remaining references to the deleted scripts are in comments recording what was replaced (lesson 259).

The `features.json` entries all have `state: "pending"` with empty `evidence` fields. This is a pre-archive state and does not block review, but the archive command may expect them populated — flagged as a Minor finding.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location |
|----------|---------|------|---------|----------|---------------------------|-------------|
| Minor | THEORETICAL | copyTree | `copyTree` strips CR from all files in the dispatcher tree, including `lib/*.sh`. Current dispatchers are text-only, so this is harmless today. If a binary file is ever added to `git-hooks/`, the NUL-byte check in `isText` would protect it — but only because that helper exists. The comment explaining this is inside `copyTree` and does not reference the dispatcher contract, so a future author adding a non-text file might not notice the implicit assumption. | Code read of `copyTree` and `isText` in `hooks.go` | `TestInstallNormalisesCRLF` (covers the happy path; no test for binary-in-dispatcher) | code (add a comment in `dispatcherTree` test helper noting the assumption, or add a test with a binary file) |
| Minor | THEORETICAL | partial-failure | `deployHooks` + `wireHooksPath` is not transactional: if deployment succeeds but wiring fails (e.g., git config write permission denied), the dispatcher sits deployed but `core.hooksPath` is not updated. On re-run the deploy is idempotent and the wiring retries. The error message propagates correctly. Not a blocker because the shell twins had the same shape and re-run recovers, but a future caller that swallows the error would leave the guard inactive silently. | Code read of `install()` in `hooks.go` | `TestInstallReportsAWiringFailure` (covers wiring failure; no test for deploy-then-wire-partial state) | code (document the non-transactional contract in the `Install` godoc) |
| Minor | THEORETICAL | cross-platform coverage | The `C:\` drive-root test case in `TestInstallRefusesUnsafeDestinations` is skipped on Linux (`windowsOnly: true`). The Pester suite also only ran on Windows for this case. The Go code's `filepath.Dir(clean) == clean` check is inherently cross-platform, but the test proves it only on the platform where the risk is highest. A Linux-side assertion that `filepath.Clean("C:\\")` evaluates to a root-equivalent would be cheap and close the gap. | Code read of `TestInstallRefusesUnsafeDestinations` | `TestInstallRefusesUnsafeDestinations` (the `C:\` subtest is skipped on Linux) | tests |
| Minor | SPECULATIVE | spec artifacts | `features.json` has all 11 entries at `state: "pending"` with empty `evidence`. The task `[x] features.json entries non-vacuous` is checked, and the verification commands in `features.json` are well-formed (`go test -run ... \| grep -q '^--- PASS:'`). The state was likely intended to be updated by `dotf spec archive` or a follow-up. If `dotf spec archive` reads this state, it may reject the archive. | File read of `features.json` | N/A | spec artifacts (update `state` to `"pass"` and populate `evidence` before archive) |
| Question | THEORETICAL | hooksPath error masking | `wireHooksPath` treats a non-zero exit from `git config --global --get core.hooksPath` as "unset" rather than "git is broken." This is correct for the normal case (git exits 1 on an unset key) but would mask a genuinely broken git installation. The integration test (`TestInstallAgainstRealGit`) proves the fake matches the real binary's dialect for the unset case. A broken-git case is outside this spec's scope but worth naming. | Code read of `wireHooksPath` | `TestInstallAgainstRealGit` (covers normal dialect; no test for git-binary-absent) | — (surface only; no action required for this spec) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All 8 acceptance criteria verified with named tests; negative paths (unsafe dests, missing source, self-mirror, first install, wiring failure) all covered; mutation testing documented for 15 guards |
| Verification       | B | Evidence is reproducible with `go test -v` and `bats`, but `features.json` entries are still `pending` and `verification.md` does not name a test for AC7 (metric delta) |
| Scope              | A | Diff matches proposal exactly: hooks package, command wiring, setup repointing, doctor remedy update, twin deletion, prose sweep, two new guard scripts. No creep into unrelated areas |
| Reliability        | A | Error paths handled, idempotent, partial-failure is recoverable, `os.SameFile` for path equivalence, clean-mirror for stale hook pruning, CR-stripping for CRLF taint |
| Maintainability    | A | Clear naming, all functions <40 lines, `gitRunner` seam is idiomatic, comments explain WHY not WHAT, no dead code, cyclomatic complexity well under 10 |
| Handoff-readiness  | B | Spec files complete, doctor remedy updated, guard script enforces ADR-020 §5; lesson capture deferred (noted as "not on its own" in verification.md); ADR-020 §5 amendment landed in the PR |

### Verdict
PASS

### Recommended next steps (before archive)
- Update `features.json` entries: set `state: "pass"` and populate the `evidence` field with the test output for each feature. `dotf spec archive` may reject `pending` states.
- No Blocker or Major findings — archive is advisable once `features.json` is updated.
