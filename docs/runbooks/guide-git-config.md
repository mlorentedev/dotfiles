---
id: guide-git-config
type: runbook
status: active
created: "2026-10-08"
---

# Git config and GitHub credentials

> Every OS. The one manual step is `gh auth login`: it is a consent, and nothing should give it for you. Everything after it is `dotf`'s, and `dotf doctor` checks it.

## Who owns which file

| File | Owner | Written by |
|---|---|---|
| `git/dotfiles.gitconfig` (repo) → `~/.config/git/dotfiles.gitconfig` | the dotfiles | `dotf deploy` (`gitconfig` entry), replaced whole |
| `~/.gitconfig` | git, gh and you | `git config --global`, `gh auth setup-git`, `dotf hooks install` and converge's `git-hooks` step (`core.hooksPath`), and the `include.path` line `dotf converge` adds |

`~/.gitconfig` is co-owned, so nothing should deploy over it (lesson 366). The dotfiles' settings reach git through one line in it:

```ini
[include]
	path = ~/.config/git/dotfiles.gitconfig
```

Both setup scripts run `dotf deploy` and then `dotf converge --only git-config`; neither copies a file over `~/.gitconfig`. A `dotf` older than the `--only` flag (0.65.0 and before) rejects the call, and setup warns: re-run setup once a release moves `DOTF_VERSION`, or run the two steps below by hand with a newer `dotf`.

The deployed file is deliberately not `~/.config/git/config`. Git reads that path by itself, and on a machine with no `~/.gitconfig`, `git config --global` writes into it; the next deploy would then erase what git wrote.

## The GitHub credential helper

`gh auth setup-git` writes the helper with the absolute path of the gh that runs it: `/opt/homebrew/bin/gh` on macOS, `/usr/bin/gh` from apt, the quoted `gh.exe` path on Windows. A bare `!gh auth git-credential` works in a terminal and fails in every process that does not inherit the shell's PATH: a GUI app such as Obsidian (obsidian-git), launchd, cron, a scheduled task. Measured on the Mac on 2026-10-08: a Dock-launched app gets `PATH=/usr/bin:/bin:/usr/sbin:/sbin`, gh is not on it, and git falls back to asking for a username (#2207).

To check what such a process sees:

```bash
env -i HOME="$HOME" PATH=/usr/bin:/bin:/usr/sbin:/sbin GIT_TERMINAL_PROMPT=0 \
  git -C <a GitHub checkout> ls-remote origin HEAD
```

Exit 0 means the helper works without your shell.

To check the include is effective, ask what a commit sees. `git config --global` alone ignores includes:

```bash
git config --global --includes user.name
```

## New machine

1. `gh auth login`. Grant what the board tooling needs as well; #2204 tracks requesting every scope in this one step.
2. `dotf deploy` installs `dotfiles.gitconfig`.
3. `dotf converge --only git-config` adds the include and, with gh logged in, runs `gh auth setup-git`. A second run reports `0 changed`.

## Checks and repair

- `dotf doctor`, section `Git config (global)`:
  - **FAIL**: the include is missing, the file it names is not deployed (run `dotf deploy`), or the helper is not gh's absolute form while gh could fix it.
  - **WARN**: the helper is wrong and gh is not on PATH or not logged in. The message names the login.
- `dotf doctor --fix` runs the same repair as converge. Both use the predicate in `cli/internal/gitconfig` (lesson 368).
