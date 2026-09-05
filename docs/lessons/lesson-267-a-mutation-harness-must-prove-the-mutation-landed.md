# Lesson 267 — A mutation harness must prove the mutation landed

**Date:** 2026-09-04
**Context:** CI-002 (#1478), PR #1482 — guarding the pi package reconcile skip.

## What happened

Five new assertions in `tests/pi-packages.bats` guarded a CI change. Because a test that
passes on first write is not evidence, each was mutation-tested: break the thing, confirm
the test goes red.

Four of them held. Then a review found a real gap the suite had missed — `versions.conf`
was absent from the `pi` path filter — and fixing it meant mutating the filter entries
too. The harness reported:

```
drop ai/pi/**:          !!! NOT CAUGHT !!!
drop setup-linux.sh:    !!! NOT CAUGHT !!!
drop setup-windows.ps1: !!! NOT CAUGHT !!!
drop versions.conf:     !!! NOT CAUGHT !!!
```

Four assertions, all apparently vacuous. The obvious reading was that the tests were
worthless and had to be rewritten.

**The tests were fine. The harness was broken.** `setup-windows.ps1` appears **twice** in
`ci.yml` — once in the `code` filter, once in the new `pi` filter — and the mutation did
`s.replace(entry, '', 1)`, deleting the **first** occurrence. It removed a line from a
different filter, left the `pi` block untouched, and the assertions correctly passed
against an unmutated block.

Re-run targeting inside the `pi:` block by line position, all four failed as they should.

## The lesson

**"The mutation was not caught" and "the mutation was not applied" are indistinguishable
from outside the harness.** Both produce a green suite and the word `NOT CAUGHT`. One
means the test is worthless; the other means the test is fine and the harness lied. They
demand opposite responses, and the wrong response — rewriting working assertions — costs
more than the bug ever would have.

So a mutation harness needs its own assertion, and it is not the suite's exit code:

**Assert the mutation changed the thing it was aimed at, before running the suite.**

`assert anchor in source` is not that assertion. It proves the string exists *somewhere*
in the file, which is exactly the property that was true and useless here. The anchor must
be located the same way the code under test locates it — by block, by line range, by
structure — or the harness is testing a different file region than the test is.

## The generalisation, and why it recurred four times in one evening

This is the same failure as the four others found in the same session, at a different
altitude:

| where | what it measured instead |
|---|---|
| `features.json` counted `--- PASS: Test` including indented subtests | 18 where 4 was expected |
| `grep -n X f \| grep -v '^ *#'` "filters comments" | `grep -n` prefixes a line number, so no line starts with `#` — nothing is filtered |
| the `awk` slice terminated on "the next 12-space key" | that never fires, so the slice ran past the block and matched unrelated lines |
| `features.json` asserted `grep -c '^ok ' \| grep -qx 22` | adding a 23rd test broke the spec's evidence, for a reason unrelated to the behaviour |
| **this one** | the mutation deleted a line in a different block |

Every member is **a green result that measured the wrong thing**, and none of them is
visible from a passing run. The recurring shape is a check whose *target* is specified by
a pattern that matches more, or less, or elsewhere than intended — while the *assertion*
on top of it is perfectly correct.

The verification is never the assertion alone. It is the assertion **plus** a proof that
it was pointed at the right thing.

## What to do

- **Mutate by structure, not by string.** Locate the block, then edit inside it. A
  repo-wide `replace(x, '', 1)` will find the wrong `x` eventually, and silently.
- **Make the harness report what it changed**, not just that it changed something. A diff
  of the mutation is two lines of code and turns `NOT CAUGHT` into a fact rather than a
  hypothesis.
- **Distrust a unanimous `NOT CAUGHT`.** One vacuous assertion is plausible; four
  simultaneously is a harness fault far more often than a test fault.
- **Never assert a count where the thing counted is expected to grow.** Assert the
  property, and assert each guarantee by name.
- **Prove both directions.** A guard must fail when the guarantee is removed *and* survive
  when unrelated things are added. Only one of those was ever checked here, and it was the
  wrong one.

## References

- PR #1482 (`46af234`, `e0b0ad1`), spec `specs/CI-002/`
- Lesson 265 — a correct measurement answering the wrong question
- Lesson 266 — two measurements agreeing is not reproducibility
- #1422 — the same family in the triage queue: a comparison pointed at `created_at`, which
  never moves when a reviewer edits in place
