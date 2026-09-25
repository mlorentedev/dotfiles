---
id: lesson-294
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, git, windows, worktree, hooks]
---

# 294 — A Windows worktree is not automatically readable by POSIX Git

## What happened

`check-doc-paths.sh` discovers instruction files through `git ls-files`. From
WSL Bash inside an external worktree created by Git for Windows, POSIX Git read
the worktree's `.git` file as a literal POSIX path even though its `gitdir:`
pointer used `C:/...`. Discovery returned no files and the guard exited with a
usage error; the pre-commit hook then appeared to hang at the documentation
check.

## The rule

When a Bash guard needs Git metadata, first use its normal POSIX Git path. If
that cannot resolve the worktree and `git.exe` is available, run Git for
Windows from the worktree directory instead. Test the no-argument discovery
path, not only explicit file arguments: the hook invokes the former.

