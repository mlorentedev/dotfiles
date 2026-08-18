---
id: lesson-102-per-repo-git-hooks-can-t-enforce-a-machine-wide-in
type: lesson
status: active
created: "2026-06-17"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 102: Per-repo git hooks can't enforce a machine-wide invariant — core.hooksPath + a chaining dispatcher is the keystone (GUARD-001)

**Context**: Building GUARD-001 so agent memory (`MEMORY.md`, `memory/`, session handoffs) can only ever be committed to the vault, never leak into a code repo — the gap that let `MEMORY.md` reach the ts-bridge repo.

**Problem**: A per-repo `.git/hooks/pre-commit` only protects repos where it was installed; it cannot enforce an invariant across *every* repo on the machine, and a freshly-cloned or newly-`init`ed repo is unprotected by default. `pre-commit install` is per-repo too.

**Solution**: One global `git config --global core.hooksPath <dir>` points every repo at a single tracked dispatcher dir. Because a global `hooksPath` makes git ignore each repo's `.git/hooks/`, the dispatcher runs its global concern (the memory-sink guard) and then `exec`s the literal `.git/hooks/<type>` — the chaining is what keeps per-repo hooks (gitleaks, the pre-commit framework) alive. `dotf init` also bakes a `MEMORY.md`/`memory/` block into a new repo's `.gitignore` so it is born convention-correct. Wiring is **safety-first**: a global setting has machine-wide blast radius, so an unrelated pre-existing `core.hooksPath` is never clobbered — `dotf doctor` WARNs and preserves it, wiring only when unset and only under `--fix`.

**Rule**: To enforce an invariant across *all* repos, reach for global `core.hooksPath` + a chaining dispatcher, never a per-repo hook. When a tool writes a global/shared setting, treat any pre-existing value as sacred: detect-and-warn, wire only when absent, and gate the write behind `--fix` — never silently overwrite something with machine-wide reach.
