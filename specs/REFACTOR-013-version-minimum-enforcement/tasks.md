---
id: "REFACTOR-013-version-minimum-enforcement"
type: tasks
status: active
created: "2026-06-20"
template_version: "1.0"
---

# Tasks — REFACTOR-013

- [x] **T1** Add `version_gte` to `scripts/utils.sh` (pure `sort -V`; empty-pin
      and empty-installed edge cases). → AC1
- [x] **T2** Add `tests/version-gte.bats` covering equal / newer / older /
      numeric-not-lexical / component-count / empty-pin / empty-installed. → AC1
- [x] **T3** `setup-linux.sh`: opencode `!=` → `! version_gte`; keep the
      shadowing-install re-query. → AC3, AC5
- [x] **T4** `setup-linux.sh`: yarn `!=` → `! version_gte`. → AC3, AC5
- [x] **T5** `setup-linux.sh`: pi gains a below-minimum upgrade branch (was
      presence-only `[ ! -x $PI_BIN ]`). → AC4, AC5
- [x] **T6** `setup-windows.ps1`: add `Test-VersionAtLeast` helper. → AC2
- [x] **T7** `setup-windows.ps1`: opencode tools loop, yarn, pi switch `-ne` →
      `-not (Test-VersionAtLeast …)`. → AC3, AC5
- [x] **T8** Update regression tests that asserted the old `!=` / `-ne` strings
      (`tests/opencode.bats`, `tests/setup-windows.bats`) to assert the new
      comparator AND the absence of the exact-match form. → AC3
- [x] **T9** Verify: `bash -n setup-linux.sh`, PSScriptAnalyzer on
      `setup-windows.ps1`, full bats suite. → AC6
