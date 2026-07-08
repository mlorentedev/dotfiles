---
tags: [spec, tasks]
created: "2026-06-21"
---

# Tasks - CLI-019

> Build-then-delete, two PRs (strangler-fig). PR-A (this branch) ports the drift
> check into `dotf doctor`, build-only. PR-B repoints callers + deletes the twins.

## Setup

- [x] Branch from main: `feat/dotf-doctor-deploy-drift`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] Open questions resolved (allowlist-sync residual documented as a non-blocking follow-up)

## PR-A — port the drift check (build-only, this branch)

- [x] Failing table test `checks_deploy_drift_test.go` (agree / managed-drift / unmanaged-ignored / half-present / no-deploy / no-git / git-error) + `TestIsManagedDeployPath`
- [x] `checkDeployDrift` in `checks_deploy.go` — `git ls-files` via the `CommandOutput` seam, byte-compare via `filesEqual`, SKIP when repo/deploy/`.git` absent, WARN on git failure
- [x] `resolveRepoDir` (`DOTFILES_REPO_DIR` → walk-up to `.git`) + `isManagedDeployPath` allowlist
- [x] Register in `doctor.go` full sweep (after `checkHarnessDrift`; NOT in `--quick`)
- [x] `go test ./internal/doctor/...` green; `go vet` + golangci-lint clean
- [ ] PR-A opened (references #488, does NOT close it)

## PR-B — repoint callers + delete diff-check (separate PR, closes #488)

- [x] Repoint `ci.yml` — only stale comments referenced `diff-check` (no direct
      invocation); generalized them. The Windows CI step still runs
      `healthcheck.ps1`, whose §11 now SKIPs; the dotf-doctor switch is #509.
- [x] Repoint `setup-linux.sh` + `setup-windows.ps1` — removed the
      `setup-windows.ps1` diff-check deploy block; `setup-linux.sh` had only a
      BUG-021 comment (cleaned). Both already invoke `dotf doctor` post-setup
      (Linux today; Windows via #509).
- [x] Repoint `powershell/profile.ps1` `dch` function **and** the Linux
      `.zsh/aliases.zsh` `dch` alias → `dotf doctor` (the task list omitted the
      Linux alias; both are now repointed). Also fixed `env-contract.json`'s
      `DOTFILES_REPO_DIR` description.
- [x] `git rm scripts/diff-check.sh scripts/diff-check.ps1 tests/diff-check.bats`
      (no Pester existed for the twin).
- [x] Guard-grep clean for `diff-check` — new `tests/setup-windows.bats`
      "production callers no longer reference diff-check" test, scoped to the 5
      caller files; `scripts/healthcheck.ps1` + `tests/healthcheck-ps1.bats` are
      the lone residual, excluded and owned by #509. bats suites green
      (setup-windows 109, setup-linux, healthcheck-ps1 36); `go test
      ./internal/doctor/...` green.
- [ ] (hardening, deferred) grep-guard test pinning the Go allowlist to
      `setup-linux.sh`'s copy block — kept out of PR-B to stay atomic; tracked as
      a follow-up on #488.

## Closing

- [ ] Both PRs merged → archive spec, close #488
- [ ] Sequenced BEFORE #509 (CLI-018 PR-B healthcheck.ps1 deletion) so no Windows drift coverage is lost

## Note

§11 has two halves: harness/skill drift (`checkHarnessDrift`, already ported in
CLI-012) and repo↔deploy drift (`checkDeployDrift`, this spec). CLI-018 PR-B
(#509) deletes `healthcheck.ps1` and must land AFTER PR-A.
