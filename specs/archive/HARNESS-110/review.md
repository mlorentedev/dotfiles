---
spec: "HARNESS-110"
verdict: "PASS"
reviewed_sha: "ca572f9779b56e275c12142cec039cc734b921b0"
reviewer: "nan/deepseek-v4-flash"
date: "2026-09-02"
---

## Adversarial review

**Scope**: HARNESS-110 (harness-role-join)
**Sources**: `specs/HARNESS-110/{proposal,tasks,verification}.md` + `features.json`; diff `745f320..HEAD` (3 commits: `7785c45`, `da2e0e9`, `ca572f9`).

### Spec and task alignment

- All 8 acceptance criteria map cleanly to the diff. AC1–AC3 live in `roles.go` (`ResolveRoles`, `FormatSuggestion`, `entrySkill`), AC6 in `hookpayload.go` (`PromptFromHookPayload`), AC5/AC7 in `harness_suggest_hook.go` (`runSuggestFromHook`, `harnessRoot`), the command wiring in `harness.go` (`--from-hook`, `--repo-root`), and AC4 in `harness/manifest.json` (`agents.bind[claude].emit_hooks[suggest-role]`). No orphan file, no scope creep.
- `tasks.md` implementation boxes are all ticked except "PR opened" (`[ ]`) — expected, since this is the pre-archive gate and the PR is still open.
- `features.json` has 8 entries, each with a non-vacuous `go test -run <named-test>` command; all `state: "pending"` (correct — only the harness sets `passing`).
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file.

### What I actually ran (claims, not assertions)

