---
id: lesson-166-git-rev-parse-echoes-an-option-it-does-not-underst
type: lesson
status: active
created: "2026-08-08"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 166: `git rev-parse` echoes an option it does not understand back at you, and exits 0

**Context**: Fixing hook resolution in a linked worktree (`#776`) by asking git for the shared git dir instead of assuming `$toplevel/.git`.

**Problem**: `--git-common-dir` arrived in git 2.5. On anything older, `git rev-parse` does not error — it treats the unrecognised flag as a revision-ish argument and prints it back:

```console
$ git rev-parse --git-nonexistent-dir
--git-nonexistent-dir
$ echo $?
0
```

So a naive `common_dir=$(git rev-parse --git-common-dir)` yields the literal string `--git-common-dir` as its "path", resolves hooks beneath it, finds nothing, and skips silently — the exact failure being fixed, reintroduced for older git.

**Solution**: Validate that the answer is a real directory before using it, and fall back to the classic layout otherwise. Separately, `--git-common-dir` answers *relative to the cwd* in an ordinary checkout and *absolute* in a linked worktree; asking from `$toplevel` makes both forms resolve against the same base, which avoids needing `--path-format=absolute` (git 2.31+) and so introduces no version floor of its own.

**Rule**: `git rev-parse` is a parser, not a validator: an exit code of 0 means "I produced output", not "I understood you". Never consume its output without checking the *shape* of what came back. This generalises to any tool whose contract is "echo the resolved form of your arguments" — the ones that gracefully pass through what they cannot interpret are exactly the ones that turn a version-compatibility problem into a silent wrong answer.
