---
id: lesson-273
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, testing, cross-platform, review, guards]
---

# 273 — The fix and the detector can be the same edit

## What happened

`test (windows-latest)` was red on #1515. Two sibling functions disagreed about path separators:
`ResolveMainRepoRoot` normalised git's answer through `filepath.Dir/Abs`, so it returned
backslashes on Windows; `ResolveWorktreeRoot` returned git's **raw stdout**, and git emits
forward slashes on every platform including Windows. On Linux the two conventions coincide,
which is why nothing local could see it.

The failing test compared `t.TempDir()`'s Windows 8.3 short form (`C:\Users\RUNNER~1\…`) against
git's long form (`C:\Users\runneradmin\…`). The obvious repair is a helper that resolves both
spellings before comparing:

```go
func samePath(a, b string) bool {
    return normalizePath(a) == normalizePath(b)   // EvalSymlinks, then filepath.Clean
}
```

That is correct for the 8.3 problem and **it makes the separator bug invisible**, because
`filepath.Clean` on Windows rewrites `/` into `\`. With `samePath` in place, the fixed code and
the unfixed code both pass.

The repair for the symptom would have removed the only thing that could detect the disease, in
the same commit that cured it. `CodeRabbit` independently proposed `os.SameFile`, which has the
identical property: any identity comparison is blind to how the path was spelled.

## Why it matters

**A comparison has a shape, and normalising it to make it correct also decides what it can no
longer see.** The separator convention had to be asserted *separately*, against the raw
resolver output, never through the comparison:

```go
if path != filepath.Clean(path) { … }   // fires only on Windows; that is the point
```

The general form: when a test fails for two reasons at once, fixing the comparison usually
addresses the *softer* one, and the harder one leaves with it silently.

## The shape recurred five more times in one session

The same session produced five further instances, four of them the author's own, and **not one
was found by reading**:

| What | What caught it |
|---|---|
| `render_region` written as a pipe swallowed a `return 1` | three unrelated coverage tests |
| Two of three `! grep -q` in a new bats test asserted nothing — bash exempts `!` from `set -e` | the repo's own `tests/guard-bats-negation.bats` (#1034) |
| Changing a return type broke the `//go:build !linux` test file; `go test ./...` on Linux stayed green because it never compiles it | `GOOS=darwin go vet` |
| A new wikilink guard reported correct state as broken — two link conventions exist, it knew one | running against the real tree, not a fixture |
| A mitigation counter that is non-zero on **every** Linux machine, and therefore signals nothing | the round-2 adversarial reviewer |

The last is the sharpest. It was written as the answer to a reviewer's finding that an
unreadable `/proc/<pid>/cwd` silently reads as "nobody is there"; the fix counted those and
reported the count. `/proc/1/cwd` is unreadable to every non-root caller, so the count is never
zero — a permanently-lit warning light is a warning light that has been removed.

## What to do about it

1. **When a repair makes a test pass, ask what it stopped asserting.** If the answer is "the
   thing that was broken", the assertion has to live somewhere the repair does not reach.
2. **Mutate the fix, not the tests.** Revert the change and re-run: if the suite stays green,
   the detector is gone. Assert the mutation's anchor first, or a no-op patch reports as a clean
   result — see [[lesson-267]].
3. **A guard that cannot fail on the platform CI runs is still worth having** — it just has to
   be honest about where it fires. Say so in the file, so nobody later "simplifies" it for being
   permanently green locally.
4. **Run guards against the real tree.** Two of the five above were only reachable there; a
   fixture is built to match the assumption being tested.

## Related

- [[lesson-267]] — a mutation harness must prove the mutation landed.
- [[lesson-268]] — a refused question and a negative answer print the same thing. This is its
  family: the reassuring result and the broken one are the same bytes.
- [[lesson-272]] — fail-closed worktree garbage collection, whose Gate f supplied the last row.
