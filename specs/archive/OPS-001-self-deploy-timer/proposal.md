---
id: "OPS-001-self-deploy-timer"
type: spec
status: archived
created: "2026-06-09"
issue: "dotfiles#295"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# OPS-001-self-deploy-timer

> **Naming**: file lives at `<repo>/specs/OPS-001-self-deploy-timer/proposal.md`. `OPS-001-self-deploy-timer` is `AREA-NNN-slug`.

## Why

Dotfiles changes (new tools, version bumps, config edits) merge to `main` but only reach a machine when someone remembers to `git pull` and re-run `setup`. On a multi-machine setup that drift is silent and unbounded — a fix can sit undeployed for weeks. An opt-in timer that periodically pulls the repo and re-runs the *idempotent* setup closes the loop with zero manual steps (Standing Order #1: automate, don't instruct), while staying safe enough to never touch local work-in-progress.

## What

A new opt-in **self-deploy** mechanism:

- A dedicated entrypoint `scripts/dotfiles-selfupdate.sh` that: guards against a dirty worktree, `git fetch`es, fast-forwards `main` **only** when it can, and re-runs the platform `setup` **only when `HEAD` actually moved**. It never merges, rebases, or resets — anything that is not a clean fast-forward is logged and skipped.
- A systemd `--user` timer + oneshot service (`systemd/dotfiles-selfupdate.{timer,service}`) firing **daily** with `Persistent=true` (a suspended/off laptop runs the missed slot on next boot).
- Opt-in wiring in `setup-linux.sh` gated on `DOTFILES_AUTODEPLOY`: `1` installs + enables the timer, `0` disables + removes it, **unset leaves the machine untouched** (default OFF).
- Windows parity authored: `scripts/dotfiles-selfupdate.ps1` + a daily Scheduled Task registered by `setup-windows.ps1` under the same env gate (runtime verification deferred to a Windows-box session).

Observable change: after `DOTFILES_AUTODEPLOY=1 ./setup-linux.sh`, the machine converges to `origin/main` within ~24h on its own, and **never** does so when the worktree is dirty or history has diverged.

## Out of scope

- Auto-applying breaking config changes without review — the timer redeploys whatever is on `main`; gating *what* lands on `main` is the PR review's job, not the timer's.
- Touching `dotfiles-sync.sh` — it stays the push/deploy tool (`local repo → remote`, `repo → ~/.dotfiles`); self-update is the opposite direction and gets its own script.
- Merge/rebase/stash/reset recovery strategies — explicitly rejected (see Risks). Non-fast-forward is a skip, not a fix.
- Pulling from the vault — self-update pulls from the **git remote** only.

## Risks / open questions

- **Clobbering local work** → guarded: dirty worktree (`git status --porcelain` non-empty) → log + skip, exit 0. This is the primary failure mode and the core guard test.
- **Diverged history / local commits** → `--ff-only`; non-ff → log + skip, exit 0. No unattended merge/rebase/reset (rejected: rewriting history or auto-merging in a timer is how you lose work silently).
- **Network failure** → `git fetch` failure → log + skip, exit 0 (transient; next slot retries).
- **Redundant setup churn** → setup runs **only if `HEAD` moved**; an already-current repo is a no-op (exit 0, no setup).
- **Real setup failure surfacing** → if setup *runs and fails*, exit **non-zero** so `systemctl --user status dotfiles-selfupdate` and the journal show it (distinct from the benign skips above).
- **Repo path assumption** → script defaults `DOTFILES_REPO_DIR=$HOME/Projects/dotfiles` (the documented repo location); overridable via env.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1 — Dirty worktree: `dotfiles-selfupdate.sh` against a repo with uncommitted changes logs a skip and exits 0 **without** fetching or running setup.
- [ ] AC2 — Diverged/non-ff: when local `main` cannot fast-forward to the remote, it logs a skip and exits 0 without running setup.
- [ ] AC3 — Already current: when `HEAD == @{u}`, it exits 0 and does **not** run setup.
- [ ] AC4 — Clean fast-forward: when the remote is ahead and ff is possible, it fast-forwards and runs setup exactly once.
- [ ] AC5 — Setup failure: when the injected setup command exits non-zero, the script exits non-zero.
- [ ] AC6 — Opt-in gate: `DOTFILES_AUTODEPLOY` unset → `setup-linux.sh` neither installs nor enables the timer; `=1` installs + enables; `=0` disables + removes. (Asserted by a guard test on the setup block / unit presence.)
- [ ] AC7 — Units valid: `systemd/dotfiles-selfupdate.{service,timer}` exist, the timer is `OnCalendar=daily Persistent=true`, and `systemd-analyze verify` (or a structural grep guard) passes.
- [ ] AC8 — Cross-shell: `dotfiles-selfupdate.sh` passes shellcheck and the bats suite under both bash and zsh; `.ps1` files are ASCII-only.

## References

- GitHub: `dotfiles#295` (work-gate, bitácora Project)
- Reference pattern in-repo: `systemd/hive-upgrade.{service,timer}` + its install block in `setup-linux.sh` (AI-023 / hive#176)
- ADR: `docs/adr/adr-005-*` (`~/.dotfiles` is a deployed copy, not a git repo — why self-update targets `~/Projects/dotfiles`)
- Related patterns: `00_meta/patterns/pattern-config-defaults.md` (opt-in defaults), `pattern-shell-standards.md`

<!-- archived 2026-06-09 — PR: https://github.com/mlorentedev/dotfiles/pull/303 -->
