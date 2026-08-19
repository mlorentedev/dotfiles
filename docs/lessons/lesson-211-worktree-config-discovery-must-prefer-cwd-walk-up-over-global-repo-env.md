# Lesson 211 — Worktree Config Discovery Must Prefer CWD Walk-Up Over Global Repo Env

> **Date:** 2026-08-18  
> **Area:** CLI / Harness / Worktree Isolation  
> **Keywords:** worktree, path resolution, config discovery, LoadTriggers, env fallback

## Symptom

When running tests or CLI commands (`dotf harness suggest`) inside an isolated worktree (`dotfiles-wt-<slug>`), new trigger definitions added to `harness/triggers.json` inside the worktree were ignored. Commands instead resolved against the parent repository root defined in `$DOTFILES_REPO_DIR` or `~/.config/dotfiles/machine.json`.

## Root Cause

`LoadTriggers` called `env.RepoDir()` first. In development environments where `DOTFILES_REPO_DIR` is set to the primary checkout (`~/Projects/dotfiles`), the global path resolution wins over the active worktree checkout. This caused commands run from inside a worktree to read stale configuration from the parent repository instead of the worktree's modified files.

## Resolution

Inverted the discovery precedence in `LoadTriggers`:
1. If an explicit `repoRoot` argument is passed (e.g. from unit tests), read from it directly.
2. Otherwise, walk up from the current working directory (`os.Getwd()`) to locate any enclosing `harness/triggers.json` within the active worktree.
3. Fall back to `env.RepoDir()` only if no local config exists in the directory tree.
4. Fall back to embedded `defaultTriggersJSON` if neither is found.

## Rule

When resolving repo-local configuration files in CLI tools, always walk up from `cwd` before consulting global machine environment variables to preserve git worktree isolation.
