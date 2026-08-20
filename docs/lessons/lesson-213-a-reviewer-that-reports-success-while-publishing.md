---
id: lesson-213
type: lesson
status: active
created: "2026-08-20"
owner: manu
tags: [lesson, ci, review, guards, inference]
---

# 213 — A reviewer that reports success while publishing nothing, in two shapes

**Context**: `pr-agent` is this repo's fallback reviewer, shipped so CodeRabbit's
account-wide quota stops being the constraint on throughput. On 2026-08-20 six pull
requests in one working session carried a green `review` job and **no review**:
#1096, #1100, #1101, #1103, #1104, #1105. The checks list said pass. Finding it took
a log dive.

**Root cause**: the NaN cluster allows **five concurrent requests per model**, shared
with pi's TUI, `qq` and hive embeddings. Parallel PR activity exhausts it, and
PR-Agent swallows the resulting failure into a clean exit.

**The part worth generalising is that it failed in two different shapes**:

    # shape 1 — an explicit error
    litellm.RateLimitError: deepseek-v4-flash concurrency limit: max 5 simultaneous requests.
    Failed to review PR: Failed to generate prediction with any model of [...]

    # shape 2 — silence
    02:57:13  "PR diff"        <- last line the container emitted
    03:03:18  "Complete job"   <- six minutes later, no error at all

**A guard written against the error would have caught half of them.** Grepping the
log for `RateLimitError` catches #1100 and misses #1101 entirely, because there was
nothing to find. The guard that works asks the *consequence* — did a review get
published? — and is therefore blind to how the failure arrived, including shapes
nobody has seen yet.

**Generalises to**: any check whose subject is a third-party tool. You do not
control its failure modes and you cannot enumerate them, so assert on the artifact
it was supposed to produce, never on the symptom you happened to observe. The same
reasoning is why `review-attestation` asks *"did a review happen"* rather than
*"did the reviewer report an error"*.

**Two hypotheses were tested and refuted first**, which is worth recording because
both were plausible and both were wrong: that the head being a merge commit
suppressed the run (`push_trigger_ignore_merge_commits` — refuted: every PR in both
the working and failing sets had `parents=2`), and that the runs were cancelled
(refuted: they reported `success`).

**Fixed by**: #1109 (fail the job when no review was published), #1110 (halve the
demand by turning off `auto_improve`, serialise inference repo-wide, and add a
same-tier fallback on a second model — the limit being **per model** is what makes
that a real fallback rather than a rename).

**Related**: [[lesson-212]] — an invalid instrument is indistinguishable from an
absent guard. This is its sibling: a *silent* instrument is indistinguishable from a
passing one.
