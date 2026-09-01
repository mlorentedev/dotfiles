---
spec: "WIN-013-scripts-dir-contract"
verdict: "PASS"
reviewed_sha: "389aaf50f3d0246b56b1e27871094694dbf44701"
reviewer: "nan/deepseek-v4-flash"
date: "2026-08-29"
---

## Adversarial review

**Scope**: WIN-013-scripts-dir-contract
**Sources**: proposal.md, tasks.md, verification.md, features.json; implementation commit `2d87c7b` (PR #1356, merged) + the files it touches (`setup-windows.ps1`, `env-contract.json`, `.github/scripts/doctor-gate.ps1`, `doctor-gate-known-failures.txt`, tests).

### Spec and task alignment

- All task boxes in `tasks.md` are ticked, and each maps to a named, present test or guard in the reviewed tree — no `[x]` with no diff evidence.
- Five acceptance criteria, each with a named verification. AC1-AC4 are proven by the committed tests; AC5 is proven by a cited CI run. Details and the residual gaps are the findings below.
- No `[AGENT-DRAFT]` / `[AGENT-SUGGESTION]` tags remain in any spec file — the archive gate's own marker check is clean.
- The implementation moves Windows scripts to `$DotfilesDest\scripts` (`~\.dotfiles\scripts`), updates `required_path_entries.windows[0]` to the same, and I re-verified the two `SCRIPTS_DIR` consumers the proposal claims "already agree": `powershell/profile.ps1:270` and `setup-windows.ps1:2229` both derive `$env:DOTFILES_DIR\scripts` where `DOTFILES_DIR=~\.dotfiles`. So the fourth value does agree in the code, even though AC1's automated test does not pin it (finding 3).

### Findings

| Severity | Reality | Area | Finding | Evidence | Test (named, or UNTESTED) | Fix location (code / tests / spec / vault) |
|----------|---------|------|---------|----------|---------------------------|---------------------------------------------|
| Minor | REAL | docs | `powershell/profile.ps1:202` still reads "The setup-windows.ps1 script adds ~/scripts to PATH", but the change makes setup add `~\.dotfiles\scripts`. A doc the change rendered false was left stale. | read of `powershell/profile.ps1` lines 200-204 vs `setup-windows.ps1:1739-1743` (PATH write targets `$ScriptsDir`) | UNTESTED (comment-only) | code/docs |
| Minor | THEORETICAL | verification | AC2's destructive `Remove-Item` sweep (7 retired names × both dirs + 5 deployed from the legacy dir) is proven only by static presence greps, never by executing the removal loop. A runtime regression (exception mid-loop, path-join error) would pass the green gate unreported. | `tests/setup-windows.bats` "(WIN-013)" asserts names/`$LegacyScriptsDir`/`foreach` loop strings, no execution; verification.md discloses setup was not run on a real box pre-merge | `@test "setup-windows.ps1 removes every retired script and the legacy ~\scripts copies of the live ones (WIN-013)"` — structural only, does not run the sweep | tests (extract sweep to a unit-testable helper) |
| Minor | REAL | spec-gap | The proposal's Why lists four agreeing values; AC1's test pins three (contract default, `required_path_entries.windows[0]`, setup `$ScriptsDir`) but not the doctor/profile fallback. Code agrees (verified by hand), so this is a coverage gap, not a defect. AC1 names three, so no AC violation. | `tests/env-contract.bats` "(WIN-013)" checks 3 sources; fallback asserted in prose only | `@test "SCRIPTS_DIR: ... (WIN-013)"` — covers 3 of 4 | tests / spec |
| Minor | SPECULATIVE | test | AC4 rewrites the MEM-002 / CLI-018 / CLI-019 guards from "no mention" to deployment-only (`Copy-Item.*name`, `&.*name`). A retired script re-introduced via another deploy verb (dot-source, `Move-Item`, `Invoke-Expression`) would evade all three. Deliberate, documented weakening; the residual slot is low-risk and the names still cannot be deployed by the two verbs setup actually uses. | `tests/setup-windows.bats` MEM-002/CLI-019/CLI-018 guards; commit message explains the guard change | MEM-002, CLI-019, CLI-018 named guards | tests |
| Minor | REAL | verification | AC5 CI evidence (run 33201808112, test-windows `0 known runner-only FAIL(s)`) is from branch commit `1076ed0`; the merged `2d87c7b` differs by one docs-only commit (`docs(spec): record the WIN-013 gate evidence from the runner`) that cannot affect the doctor gate. Evidence carries; worth a one-line note so a future reader is not surprised by the sha mismatch. | `verification.md` AC5 vs `git show 2d87c7b --stat` (docs-only delta) | UNTESTED (CI evidence) | spec |

No Blocker or Major findings. No finding is REAL-with-destructive-impact; the strongest (finding 2) is a THEORETICAL verification gap on code that runs only post-merge, and it is disclosed in verification.md rather than hidden.

### Evaluator rubric

| Dimension | Grade (A-D) | Rationale (one line) |
|-----------|-------------|----------------------|
| Correctness        | B  | ACs met with named tests; negative paths (empty known-failures list, legacy PATH untouched) covered, but AC1 is happy-path equality and AC2 is static-only |
| Verification       | B  | Reproducible bats/Pester/CI commands with outputs and a run ID; no real work-box run, disclosed and scheduled post-merge on #1310 |
| Scope              | A  | Diff maps 1:1 to AC1-AC5 plus the necessary doctor-gate empty-list fix (AC3's direct consequence); test-infra change is minimal and purposeful |
| Reliability        | B  | Sweep is idempotent (`Test-Path` guards, `Select-Object -Unique`, `-ErrorAction SilentlyContinue`), PATH untouched; runtime-unverified (finding 2) |
| Maintainability    | A  | Short, clear, well-commented; loop and list are straightforward; negligible cyclomatic complexity |
| Handoff-readiness  | B  | `features.json` evidence recorded, `status: verifying`, archive checklist present, promotion candidates declined with reasons |

Aggregation: all B or above, no D, no C → **PASS**.

### Verdict
PASS

### Recommended next steps (before archive)
- Apply finding 1 (fix the stale `profile.ps1:202` comment) — a one-line docs correction; do it in the same change that archives, or ticket it so it does not float.
- Consider (non-blocking) a unit-testable extraction of the sweep (finding 2) so the destructive path has a runtime test; this is the strongest gap but not a gate.
- Extend the AC1 bats test to pin the profile-fallback value (finding 3) if convenient.
- Update the `date:`/`reviewed_sha:` tracking: this review records the merged HEAD `389aaf5`; the AC5 CI evidence predates it by one docs-only commit — add the one-line note from finding 5 when archiving.
- Then resume the normal archive path: `dotf spec archive` / `/spec archive`, which will refuse only if the contract files change after this sha.
