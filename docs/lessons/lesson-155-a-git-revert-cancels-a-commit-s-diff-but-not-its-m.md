---
id: lesson-155-a-git-revert-cancels-a-commit-s-diff-but-not-its-m
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 155: A `git revert` cancels a commit's diff but not its message, and GitHub auto-close reads both

**Context**: #768 (the archive-on-merge dogfood sweep) merged into #767's own branch instead of `main` by mistake. Unstacking it meant `git revert`ing #768's squash commit out of #767's branch, then re-landing its content later as a fresh PR (#775) once #767 reached `main`.

**Problem**: #767 was never supposed to close #670 — its PR body said `Refs #670` throughout. It closed anyway, 14 minutes before the PR that actually finished the work even existed. Cause: GitHub's squash-merge commit message concatenates every individual commit's message from the branch, including ones a `revert` cancels the *diff* of but does not remove from history. #768's original squash commit had ended with a literal `Closes #670` trailer; that commit stayed an ancestor of #767's branch after the revert, so its trailer text got pulled into #767's own squash-merge commit message — and GitHub's auto-closer scans merge-commit messages, not just the live PR body.

**Solution**: Nothing to fix here — #670's early closure was harmless in the end, since #775 genuinely closed it minutes later for the right reason. Recorded as a lesson, not a bug: the outcome was correct by luck (a real closing PR existed shortly after), not by design.

**Rule**: A `git revert` undoes a commit's effect on the tree, not its presence in history or its message text. Anything that scans commit history for meaning — GitHub's closing-keyword auto-linker, a changelog generator, this very repo's own `check-spec-gate.sh` — can still see a reverted commit's words. When unstacking a mis-based PR, treat the revert as reversible *content*, not as if the commit never happened.
