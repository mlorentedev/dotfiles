---
id: "ADR-019-self-deploy-fast-forward-only"
type: adr
status: accepted
owner: manu
date: "2026-06-09"
extends: [adr-012-deploy-strategy-copy-with-drift-assertion]
tags: [architecture, decision, self-deploy, systemd, scheduled-task, opt-in, ops]
created: "2026-06-09"
---

# ADR-019: Self-deploy is fast-forward-only, opt-in, and idempotent-setup-gated

> An opt-in timer (systemd `--user` on Linux, Scheduled Task on Windows) keeps a machine
> converged to `origin/main` by pulling and re-running the idempotent setup — but only via a
> clean fast-forward, and only when `HEAD` actually moved. Closes [#295](https://github.com/mlorentedev/dotfiles/issues/295) (OPS-001).

## Status

Accepted.

## Context

Dotfiles changes merge to `main` but reach a machine only when someone manually `git pull`s and
re-runs `setup`. On a multi-machine setup that drift is silent and unbounded. Standing Order #1
("automate, don't instruct") calls for closing the loop. The hazard: an unattended `git pull` on a
repo where the user does real work can clobber uncommitted changes or local commits, and re-running
a full `setup` blindly is wasteful.

`dotfiles-sync.sh` was *assumed* to be the base, but it pushes (`local → remote`) and rsyncs
(`repo → ~/.dotfiles`) — the opposite direction. Self-deploy needs `remote → local repo` + setup.

## Decision

1. **Dedicated entrypoint** `scripts/dotfiles-selfupdate.sh` (+ `.ps1` parity), separate from
   `dotfiles-sync.sh` (SRP: opposite data-flow directions stay in different scripts).
2. **Fast-forward only.** `git fetch` then a `--ff-only` merge. Anything that is not a clean
   fast-forward — dirty worktree, diverged history, no upstream, unreachable remote — is **logged
   and skipped with exit 0**. The script never merges, rebases, stashes, or resets. An unattended
   history rewrite is how you lose work silently; it is explicitly rejected.
3. **Setup runs only when `HEAD` moved.** An already-current repo is a no-op (no setup churn).
4. **Real failures surface.** Only a setup command that *runs and fails* exits non-zero, so
   `systemctl --user status dotfiles-selfupdate` (and the journal) distinguish a genuine failure
   from a benign skip.
5. **Opt-in, default OFF.** Gated on `DOTFILES_AUTODEPLOY`: `1` installs + enables, `0` disables +
   removes, unset is a no-op. A normal `setup` run never installs the timer.
6. **Cadence: daily**, `Persistent=true` (a suspended/off machine runs the missed slot on next
   boot), `RandomizedDelaySec` to spread multiple machines.

## Consequences

- **Safe by construction:** the common "I have local edits" case can never be clobbered — it is the
  first guard, and it is the core regression test (incident → guard).
- A machine with local commits ahead of `origin` simply stops auto-deploying until the user
  reconciles — surfaced via the journal, not papered over.
- Breaking config changes are gated by PR review on `main`, not by the timer (out of scope here).
- Mirrors ADR-012's philosophy: deploy is mechanical and idempotent; correctness comes from refusing
  to act on an ambiguous state rather than from clever recovery.

## Alternatives rejected

- **Stash → pull → pop:** leaves unattended stash-pop conflicts; defeats the safety goal.
- **`--ff-only` + auto-rebase:** rewrites local history in a timer; silent and dangerous.
- **Hard reset to remote:** clobbers local work outright.
- **Extend `dotfiles-sync.sh` with a `--self-update` mode:** mixes opposite directions in one script.
