---
id: "lesson-599-a-provisional-binding-needs-a-probe-for-the-condition-that-retires-it"
type: lesson
status: active
title: "A provisional binding needs a probe for the condition that retires it"
created: "2026-11-14"
---

# A provisional binding needs a probe for the condition that retires it

## Context

On 2026-10-14 agy stopped serving `gemini-3.1-pro-high`, and the reviewer pool's
provider-diverse arm was moved to `gemini-3.8-flash-high` (#2385, #2474). The entry
said so in prose: "PROVISIONAL: when Gemini ships new models, measure them on the
same bar and replace this one if one passes", and `harness/model-map.json` routed
the top tier onto the mid id "until Gemini ships a stronger one".

Google shipped Gemini 4 Argon on 2026-10-13 and Hydrogen on 2026-11-12, and agy
1.5.8 serves both as `gemini-4-argon-{low,medium,high}` and
`gemini-4-hydrogen-{low,medium,high,xhigh}`. Nothing in the repository noticed.
The owner did, from the news, on 2026-11-14 (#3471). Both candidates passed the
admission bar on the first run (50 s and 66 s), so the binding had been stale for
as long as agy had served them, with a Pro tier one `agy models` call away.

## Lesson

A condition written into a comment has no detector. "Until X ships" names the
event that retires the binding and leaves finding that event to whoever happens to
read the news. The same catalog that would have reported it, `agy models`, was
already the source of every id the entry names; the comparison was never run.

When a binding is provisional, write down the probe that ends its term, not only
the term: the command, what it must print for the binding to stay, and where the
check runs. For a model catalog that is the list of served ids compared against
the ids the map routes, reported by `dotf doctor` when the catalog carries a
family the map does not route (#3475). A probe that exists only as a sentence is a
promise wearing a file's clothes.

Related: lesson 331 (a shared budget is watched at the account, not the binding),
lesson 588 (a ticket and an operational note written in one sitting drift from the
archive they both cite).
