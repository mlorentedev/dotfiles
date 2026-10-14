---
spec: "PI-002-windows-fixes"
verdict: "PASS-WITH-GAPS"
reviewed_sha: "ed901e6d1edcf9bd115e0402e6267f3cf6871492"
reviewer: "agy/gemini-3.8-flash-high"
date: "2026-10-14"
---

## Adversarial review

**Scope**: PI-002-windows-fixes — fix visible console window popup for `bw serve` on Windows, and align `deepseek-chat` model limits with the provider catalog to resolve `dotf doctor` drift warnings.
**Sources**: `specs/PI-002-windows-fixes/{proposal.md,tasks.md,verification.md,features.json,review-request.json}`; `git diff c8ebfec16b5359f76e9c490a0c47f0f2649dcd23...HEAD` (1,426 files, +72,801/-11,901; base commit `c8ebfec` predates multiple subsequent main merges; spec-owned sites inspected: `cli/internal/secrets/bwserve_windows.go`, `cli/internal/secrets/bwserve_windows_test.go`, `ai/pi/models.json`, and `specs/PI-002-windows-fixes/*`).

### Spec and task alignment

All three findings from Round 1 (2026-10-03, `FAIL`, `agy/gemini-3.1-pro-high`) have been addressed:

- **Round-1 Blocker (Doctor warned on deepseek limits) — RESOLVED.** `ai/pi/models.json` lines 50–51 set `contextWindow: 163840` and `maxTokens: 16384` for `deepseek/deepseek-chat`, matching OpenRouter's live API and the models.dev catalog. Verified on Linux (`msi`) and Windows 11 (`EGW-LEN029`) where `dotf doctor --verbose` reports `[Model limit drift] [ OK ] 1 models match the provider catalog`.
- **Round-1 Major (Spec named `CREATE_NO_WINDOW`, code rejected it) — RESOLVED.** `proposal.md` What #1 and `tasks.md` now explicitly describe `HideWindow: true` alongside `DETACHED_PROCESS` and reference `lesson-333` explaining why Windows ignores `CREATE_NO_WINDOW` when combined with `DETACHED_PROCESS`.
- **Round-1 Major (No Windows runtime evidence) — RESOLVED.** `verification.md` now records physical verification on corporate Windows 11 hardware (`EGW-LEN029`, 2026-10-13, dotf 0.69.0): `dotf secrets unlock` spawned `bw serve` through Scoop shim `C:\Users\mlorente\scoop\shims\bw.exe` with no console window (`hwnd=0` for shim PID 26400 and child PID 28160).
- **Static checks & tests re-run this session**: `go test ./internal/spec/...` in `cli/` exits 0 (all contract digest, provenance, and staleness tests pass); `go test -v ./internal/doctor -run TestModelLimits` exits 0 (10/10 tests pass); `go run ./cmd/dotf doctor --verbose` exits 0 (0 failed).
- **Tag scan**: Zero unresolved `[AGENT-DRAFT]` or `[AGENT-SUGGESTION]` markers in `specs/PI-002-windows-fixes/` (matches pre-flight `FindUnresolvedTags`).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Major | THEORETICAL | `verification.md` (Archive pre-flight) | `verification.md` lacks the `## Promotion candidates` section. The archive pre-flight `CheckPromotions` (`cli/internal/spec/archive.go:491`, HARNESS-160) unconditionally checks this section and will refuse `dotf spec archive` until each candidate is answered (`yes: <path>` or `no: <reason>`). Lessons 333 and 334 were already committed in `docs/lessons/` for this spec. | Code read of `cli/internal/spec/promotion.go:51` and `archive.go:491` | `TestPromotionsRefuseWhenSectionIsMissing` (`cli/internal/spec/promotion_test.go`) | spec (`verification.md` — outside the contract set: add before archiving) |
| Minor | REAL | `proposal.md` (Acceptance criteria) | AC3 (`- [ ] Next run of dotf doctor on Windows is clean.`) is unticked in `proposal.md`. The Windows run (2026-10-13, EGW-LEN029) showed `[Model limit drift] (1 checks, all ok)`, but the overall doctor run reported 1 failure in an unrelated section (`hive service status`, #2258) and 6 warnings, leaving the box unticked. | `specs/PI-002-windows-fixes/proposal.md:41` | UNTESTED | spec (contract set — tracked in `verification.md`; do NOT edit `proposal.md` now to prevent staling review) |
| Minor | REAL | `features.json` (Contract set) | `features.json` specifies `"verification": "echo 'verified in verification.md'"`, which is a vacuous verification command that performs no automated assertion. While Windows console detachment cannot be asserted on headless runners, unit tests for Go struct attributes and doctor limits exist. | `specs/PI-002-windows-fixes/features.json:5` | UNTESTED | spec (contract set — tracked in `verification.md`; do NOT edit `features.json` now to prevent staling review) |
| Minor | REAL | `cli/internal/secrets/bwserve_windows_test.go` | `TestBWServeDetachAttr_ChildHasNoConsole` skips itself when the parent test process has no attached console (`probeConsole(t, nil) != "console=attached"`). On headless CI runners (e.g. GitHub Actions `windows-latest`) and agent subprocesses, the test always skips (`SKIP: test process has no console...`), so detachment is never asserted by automated CI and relies on manual operator observation. | `verification.md:87-88` and `bwserve_windows_test.go:50-52` | `TestBWServeDetachAttr_ChildHasNoConsole` (skips in headless CI) | tests (outside contract set: consider adding a headless unit test checking `SysProcAttr` fields directly) |
| Minor | THEORETICAL | `ai/pi/models.json` (Upstream drift) | OpenRouter `deepseek-chat` limits occasionally flap between 128k/16k and 163840/16384 on models.dev (as observed in commits `946afa50` and `8f4c0bab`). Hardcoded values in `models.json` remain vulnerable to upstream catalog flapping until #902 (model-config SSOT from catalog) lands. | Git history `946afa50` vs `8f4c0bab` in `verification.md:26-38` | `TestModelLimits` (`cli/internal/doctor/checks_model_limits_test.go`) | vault (tracked under #902) |
| Minor | REAL | PR / Diff Scope | The review base `c8ebfec16b5359f76e9c490a0c47f0f2649dcd23` predates dozens of merged PRs and releases, inflating the diff between base and HEAD to 1,426 files. However, the changes directly attributable to `PI-002-windows-fixes` are tightly scoped (<50 LOC across two production files). | `git diff --stat c8ebfec16b5359f76e9c490a0c47f0f2649dcd23...HEAD` | UNTESTED | none (historical branch base artifact; tracked) |

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B | Core fixes verified on physical Windows 11 hardware (`hwnd=0`, `[Model limit drift] OK`); AC3 left unticked in `proposal.md` due to unrelated doctor failure. |
| Verification       | B | Hardware runtime verification thoroughly documented in `verification.md`; docked for vacuous `features.json` verification command and headless CI test skip. |
| Scope              | B | Diff base predates merged main history; changes at spec-owned sites match the proposal with no creep. |
| Reliability        | A | `HideWindow: true` with `CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS` correctly handles Scoop shims and avoids console termination events (`lesson-333`). |
| Maintainability    | A | Concise, clean implementation (6 LOC in `bwserve_windows.go`, CC=1) with detailed doc comments explaining Win32 API interactions. |
| Handoff-readiness  | B | Lessons 333 and 334 committed in `docs/lessons/`; `## Promotion candidates` section omitted from `verification.md` but fixable without staling the contract. |

### Verdict
PASS WITH GAPS

All previous Round 1 blockers and majors have been resolved. The open Major regarding `## Promotion candidates` is THEORETICAL (it represents an archive pre-flight check that will fail when `dotf spec archive` runs, but does not represent a defect in the code or contract, and lives in `verification.md` outside the contract set). All remaining findings are Minors, and all rubric dimensions are Grade B or above.

### Recommended next steps

The contract set (`proposal.md`, `tasks.md`, `features.json`) is **closed by this verdict**: editing it now would invalidate the review and make it stale under `dotf spec archive`.

**Required before archive (outside the contract set):**
- Add the `## Promotion candidates` section to `specs/PI-002-windows-fixes/verification.md` before running `dotf spec archive`. Answer each candidate line:
  - Lesson: `yes: docs/lessons/lesson-333-createprocess-detached-ignores-no-window.md` (and `docs/lessons/lesson-334-windows-prompts-to-open-extensionless-scripts.md`)
  - ADR: `no: bug fixes and configuration alignment, no new architectural pattern or public contract`
  - Pattern: `no: Windows process creation quirk and provider model limits, already captured in lessons 333 and 334`
  Because `verification.md` is excluded from the contract set (`cli/internal/spec/review.go:26`), editing it to add this section will **not** stale this review.

**Tracked in the contract set (disposition in `verification.md` or follow-up issues, do NOT edit now):**
- AC3 in `proposal.md`: Document disposition in `verification.md` explaining that the spec-owned section (`[Model limit drift]`) is clean, while the general doctor run had 1 failure in an unrelated section (#2258).
- `features.json`: In future specs, point `"verification"` to concrete Go test commands rather than an echo placeholder.
- Headless test coverage: Consider adding a headless unit test to `bwserve_windows_test.go` asserting `SysProcAttr` fields directly, complementing `TestBWServeDetachAttr_ChildHasNoConsole`.

**Archive readiness:**
`dotf spec archive PI-002-windows-fixes` is **advisable** once the `## Promotion candidates` section is added to `verification.md`. This review is fresh against `ed901e6d1edcf9bd115e0402e6267f3cf6871492`, `PASS-WITH-GAPS` is accepted by `checkReviewGate`, `reviewer:` matches the pool exactly, and no draft tags remain. Do not modify `proposal.md`, `tasks.md`, or `features.json`.
