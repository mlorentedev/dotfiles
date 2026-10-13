---
id: "lesson-409-a-test-that-stages-in-the-real-index-races-every-other-git"
type: lesson
status: active
title: "A test that stages in the real index races every other git"
created: "2026-10-13"
---

# A test that stages in the real index races every other git

## Context
`check-doc-paths.sh` finds instruction files through the git index. To prove it, three cases in
`tests/check-doc-paths.bats` wrote a probe file into the working tree, ran `git add -N` on it and
removed it again in `teardown` with `git rm --cached`.

## The Trap
Each `git add -N` takes `index.lock` on the repository's one index. Under `bats --jobs`, the
cases ran beside each other and beside any other git process in the checkout, such as an editor,
a status line or a second session. Whoever lost the race failed with
`Unable to create '.../index.lock': File exists`. With one other git process looping in the same
worktree, the suite failed in 7 of 10 runs on `main`. Each failure looked like a flaky test, not
a shared resource, and a failure before `teardown` could leave the probe staged in the real
index.

## The Solution
- `setup` copies the real index to `$BATS_TEST_TMPDIR/index` and exports `GIT_INDEX_FILE` to
  point at it. Every git the test runs, the guard's included, inherits it, so a staged probe is
  visible to discovery and to nothing else.
- `teardown` stops calling `git rm --cached`: nothing ever reached the real index.
- A case asserts it: `env -u GIT_INDEX_FILE git ls-files` does not list the probe.

Measured with the same loop: 0 failures in 15 runs.

## Takeaways
- A test that needs staged state stages it in a private `GIT_INDEX_FILE`, never in the checkout's
  index. The index is a single file that every git process on the worktree locks.
- "Flaky under `--jobs`" means two tests share a resource. Find the resource; do not retry.
