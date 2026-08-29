---
spec: "AI-038-copilot-npm-channel"
verdict: "PASS"
reviewed_sha: "389aaf50f3d0246b56b1e27871094694dbf44701"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-28"
---

## Adversarial review

**Scope**: AI-038-copilot-npm-channel (commit `f18b69a`, #1359; reviewed at HEAD `389aaf5`)
**Sources**: `specs/AI-038-copilot-npm-channel/{proposal,tasks,verification}.md`, `features.json`; diff/`git show f18b69a`; doctor + tools sources; bats suite.

### Spec and task alignment
- AC1 (npm catalog tool, no setup install block) — met. `packages.json` declares `copilot` (`npm`, `@github/copilot`, 1.0.81); the `Id = "GitHub.Copilot"` winget row is gone from `setup-windows.ps1`; `setup-linux.sh` detect-and-act re-worded to name the catalog. Confirmed by bats: `copilot-config.bats` → @test "copilot is a packages.json catalog tool (npm) and neither setup carries an install block (AI-038, ADR-036)", `setup-windows.bats` → @test "…no longer installs GitHub.Copilot via winget…".
- AC2 (dropped) — `ai/copilot/config.json` carries no `autoUpdate` key; the CLI rewrites that file (measured), so the setting moves to `settings.json` (#1322). Consistent across spec/tasks/verification.
- AC3 (doctor row per status) — met. `checkCopilot` wired after `checkOpenCode` (`doctor.go`); `TestCheckCopilot_PinMatchByStatus` exercises 5 rows: at-pin PASS, above WARN drift, below WARN drift, no-semver WARN (no drift line), absent SKIP. Passes (re-run this session).
- AC4 (ADR-036 amendment) — met; `copilot` moved to the npm class row with a dated "Amendment 2026-08-28 (AI-038, #1321)" section.
- AC5 (box) — plausible and reproducible: winget copy satisfies the floor so install no-ops first (ADR-036 §5), after uninstall the npm copy installs, second run idempotent, `dotf tools version copilot` = 1.0.81, doctor section all ok, `copilot -p` answers. I independently re-ran the Go tests + the relevant bats cases; only `setup-linux.bats` zsh syntax cases fail and that is the disclosed missing-`zsh`-on-this-box env gap (status 127), carried by CI — not a regression.

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | scope/handoff | The duplicate-tool-name rejection added to `tools.Load` (`catalog.go`, ~12 lines + test) is bundled silently into the AI-038 commit but appears in none of proposal.md, verification.md, or the commit message, and carries no issue number — its rationale lives only in code comments + test names. It is a legitimate found-on-the-way fix to a real incident (a duplicated copilot entry shipped in the PR's own first push and misread as "already installed; skipping"), and it is tested, but the archive artifact does not record it. | code diff of `f18b69a` → `cli/internal/tools/catalog.go`; comment: duplicate "read as already installed; skipping" | `TestLoad_RejectsDuplicateToolNames`; `packages-json.bats` @test "packages.json tool names are unique" | spec artifacts (one line in verification.md), or accept as code-documented found-on-the-way |
| Minor | THEORETICAL | correctness | The new "GitHub Copilot CLI" section does not disambiguate a box that resolves AWS Copilot CLI (`Amazon.CopilotCLI`, binary also named `copilot`): `copilot --version` prints "AWS Copilot CLI version: v1.x.y", `semverOf` returns `1.x.y` (≠ "unknown"), so `checkCopilot` emits `rep.Pass("copilot in PATH: 1.x.y")` under a GitHub-heading followed by a drift WARN against the `@github/copilot` pin. It does not yield a false clean PASS (the WARN fires) and the setup scripts already declare this <1% collision out-of-scope, but the new doctor path inherits it undocumented. | code read of `checks_copilot.go` + `setup-windows.ps1` comment ("AWS Copilot CLI … also exposes itself as `copilot`. Out-of-scope… <1%") | UNTESTED | tests (banner-assert: section claims "GitHub Copilot CLI" only when the version banner contains it), or declare out-of-scope in the spec |
| Minor | REAL | spec artifact | AC2 is marked `[x]` (a completed checkbox) in proposal.md while the criterion text says "(dropped, moved to #1322)". A checked box on a non-implemented criterion reads to the gate and to readers as satisfied; the drop is real and consistent across tasks.md/features.json, but the `[x]` is a misleading marker. | proposal.md Acceptance criteria, AC2 line | n/a (prose artifact) | spec artifacts (mark `[~]`/`[ ]` with explicit drop + #1322 link) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | All AC met; negative paths (no-semver, absent, above/below drift) named-tested; two minor un-tested edges (AWS banner, AC2 marking) |
| Verification       | B | Evidence reproducible by name and re-run this session (Go + bats); box AC5 is prose but commands named |
| Scope              | B | Diff matches proposal; small related duplicate-loader hardening bundled, under-documented in spec artifacts (Minor 1) |
| Reliability        | B | Install idempotent (skip at/above pin, never downgrade), failure leaves prior state, doctor WARNs (not fails) on drift/absent/no-semver |
| Maintainability    | B | checkCopilot 25 lines, clean GoDoc, mirrors opencode, table-driven test, CC low; loader rationale lives only in comments |
| Handoff-readiness  | B | Full spec triad + ADR amendment present; loader hardening not surfaced in verification (Minor 1) |

### Verdict
PASS

### Recommended next steps (before archive)
- No action blocked: the change is small, correct, well-tested, and within the proposal (plus one related, tested loader hardening).
- Optional before archiving, to make the record airtight (all Minor): (1) add one line to `verification.md` noting the duplicate-tool-name rejection in `tools.Load` and its trigger; (2) reword AC2's `[x]` to a dropped marker with the #1322 link; (3) either add an AWS-Copilot banner test case to `TestCheckCopilot_PinMatchByStatus` or name the collision out-of-scope in the proposal.
- Commit `review.md` (reviewed_sha `389aaf50f3d0246b56b1e27871094694dbf44701`) so the archive gate sees it; then `dotf spec archive AI-038-copilot-npm-channel` is advisblable. When archiving, close bitácora #1321 with the PR link, per the archive checklist.
- `setup-linux.bats`'s two `zsh` failures are the pre-existing missing-`zsh` env gap (CI carries zsh), not a regression; no action for this change.
