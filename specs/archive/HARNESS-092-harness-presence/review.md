---
spec: "HARNESS-092-harness-presence"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "c4c43d61522f6124ddbd4451d0870addae13bdfb"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-29"
---

## Adversarial review

**Scope**: HARNESS-092-harness-presence (merged as `origin/main` @ `c4c43d6`, PR #1368)
**Sources**: `specs/HARNESS-092-harness-presence/{proposal,tasks,verification,features}.json`,
`cli/internal/harness/presence.go`, `cli/internal/harness/persona.go`,
`cli/internal/cmd/harness_presence.go`, `cli/internal/doctor/checks_presence.go`,
`scripts/compile-harness.sh`, `setup-windows.ps1`, `tests/compile-harness.bats`,
`tests/setup-windows.bats`.

### Spec and task alignment

- The four acceptance criteria (AC1–AC4) and the box criterion (AC5) map cleanly to
  `features.json` rows f1–f5, each with a named verification command and `state: verified`.
- Implementation matches the proposal's "What" precisely. The one *real* misalignment is a
  **tasks.md** bullet, not the code: see F1 below.
- The whole-name `targets:` match vs the shell's substring match is declared as an intentional
  behaviour difference in the proposal and the Go `Persona.AppliesTo` implements whole-name
  (`EqualFold`) — recorded, not preserved. No spec/code mismatch.
- **Independent re-verification this session**: `go test ./internal/{harness,cmd,doctor} -run
  'Presence|Marker'` and the full doctor/cmd/harness suites all pass; `bash -n
  scripts/compile-harness.sh` passes; no `[AGENT-DRAFT]`/`[AGENT-SUGGESTION]` tags remain.
  Most importantly, I re-rendered the roster directly from `harness/agents` via the real
  `harness.RenderPresence` and got `SHA=5e0b469a4de5feb6` — **byte-identical to the deployed
  begin marker** in the instructions files, so the box is current and doctor would report green.
  That independently confirms the core AC5 invariant (fresh region) without trusting the author's
  own log.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | spec (tasks.md) | AC3 task bullet says "absent/old dotf → ERROR, `--deploy` fails", the opposite of the accepted contract. The proposal ("loud WARN … the deploy continues") and the implementation (`deploy_agent_presence` does `return 0` on a dotf absent/predating the verb) both say WARN + continue, and a failing *running* verb is the only `--deploy` failure. A future reader of tasks.md is misled about the contract. | tasks.md "absent/old dotf → ERROR, `--deploy` fails" vs proposal.md "loud WARN … the deploy continues"; `scripts/compile-harness.sh` `deploy_agent_presence` `return 0` | `tests/compile-harness.bats` "a dotf that predates harness presence warns that presence was NOT deployed, and the deploy continues (HARNESS-092)" asserts `status 0` | spec (tasks.md) |
| Minor | THEORETICAL | code (InjectPresence) | "Byte-identical outside the markers" holds only for uniform line endings. A file containing **any** CRLF is rejoined entirely as CRLF (`bytes.Contains(raw, "\r\n") → nl="\r\n"`), so a mixed-ending file's non-region content is rewritten, and the file is force-written mode `0o644` (the old shell's `mktemp`+`mv` left it `0o600`). Target files are deployed uniformly so it rarely fires. | `presence.go` `InjectPresence`: `bytes.Contains`, `strings.Join(out, nl)`, `os.WriteFile(path, data, 0o644)` | UNTESTED (TestInjectPresence_HonoursCRLF covers uniform CRLF only; no mixed-ending or permission-preservation case) | code + tests (or document explicitly) |
| Minor | THEORETICAL | code (InjectPresence) | A file carrying a BEGIN marker but **no END** (truncated/corrupt region) falls through to the append branch, producing a second region; it self-heals only on the next run. Low likelihood (only writer is the tool). | `presence.go` scan: `case begin >= 0 && end > begin` is false when `end == -1` → `default` append | UNTESTED (no malformed-open-region test) | code + tests (add regression) |
| Minor | THEORETICAL | code (doctor/PresenceStatus) | "Current" answers "does the begin sha equal today's roster", not "does the region's own content match that sha". A region whose prose is hand-edited while its begin sha still equals the current roster reads **PASS** until the next deploy rewrites it. The marker forbids editing, so low severity. | `presence.go` `PresenceStatus`: only compares the begin line's `(sha256:…)` against `PresenceSHA(block)` | UNTESTED (TestInjectPresence_ReplacesAStaleRegionInPlace covers a *different* sha, not corrupted-same-sha content) | code + tests (or document as accepted) |

Questions / observations (not defects):
- **copilot gate asymmetry**: `DeployPresence` injects into the copilot file even when
  `requires_command` is absent; only `checkAgentPresence` gates on the binary. Harmless (the file
  is only a surface when copilot exists) but the deploy/doctor asymmetry is a deliberate-by-omission
  detail worth a comment.
- **Disclosed bats gap, environmental**: verification.md states `tests/compile-harness.bats`
  ran 31/32 on this box, the one failure ("an absent dotf warns … does not fail the deploy")
  being the reduced-PATH `jq`-under-WinGet artifact that also fails on `origin/main`; Linux CI
  (`/usr/bin/jq`) is the authoritative run. Disclosed honestly; not a regression finding. The full
  bats suite was not executed green in this session (slow on this box); the Go suites were.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | A | All ACs verified, negative paths + mutations covered, no observed defects; deployed sha independently confirmed fresh. |
| Verification       | A | Reproducible commands/outputs in verification.md; Go suites re-run green this session; box log corroborated. |
| Scope              | A | Diff matches proposal; second commit (marker-constant fix) is an in-scope rebase repair, not creep. |
| Reliability        | B | Error paths handled (absent, broken record, CRLF, verb-fail fails deploy); minor gaps: mixed-ending rewrite, malformed-region self-heal-later. |
| Maintainability    | A | Clear naming, ≤40-line functions, low cyclomatic complexity, WHY comments, single-marker SSOT + pinning test. |
| Handoff-readiness  | B | Spec complete with decisions + promotion analysis and archive checklist; tasks.md AC3 wording contradicts the accepted contract (F1). |

### Verdict
PASS WITH GAPS — no blockers, no majors; one REAL Minor spec-artifact discrepancy (F1) and three
THEORETICAL Minors (F2–F4). Rubric is all B or above (no C/D).

### Recommended next steps (before archive)
- **Fix F1 (required for spec hygiene)**: correct the tasks.md AC3 bullet to state the accepted
  contract — "absent/old dotf → loud WARN, `presence NOT deployed`, deploy continues; a failing
  verb fails `--deploy`". Updating tasks.md will bump its sha, so re-run this review's
  `reviewed_sha` window (`dotf spec archive` refuses a stale review if tasks.md changes after it).
- Optional, non-blocking: add a mixed-ending test, a BEGIN-without-END test, and a
  corrupted-same-sha doctor test (F2–F4), or add one-line comments declaring those behaviours
  accepted.
- `dotf spec archive HARNESS-092-harness-presence` is otherwise **advisable**: implementation is
  correct, tests green, the box is verified fresh (`5e0b469a4de5feb6` matches), and no Blocker/Major
  remains.
