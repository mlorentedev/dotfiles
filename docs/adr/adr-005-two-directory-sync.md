---
id: "dotfiles-adr-005-two-directory-sync"
type: adr
adr: "005"
title: Two-Directory Sync Pattern
tags: [adr, dotfiles, sync, architecture]
status: accepted
created: "2026-02-22"
owner: manu
---

# ADR-005: Two-Directory Sync Pattern

## Context

The dotfiles serve two purposes: (1) stable local installation for daily use, and (2) active development with frequent commits. Running a development repo as `$HOME/.dotfiles` risks breaking the shell environment during git operations (rebase, branch switches, failed merges).

Approaches evaluated:

| Approach | Stability | Dev workflow | Complexity |
|----------|-----------|-------------|------------|
| Single dir (`~/.dotfiles` = repo) | Low — git ops affect live config | Simple — one location | Low |
| Two dirs with manual copy | High | Tedious — easy to forget | Medium |
| Two dirs with `dotfiles-sync` | High | One command | Medium |
| Nix/Home Manager | High | Declarative but steep learning curve | High |

## Decision

Maintain two directories:

- **`~/.dotfiles/`** — Stable local installation. Shell configs source from here. Never modified directly (except by sync).
- **`~/Projects/dotfiles/`** — Development repository. All editing, commits, and PRs happen here.

The `dotfiles-sync` command handles bidirectional sync:
1. Compares timestamps of `.age` files, `env-mapping.conf`, audit log
2. Copies newer files in both directions
3. `git push` from `~/Projects/dotfiles`
4. `git pull` to `~/.dotfiles`

Secret operations (`secrets_add`, `secrets_rotate`) auto-sync to the repo if `DOTFILES_REPO_DIR` is set.

## Consequences

### Positive

- **Shell stability:** `~/.dotfiles` never sees mid-rebase states, detached HEADs, or uncommitted experiments
- **Safe development:** Can break things in `~/Projects/dotfiles` without affecting the running shell
- **Atomic updates:** `dotfiles-sync` moves all changes at once, not incrementally
- **Git-clean repo:** Development repo stays clean for PRs; local install doesn't need git at all

### Negative

- **Two copies of everything:** Disk usage doubled (~small for dotfiles, but conceptually heavier)
- **Sync discipline required:** Forgetting `dotfiles-sync` after changes means directories diverge
- **Conflict potential:** If both directories are modified independently, sync uses timestamp-based resolution (not merge)

### Mitigations

- `secrets_add` and `secrets_rotate` auto-sync immediately (most common change path)
- `dotfiles-sync --secrets-only` for quick secret-only sync
- Setup scripts handle initial clone/deployment of `~/.dotfiles`
