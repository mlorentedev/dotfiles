---
generated: true
generated_from: 00_meta/skills/using-git-worktrees/SKILL.md
generated_sha: c138ea5fd9f93613
id: using-git-worktrees-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: using-git-worktrees
description: Use when starting feature work that needs isolation from current workspace,
  or before executing implementation plans that should not affect the main working
  directory.
keywords: [git worktree, external worktree, isolated worktree, worktree safety]
paths: []
---
# Using Git Worktrees

> **Locally hardened (diverged from the superpowers upstream).** The directory-selection
> default was changed from *nested* (`.worktrees/`) to *external sibling* after a nested
> worktree was committed as a `160000` gitlink by an auto-committing repo (obsidian-git).
> Do not silently re-sync this file from upstream — re-apply the hardening. See
> [[00_meta/runbooks/runbook-worktree-safety|runbook-worktree-safety]] for the why.

## Overview

Git worktrees create isolated workspaces sharing the same repository, allowing work on
multiple branches simultaneously without switching.

**Core principle:** Worktrees live **outside** the repository's working tree. External
placement + auto-commit detection + post-create leak gate = deterministic isolation that
does not depend on any ignore rule being correct.

**Announce at start:** "I'm using the using-git-worktrees skill to set up an isolated workspace."

## The one rule that prevents leaks

**Never create a worktree inside the repository's own working tree.** A nested worktree is
a directory containing a `.git` file; any tool that runs `git add -A` (obsidian-git,
watch-and-commit hooks, CI auto-commit) stages it as a `160000` **gitlink** — embedding the
worktree into the parent branch like a submodule. A reactive `.gitignore` / `.git/info/exclude`
**loses the race** against a 5-minute auto-commit timer. The only deterministic defence is
physical: put the worktree where the parent repo cannot see it.

## Directory Selection (deterministic)

### 1. Default — external sibling

```bash
repo_root="$(git rev-parse --show-toplevel)"
project="$(basename "$repo_root")"
path="$(dirname "$repo_root")/${project}-wt-${SLUG}"   # sibling of the repo, OUTSIDE it
```

This is the default for **every** repo. It matches the existing convention
(`dotfiles-wt-<slug>`) and needs no `.gitignore` cooperation.

### 2. Optional — a configured worktree root

If `CLAUDE.md` / `AGENTS.md` specifies a worktree root, honour it:

```bash
grep -i "worktree.*\(director\|root\)" AGENTS.md CLAUDE.md 2>/dev/null
# e.g. path="$WORKTREE_ROOT/$project/$SLUG"
```

Any location is acceptable **as long as it is outside every repo's working tree.**

### 3. Nested (`.worktrees/`) — discouraged, gated

Only if the user explicitly insists on a nested worktree, AND the auto-commit gate below
passes (repo has NO auto-committer), AND `.worktrees/` is git-ignored. Otherwise refuse and
use external.

## Safety Gates

### Gate A — auto-commit detection (BEFORE creation)

```bash
auto_commit=0
[ -d "$repo_root/.obsidian/plugins/obsidian-git" ] && auto_commit=1   # Obsidian vault
git -C "$repo_root" config --get-regexp '^.*autocommit' >/dev/null 2>&1 && auto_commit=1
ls "$repo_root/.git/hooks/" 2>/dev/null | grep -q . && \
  grep -rqs 'add -A\|add \.' "$repo_root/.git/hooks/" && auto_commit=1
```

**If `auto_commit=1`:** external placement is **mandatory**. Do not offer nested. Do not
rely on excludes.

### Gate B — post-create leak check (AFTER creation, BEFORE declaring ready)

Both checks MUST pass:

```bash
# 1. The new worktree must NOT appear in the parent repo's status.
git -C "$repo_root" status --porcelain | grep -q "$(basename "$path")" && echo "LEAK: visible in parent status"

# 2. No worktree path may be tracked as a gitlink (mode 160000) in the parent index.
git -C "$repo_root" ls-files --stage | awk '$1==160000 {print "LEAK: gitlink ->", $4}'
```

