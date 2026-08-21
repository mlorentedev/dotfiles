# Lesson 215 — A parser written for one runner reads the other runner's review as empty

**Date:** 2026-08-21
**Context:** Adversarially reviewing the orchestration evidence report before an ADR consumed it.

## What happened

Two reviewers from `harness/reviewer-pool.json` were run on the same document: `nan/deepseek-v4-flash`
via the `pi` runner, and `agy/gemini-3.1-pro-high` via `agy`. Both were launched by hand, because
`dotf spec review` takes a `<feature-id>` and there is no sanctioned way to review an arbitrary
document.

A parser was written to pull the review text out of the first runner's JSONL stream. Applied to the
second, it reported:

```
### chars=0 tools_run=0 done=False
```

The review was called empty and very nearly recorded as "agy produced nothing". It had in fact
produced a **complete 4,551-character review with seven findings, one of them a Blocker**. It was
caught only by dumping the raw stream and reading it.

## Why

The two runners emit different stream schemas:

| Runner | Envelope key | Text location |
|---|---|---|
| `pi` | `"type": "message_update"` | `assistantMessageEvent.text_delta` |
| `agy` | `"event": "step_update"` | `step_update.text_delta`, plus a final `result.response` holding the whole answer |

A parser looking for `type` finds nothing in a stream keyed on `event`, and returns an empty string
rather than an error — because "no matching records" and "no content" are the same value.

## The lesson

**This is lesson 213's shape one layer down.** 213 recorded a reviewer reporting success while
publishing nothing. Here the reviewer published everything and the *reading* of it produced nothing,
which is indistinguishable at the point of use. An empty extraction must be an error, never an
empty review.

The root cause is not the parser. It is that **the launcher already solves this and was bypassed**.
`cli/internal/spec/review_launch.go` encodes the per-runner differences precisely because they are
easy to get wrong — it documents that agy's `--print` consumes a value, so any flag placed between
it and the prompt swallows the prompt, and that `pi` defaults to the `google` provider so an
explicit `--provider` is mandatory. Hand-rebuilding an invocation from a launcher's source
reproduces every bug the launcher exists to prevent, and adds the ones the launcher never had.

**Generalised:** when a sanctioned path exists but does not cover your case, the gap is a ticket,
not an invitation to re-implement it. The re-implementation will be correct in the places you
thought about and silently wrong everywhere else — and "silently wrong" here means a review that
reads as absent.

Filed as #1138 (`dotf spec review --doc <path>`, reusing the pool, the launcher and the transcript
contract, with a normalised extraction across both schemas).

## See also

- `docs/lessons/lesson-213-a-reviewer-that-reports-success-while-publishing.md`
- `cli/internal/spec/review_launch.go` — the per-runner argv contract
- `harness/reviewer-pool.json` — the `operational_note` on reasoning models returning empty bodies
