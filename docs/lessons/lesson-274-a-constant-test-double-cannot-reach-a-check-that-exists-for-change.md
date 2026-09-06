---
id: lesson-274
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, testing, toctou, test-doubles, review]
---

# 274 — A constant test double cannot reach a check that exists for change

## What happened

`dotf worktree sweep` asks Gate f — "is a live process sitting in this worktree?" — **twice**. Once
to pick candidates, and again inside `reapSingleWorktree`, under the lock, immediately before the
directory is removed. The second call is a TOCTOU guard: it exists for the window in which a shell
`cd`s in while the sweep is still deciding.

Gate f is injected through a package-level seam, and five tests drove it. An adversarial review
deleted the second check outright:

```go
if err != nil || absWT == absCwd || hostProcessInside(absWT).Inside {   // before
if err != nil || absWT == absCwd {                                      // mutated
```

The whole suite stayed **green**. Two review rounds had passed over it.

## Why it happened

Every one of those tests installed a double that returned the *same answer every call*. That
answer can be `true` or `false`, and **neither reaches the second check**:

| The double always says | What happens |
|---|---|
| `true` ("someone is inside") | the *first* gate refuses; control never gets near the re-check |
| `false` ("nobody is inside") | the re-check runs and agrees; deleting it changes no outcome |

The second check only matters when the answer **changes between the two calls**. A constant double
makes it unreachable-in-effect: present in the trace, but incapable of altering any assertion. The
mutation is invisible not because the test forgot to assert, but because the *shape of the double*
excluded the only scenario in which the code under test does anything.

The fix is one line of double, and it is the whole lesson:

```go
calls := 0
hostProcessInside = func(string) GateFReading {
    calls++
    return GateFReading{Inside: calls > 1}   // absent, then present
}
```

Plus the anchor, because this test can now go vacuous in a new way — if the fixture never becomes
reapable, Gate f is consulted zero times and the test passes having exercised nothing:

```go
if calls < 2 {
    t.Fatalf("Gate f was consulted %d time(s); the fixture never reached the re-check", calls)
}
```

## Why it matters

**The shape of a test double is part of the property being tested, not plumbing.** Choosing a
constant stub silently narrows the reachable state space, and the narrowing does not show up as a
gap — it shows up as a passing test with a confident name.

The general form: **a guard that exists because a value can change cannot be tested by a double
that holds it still.** Re-checks under a lock, retry loops, cache invalidation, optimistic
concurrency, revalidation after a slow call — every one of them is a second read whose entire
purpose is disagreeing with the first. Stub them constant and you have tested the first read twice.

There is a companion trap in the same family. `isCandidateForReap` was a bool wrapper around the
real gate that **no production caller used**; two of the tests pinned Gate f's refusal on it. A
seam into dead code reports coverage it does not have, so before trusting a gate's tests, check
that the function they call is the one production calls.

## What to do about it

1. **When a check is a re-read, the double must change its answer.** Name the call ordinal in the
   stub. If a constant stub can satisfy the test, the test is not about the re-read.
2. **Count the calls and assert the count.** It is the anchor for this class: it converts "the
   guard did not fire" into "the guard was never reached", which are different results that a
   green suite renders identically — see [[lesson-267]].
3. **Mutate by deletion, specifically.** Removing a guard entirely is a sharper probe than altering
   its condition; a condition change can still be caught by luck, a deletion cannot.
4. **Check the seam lands in production code.** `grep` the function the test drives for callers
   outside `_test.go` files.

## Related

- [[lesson-267]] — a mutation harness must prove the mutation landed.
- [[lesson-273]] — the fix and the detector can be the same edit; this is its sibling, where the
  *double* is what deletes the detector.
- [[lesson-272]] — fail-closed worktree garbage collection, the change these guards belong to.
