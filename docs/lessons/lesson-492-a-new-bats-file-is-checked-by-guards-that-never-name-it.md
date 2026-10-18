---
id: "lesson-492-a-new-bats-file-is-checked-by-guards-that-never-name-it"
type: lesson
status: active
title: "A new bats file is checked by guards that never name it"
created: "2026-10-18"
---

# A new bats file is checked by guards that never name it


**Context:** #2793 added `tests/precommit-config-real.bats` and three tests to `tests/precommit-config.bats`. Before pushing, the run did what lesson 347 asks: it ran every bats file that `git grep` found referencing the changed files. All 12 tests passed, in bash and in zsh.

**Problem:** CI's `test` job failed on two suites the grep could not find. `tests/bats-pwsh-gate.bats` refuses a `command -v pwsh` gate, because a mise shim without a version passes it and then fails every call. `tests/guard-bats-negation.bats` refuses a bare `! cmd` line. Neither suite names the new file. Each one globs `tests/*.bats` and reads whatever it finds, so a reference search returns nothing. The cost was one CI round, about 15 minutes on a congested runner pool.

**Solution:** When a change adds or edits a bats file, also run the suites that read every bats file:

```bash
bats tests/guard-*.bats tests/bats-*.bats   # 84 tests, under a minute
```

**Why:** Lesson 347's grep finds the tests that read a *product* file by name. A test file is read the other way round, by guards that enumerate the suite. That set is fixed and small, so running all of it is cheaper than working out which guards apply.
