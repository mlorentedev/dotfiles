---
id: lesson-284
type: lesson
status: active
created: "2026-09-22"
owner: manu
tags: [lesson, testing, mutation, verification, silent-failure, go]
---

# 284 — A mutation "killed" by a compile error proves nothing

## What happened

Applying the CLI-078 round-2 review, the fix added a finding kind and two mutations
were run to prove the new tests pinned it. The harness reported both **killed**. The
same run's plain `go test` line said `[build failed]`: the fix referenced a struct
field that existed only on a stacked branch, so the package did not compile. With the
mutation reverted, and with it applied, `go test` failed identically, for a reason
that had nothing to do with either mutation.

## Why it happens

A mutation harness asks one question: *did the tests fail?* A build failure answers
yes. So a broken build turns every mutation into a kill, and the report is
indistinguishable from a suite that catches everything. It is the same shape as an
absence check that passes because the check never ran (lesson 283): the harness
reads "failed" and the failure it reads is not the one it asked about.

## The rule

A mutation counts as killed only when the mutated tree **builds** and a **test**
fails. Check the build first and report a broken build as its own outcome, never
as a kill:

```bash
if ! go build ./... >/dev/null 2>&1; then echo "$name: BUILD-BROKEN"
elif go test ./pkg/ >/dev/null 2>&1; then echo "$name: SURVIVED"
else echo "$name: killed"; fi
```

And run the unmutated suite green once before the first mutation. A baseline that
is not green makes every kill meaningless.

## References

- CLI-078 review round 2, applied in #1600
- lesson 283, same family: a check that reads "failed" without asking why
