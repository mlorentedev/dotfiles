# Lesson 246 — A guard that reads its own evidence from a file allowed not to exist switches itself off exactly when it is needed

**Date:** 2026-08-30
**Context:** HARNESS-094 (#1383, PR #1388). `VerifyReviewProduced` was added to make `dotf spec review` fail when a foreground run ends without writing `review.md`. Its first version answered "what did review.md hold before this run?" by reading `review-request.json` — a sidecar whose own writer is documented to fail with a *warning*, never a refusal. Caught in review before it shipped.
**Category:** guards, degradation, review

## What happened

The sidecar exists for a good reason and degrades for a good reason. It records
the head the reviewer was pointed at and the digest `review.md` carried at
launch, and `WriteReviewRequest` deliberately does not refuse when it cannot be
written:

> A failure to write it is a WARNING, not a refusal: losing the guard is worse
> than losing the review, but refusing to launch because a sidecar could not be
> written would make the guard a liability the first time a spec dir is
> read-only.

That reasoning is sound for the *archive* gate, which treats a missing sidecar as
"not asserted" and moves on. It is not sound for a second guard that reads the
same file. Compose the two and one cause — a spec dir that cannot be written —
takes out both, and the second one silently:

| | with sidecar | without sidecar |
|---|---|---|
| archive gate | refuses a review.md that never moved | says nothing (honest: nothing was asserted) |
| launcher check, first version | refuses | **says nothing, and exits 0** |

A read-only spec dir is exactly the state where a review is most likely to end
with nothing written, and it was the state in which the new check turned itself
off.

## The rule

**Evidence a guard needs at time T must be captured at time T, not re-read from
storage that was permitted to fail.** The launcher held the pre-launch digest in
its own hand before it launched anything; nothing had to reach disk for it to
know. The fix was to keep it:

```go
// Held in memory for the whole launch, because the sidecar write below is
// allowed to fail with a warning.
digestBefore := spec.ReviewDigest(specDir)
```

and to pass it in, so `VerifyReviewProduced` never opens the sidecar at all. The
persisted copy remains what a *later* process reads, which is the only reader
that has no alternative.

Two smaller things fell out of it, both worth keeping:

- **One definition of the fact.** `ReviewDigest` is now the single function that
  answers "the digest of review.md", used by the writer and the checker.
  Previously the same `fileDigest(filepath.Join(...))` expression appeared twice.
- **Fail-open and fail-closed are per-caller, not per-value.** `fileDigest`
  returns `""` for both "absent" and "unreadable", documented as failing *open*
  into "the digest moved" for the archive gate. The launcher's check reads the
  same `""` and fails *closed* — no file means no verdict. Same helper, opposite
  postures, each correct for its caller.

## How it was caught

Not by a test — every test passed, because they all seeded a sidecar. A reviewer
read the composition, not the function, and asked what happens when the warning
path fires. Worth noting for the shape: this class survives unit tests by
construction, because the fixture that sets up the guard's happy path is the same
fixture that hides the degraded one. Ask what the *other* branch of every
warning-level failure does to the checks that come after it.
