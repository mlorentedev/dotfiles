---
tags: [spec, verification]
created: "2026-06-21"
---

# Verification - CLI-019 (PR-A)

## Evidence (PR-A — build-only)

- [x] **Drift section ports diff-check** → `checkDeployDrift` in
  `cli/internal/doctor/checks_deploy.go`; registered in `doctor.go`'s full sweep
  after `checkHarnessDrift` (NOT in `--quick`).
- [x] **Table test green** → `go -C cli test -run 'TestCheckDeployDrift|TestIsManagedDeployPath' ./internal/doctor/`:

  ```
  --- PASS: TestCheckDeployDrift (0.12s)
      all managed equal → pass
      managed drift → fail naming the path
      unmanaged tracked file drift → ignored
      managed file absent in deploy → not drift
      no deploy-dir → skip
      not a git repo → skip
      git ls-files fails → warn, no crash
  --- PASS: TestIsManagedDeployPath
  ok  github.com/mlorentedev/dotfiles/cli/internal/doctor
  ```

- [x] **Package suite green** → `go -C cli test ./internal/doctor/...` ok
  (full-sweep tests still pass: `checkDeployDrift` SKIPs in that context).
- [x] **vet + lint clean** → `go vet ./internal/doctor/`; `golangci-lint run ./internal/doctor/` (no findings).

## Acceptance criteria → test

| Criterion | Covered by |
|---|---|
| Full sweep prints "Repo↔deploy-dir drift" section | `checkDeployDrift` `rep.Section(...)` + table output |
| Managed file differs → FAIL naming the path | `TestCheckDeployDrift/managed_drift_→_fail_naming_the_path` |
| All managed equal → PASS | `TestCheckDeployDrift/all_managed_equal_→_pass` |
| Unmanaged tracked file differs → ignored | `TestCheckDeployDrift/unmanaged_tracked_file_drift_→_ignored` |
| Missing repo / deploy / non-git → SKIP | `…/no_deploy-dir_→_skip`, `…/not_a_git_repo_→_skip` |
| `git ls-files` failure → WARN (no crash) | `…/git_ls-files_fails_→_warn,_no_crash` |

## Test status

- Suite: `go -C cli test ./internal/doctor/...` → ok (no regressions).
- Manual smoke deferred to PR-B (where a deployed box runs `dotf doctor` against
  a real repo↔deploy-dir pair); PR-A is additive Go logic verified by tests.

## Decisions made during implementation

- **SKIP, not exit-2, when a side is absent.** The shell twin `exit 2`'d on
  missing repo/deploy/`.git`; folded into `dotf doctor` (runs on CI / fresh
  boxes) those are expected, so they degrade to SKIP — matching `checkTmux`/`checkVault`.
- **`git ls-files` via the `CommandOutput` seam**, not an FS walk — keeps the
  "tracked files only" semantics of diff-check while staying testable git-free.
- **Allowlist ported verbatim** with the mirror-setup warning comment. A
  grep-guard pinning it to `setup-linux.sh`'s copy block is a PR-B follow-up.

## Evidence (PR-B — delete + repoint)

- [x] **Twins deleted** → `git rm scripts/diff-check.sh scripts/diff-check.ps1
  tests/diff-check.bats` (no Pester existed).
- [x] **Production callers repointed/cleaned** → `setup-windows.ps1` diff-check
  deploy block removed + BUG-021 comment generalized; `setup-linux.sh` BUG-021
  comment cleaned; `powershell/profile.ps1` `dch` now wraps `dotf doctor`;
  `.zsh/aliases.zsh` `dch` alias → `dotf doctor`; `ci.yml` comments generalized;
  `env-contract.json` `DOTFILES_REPO_DIR` description updated.
- [x] **Guard test green** → `tests/setup-windows.bats` gains 3 CLI-019 checks
  (no-deploy / `dch`-wraps-`dotf doctor` / production-callers-clean). The guard
  is scoped to the 5 caller files; written with explicit `if grep …; then
  return 1; fi` (a bare `! grep` is exempt from bats errexit).
- [x] **Suites green** → `bats tests/setup-windows.bats` (109), `bats
  tests/setup-linux.bats`, `bats tests/healthcheck-ps1.bats` (36, §11 residual
  still asserts the un-touched `healthcheck.ps1`). `go test
  ./internal/doctor/...` ok. (`go test ./...` shows the pre-existing #461
  vault-template drift FAILs only — unrelated; no Go touched here.)

### Sequencing decision (PR-B before #509)

Kept the spec order at the user's call. Because Windows does not yet invoke
`dotf doctor` (that switch is #509), deleting `diff-check.ps1` opens a
**transient Windows repo↔deploy drift gap** — `healthcheck.ps1` §11 degrades to
SKIP — until #509 lands right after. Linux is unaffected (`setup-linux.sh`
already runs `dotf doctor` with PR-A's `checkDeployDrift`). The lone residual
(`healthcheck.ps1` + `tests/healthcheck-ps1.bats`) is excluded from the guard and
owned by #509, which tightens it on deletion.

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no (faithful port; the build-then-delete +
  sequencing lessons already exist from CLI-012/CLI-020).
- [ ] ADR-worthy? no (executes ADR-020, no new decision).
- [ ] New pattern? no.

## Archive checklist (after PR-B merges)

- [ ] `proposal.md` frontmatter `status: archived`
- [ ] Folder moved: `specs/CLI-019/` → `specs/archive/CLI-019/`
- [ ] #488 closed; PR links recorded
