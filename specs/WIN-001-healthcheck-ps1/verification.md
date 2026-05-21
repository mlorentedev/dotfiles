---
tags: [spec, verification]
created: "2026-05-21"
---

# Verification - WIN-001-healthcheck-ps1

## Evidence

Each acceptance criterion from `proposal.md` mapped to a concrete artifact.

- [x] `scripts/healthcheck.ps1` exists, valid PowerShell -> `pwsh -NoProfile -Command "[scriptblock]::Create((Get-Content -Raw scripts/healthcheck.ps1)) | Out-Null"` -> "PARSE OK"
- [x] PSScriptAnalyzer clean -> `Invoke-ScriptAnalyzer -Path scripts/healthcheck.ps1 -Settings .PSScriptAnalyzerSettings.psd1 -Severity Error,Warning` -> empty result
- [x] 12 sections numbered 1/12..12/12 -> bats test `healthcheck.ps1 has all 12 sections numbered 1/12..12/12` (tests/healthcheck-ps1.bats)
- [x] Sec 9 (tmux), 11 (ghostty), 12 (drift) emit SKIP with explanation -> bats tests `section 9/12 emits SKIP`, `section 11/12 emits SKIP with TERM-002 reference`, `section 12/12 emits SKIP with REFACTOR-003 reference`
- [x] `[CmdletBinding()]`, `Set-StrictMode -Version Latest`, `$ErrorActionPreference = 'Continue'` -> bats tests for each
- [x] Write-Pass / Write-Fail / Write-Skip / Write-Section helpers defined -> bats tests for each
- [x] `$script:Passed` / `$script:Failed` / `$script:Skipped` counters + summary line -> manual smoke output: `Results: 59 passed, 18 failed, 15 skipped`
- [x] Exit code policy: exit 0 if Failed==0, exit 1 otherwise -> bats test `gates exit 1 on $script:Failed counter`
- [x] `setup-windows.ps1` invokes healthcheck.ps1 after doctor (section 8d), non-fatal -> bats tests `WIN-001: setup-windows.ps1 invokes healthcheck.ps1 after doctor`, `healthcheck block is non-fatal (Write-Warn, no exit)`, `healthcheck block runs after doctor block`
- [x] `powershell/profile.ps1` exports `hc` function -> bats tests `profile.ps1 defines hc function`, `hc function references SCRIPTS_DIR\healthcheck.ps1`
- [x] `tests/healthcheck-ps1.bats` parity asserts written -> file exists, 31 asserts (structural + PSScriptAnalyzer + cross-OS parity)
- [x] Empirical smoke on this Windows machine -> output: 12 sections rendered, K=15 skipped (sec 9/11/12 baseline + JAVA_HOME/MAVEN_HOME/etc unset + tools not installed), exit 1 due to legitimate environment gaps (no docker/kubectl/terraform/direnv on this Windows box; these are WIN-002 sweep findings, not WIN-001 bugs)

## Test status

- Manual structural assertion sweep: all PASS (matching bats coverage exactly via grep).
- PSScriptAnalyzer (Error+Warning) clean on `scripts/healthcheck.ps1`, `setup-windows.ps1`, `powershell/profile.ps1`.
- Empirical end-to-end: `pwsh -NoProfile -File scripts/healthcheck.ps1` -> 59 passed / 18 failed / 15 skipped / exit 1. The 18 FAILs are legitimate Windows-environment findings (no docker, no JAVA_HOME, etc.), NOT script bugs — they feed WIN-002.
- bats suite runs in CI (no local bats binary on this Windows machine; trusting GitHub Actions linux job).

## Decisions made during implementation

- **`$PROFILE` indirection in sec 4.** Initial draft hardcoded `Documents\PowerShell\profile.ps1` but `setup-windows.ps1` deploys to `$PROFILE` which under pwsh 7 resolves to `Microsoft.PowerShell_profile.ps1`. Switched to `Test-DeployedFile $PROFILE` so the check tracks whatever the running shell would source. Cross-version safe (works under PS 5.1 if someone bypasses the BUG-005 reexec).
- **`vault-health.sh` SKIP, not FAIL.** Linux side has `vault-health.sh` but there's no `.ps1` sibling yet. On Windows we SKIP with the explanation "Linux-only script (no .ps1 sibling)". Promoting this to a full FAIL would be a behavior change beyond the proposal — left as future work.
- **No `Pester` tests.** Backlog originally mentioned Pester. Switched to bats following the established pattern (`knowledge-crystallize-ps1.bats`, `obs-cli-ps1.bats`) — bats invokes pwsh for PSScriptAnalyzer, rest is grep-static. Cross-OS parity with `healthcheck.bats` is the prize; Pester would diverge from house style.
- **Sibling tickets opened mid-PR (not deferred to post-merge).** Decision shaped by user feedback: "tambien deberiamos abrir los tickets para que se haga en linux y haya paridad total no?". Added `WIN-001b-healthcheck-auto-wire-linux` and `REFACTOR-003-diff-check-ps1` to vault `11-tasks.md` while the design context was fresh, instead of letting them surface as "implicit follow-ups" 2 sprints later.

## Promotion candidates

- [x] Lesson for `90-lessons.md`: **"vault_patch timeout != patch not applied"** — Hive `vault_patch` returned `timed out after 60s` on the first call, but the patch had actually committed; retry failed with `find text not found` because the file was already in its new state. Rule: on `vault_patch` timeout, re-read the file before retrying. Surface lesson during handoff.
- [ ] ADR-worthy decision? No — WIN-001 follows existing patterns (`doctor.ps1`, `obs-cli.ps1`, etc.), no new architectural commitment.
- [ ] New pattern candidate? No — single-file port, no recurrence yet.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/WIN-001-healthcheck-ps1/` -> `specs/archive/WIN-001-healthcheck-ps1/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (lesson capture via `capture_lesson`)
