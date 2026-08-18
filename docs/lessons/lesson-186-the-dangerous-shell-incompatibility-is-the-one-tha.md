---
id: lesson-186-the-dangerous-shell-incompatibility-is-the-one-tha
type: lesson
status: active
created: "2026-08-09"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 186: The dangerous shell incompatibility is the one that answers wrongly instead of failing

**Context**: A sweep for git worktrees across every repo under two parent directories, run from an agent's zsh: `for base in /home/manu/Projects /home/manu/Projects/Workspace; do for r in "$base"/*/; do …; done; done`. It printed nothing, and "nothing" was read as "no repository has extra worktrees" — while `kubelab` had one the whole time.

**Problem**: two zsh behaviours, both silent. `/home/manu/Projects/Workspace` does not exist on this machine, and zsh's default `NOMATCH` makes an unmatched glob an **error that aborts the entire compound command** — so the first `base` never finished and the second never ran. Earlier in the same session the sibling gotcha had already fired: zsh does not word-split unquoted parameters, so `set -- $pair` inside a loop assigned the whole line to `$1`, and a later `git diff … -- $files` passed a newline-joined blob as one pathspec. Neither printed an error. Each produced a plausible, empty, wrong result.

The repo's prohibited-patterns table exists for exactly this class, and every row in it — `echo -e`, `declare -g`, `&>/dev/null`, `((count++))` under `set -e` — describes a construct that **fails loudly** in the other shell. Nothing in it covers constructs that succeed while returning the wrong answer, which is the more dangerous half, because a crash gets fixed and a wrong answer gets acted on.

**Solution**: for throwaway analysis, run the loop under `bash -c 'shopt -s nullglob; …'` rather than porting glob semantics in the head; the corrected sweep found the worktree immediately. For anything that ships, the existing rules apply: quote every expansion, never rely on word splitting, and guard a glob whose directory may not exist. The deeper fix is not in this entry: the table that would have warned lives in `.claude/CLAUDE.md`, which `.gitignore` excludes, so it is neither versioned nor visible to any agent other than Claude on this one machine.

**Rule**: When a bash/zsh difference is catalogued, record which way it fails. A construct that errors out is a nuisance; one that silently returns an empty set is a defect generator, because emptiness reads as a finding. Two to treat as first-class: an unmatched glob under zsh's default `NOMATCH` aborts the whole command rather than expanding to nothing, and an unquoted `$var` does not word-split. Both turn a loop into a no-op that reports success. Corollary for agents: before believing an empty result from a shell sweep, re-run it in the other shell — the cost is one command and the failure mode is a confident wrong answer.
