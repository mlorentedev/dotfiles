---
id: lesson-282
type: lesson
status: active
created: "2026-09-08"
owner: manu
tags: [lesson, tooling, github, bitacora, verification, silent-failure, gh-cli]
---

# 282 — A listing that truncates answers instead of failing, so the short read becomes the finding

## What happened

The `new-ticket` skill used `gh project item-list` twice, with hardcoded limits,
to answer two questions about the bitácora:

| Call site | Limit | What it was asked |
|---|---|---|
| Step 5, "confirm the fields landed" | `--limit 400` | is issue N on the board, with which fields? |
| Priority calibration | `--limit 1000` | what is the board's priority distribution? |

The board holds **3311 items**. `gh project item-list` returns them oldest-first
and stops at `--limit` — with no warning, no truncation flag, and exit 0. So the
first call sampled 12% of the board and the second 30%, and both answered
confidently from what they had.

Filing one ticket surfaced both failures in the same session:

- Step 5 printed **`NOT FOUND ON BOARD`** for an item whose four fields had just
  been set successfully, one command earlier, in the same script.
- An improvised board `ID` scan at `--limit 1000` reported `HARNESS-111` as the
  highest existing id while `HARNESS-120`…`134` already existed — an id proposal
  **23 numbers deep into occupied space**. Only the paginated REST title scan
  caught it, which is the sole reason no duplicate was filed.

## Why it survived

Both call sites had been correct once. `--limit 400` was written when the board
was smaller than 400 items, and it kept working, and then it kept *answering*.
There is no moment where a truncating listing announces that it stopped being
right: the failure mode is not an error, it is a smaller true answer to a
different question than the one asked.

That is the same class the repo's shell-compatibility table already names —
`${PIPESTATUS[0]}` expanding to nothing under zsh, an unmatched glob aborting a
loop, `set -- $var` producing one field instead of N. Every one of them returns
*empty or short*, and every one of them gets read as a finding. The distinguishing
property is not the language or the API; it is that **the failure path and the
"nothing matched" path are the same output**.

## The fix

- **Verify by identity, not by search.** Step 5 now reads the item by its node id
  — already in hand from the step that created it — via a single GraphQL query.
  Exact, cheap, and immune to board growth. Searching a list for something you
  already hold the address of is the mistake.
- **Delete a query you cannot make correct, don't grow the limit.** The obvious
  repair for the priority calibration was to read the whole board and assert the
  count. That was written, run, and it *failed*: reading 3311 items trips the
  GraphQL secondary rate limit outright (`API rate limit already exceeded`), and
  it would burn the quota the create step needs a minute later. The query was
  removed and replaced with a dated, explicitly-labelled sample. A stated
  measurement that is stale beats a query that either mis-samples in silence or
  breaks the step that follows it.
- **Name the authoritative scan.** The paginated REST title scan is the one that
  decides ids; the board listing is forbidden for that purpose. *A truncated
  board scan is worse than none, because it answers.*

## The rule

**Before trusting a short or empty result, establish that the tool could have
returned a long one.** For any paginated or limited API, that means one of: read
by identity instead of listing; page exhaustively and assert the count against a
declared total; or delete the question. A hardcoded `--limit` against a growing
collection is a correct answer with an expiry date nobody is watching.

The tell is in the shape of the failure, not the domain: **if "broken" and
"nothing found" print the same thing, the caller cannot tell them apart, and it
will report the second one.**

## References

- PR #1587 — the fix, with both failures reproduced.
- Issue #1586 — the ticket being filed when it surfaced.
- `00_meta/skills/new-ticket/SKILL.md` (`mlorentedev/knowledge@e65645e5`, `@c3580e13`).
- `.claude/CLAUDE.md`, the prohibited-pattern table — *"the last five rows fail silently: they
  return an empty or single-element result instead of an error, and empty reads as a finding."*
- Lesson 263 — never hand-split `--paginate` output; the same class, one layer down.
