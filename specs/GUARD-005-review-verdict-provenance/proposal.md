---
id: "GUARD-005-review-verdict-provenance"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-21"
issue: "mlorentedev/dotfiles#1157"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, guard, review, provenance, archive]
template_version: "1.0"
---

# GUARD-005-review-verdict-provenance

## Why

`dotf spec review` launches **detached**, so it can only ever report that the runner *started*. When
a run ends without writing a verdict, the command exits 0 and the **previous round's `review.md`
stays on disk** — carrying a verdict, a `reviewed_sha` from a tree that may no longer exist, and
nothing to distinguish it from a fresh review that reached the same conclusion. `dotf spec archive`
then accepts it.

This is not hypothetical. Measured **2026-08-22, twice within an hour**, on both pool arms:

| Arm | What happened | What the launcher reported |
|---|---|---|
| `pi` / `nan/deepseek-v4-flash` | 51 tool executions, `agent_settled`, no verdict written | success |
| `agy` / `gemini-3.1-pro-high` | `{"status":"ERROR","error":"Individual quota reached"}` | success |

In the same session this nearly caused an archive on a stale PASS: only the file's **mtime** gave it
away. A guard that depends on a human noticing a timestamp is not a guard.

Two adjacent facts make it worse. The reviewer **authors its own frontmatter**, so `reviewed_sha` is
a *claim* rather than a measurement — one round stamped `date: 2026-08-20` on a review run on the
21st. And #1153 records the structural version of the same hole: nothing compares the merged sha to
the reviewed one.

## What

The launcher writes `review-request.json` beside the spec **before the reviewer starts**, recording
the three facts the reviewed party does not author: the head it was pointed at, the pool member
resolved, and the **SHA-256 of `review.md` as it stood at launch**.

`dotf spec archive` then refuses when:

1. **the digest has not moved** — the reviewer wrote no verdict, and what is on disk is the previous
   round's;
2. **the frontmatter's `reviewed_sha` disagrees with the launched head** — the claim contradicts the
   measurement;
3. **the verdict is signed by a different pool member than the one launched** — which
   `checkReviewerPool` cannot see, because both ids are admitted.

**The digest is what makes this runner-agnostic, and that is the central design choice.** Parsing the
transcript for a completion signal would need a schema *per runner* — `pi` emits
`{"type":"agent_settled"}`, `agy` emits `{"event":"result",...}` — and `docs/lessons/lesson-215`
records precisely that failure: a parser written for one runner read the other's review as empty.
*"Did the file change?"* needs no schema and cannot rot when a runner changes its output format.

**The sidecar is committed, not ignored.** `.gitignore` drops `review-transcript.jsonl` (1.4 MB of
reproducible detail); the sidecar is four fields of measurement. Keeping it means an archived spec
carries *what was asked, of whom, at which head* without the bulk — which is the auditable half of
the tension #1010 names, where the docs call transcripts auditable and the ignore rule calls them
disposable.

## Out of scope

- **Detecting failure at launch time.** A detached launcher provably cannot observe an outcome that
  happens after it returns. The guard therefore lives where the verdict is **consumed** (`spec
  archive`), not where it is produced. This is the same reasoning that puts #1153's sha check there.
- **#1153's merged-vs-reviewed comparison.** Adjacent and complementary; this spec supplies the
  sidecar it would build on.
- **Making the reviewer's own frontmatter trustworthy.** It stays a claim; the fix is to have a
  measurement to check it against, not to believe it harder.
- **Reviving a run that died.** The guard reports; re-running is the operator's call.

## Risks / open questions

- **A guard that refuses too much gets bypassed.** `--force-without-review` already exists and every
  message names it, so the escape is explicit rather than invented under pressure. An **absent**
  sidecar is silent by design: reviews predating this guard and hand-written ones stay governed by
  the verdict, staleness and pool checks. Refusing them would invalidate every review on disk to
  close a hole none of them demonstrably has.
- **A damaged sidecar must not read as an absent one.** That would drop the guard exactly when the
  file carrying it is broken — the C15 shape. An unparseable sidecar is a loud error.
- **Writing the sidecar must not be able to block a review.** A read-only spec dir would otherwise
  turn a guard into a liability, so a write failure is a warning and the launch proceeds. It
  degrades to "provenance not asserted", which is honest.
- **`HeadSHA` returns `""` outside a checkout** rather than erroring, for the same reason.

## Acceptance criteria

- [x] The launcher writes `review-request.json` before the reviewer starts, holding the launched
      head, the resolved pool member, and the digest of `review.md` at launch.
- [x] A run that writes no verdict is refused at archive, with a message naming *that* cause rather
      than the sha mismatch it would otherwise surface.
- [x] A verdict claiming a `reviewed_sha` the launcher never pointed at is refused, contrasting the
      claim with the measurement.
- [x] A verdict signed by a pool member other than the one launched is refused.
- [x] No sidecar means the guard is not asserted — not that the review is bad.
- [x] An unparseable sidecar fails loudly rather than silently disabling the guard.
- [x] A first review (no previous `review.md`) records an empty digest, so it is never mistaken for
      an unchanged one.
- [x] A sidecar write failure warns and lets the review launch.
- [x] Verified end to end against the real binary, not only in unit tests.

## References

- Bitácora board: mlorentedev/dotfiles#1157
- `docs/lessons/lesson-215-a-parser-for-one-runner-reads-the-other-runners-re.md` — why the digest
  beats transcript parsing
- Related: #1153 (merged-vs-reviewed sha, builds on this sidecar), #1010 (transcript lifecycle),
  #1156 (the fallback arm that could not cover the primary's silent failures)