**If either reports a LEAK:** the worktree is nested/embedded — move it out with
`git worktree move` and run [[00_meta/runbooks/runbook-worktree-safety|runbook-worktree-safety]]
to remove the gitlink. Do not proceed.

## Creation Steps

### 1. Create the worktree (new branch)

```bash
git worktree add "$path" -b "${USER_PREFIX}/${SLUG}"   # e.g. mlorentedev/research-agents
cd "$path"
```

### 2. Run project setup (auto-detect)

```bash
[ -f package.json ]   && npm install
[ -f Cargo.toml ]     && cargo build
[ -f requirements.txt ] && pip install -r requirements.txt
[ -f pyproject.toml ] && poetry install
[ -f go.mod ]         && go mod download
# Knowledge vault / docs repo: no build step — skip.
```

### 3. Verify clean baseline

Run the project-appropriate check (`npm test`, `cargo test`, `pytest`, `go test ./...`).
For a non-code repo (vault), a structural validator or "clean tree" is the baseline.

**If it fails:** report and ask whether to proceed. **If it passes:** report ready.

### 4. Report

```
Worktree ready at <full external path>
Branch <prefix>/<slug>  (forked from <base>)
Gate A: <auto-commit repo? y/n>   Gate B: no leak ✓
Ready to work in parallel.
```

## Teardown (deterministic)

```bash
git worktree remove "$path"            # add --force only if intentionally discarding work
# If a nested fallback was ever used: remove its line from .git/info/exclude
git -C "$repo_root" ls-files --stage | awk '$1==160000'   # must be empty
git worktree prune
```

## Quick Reference

| Situation | Action |
|-----------|--------|
| Any repo | External sibling `<repo>-wt-<slug>` (default) |
| Auto-commit repo (obsidian-git etc.) | External **mandatory** — Gate A blocks nesting |
| User insists on nested | Allow only if no auto-committer AND `.worktrees/` ignored |
| Gate B reports gitlink/visible | Move out + run runbook-worktree-safety |
| Baseline fails | Report + ask |
| Done | `git worktree remove` + re-audit gitlinks + prune |

## Common Mistakes

### Nesting the worktree inside the repo
- **Problem:** `git add -A` embeds it as a `160000` gitlink; reactive excludes lose the race.
- **Fix:** External sibling, always. Gate A makes it mandatory for auto-commit repos.

### Trusting an exclude to protect a nested worktree
- **Problem:** Non-deterministic — the auto-commit timer can fire before the exclude lands.
- **Fix:** Don't rely on ignores for isolation; rely on physical placement (Gate A/B).

### Declaring "ready" without the leak gate
- **Problem:** A leak is silent for up to one auto-commit cycle, then it's in history.
- **Fix:** Gate B (parent status + gitlink audit) before reporting ready.

### Hardcoding setup commands
- **Problem:** Breaks on projects using different tools.
- **Fix:** Auto-detect from project files.

## Red Flags

**Never:**
- Create a worktree inside a repo's working tree (especially an auto-commit repo)
- Use a reactive exclude/.gitignore as the isolation mechanism
- Skip Gate B before declaring ready
- Skip baseline verification

**Always:**
- Place worktrees externally (sibling `<repo>-wt-<slug>`)
- Run Gate A (auto-commit detection) before creating
- Run Gate B (status + `160000` audit) after creating
- Tear down with `git worktree remove` + gitlink re-audit + prune

## Integration

**Called by:**
- **brainstorming** (Phase 4) — REQUIRED when design is approved and implementation follows
- **subagent-driven-development** / **executing-plans** — REQUIRED before executing tasks
- Any skill needing an isolated workspace

**Pairs with:**
- [[00_meta/runbooks/runbook-worktree-safety|runbook-worktree-safety]] — leak detection + remediation
- **finishing-a-development-branch** — cleanup after work complete
