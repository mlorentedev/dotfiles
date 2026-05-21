---
tags: [spec, verification, cross-os-parity, drift-detection]
created: "2026-05-21"
---

# Verification - REFACTOR-003-diff-check-ps1

## Evidence (per acceptance criterion)

To be filled post-implementation. Template:

- [ ] **`diff-check.ps1` exists + mirrors allowlist**: line range + grep cross-check vs `diff-check.sh`.
- [ ] **Exit codes match**: pwsh repro of "no drift" -> 0, "drift" -> 1, "missing dir" -> 2.
- [ ] **healthcheck.ps1 sec 12 invokes**: diff between current SKIP block and new PASS/FAIL block.
- [ ] **`dch` alias in profile.ps1**: function definition + verification it resolves.
- [ ] **setup-windows.ps1 deploys**: Copy-Item line + post-setup `Test-Path "$ScriptsDir\diff-check.ps1"`.
- [ ] **Bats**: count of new asserts + CI green.
- [ ] **Lint**: PSScriptAnalyzer + AST clean + ASCII-only.

## Test status

### Pre-fix state (Windows daily-driver, captured 2026-05-21)

```
PS> healthcheck.ps1
[12/12] Repo - Deploy-Dir Drift
  SKIP: diff-check - diff-check.ps1 not implemented (REFACTOR-003 queued)
```

### Empirical post-impl run

To be filled.

### Lint results

To be filled.

## Decisions made during implementation

- **Faithful port over Windows-specialised allowlist**: per anti-scope in proposal, mirror bash allowlist exactly. Linux-only items (.zshrc, .bashrc, tmux.conf) get implicitly skipped on Windows via `Test-Path` guard. Refining the per-OS allowlist is REFACTOR-003b if motivation appears.
- **`Get-FileHash SHA256` over byte-by-byte `Compare-Object`**: faster on Windows + deterministic + matches the bash side's `cmp -s` semantic (byte-equal). Acceptable for files <10MB (every managed config file is <100KB).
- **Non-fatal in healthcheck**: sec 12 surfaces drift as FAIL line but doesn't break exit code (mirror WIN-001's non-fatal pattern from PR #71).
- **`dch` function vs alias**: PowerShell `Set-Alias` cannot accept arguments, so a function-form `dch { ... }` is required to forward `-VerboseOutput` / `-Help`. Matches Linux bash function semantic.

## Promotion candidates

- [ ] Lesson for `90-lessons.md`? **possibly** — "when porting a bash drift-detector to PowerShell, prefer Get-FileHash SHA256 over byte-by-byte Compare-Object: faster, deterministic, and matches `cmp -s` semantics".
- [ ] ADR? **no** — tactical mirror, no new architectural decision.
- [ ] New pattern? **no** — `pattern-cross-os-script-mirror` would cover it if formalised, but not needed yet.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`.
- [ ] Folder moved: `specs/REFACTOR-003-diff-check-ps1/` -> `specs/archive/REFACTOR-003-diff-check-ps1/`.
- [ ] Vault `11-tasks.md` REFACTOR-003 entry ticked ✓ with PR link.
- [ ] Healthcheck sec 12 transition documented as proof.
