---
id: "REFACTOR-013-version-minimum-enforcement"
type: verification
status: active
created: "2026-06-20"
template_version: "1.0"
---

# Verification — REFACTOR-013

| AC | Evidence | Result |
|----|----------|--------|
| AC1 | `tests/version-gte.bats` — 8/8 pass (equal, newer, older, `1.10.0>=1.9.0`, `1.2<1.2.0`, empty-pin, empty-installed, `1.22.22>=1.22.9`). | ✅ |
| AC2 | `Test-VersionAtLeast` unit cases via pwsh — 10/10 (incl. `weird-tag` non-semver → string fallback, no throw). | ✅ |
| AC3 | `tests/opencode.bats` test 7 + `tests/setup-windows.bats` test 51 assert the new comparator AND `! grep` the removed `!=` / `-ne` form. Both pass. | ✅ |
| AC4 | `setup-linux.sh` pi block: added below-minimum upgrade branch (was `[ ! -x $PI_BIN ]` presence-only). | ✅ |
| AC5 | `version_gte`/`Test-VersionAtLeast` return satisfied for newer-than-pin → reconcile branch is skipped (no downgrade). | ✅ |
| AC6 | `bash -n setup-linux.sh` OK; `setup-windows.ps1` parses clean + PSScriptAnalyzer (Error,Warning) → no issues. | ✅ |

## Test run (local, Windows git-bash)

- `version-gte.bats` 8/8, `opencode.bats` 45/45, `setup-windows.bats` 108/108.
- `setup-linux.bats` 79/80 — the single failure is `setup-linux.sh valid zsh
  syntax`, which fails only because **zsh is not installed on this dev machine**
  (`zsh: command not found`); it fails identically on the unmodified `main`
  tree, and CI installs zsh (`.github/workflows/ci.yml`), so it passes there.
  `version_gte` is POSIX-only (`printf` / `sort -V` / `head` / `[ ]`).

## Notes

- bats/obsidian (Linux install *latest*, pin not wired into the install) and the
  Windows-CI-only AGE/EZA/ZOXIDE download pins are out of scope per the proposal.
- The authoritative gate is CI (Linux + windows-latest), where zsh and real
  symlinks exist.
