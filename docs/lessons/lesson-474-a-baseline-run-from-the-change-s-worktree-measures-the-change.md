---
id: "lesson-474-a-baseline-run-from-the-change-s-worktree-measures-the-change"
type: lesson
status: active
title: "A baseline run from the change's worktree measures the change"
created: "2026-10-17"
---

# A baseline run from the change's worktree measures the change


**Context:** #2136 moves gh from a `system` catalog entry to a `# mise: cli` pin. The live check was the usual one: plan with the installed `dotf` (the baseline), plan and apply with the branch build, and re-plan for `0 to change`.

**Problem:** The "baseline" plan, run with the released 0.73.0 from the branch worktree's `cli/`, already reported `mise config to write; to install: gh`, the branch's own change. `converge` resolves its checkout from the working directory first (`convergeCheckout`): the dotfiles checkout the cwd is in wins over `DOTFILES_REPO_DIR`. The released binary had read the branch's `versions.conf`. The same plan with `--repo` set to a checkout of main reported something else: `to prune, asked for by no pin: gh@2.102.0`, an install left by an earlier `mise exec gh@2.102.0` probe.

**Solution:** A baseline names both halves explicitly: the released binary AND a checkout of the base (`--repo <main checkout>`), run from a directory outside every worktree (`cd ~`). The branch run names its own worktree the same way. Then the two plans differ only in the change under test.

**Why:** A binary is half of what a tool run measures; the input it reads is the other half, and `dotf` picks its input from the cwd. Swapping the binary while the cwd stays in the branch compares the change with itself, and agreement then looks like a precondition.

**Amended 2026-10-18 (#2711):** `convergeCheckout` now prefers the declared `DOTFILES_REPO_DIR` when it exists, and prints which tree it converges when the cwd is another checkout, so the mechanism above is historical for `converge`. The discipline stands: `dotf deploy` and `dotf harness mirror` still read the cwd first (#1418), and a baseline names its binary and its tree whatever the default.
