---
id: lesson-161-a-linked-worktree-s-checkout-is-not-self-contained
type: lesson
status: active
created: "2026-08-07"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 161: A linked worktree's checkout is not self-contained — its `.git` is a file, and tools that assume a directory all fail together

**Context**: Working the dotfiles repo through `git worktree`, the standing convention here because parallel sessions share the main checkout. Two unrelated attempts hit the same wall the same day: bind-mounting a worktree into a Linux container to run git commands, and copying a worktree on Windows with `robocopy /XD .git`.

**Problem**: In a linked worktree `<toplevel>/.git` is not a directory — it is a one-line file containing an absolute `gitdir:` pointer into the main checkout's `.git/worktrees/<name>`. Everything downstream of "the `.git` directory is here" breaks, and each breaks in its own idiom. Bind-mounting the worktree alone carries the pointer but not its target, so every `git` call inside the container aborts. `robocopy /XD .git` never excludes it, because `/XD` matches **directories** and this `.git` is a file — so the pointer gets copied and the copy silently references a path that means nothing where it landed. The absolute, Windows-shaped `gitdir:` makes both worse: even mounting the target would not fix a path the container cannot resolve. The same assumption also lived in this repo's own code — `chain-local-hook.sh` resolved `$toplevel/.git/hooks/<type>` and so skipped every local hook in a worktree (#776), and `dotf doctor`'s vault gate `isDir`-tests the same path and reports SKIP for a vault it never checked (#806).

**Solution**: Treat a worktree as a checkout whose git state lives elsewhere. Ask git rather than the filesystem — `git rev-parse --git-common-dir` for shared state such as hooks, `--git-dir` for per-worktree state — and never infer the layout from `<toplevel>/.git`. For containers, mount the main checkout too, or work from a plain clone. For `robocopy`, exclude it as a **file** (`/XF .git`) as well as a directory.

**Rule**: Ask of any code touching `.git`: does it still hold when `.git` is a *file*? The check is cheap and the failures are silent — a skipped hook, a green SKIP, a copy that references a machine that is not this one. And note the shape: a single wrong assumption about a filesystem layout does not produce one bug, it produces a family of them across every tool that shares the assumption, all latent until someone actually uses the layout. Related: #776, #806, #761.
