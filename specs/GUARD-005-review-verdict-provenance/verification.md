---
tags: [spec, verification, templates]
created: "2026-08-21"
---

# Verification - GUARD-005-review-verdict-provenance

## Evidence

All 7 `features.json` verifiers executed and passing; each propagates the runner exit status and
pins its test by unique name (lesson 217).

**The end-to-end run is the part unit tests cannot establish**, so it was done against a binary
built from this branch, reproducing the exact failure that nearly caused a bad archive earlier the
same day:

```console
1. seeded a stale PASS in review.md (reviewed_sha deadbeef..., date 2026-08-20)
2. dotf spec review <spec> --reviewer agy/...   # reviewer wrote nothing (quota exhausted)
3. dotf spec archive <spec>

Error: review.md has not changed since the review was launched - the reviewer wrote no verdict
what is on disk is the PREVIOUS round's, which is not a review of this change
re-run /adversarial-review (a run ended by a turn limit or a rate limit leaves exactly this state),
or pass --force-without-review
```

The sidecar written at launch, verified on disk:

```json
{
  "reviewed_sha": "a33f856814c9bbf3c923eda2968bfd7b1f4cc92e",
  "reviewer": "agy/gemini-3.1-pro-high",
  "requested_at": "2026-08-22T05:17:39Z",
  "review_digest_before": ""
}
```

## Test status

- `go build ./... && go vet ./...` -> clean
- `go test ./... -count=1` -> every package ok; the spec package suite passes unchanged, which is
  the evidence that inserting a provenance check ahead of the verdict and staleness checks did not
  disturb them
- The proof artifacts (seeded review.md, sidecar, transcript) were removed after the run; the
  working tree carries only the change.

## Decisions made during implementation

- **The guard lives where the verdict is CONSUMED, not where it is produced.** A detached launcher
  provably cannot observe an outcome that happens after it returns, so no amount of work in
  `spec review` can close this. The same reasoning places #1153's sha check in `archive`.
- **The digest, not the transcript.** A completion signal parsed from the transcript needs a schema
  per runner - `pi` emits `{"type":"agent_settled"}`, `agy` emits `{"event":"result",...}` - and
  lesson 215 records that exact parser reading the other runner output as empty. "Did the file
  change?" needs no schema and survives a runner changing its format.
- **Order matters inside the gate.** The digest check runs FIRST: a reviewer that wrote nothing
  leaves the previous round sha in place, so the sha mismatch would be reported instead and send
  the reader hunting for a rebase that never happened.
- **The sidecar is committed, not gitignored.** The transcript is 1.4 MB of reproducible detail and
  is ignored; the sidecar is four fields of measurement. Keeping it means an archived spec carries
  what was asked, of whom, at which head - the auditable half of the tension #1010 names.
- **An absent sidecar is silent; a damaged one is loud.** Silence keeps the guard from invalidating
  every review already on disk. Loudness on damage is C15: reading a broken file as "absent" drops
  the guard exactly when the file carrying it is broken.
- **A sidecar write failure warns and lets the launch proceed.** A read-only spec dir would
  otherwise turn a guard into a liability; it degrades to "provenance not asserted", which is
  honest.

## Promotion candidates

- [ ] Lesson for `docs/lessons/`? no - the transferable point (a detached launcher cannot observe
      its own outcome, so the guard belongs at the consumer) is recorded in the proposal and in the
      code comment where someone changing it will read.
- [ ] ADR-worthy? no - no new architectural decision; this closes a hole in an existing gate.
- [ ] Vault pattern? not yet - if a second detached-launcher guard needs the same shape, promote it.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved to `specs/archive/GUARD-005-review-verdict-provenance/`
- [ ] Bitacora #1157 closed with the PR link
- [ ] `/adversarial-review GUARD-005-review-verdict-provenance` run and PASSing
