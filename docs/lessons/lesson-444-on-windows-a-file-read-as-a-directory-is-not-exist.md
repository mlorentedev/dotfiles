---
id: "lesson-444-on-windows-a-file-read-as-a-directory-is-not-exist"
type: lesson
status: active
title: "On Windows a file read as a directory is ErrNotExist: decide absence with Stat"
created: "2026-10-15"
owner: manu
tags: [go, windows, cross-platform, fail-closed]
---

# On Windows a file read as a directory is ErrNotExist: decide absence with Stat

**Context:** CLI-118 PR 3 (#2556) makes `dotf worktree done` and `sweep` refuse to remove a worktree whose Claude Code transcript is recent. After review, a transcript store that cannot be read must refuse too, while a store that does not exist still means "no session". `RecentSession` told the two apart with `errors.Is(err, fs.ErrNotExist)` on `os.ReadDir`'s error.

**Problem:** The test made the store unreadable portably by pointing it at a regular file. On macOS and Linux `ReadDir` fails with ENOTDIR and the refusal held. On `windows-latest` both fail-closed tests failed: the error matched `fs.ErrNotExist`, so the unreadable store read as absent and `done` removed the worktree. The code never asked whether the path existed; it inferred it from an error whose mapping is the platform's.

**Solution:** On any `ReadDir` error, `os.Stat` the path: only a Stat that says `ErrNotExist` makes it absent, and every other case returns the error. The rule is now "absent means Stat cannot find it", which holds on every OS, instead of "absent means ReadDir's error looks like not-found".

**Why:** A fail-closed check is only as closed as its narrowest "this is fine" branch, and an error-class test is that branch. Error classes are mapped per OS from native codes (`ERROR_PATH_NOT_FOUND`, `ERROR_DIRECTORY`), so a branch keyed on one inherits every native code that maps to it. When absence is the permissive answer, establish it with the call that asks exactly that question. The Linux and macOS runs could not see this; only the Windows leg could.
