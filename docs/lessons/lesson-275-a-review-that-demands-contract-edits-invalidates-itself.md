---
id: lesson-275
type: lesson
status: active
created: "2026-09-05"
owner: manu
tags: [lesson, sdd, adversarial-review, spec-archive, gates]
---

# 275 — A review that demands contract edits invalidates itself

## What happened

BUG-093 spent four consecutive review rounds at FAIL. Round 5 — the first run after the launcher
learned to resolve a `base_sha` (#1535) — returned **PASS-WITH-GAPS**, and its recommendation list
was headed *"Recommended next steps (before archive)"*. Four of the seven asked for edits to
`features.json` and `proposal.md`.

Applying them, then archiving:

```
Error: review.md is stale: features.json, proposal.md changed after reviewed_sha 5a68096
re-run /adversarial-review against the current head, declare `review: waived` ..., or pass --force-without-review
```

The archive gate measures staleness against exactly `proposal.md`, `tasks.md` and `features.json`.
So the review had asked for the edits that invalidate the review.

Measured on the diff, before assuming the gate was wrong: **zero acceptance criteria and zero
`behavior` fields changed**. All ten changed lines were `evidence` (6) and `verification` (4)
strings, plus prose. That measurement is what made the next step decidable.

## Why it happened

The heading came from the skill template, not from the reviewer's judgment — `SKILL.md:282` shipped
it, and `:153` said a Major requires a *"spec update ... before archive"*. Every reviewer, on every
spec, was being told to demand pre-archive contract edits regardless of verdict.

The template had no notion of **which set a fix lands in**, so it could not distinguish the case
where a contract edit is the whole point (FAIL: fix, then re-review) from the case where it is
self-defeating (PASS: the verdict *means* the gaps are tracked, not fixed).

## The trap: the coarse check looks like the bug

The obvious fix — make staleness field-granular, so `evidence`/`verification` edits stop
invalidating a review — is wrong, and attractively so. `verification` is the command the reviewer
**ran**. Exempting it opens precisely the bypass the code comment at `review.go:187` already names:
get a pass, rewrite in the working tree, archive. And `proposal.md` prose has no field structure to
key on, so half the case is unfixable that way anyway.

**The gate was right.** A verdict is about a *state*. The review's own findings quote text — "f8
says 9, it is 10", "the *What* claims more than the code does" — that must still be on disk when
the spec archives, or the archived artifact is internally inconsistent. Editing the reviewed files
to correct a stale number creates a worse inconsistency than the one it removes.

Two other exits were also wrong: `--force-without-review` discards a review that exists and passed,
and `review: waived` declares the requirement does not apply at all — `checkReviewGate` returns on a
waiver with a reason *before* it calls `FindReview`, so a passing review sitting in the folder is
never even read. Both walk past a verdict that was earned, in the opposite direction from the stale
number.

## What to do instead

Restore the contract files to `reviewed_sha`, and disposition each recommendation in
`verification.md` — applied, ticketed, or declined with a reason. `verification.md` is **excluded
from the staleness check for exactly this purpose**; it is where Definition of Done §4 lands for a
spec review. Improvements outside the contract set — code, tests, scripts committed beside the spec
— stay, because none of them is measured.

## The rule

> **Where the fix lands decides when it can be applied.** `proposal.md`, `tasks.md` and
> `features.json` are the contract set. Under FAIL, editing them is the point and a re-review
> follows. Under a passing verdict the set is closed: recommendations become dispositions in
> `verification.md` or follow-up tickets. A review that asks for a contract edit alongside a passing
> verdict has asked to be thrown away.

## Fixed in

- `harness/skills/adversarial-review/SKILL.md` and the vault source `00_meta/skills/` — the
  "(before archive)" heading dropped, the Major clause reworded, the contract set and the routing
  rule stated (#1533).
- `cli/internal/spec/review.go` — the staleness refusal now names the exit that keeps the review
  **first**, ahead of the three overrides, guarded by
  `TestStaleRefusalOffersTheExitThatKeepsTheReview` (mutation-proven against the old message).

## Related

A second, smaller finding from the same spec, recorded because it cost a false verification: the
CI shellcheck job covers `scripts/` and the root scripts only, so the `.sh` files that live inside
spec folders are unlinted — and those are the scripts that *produce archive evidence*. `flip-proxy.sh`
shipped with `for f in $FILES`, a row in this repo's own prohibited-pattern table, which left the
tree flipped and made `mutate.sh` report five caught mutations as survivors. Ticketed as #1544.

See also [274](lesson-274-a-constant-test-double-cannot-reach-a-check-that-exists-for-change.md) —
the same spec, the same failure mode one layer down: a check that exists for a state change cannot
be reached by a double that never changes.
