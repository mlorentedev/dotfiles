# 255 - Truncation, not hostile input, is what made the collision guard load-bearing — and without it the measurement would have confirmed the opposite

**Date:** 2026-09-01
**Area:** harness, orchestrator gate, path construction

## What happened

`harness.StatePath` builds the gate's per-scope consumption ledger filename. It
maps every character outside `[A-Za-z0-9_-]` to `_`, **truncates the result to 48
characters**, and appends the first four bytes of `sha256(scope)` as hex:

```go
if len(safe) > 48 { safe = safe[:48] }
sum := sha256.Sum256([]byte(sessionID))
return filepath.Join(stateDir, "gate", safe+"-"+hex.EncodeToString(sum[:4])+".json")
```

The scope is `session_id + "-" + agent_id` when a subagent is acting. A session id
is a 36-character UUID, so with the joining hyphen only **11 characters of the
agent id survive the truncation**.

Two subagents were dispatched in one session to measure whether `agent_id` is
per-invocation or stable per persona — the check the code itself names as a
precondition for promoting any skill to `enforce: block`. Both wrote a ledger
entry:

```
e4edd8fa-dc36-4cc5-bd73-5047c2e5b737-agate-probe-079740d4.json  {"skills":{"genre-picker":true}}
e4edd8fa-dc36-4cc5-bd73-5047c2e5b737-agate-probe-8fb525a3.json  {"skills":{"research-prompt":true}}
```

The readable prefixes are **byte-identical, both exactly 48 characters**. The two
files exist as separate files only because the digests differ. The measurement
succeeded — the ids are distinct, per invocation — and it succeeded entirely on
the strength of the appended hash.

## Why this is the lesson and not just a bug

The digest was added because a PR reviewer on #1272 objected to character-mapping
alone, and the argument that carried was adversarial:

> `a/b` and `a.b` both flatten to `a_b`, so one session's consumption record would
> open another session's gate. Well-behaved harnesses send UUIDs and would never
> hit it, but a session id is attacker-adjacent input that lands in a filesystem
> path, and "it does not happen with well-behaved input" is not a property a path
> builder should rely on.

The reviewer was right, and right for a reason neither side stated. **No hostile
input was required.** The collision arrived on the first real measurement, from
two ordinary auto-generated agent ids, because a fixed 48-character cap discarded
the only part that distinguished them. The defence was accepted as a hardening
against a rare adversarial case and turned out to be load-bearing on the normal
path — a distinction that matters, because defences argued only on rare cases are
the ones that get traded away under review pressure.

**What makes this worth writing down is the shape of the failure it prevented.**
Had the digest not been there, the second dispatch would have written to the first
one's file. The directory would have shown **one** entry for two dispatches, and
the honest reading of that evidence is *"the ledger key is shared, so the second
dispatch inherits the first's consumption"* — which is precisely the
over-permissive failure the code documents at `gate.go:238-247` as the thing this
measurement exists to rule out.

So the missing digest would not have produced a confusing result or an obvious
crash. It would have produced **the exact symptom of the real defect being tested
for**, sourced from a path-construction bug instead of the harness, and we would
have "confirmed" it and gone off to fix a harness that was behaving correctly. A
false negative indistinguishable from the true positive is the most expensive
possible failure of an instrument.

## What this does not license

**The truncation is not itself wrong and should not be removed.** Its stated
purpose is that the directory stays diagnosable by eye, and it achieves that: the
session id is legible in every filename, which is how the three sessions writing
into this ledger were traced back to their projects. The lesson is not "do not
truncate" — it is that a truncated key is a *display* key, and correctness has to
live in the part that is not truncated.

**Nor does a passing measurement retire the concern.** The prefix collision is
still there, silently, on every subagent dispatch. Anything that later reads this
directory by prefix — a cleanup routine, a doctor check, a human running `ls
<session>-*` — will treat two scopes as one. The collision was defused for the
file path and nowhere else.

## Rule

When a defence is accepted on adversarial grounds, check whether the **ordinary**
path already triggers it; if it does, the defence is load-bearing and its
rationale should say so, because the adversarial framing invites someone to trade
it away later. And when a key is truncated for readability, verify the truncation
preserves what distinguishes — otherwise the instrument fails by producing the
very reading it was built to test for, which no amount of care in interpreting the
result can catch.
