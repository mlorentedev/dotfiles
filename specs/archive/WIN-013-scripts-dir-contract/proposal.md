---
id: "WIN-013-scripts-dir-contract"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-08-28"
issue: "mlorentedev/dotfiles#1310"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, windows, env-contract, setup]
template_version: "1.0"
---

# WIN-013-scripts-dir-contract

## Why

Four values named the Windows scripts directory and disagreed: the env
contract's `SCRIPTS_DIR` default (`$env:USERPROFILE\.dotfiles\scripts`),
`required_path_entries.windows` (`$env:USERPROFILE\scripts`), `setup-windows.ps1`'s
`$ScriptsDir` (`$env:USERPROFILE\scripts`) and the doctor/profile fallback
(`$env:DOTFILES_DIR\scripts`). Setup deployed to `~\scripts`, nothing ever
created the contract path, and `dotf doctor` FAILed `SCRIPTS_DIR=... (path does
not exist)` on every fresh Windows box — the test-windows gate (#1308) had to
allow-list it. On the work box `~\scripts` also held seven scripts retired by
earlier tickets that nothing had ever removed. Issue #1310 (folds #148).

## What

`setup-windows.ps1` deploys scripts to the contract's directory, under the
deploy dir (`$DotfilesDest\scripts`, the same shape as Linux's
`$HOME/.dotfiles/scripts`); `required_path_entries.windows` names the same
directory; the profile fallback already agrees. Every run removes, idempotently,
the retired scripts from both locations and the pre-contract copies of the
scripts setup now deploys elsewhere. The gate's WIN-013 row is gone in the same
change, so a stale row cannot survive it. The legacy `~\scripts` directory and
its User PATH entry are left in place.

## Out of scope

- Pruning the `~\scripts` entry from the User PATH (the 2048-char hazard #148
  named); a later ticket may do it with a measured PATH length.
- Deleting the legacy directory itself — it may hold the user's own scripts.
- `profile.ps1` consumers of `SCRIPTS_DIR` (`hc`, `project-init`): they read the
  variable, which the contract renders; unchanged.

## Risks / open questions

- A box whose `~\.dotfiles\scripts` already exists with stale content: setup
  overwrites the deployed set with `Copy-Item -Force`, as before.
- The gate list is now empty: `Read-KnownFailures` returns `@()`, which
  `Test-DoctorGate` accepts (`AllowEmptyCollection`), and the Pester guard that
  required at least one entry now asserts the parser's count instead.

## Acceptance criteria

- [x] AC1 — the contract's Windows `SCRIPTS_DIR` default, `required_path_entries.windows[0]`
  and setup's `$ScriptsDir` name the same directory (`~\.dotfiles\scripts`).
- [x] AC2 — setup removes the seven retired scripts from both locations and the
  legacy copies of the five deployed scripts, on every run, without touching the
  User PATH.
- [x] AC3 — the doctor gate's WIN-013 row is removed in the same change and the
  gate tooling accepts an empty list.
- [x] AC4 — the existing "no longer deploys X" guards measure deployment
  (`Copy-Item` / invocation), not mention, so the removal list can name X.
- [x] AC5 — the test-windows gate passes with `0 known runner-only FAIL(s)`.

## References

- Bitácora board: #1310 (consolidates #148)
- ADR-025 (paths resolve at setup), #1305 (`dotf harness mirror` took the same direction for the deploy dir)
- `env-contract.json`, `setup-windows.ps1`, `.github/scripts/doctor-gate-known-failures.txt`

<!-- archived 2026-08-28 — PR: https://github.com/mlorentedev/dotfiles/pull/1356 -->
