---
id: "lesson-410-bash-c-reads-no-startup-file"
type: lesson
status: active
title: "bash -c reads no startup file, so no file of ours can set its PATH"
created: "2026-10-13"
---

# bash -c reads no startup file, so no file of ours can set its PATH

## Context
`mise activate` lives in `.zshrc` and `.bashrc`, and only interactive shells reach it. An
agent, a script or a command over ssh runs a non-interactive shell, which never ran it, so a
green "at their pin through mise" from doctor said nothing about what those callers run
(#2013 W2b). The fix puts mise's shims at the head of PATH in a file every such shell reads.

## The Trap
Which file a non-interactive shell reads depends on the shell and on how it was started, and
the forms that read nothing look identical to the ones that do:

| Invocation | Reads |
|---|---|
| `zsh -c` | `~/.zshenv` only (`$ZDOTDIR/.zshenv` when set) |
| `bash -lc` | `/etc/profile`, then the first of `~/.bash_profile`, `~/.bash_login`, `~/.profile` |
| bash started by sshd for a command | `~/.bashrc`, non-interactively: the rc's interactive check returns before anything below it |
| `bash -c`, `sh -c` | nothing (bash reads `$BASH_ENV` if set, which nobody sets) |

On macOS, `/etc/profile` and `/etc/zprofile` run `path_helper`, which rebuilds PATH from
`/etc/paths` and moves the inherited entries after it, so a login shell's PATH is not the one
it was given.

A probe that starts `bash -c` from an empty environment therefore always reports the shims
missing, whatever the dotfiles deploy, and a probe that inherits doctor's own PATH always
reports them present, because doctor ran from an activated terminal.

## The Solution
- `.zsh/path.sh` is the one POSIX `sh` file that moves the shims, then `~/.local/bin`, to
  the front of PATH, removing any later copies. `.zshenv` sources it, `.profile` sources it
  last (after the lines that prepend `~/.local/bin`), and `.bashrc` sources it before its
  interactive check.
- `bash -c` and `sh -c` are covered by inheritance only: they get the shims when their parent
  is one of the shells above. Cron is not such a parent, so its PATH belongs to the job
  definition, not to a startup file.
- Doctor's probe starts `zsh -c` and `bash -lc` with only `HOME` and `PATH=/usr/bin:/bin`, so
  it measures what the deployed files give a fresh shell, and leaves `bash -c` out on purpose.
  `tests/shell-path-head.bats` runs the real shells the same way.
