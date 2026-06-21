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

## Promotion candidates

- [ ] Lesson for `docs/lessons.md`? no (faithful port; the build-then-delete +
  sequencing lessons already exist from CLI-012/CLI-020).
- [ ] ADR-worthy? no (executes ADR-020, no new decision).
- [ ] New pattern? no.

## Archive checklist (after PR-B merges)

- [ ] `proposal.md` frontmatter `status: archived`
- [ ] Folder moved: `specs/CLI-019/` → `specs/archive/CLI-019/`
- [ ] #488 closed; PR links recorded
