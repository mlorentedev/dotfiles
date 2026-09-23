---
id: lesson-286
type: lesson
status: active
created: "2026-09-23"
owner: manu
tags: [lesson, testing, mutation, go, resources, silent-failure]
---

# 286 — A mutation that deletes a loop bound is a memory bomb, not a failing test

## What happened

Mutation-testing the CLI-080 review fix (`reconcile --apply` now runs up to two
passes), one mutation deleted the condition that ended the loop. The loop was
written `for pass := 1; ; pass++` with its bound in the body. Against the test
double that never converges, the mutated loop never ended. Each pass appended a
printed plan to a `bytes.Buffer`, so the test binary grew until the kernel's OOM
killer stepped in. Four times, between 20:29 and 20:50: `cmd.test` at 8–9 GB RSS,
killed inside the terminal host's cgroup scope (`app-orca-…: Failed with result
'oom-kill'`). The terminal host went down with it, and so did the session running the
battery. Each restarted session re-ran the same battery and hit the same wall.

A second-order effect was worse. A battery left running in the background by a
killed session kept applying and reverting mutations after the next session had
"verified" the tree clean. The next checkpoint commit captured a mutated file.

## Why it happens

A mutation harness expects each mutant to end in one of two ways: tests pass or
tests fail. A mutant that removes a loop's exit gets a third outcome, *does not
end*. `go test`'s default 10-minute timeout is no protection when memory runs out
first, and the OOM killer chooses its victim by cgroup, not by blame. Here the
victim was the whole scope the harness ran in.

## The rule

1. **Bound loops in the header.** `for pass := 1; pass <= max; pass++`, with any
   early exit in the body *in addition*. Then no edit to the body, whether a
   mutation, a refactor or a merge, can make the loop unbounded.
2. **Run mutation tests in their own resource box**:
   `systemd-run --user --scope -p MemoryMax=1500M -p MemorySwapMax=0 -- env GOMAXPROCS=4 go test -p 1 -timeout 90s …`.
   A runaway mutant then kills only its own scope.
3. **One mutant per invocation, synchronously, restoring from `git checkout HEAD`
   in a `trap`.** A background battery outlives the session that started it, and
   a restore from a scratch copy is only as good as that copy.
4. **Before trusting "the tree is clean", check that no harness is still running**
   (`pgrep -af 'go test|\.test'`), not just that files match.

Refs: CLI-080 (`applyUntilConverged`), lesson 284 (a kill from a compile error
proves nothing; this is its resource-side sibling).
