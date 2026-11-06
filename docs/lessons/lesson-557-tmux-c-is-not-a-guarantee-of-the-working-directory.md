---
id: "lesson-557-tmux-c-is-not-a-guarantee-of-the-working-directory"
type: lesson
status: active
title: "tmux -c is not a guarantee of the working directory"
created: "2026-11-06"
---

# tmux -c is not a guarantee of the working directory

## Context

`dotf spec review` runs the reviewer in a disposable review worktree (HARNESS-142). The launch started the
detached tmux session with `tmux new-session -c <worktree>`, and nothing else put the reviewer there.

## Problem

On macOS, one tmux server had been running for days, started from another session's worktree. Every new
session it created had an unreadable working directory, whatever `-c` said:

- `pwd -P` printed `.`;
- mise warned `Current directory does not exist or is not accessible: .`;
- a pi reviewer died on startup with `error: The current working directory was deleted`.

An agy reviewer ran normally against the same server. It uses absolute paths and `cd`s itself, so it worked
from the wrong directory without anyone noticing: its init event named the other worktree. The acceptance run
of #3213 used agy, so it verified agy's tolerance of a wrong start directory, not the start directory.

A fresh server (`tmux -L <name> new-session -c <worktree>`) started in the worktree, and pi ran.

## Solution

`spec deadline`, the first process inside the session, changes into `--worktree` itself before it starts the
reviewer. It refuses, naming the directory, when it cannot. The reviewer inherits deadline's working
directory, so every runner starts in the worktree without depending on tmux (#3220).

## Takeaway

A working directory handed over by a long-lived server is a request, not a fact. Have the first process you
own enter the directory itself, and fail loudly if it cannot.

A run that passes with one client proves only that the client tolerated the environment. Before reading it as
coverage, check the environment from inside the run: the working directory, the user, the env.
