---
tags: [spec, tasks, cli, go, convergence, sync, update]
created: "2026-07-01"
---

# Tasks - CLI-027-dotf-sync-update

> TDD order. Two PRs (atomic-PR + de-risk). Tick as you go.

## PR 1 — `dotf update` (selfupdate port)

### 1a. Build the noun (no deletes — build-beside-the-shell)  ✅ DONE

- [x] `internal/update/update.go` — `Run(cfg, Deps)` with a Git + RunSetup seam; every non-actionable branch is a skip (nil err); only a real setup failure errors.
- [x] `internal/update/update_test.go` — table over all 9 branches (not-a-repo / dirty / offline / no-upstream / current / diverged / ff-failed / updated / setup-failed); asserts skip branches never re-run setup. **9/9 green.**
- [x] `internal/cmd/update.go` — cobra `dotf update`; production Git (`git -C repo`) + OS-aware setup exec (POSIX direct / `pwsh -File` on Windows); `repoForUpdate()` resolves via the ADR-025 seam with the `$HOME/Projects/dotfiles` bare-scheduler-env fallback (matches the shell twins). Registered in `root.go`.
- [x] Builds, vets, gofmt-clean; the #664 help guard renders `dotf update --help`.

### 1b. Cutover (repoint every caller, then delete)  ⏳ PENDING

All callers of the deleted scripts mapped (guard-grep `dotfiles-selfupdate`):

- [ ] `systemd/dotfiles-selfupdate.service` — `ExecStart` → `%h/.local/bin/dotf update` (mirror hive-upgrade); refresh the comment. (env: `repoForUpdate()` fallback covers the bare --user env; no `Environment=` needed.)
- [ ] `setup-windows.ps1` — `DotfilesSelfUpdate` task action → run `dotf update` directly (drop the `.ps1` copy-to-ClaudeHome logic).
- [ ] `tests/dotfiles-selfupdate-install.bats` — repoint the service assertion (`dotfiles-selfupdate.sh` → `dotf update`), the Windows task assertion (`.ps1` → `dotf update`); remove the "`.ps1` exists + ASCII-only" test (script gone). Keep the unit/timer/gate assertions (units are floor).
- [ ] `docs/runbooks/guide-self-deploy-timer.md` — operational refs `scripts/dotfiles-selfupdate.sh` → `dotf update` (the #664 dangling-doc-ref lesson: a runbook must not name a deleted script).
- [ ] `setup-linux.sh` — the OPS-001 comment mentioning `dotfiles-selfupdate.sh` no-ops → `dotf update`.
- [ ] Delete `scripts/dotfiles-selfupdate.{sh,ps1}`.
- [ ] Guard-grep: no non-historical reference to the deleted scripts remains (ADRs/audit planning docs are point-in-time snapshots, left as-is).
- [ ] Verify: `bats tests/dotfiles-selfupdate-install.bats` green; `go test ./...`; **Windows smoke** — register the task, run it, confirm `dotf update` executes (I'm on the Windows box).

## PR 2 — `dotf sync` (secrets newest-wins + repo→deployed deploy)

- [ ] **Resolve the open question first:** confirm the Windows repo→deployed model vs Linux `rsync --delete` (proposal Risks). Then scope + build `dotf sync`; delete `dotfiles-sync.{sh,ps1}` + tests.

## Machine-readable features

`features.json` emitted per PR as each lands (harness gates `passing` on captured evidence).
