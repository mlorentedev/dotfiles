---
id: "lesson-491-filepath-glob-reads-an-unreadable-directory-as-no-match"
type: lesson
status: active
title: "filepath.Glob reads an unreadable directory as no match"
created: "2026-10-18"
---

# filepath.Glob reads an unreadable directory as no match

**Context:** The session-start brief warns when the newest lesson under `docs/lessons/` is older than the staleness limit. It found that lesson with `filepath.Glob` over `lesson-*.md` and `*/lesson-*.md` (#2690).

**Problem:** `filepath.Glob` returns only `ErrBadPattern`. Any I/O error while it walks, such as permission denied on the directory or on a category, is dropped, so the result is an empty match list. The brief took that list as "no lessons", fell back to `docs/lessons.md`, and printed nothing. The warning was silenced, and the brief gave no sign that the check had not run.

**Solution:** Enumerate with `os.ReadDir`, which returns the error. A missing directory (`fs.ErrNotExist`) still means no lessons. Any other error becomes one line in the brief that names the directory and the error: session start never blocks, but it does not go quiet either. A test locks the directory and then a category, so both paths fail if the error is dropped.

**Why:** In Go, use `filepath.Glob` only where an unreadable directory may honestly read as no match. Where an empty result drives a decision ("nothing to warn about", "nothing to deploy"), read the directory yourself so an error stays an error. Other `filepath.Glob` call sites in `cli/` that drop errors this way are tracked in #2797.
