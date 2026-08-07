---
id: "BUG-028-setup-refuse-in-place"
type: spec
status: archived
created: "2026-07-09"
tags: [spec, proposal, setup, install-layout, guard, linux]
template_version: "1.0"
---

# BUG-028-setup-refuse-in-place

Refuse the in-place install layout (repo cloned into `~/.dotfiles`) that silently
corrupts setup, and point the README at the supported copy layout.

## Why

The README Quick Start cloned the repo INTO `~/.dotfiles` — which is the **deploy
target** setup writes into, not the checkout. With `CURRENT_DIR == DOTFILES_DIR`:

- The env-contract deploy `cp -f "$CURRENT_DIR/env-contract.json"
  "$DOTFILES_DIR/env-contract.json"` (setup-linux.sh) becomes `cp -f X X`, which
  fails "same file"; under `set -euo pipefail` setup aborts mid-run (exit 1), so
  env generation and the final `dotf doctor` never run.
- Worse, `scripts/install-git-hooks.sh` `deploy_git_hooks` clean-mirrors with
  `rm -rf "$dest"` then `cp -rf "$src/." "$dest/"`. When `src` and `dest` are the
  same directory it deletes the dispatcher and copies nothing back — **emptying**
  it while still logging `[SUCCESS]`. The next setup run then refuses ("no
  pre-commit dispatcher"), and doctor's fix hint ("run dotfiles setup") is
  circular. GUARD-001 ends up inert.

CI only exercised the `~/dotfiles-repo` layout (`tests/Dockerfile.integration`),
so the documented path was the untested one.

Decision (recorded this session): support **one** layout — clone to a separate
checkout dir and deploy into `~/.dotfiles` — and make setup refuse the in-place
layout with a clear message. Chosen over "make in-place safe" because a single
well-defined layout is smaller surface, and the code already assumes
`CURRENT_DIR != DOTFILES_DIR`.

Source: issue #695 (audit process-audit-2026-07-07 §4 P2 + §6 Q1).

## What

- **setup-linux.sh**: an early preflight (right after `DOTFILES_DIR` is defined,
  before any directory is created or file copied) fails fast with exit 1 and a
  three-line remediation when `CURRENT_DIR == DOTFILES_DIR`.
- **scripts/install-git-hooks.sh**: `deploy_git_hooks` detects `src -ef dest`
  (same directory) and no-ops the destructive mirror (refreshing exec bits only)
  — defense in depth, so the function is safe even if reached outside setup.
- **README.md**: the Linux Quick Start clones to `~/dotfiles-repo`, with a note
  that `~/.dotfiles` is the deploy target, plus the `git`/`curl` requirement.
- **Tests**: a behavioral `setup-linux.bats` test (in-place clone -> exit 1 with
  the message) and an `install-git-hooks.bats` test (same-dir mirror is a no-op,
  dispatcher not emptied).

## Out of scope

- **Windows.** `setup-windows.ps1` sets `$DotfilesDir = $PSScriptRoot`: the
  checkout *is* the dotfiles dir, so there is no separate deploy dir to collide
  with and no same-file copy. The bug is Linux-only; the asymmetry is intentional
  and mirrors the deploy-model difference between the two OSes. No Windows
  preflight is added.
- **Full README truth pass** (#677). This PR touches only the Linux Quick Start
  block and the `curl` requirement note that made setup die even earlier; the
  broader README audit stays in #677.
- **Migrating an existing in-place checkout** for a user who already cloned into
  `~/.dotfiles`. They get the clear refuse message and re-clone; setup does not
  attempt to move their checkout.

## Risks / open questions

- A user who followed the OLD README (cloned into `~/.dotfiles`) now hits the
  refuse on the next run. This is strictly better than the previous silent
  corruption (fail loud, with the exact fix command).

## Acceptance criteria

- [x] setup-linux.sh exits 1 with an actionable message when
      `CURRENT_DIR == DOTFILES_DIR`, before any destructive step.
- [x] `deploy_git_hooks` is a no-op (dispatcher preserved) when `src -ef dest`.
- [x] README Linux Quick Start clones to `~/dotfiles-repo` and names the
      `git`/`curl` requirement.
- [x] Behavioral bats: in-place refusal (setup-linux.bats) + same-dir no-op
      (install-git-hooks.bats); both green locally.
- [x] `bash -n` clean on both scripts.

## References

- GH issue: [#695](https://github.com/mlorentedev/dotfiles/issues/695)
- Related: #677 (README truth pass), #694/BUG-026 (setup must not write into the
  checkout — same "deploy != checkout" invariant, other direction)
- Integration harness (copy layout, the tested one):
  `tests/Dockerfile.integration`
