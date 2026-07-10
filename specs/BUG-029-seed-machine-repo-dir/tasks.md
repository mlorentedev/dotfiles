---
id: "BUG-029-seed-machine-repo-dir"
type: spec
status: implementing
created: "2026-07-10"
tags: [spec, tasks]
template_version: "1.0"
---

# Tasks — BUG-029-seed-machine-repo-dir

- [x] T1. `env` package: `SetMachinePath(contractPath, machinePath, key, value)`
  — load-or-init machine.json, validate `key` against the contract, merge
  preserving other keys, atomic write. Unit tests first (TDD).
- [x] T2. `dotf env set <KEY> <VALUE>` cobra subcommand wiring `SetMachinePath`;
  help text as the write-side counterpart of `env path`.
- [x] T3. setup-linux.sh + setup-windows.ps1: seed `DOTFILES_REPO_DIR` before
  `dotf env generate`, inside the existing dotf-present guard.
- [x] T4. B: `repoForUpdate()` + `mem` resolvers fall back to `RepoDir()` walk-up
  when the cascade value is not an existing dir.
- [x] T5. Doctor check: `DOTFILES_REPO_DIR` resolves to a real checkout (dir with
  `.git`); FAIL + actionable hint otherwise. Unit test.
- [x] T6. Guards: integration bats in `verify-setup.bats` (machine.json seeded +
  resolves to a real dir).
- [x] T7. Local verification: `go build ./...`, `go test ./...`,
  `bash -n setup-linux.sh`. (bats integration validates in Linux CI.)
- [ ] T8. CI green (`lint`, `lint-powershell`, `test`, `integration`,
  `spec-gate`).