- `go build ./...` — exit 0.
- `go vet ./...` and `GOOS=windows go vet ./...` — both exit 0.
- `golangci-lint run ./...` (pinned 2.12.2 the repo mandates) — **0 issues**.
- All 8 feature tests pass on `-count=1`: `TestResolveRoles`, `TestResolveRolesAmbiguity`, `TestRoleJoinDrift`, `TestRoleJoinLatencyBudget`, `TestManifestEmitsPromptHook`, `TestFormatSuggestion`, `TestPromptFromHookPayload*`, `TestSuggestFromHook*`.
- **Mutation A** (force `ResolveRoles` to return empty → simulated collapsed join): `TestRoleJoinDrift` goes red (`resolving rules = 0, want at least 16 of 18`). The AC3 floor guard is **non-vacuous**. Reverted.
- **Mutation B** (make a `LoadTriggers` failure propagate instead of fail-open): `TestSuggestFromHookNeverExitsNonZero` red on `triggers.json_unparseable` and `persona_record_unreadable`. AC7's fail-open is **asserted, not inspected**. Reverted.
- **Mutation C** (delete the `p.Kind != "invocable"` filter): all tests still green. (See finding 2.)
- **AC4 consequence**: ran `dotf harness bind --harness claude` against a temp HOME → emitted `.claude/settings.json` contains `"UserPromptSubmit": [{"command": ".../dotf harness suggest --from-hook", "timeout": 5, "type": "command"}]`. Hook is propagated by the manifest deploy path.
- **AC6 field de-risk**: fetched Anthropic hooks docs — the `UserPromptSubmit` payload field holding the prompt is **`prompt`**, and exit code 2 **"blocks prompt processing and erases the prompt"**. Both the primary candidate spelling and the severity of the fail-open guard are confirmed against the source of truth.
- **AC8 real-path latency**: measured the actual per-prompt path (incl. `LoadTriggers` + `LoadPersonas` disk/YAML) over 30 runs: **avg 7.13ms / max 9.14ms** < 20ms budget.
- **Probe**: 16/18 rules resolve; `code-complexity-and-refactor → [builder, reviewer]`, `spec-driven-development → [planner, reviewer]`, and `shell-standards` / `powershell-ascii-only` (no skills) → empty/nobody. Matches the proposal's measurements. Reverted probe.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | tests (AC7) | The `persona_record_unreadable` test row never reaches the unreadable-persona branch: it shares `brokenRoot` with `triggers.json_unparseable`, so `LoadTriggers` fails first. | Reproduced — the row's stderr is `no triggers … parse triggers JSON`, never `no personas`. Built a **valid-triggers/broken-persona** root separately: `no personas … frontmatter is not valid YAML`, exit 0 (the fail-open behavior is correct; only the named row's coverage is fake). | `persona_record_unreadable` **UNTESTED** (does not exercise its branch) | tests — give that row its own root with a valid `triggers.json` |
| Minor | THEORETICAL | persona (AC1) | The `kind: "invocable"` filter is load-bearing only for a future case; no current test goes red if it is removed. | Mutation C: removed the filter → all `TestResolveRoles*`/`TestRoleJoinDrift` green. `hermes-nan`'s only skill `agent-lifecycle` is absent from `triggers.json`, so it never matches today. The code comment admits this ("precisely why this filter goes in now"). | `TestResolveRoles` **UNTESTED** for the invocable restriction | tests — fixture where a non-invocable persona's skill is on a rule, assert exclusion |
| Minor | THEORETICAL | verification (AC4) | AC4's stated standard ("verify by consequence: deploy, start a session, observe — not by asserting the file contains a string") is not met by an automated test; only the manifest *declaration* is asserted. | I verified the consequence manually (bind emits the hook with `--from-hook`, timeout 5), so the property holds. But the committed test is exactly the "file contains a string" form the spec says is insufficient, and `verification.md`'s AC4 row cites that test. | `TestManifestEmitsPromptHook` **UNTESTED** for emission consequence | tests — extend the bind-consequence tests to assert the emitted `settings.json` holds `suggest-role` on `UserPromptSubmit` |
| Minor | THEORETICAL | performance (AC8) | The latency test measures only the in-memory match+join (`Suggest`+`ResolveRoles` loaded once outside the loop), excluding the per-prompt `LoadTriggers`+`LoadPersonas` disk/YAML work the hook also does. | Real path measured **avg 7.13ms / max 9.14ms** (< 20ms), so the budget holds with headroom. The test would not catch a regression in the load path. | `TestRoleJoinLatencyBudget` | tests (optional) — include one cold iteration; or spec — note the test measures match+join only |

**Strengths that directly mitigate documented risks** (not balanced praise):
- AC7 fail-open (never exit non-zero) is **verified non-vacuous** and directly neutralizes the spec's stated data-loss risk; the exit-2 "erases the prompt" semantics are confirmed against Anthropic docs.
- AC6's first candidate is the **actual** Claude Code field (`prompt`), confirmed against docs — directly de-risking the spec's named "#1 open question."
- AC4 emission-by-consequence is verified (bind → settings.json), and the suggester's `harnessRoot` resolution mirrors `loadGatePersona` exactly (DOTFILES_DIR → `$HOME/.dotfiles`), so the hook and gate agree on where personas live.
- AC8 holds on the real path (7ms), not merely on the trimmed test slice.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness | B | All 8 ACs functionally verified, no observed defect; minor negative-path coverage gaps (invocable filter untested; unreadable-persona test row masked), though behavior is correct |
| Verification | B | Every claim reproduced (build/vet/lint/tests/mutations/end-to-end); AC4's own "consequence" standard is not evidenced in-file — only the declaration is |
| Scope | A | Diff matches proposal exactly; no unrelated changes, no creep |
| Reliability | A | Fail-open on every path (mutation-verified), bounded 5s timeout, no writes/idempotent, root resolution identical to the gate |
| Maintainability | A | Clear naming, heavy WHY-comments, low cyclomatic complexity, golangci-lint 0 issues, functions bounded |
| Handoff-readiness | B | Decisions + corrections recorded in spec; promotion capture (lesson/ADR/pattern) and PR-open task still deferred to archive |

### Verdict
**PASS**

### Recommended next steps (before archive)
None of the findings is a Blocker or Major, and no rubric dimension is below B, so archive is not blocked. The two lowest-cost, highest-value hardening items before you archive:
1. Split the AC7 "persona record unreadable" fixture so that row actually reaches `LoadPersonas` (right now it is a silently vacuous case, which is exactly the class of fake coverage this spec exists to prevent elsewhere).
2. Add a regression test that asserts a non-invocable persona whose skill lands on a rule is *excluded* (AC1's restriction is currently untested).
3. (Cheap) Assert in a bind-consequence test that the emitted `.claude/settings.json` carries `suggest-role` — the spec's own stated standard for AC4. The latency scope question can be left as a one-line note; the real path is comfortably under budget.
