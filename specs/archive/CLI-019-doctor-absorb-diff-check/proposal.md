---
id: "CLI-019"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-06-21"
issue: "mlorentedev/dotfiles#488"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# CLI-019 — dotf doctor absorbs repo↔deploy-dir drift, delete diff-check

## Why

`dotf doctor` (CLI-012) consolidated the diagnostics twins, but deliberately
left ONE section unported: the repo↔deploy-dir drift check (healthcheck §11),
which still lives as the standalone `diff-check.{sh,ps1}` twins. Those twins are
the only thing keeping that coverage, and they are exactly the kind of per-OS
shell duplication ADR-020 exists to eliminate. They are also a **blocker for
CLI-018 PR-B (#509)**: deleting `healthcheck.ps1` (whose §11 invokes
`diff-check.ps1`) would open a Windows drift-coverage gap unless `dotf doctor`
covers it first.

## What

`dotf doctor` gains a **"Repo↔deploy-dir drift"** section that reproduces
`diff-check` natively in Go: for each git-tracked file under the managed
allowlist, it byte-compares the repo copy against the deployed `~/.dotfiles`
copy and FAILs on any divergence (the "edited the repo but never re-ran setup"
trap). Then the standalone `diff-check.{sh,ps1}` + their tests are deleted and
every caller (CI, both setups, the PowerShell `dch`/`hc` wiring) is repointed at
`dotf doctor`.

Build-then-delete, two PRs (strangler-fig discipline, lesson 2026-06-14):

- **PR-A (this branch):** add `checkDeployDrift` to the full sweep, build-only.
  Nothing deleted, no caller repointed — purely additive coverage. This is the
  unit #509 depends on.
- **PR-B:** repoint `ci.yml` + `setup-linux.sh` + `setup-windows.ps1` +
  `powershell/profile.ps1`, `git rm scripts/diff-check.{sh,ps1}` + bats/Pester,
  guard-grep clean for `diff-check`.

## Out of scope

- Deleting `diff-check.{sh,ps1}` / repointing callers — that is **PR-B**.
- Deleting `healthcheck.ps1` / `doctor.ps1` — that is **CLI-018 PR-B (#509)**,
  sequenced *after* this lands so no drift coverage is lost.
- The `--verbose` per-file unified diff `diff-check` prints — `dotf doctor`
  reports the drifted *path* + remediation; a full diff dump is not ported (the
  remediation is always "run setup", and `git diff` covers forensics).

## Risks / open questions

- **Allowlist sync (the one real judgment call).** `diff-check`'s managed set
  (`versions.conf .zshrc .bashrc .profile .gitconfig tmux.conf` + `.zsh/ ssh/
  scripts/ sensitive/`) MUST mirror setup's copy block, and the shell twin kept
  them in sync only by a comment. The Go port inherits that coupling. Decision:
  port the allowlist verbatim with the same warning comment for PR-A; a
  grep-guard test that pins the Go allowlist to `setup-linux.sh`'s copy block is
  noted as a follow-up hardening (not blocking).
- **REPO_DIR resolution differs from the shell.** `diff-check` used
  `DOTFILES_REPO_DIR` → parent-of-script. Go has no script dir, so it resolves
  `DOTFILES_REPO_DIR` → walk-up-to-`.git` from CWD (the `findRepoRoot` already in
  `config.go`). When neither resolves (e.g. a deployed box where the repo is
  absent), the section SKIPs — `dotf doctor` runs in many contexts and a missing
  repo is not a failure (the shell `exit 2` becomes a SKIP).
- **`git ls-files`** is shelled out (faithful: tracked files only), via the
  injectable `sys.CommandOutput` seam so tests stay network-/git-free.

## Acceptance criteria

- [ ] `dotf doctor` (full sweep, not `--quick`) prints a "Repo↔deploy-dir drift"
      section.
- [ ] A managed file that differs between repo and deploy-dir → FAIL naming the path.
- [ ] All managed files equal → PASS.
- [ ] An unmanaged tracked file that differs → ignored (no FAIL).
- [ ] Missing repo / missing deploy-dir / non-git repo → SKIP (not FAIL).
- [ ] `git ls-files` failure → WARN (not a crash).

## References

- Issue: mlorentedev/dotfiles#488 ; sequenced before #509 (CLI-018 PR-B)
- Twin being ported: `scripts/diff-check.sh` + `scripts/diff-check.ps1`
- Deferral origin: `cli/internal/doctor/checks_deploy.go` `checkHarnessDrift` ("MINUS the diff-check")
- ADR-020 (Go convergence) ; lessons 2026-06-14 (build-then-delete) + 2026-06-21 (parity gate)
