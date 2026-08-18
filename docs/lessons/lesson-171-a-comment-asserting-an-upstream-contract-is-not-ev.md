---
id: lesson-171-a-comment-asserting-an-upstream-contract-is-not-ev
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 171: A comment asserting an upstream contract is not evidence of that contract

**Context**: `chain-local-hook.sh` chains the machine-wide GUARD dispatcher to repo-local hooks. Where a repo has no local hook for the stage, it falls back to invoking `pre-commit hook-impl` directly, deliberately omitting `--hook-dir`. A comment above that branch explains the choice: *"omitting --hook-dir is its supported dispatcher path (upstream marks that branch 'git 2.54+ hooks')"*.

**Problem**: On the installed pre-commit 4.4.0 the call is fatal. `hook_impl` feeds `hook_dir` straight into `os.path.join(hook_dir, f'{hook_type}.legacy')` *before* running any configured hook, so `None` raises `TypeError` and the stage exits 3. The fallback fires for every stage a repo has not locally installed, so `git commit` aborted in **any** repo on the machine carrying a `.pre-commit-config.yaml` — confirmed in a throwaway repo where the commit was simply never created. The dispatcher is installed globally through `core.hooksPath`, making the blast radius every repository on the box, and it shipped green because the tests only exercised the branch where a local hook exists.

**Solution**: Pass `--hook-dir "$common_dir/hooks"` — the value pre-commit's own generated hook computes, and one already resolved a few lines earlier for the local-hook probe. Filed as BUG-055 with the guard it implies: a test that runs a real `git commit` in a temp repo carrying a config and **no** installed hooks, asserting the commit exists afterwards.

**Rule**: Same family as the structural-assertion lesson above, one level up. There, prose inside an artifact was mistaken for evidence about that artifact; here, prose inside *our* code was mistaken for evidence about *someone else's*. A sentence naming an upstream behaviour is a claim to be tested against the installed version, never a citation. The cheapest available test — running by hand, once, the exact command the code will run — would have caught this before it reached a machine-wide hook.
