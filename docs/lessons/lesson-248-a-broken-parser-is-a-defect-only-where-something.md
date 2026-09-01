# Lesson 248 — A broken parser is a defect only where something reads it

**Date:** 2026-08-27
**Context:** HARNESS-045 AC7, PR #1319

## What happened

The spec recorded, and three sessions carried forward, that **four parsers read
`skills:` and three break silently** on the mapping form. The number had been
counted once and repeated since. Building the guards, each claim was executed
against a record in the mapping form instead of read:

| Parser | Claimed | Measured |
|---|---|---|
| `compile-harness.sh:1016` | breaks | **breaks** — returns empty, renders `MUST consume: none` |
| `check-roster-consistency.py:64` | breaks | **breaks** — returns `[]`, guard passes vacuously |
| `doctor.readAgentFrontmatter` | breaks | **not a defect** |

The third does skip indented lines, exactly as claimed. But its only consumers
are `fm["model"]` and `fm["targets"]`. **It never reads `skills`.** A `grep` for
its call sites took seconds; the claim had survived three sessions.

## The lesson

**A parser's inability to read a field is a property of the parser. Whether that
is a defect is a property of its consumers.** The two get conflated because the
parser is what you are looking at when you notice.

Had the guard been built anyway, it would have been worse than wasted: a guard
over a non-consumer is machinery that must be maintained, that fails for reasons
unrelated to anything real, and whose presence implies a risk that does not
exist. The repo's own amendment covers this — meta-work gets the cheapest honest
fix, and deletion beats construction.

The reciprocal is the sharper half: **`compile-harness.sh` was under-stated, not
over-stated.** It was listed third of three, and it is the one that disarms
enforcement on every harness simultaneously, because it feeds the presence block.
A count treats three findings as interchangeable. Measuring the consequence
ranks them.

## What to do instead

- Before guarding a reader, **grep its call sites and name what consumes the
  field.** If nothing does, say so and build nothing.
- Report breakage **by consequence, not by count.** "Two parsers break" is a
  tally; "the presence block renders `none` on four harnesses" is a severity.
- A number carried across sessions is a **claim, not a measurement.** Re-derive
  it the first time you act on it. See [lesson 235] — the same failure with the
  instrument one layer out.

## A second instance, same session

`.gemini/GEMINI.md` has a hard **12000-character** cap. `wc -c` reported
**12029** — apparently a live breach, and it nearly went into a report as one.
`wc -m` reported **11956**: under the cap, with 44 characters spare. The file is
full of `—` and `→`, three bytes each.

**Measure in the unit the constraint is declared in.** A byte count against a
character cap is not a conservative approximation; on this file it inverted the
answer. The 44 characters of real headroom also settled a design question that
had been argued as preference: annotating 35 skills with `(block)`/`(warn)` adds
roughly 250 characters, so the presence line carries ids only. The constraint had
already decided it; it just had not been measured yet.
