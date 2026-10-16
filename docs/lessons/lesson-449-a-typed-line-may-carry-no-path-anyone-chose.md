---
id: "lesson-449-a-typed-line-may-carry-no-path-anyone-chose"
type: lesson
status: active
title: "A typed line may carry no path anyone chose"
created: "2026-10-16"
owner: manu
tags: [lane, herdr, shell, windows, cli]
---

# A typed line may carry no path anyone chose

**Context:** `dotf lane next` types `dotf lane exec --agent X --prompt-file <path>` into the successor's pane (CLI-119, #2528). The first version guarded the path with a "plain word" check and wrote the prompt under the repository's git directory. Lesson 438 had already made the *binary* path a plain word for the same reason.

**Problem:** The guard refused any repository under a path with a space, so `lane next` failed outright for `C:\Users\Jane Doe\...` and the error told the user to change a `PromptDir` that was derived, not configurable (#2537). The first idea, moving the prompt to a space-free directory, only moves the failure: every absolute location the tool could pick (git directory, `$TMPDIR`, `$HOME`, the worktrees root) can hold a space on Windows. Two reviews missed it because the guard was exercised on synthetic space-free paths only.

**Solution:** The line names a fixed file relative to the pane's directory, `.dotf-lane-prompt.md`, and the prompt is written *into the new worktree* (excluded from `git status` like the lease) after the backend says where that worktree is. A relative name has no space by construction. The commands printed for a person are a different channel: they go into one shell the person chose, so paths are double-quoted there, and a character live inside double quotes is refused.

**Why:** When a tool types into a shell it does not know, the only path it can be sure of is one it did not choose. Guarding a path (accept or refuse) is weaker than not sending one. Keep a plain-word check only for what cannot be avoided, and make it per OS: a backslash is a separator on Windows and an escape in zsh and bash (#2587). A test for such a guard needs a path with a space in it, not a path that merely passes.
