# Lesson 222 — A coupling's scope is measured, not inferred from where you saw it fail

**Date:** 2026-08-23
**Area:** guards / CI / cross-repo contracts
**Severity:** medium — fixes that look complete because the place you were looking went green

## What happened

Three sightings in one session, in three different systems. Each is a case of a
thing being declared in one place and consumed in more places than the person
fixing it had counted.

**1. A backend declared with zero reachable providers.** `harness/model-map.json`
names hive as a dispatch backend. `worker_status` on 2026-08-22 reported *Ollama:
offline · OpenRouter: no API key · Available Models: none*. Both providers had
been dead for an unknown length of time — Ollama not running locally, OpenRouter
retired upstream in August 2026 — and nothing noticed, because every surface
reported the backend by whether a client object existed rather than by whether it
answered.

**2. A rule enforced by a counter that does not implement it.** `#1180` decided
the atomic-PR cap counts **executable** lines. `spec-gate` still counts comment
lines (`#1186`, open). A 44-executable-line change was refused at "74 LOC". The
rule is documented, the enforcement measures something else, and the gap only
shows up as a refusal you have to argue with.

**3. A diagnostic instrument gating merges — in two places, not one.**
`test_classify_cancellation_race` spawns a real hive subprocess and reports how
20 cancellation races distribute. Its own docstring says it is *"NOT a pass/fail
correctness test"*. It failed on a release PR carrying only a version bump, on
3.13 while 3.12 went green.

The third one is the instructive one, because the *fix* repeated the mistake. The
first attempt excluded the test from the default suite with a new `diagnostic`
marker, verified in both directions locally, and shipped. CI went red on both
runners: the test also had a **dedicated step** in the `cross_worker_lock` job
that nobody had looked at. A nodeid does not escape `addopts`, so that step
collected nothing and pytest exited 5.

Had that step not broken loudly, the ticket would have closed with the second
gate still standing, and the next release would have failed for the same reason
with the fix already "done".

## Why the mistake is easy

Because the symptom has a location, and the location feels like the scope. The
release PR went red in `check (3.13)`, so `check` is where attention went. The
question that would have found the second gate is not *"where did it fail?"* but
*"who else consumes this?"* — one `grep` over the workflow file, thirty seconds.

The same grep would have found that `_try_worker` has **three** call sites and
two tools, not the one that prompted the change: `delegate_task`'s inference
path, `delegate_task`'s vault-summarization path, and `capture_lesson`. That one
was caught before shipping, and only because it was looked for.

## The rule

**Before fixing a coupling, enumerate its consumers. After fixing it, prove the
enumeration was complete.**

Concretely, and cheaply:

- `grep` for the symbol, the marker, the filename, the env var — across
  workflows and config, not only source. CI YAML is the surface that gets
  skipped, and it is where the second gate lived.
- State the count in the commit message. "Three call sites, two tools" is a
  claim a reviewer can check; "updated the callers" is not.
- Prefer a guard that fails when the enumeration is wrong. The `diagnostic`
  marker was asserted **in both directions** — that the default run deselects it
  *and* that `-m diagnostic` selects it — which is why the deselection count
  moving from 63 to 64 was visible evidence rather than a hope.

## Relation to Lesson 220

[Lesson 220](lesson-220-four-defects-one-shape-a-thing-verified-by-a-proxy.md)
names the sibling shape: *a thing verified by a proxy that lives somewhere else*,
with the diagnostic question **"what would this still pass on if the thing it
checks were broken?"**

This one is the same family from the repair side. 220 is about a check that does
not observe its subject; 222 is about a fix aimed at one of its subject's
consumers. The questions pair up:

| | Question |
|---|---|
| **220** — writing a check | What would this still pass on if the thing it checks were broken? |
| **222** — repairing a coupling | What else consumes this, and how do I know I found all of it? |

## Evidence

- `worker_status` output, 2026-08-22 — zero reachable models
- `mlorentedev/hive#386` — release PR red on `check (3.13)`, green on 3.12
- `mlorentedev/hive#389` — the first commit de-gated one place; CI caught the second
- `#1180` (the rule), `#1186` (the counter that does not implement it)
- `mlorentedev/hive#388` — the ticket, whose own text said "one place"
