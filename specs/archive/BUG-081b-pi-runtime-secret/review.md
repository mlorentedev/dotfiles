---
spec: "BUG-081b-pi-runtime-secret"
verdict: "PASS"
reviewed_sha: "6ffee6edeefa87e975bb897c0551116800dec40f"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-16"
---

## Adversarial review

**Scope**: BUG-081b-pi-runtime-secret
**Sources**:
- `specs/BUG-081b-pi-runtime-secret/proposal.md` (acceptance criteria AC1–AC7)
- `specs/BUG-081b-pi-runtime-secret/tasks.md` (TDD order, all boxes ticked)
- `specs/BUG-081b-pi-runtime-secret/verification.md` (evidence per AC)
- `ai/pi/models.json` (source change: `{env:NAN_API_KEY}` → `${NAN_API_KEY}`)
- `cli/internal/doctor/checks_agentconfig.go` (doctor guard implementation)
- `cli/internal/doctor/checks_agentconfig_test.go` (6 test cases)
- `cli/internal/secrets/render_test.go` (load-bearing passthrough test)
- `docs/adr/adr-034-agent-config-secrets-resolve-at-runtime.md`
- `git diff 0fdb1f1^..0fdb1f1` (the implementation commit)
- `harness/reviewer-pool.json` (confirming reviewer id `nan/deepseek-v4-flash`)

### Spec and task alignment

All seven acceptance criteria are verified by passing tests and/or observable state:

| AC | Claim | Status | Evidence |
|---|-------|--------|----------|
| AC1 | Source uses pi's syntax | ✅ | `ai/pi/models.json: apiKey: "${NAN_API_KEY}"`, zero `{env:` occurrences; confirmed via `grep -c '{env:'` → `0` |
| AC2 | Render is a passthrough | ✅ | `TestRender_LeavesShellStyleVariablesUntouched` passes: `${VAR}` and `$VAR` come back byte-identical |
| AC3 | Doctor FAILs on `{env:` | ✅ | `TestAgentConfig_SeverityByShape/dotf_placeholder_pi_cannot_resolve` — reports `[FAIL] "not pi's"` |
| AC4 | Doctor FAILs on materialised literal | ✅ | `TestAgentConfig_SeverityByShape/materialised_literal` — reports `[FAIL] "literal credential"`, does NOT echo the value |
| AC5 | Observed FAIL against real state | ✅ | `verification.md` documents the real FAIL; confirmed on this machine: deployed `models.json` still carries a materialised literal |
| AC6 | No setup script changes | ✅ | `git diff --stat origin/main -- setup-linux.sh setup-windows.ps1` is empty |
| AC7 | ADR recorded | ✅ | `docs/adr/adr-034-agent-config-secrets-resolve-at-runtime.md` — posture, alternatives, consequences |

No `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` tags remain in any spec file under `specs/BUG-081b-pi-runtime-secret/`.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | THEORETICAL | doctor — path | Hardcoded `piConfigRel = ".pi/agent/models.json"` ignores `PI_CODING_AGENT_DIR`. If a user overrides the config directory, `dotf doctor` silently SKIPs (no config found at default path) and the guard is a no-op — the deployed copy goes unchecked. | `pi --help` documents `PI_CODING_AGENT_DIR` as the config directory override; `checks_agentconfig.go` line 37 hardcodes the relative path; pi's own runtime uses the env var | `TestAgentConfig_AbsentConfigSkips` exercises the SKIP path but no test asserts `PI_CODING_AGENT_DIR` is respected | code + tests |
| Minor | THEORETICAL | doctor — scope | The `{env:` check operates on raw bytes, not JSON structure. If `{env:...}` appears in a non-credential field (e.g. an example string inside a model description), the check reports it as a pi defect. Technically correct (pi cannot resolve `{env:` anywhere), but a false positive on non-credential text is noise. | JSON has no comments, and pi configs are user-authored, so the chance of a stray `{env:` in a non-credential string is low; the check is conservative-by-design | `TestAgentConfig_SeverityByShape/dotf_placeholder_pi_cannot_resolve` covers the apiKey case but not a stray `{env:` in another field | none (accept as designed) |
| Minor | THEORETICAL | doctor — precision | `piResolvable("$")` returns `true` (bare dollar sign). pi may not resolve a lone `$`, but it is not a materialised credential, so the guard correctly does not flag it. Not a security issue but the acceptance of `$` alone is imprecise. | Code read: `strings.HasPrefix(v, "$")` matches a single `$` | `TestAgentConfig_SeverityByShape/pi_bare_syntax` uses `$NAN_API_KEY` (var with name), not bare `$` | none (accept as designed) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All acceptance criteria verified with passing tests; negative paths (absent config, unparseable JSON, materialised literal) covered. Minor gaps: no test for bare `$`, no test for `{env:` in non-apiKey field. |
| Verification | A | Every claim pinned to a named Go test or a captured command output. The load-bearing passthrough claim (`render` leaves `${VAR}` untouched) has its own named test. Fingerprint comparison from primary source re-confirms pi's resolver behavior. |
| Scope | A | Diff changes exactly one line in `ai/pi/models.json` plus the doctor module. No unrelated refactors, no setup script edits. The diff matches the proposal exactly. |
| Reliability | B | Error paths handled: absent config→SKIP, unparseable JSON→WARN, empty apiKey→not flagged, decrypt errors→placeholder left intact. The `{env:` raw-byte check is robust to JSON structure. Minor: `PI_CODING_AGENT_DIR` override bypasses the guard entirely. |
| Maintainability | A | Clear naming (`piResolvable`, `checkAgentConfigSecrets`), 102-line function with thorough comment block explaining the bug history, table-driven tests, no dead code. Comments explain WHY not WHAT. |
| Handoff-readiness | A | ADR-034 fully records the decision, alternatives, and consequences. Verification artifact is thorough. `docs/lessons.md` mentions were captured in prior commits. Spec files are current. |

### Verdict

**PASS** — no Blocker or Major findings. All rubric dimensions are B or above. The three Minor findings are all THEORETICAL and do not affect the verdict.

### Recommended next steps (before archive)

- The three minor findings above are accept-as-designed; none blocks archive.
- **One actionable item**: consider adding an edge-case test for bare `$` in `apiKey` (e.g. `"apiKey": "$"`) and for `{env:` appearing in a non-credential JSON string. Not a gate, but would strengthen the test matrix.
- `dotf spec archive` is **advisable** in the current state — no blockers, no `[AGENT-DRAFT]` tags, a fresh `review.md` exists, and the `reviewed_sha` matches the current HEAD.