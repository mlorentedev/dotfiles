---
id: "lesson-467-a-converge-from-one-worktree-undoes-what-another-branch-converged"
type: lesson
status: active
title: "A converge from one worktree undoes what another branch converged"
created: "2026-10-17"
---

# A converge from one worktree undoes what another branch converged


**Context:** PLAT-003 verified its new catalog entry with `dotf converge --only tools --repo <worktree>`, the branch build run against the branch checkout. Several sessions were working other branches on the same Mac.

**Problem:** The machine's deployed state is one copy, shared by every worktree. `~/.config/mise/conf.d/dotfiles.toml` already carried `zsh-autosuggestions`, which another session's branch (#2667, unmerged) had converged. This branch's `versions.conf` does not mark it, so the tools step rewrote the config without it and asked mise to prune the install, then failed with `after mise prune, still installed: http:zsh-autosuggestions@0.7.1`. The plan had said only "mise config to write", which reads as routine drift, not as "this removes another branch's tool". The baseline plan with the released `dotf` against `main` showed the same line, so the drift was not this branch's.

**Solution:** Before applying a converge step from a branch, save the files it owns (`cp` the mise config) and diff them after; restore anything the branch did not mean to change. Verify a new reconciler with `--only <that step>` when the shared steps carry another branch's state, and report the shared drift separately from the change under test.

**Why:** A worktree isolates the checkout, not the machine. Every converge is a write to the one machine, and the last branch to run wins. A plan line that names a file, not the entries it drops, hides that.
