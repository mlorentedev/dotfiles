---
id: lesson-293
type: lesson
status: active
created: "2026-09-24"
owner: manu
tags: [lesson, harness, gate, state, guard]
---

# 293 — An empty key is still a key: define it as "no storage", never as a bucket

## What happened

The gate keys its consumption ledger, its dispatch map and its decision journal
by session id. A payload that names no session hashes the empty string, and the
digest of nothing is a perfectly good filename, `unknown-e3b0c442`. Every such
payload shared one ledger.

Reproduced against the old code: two unrelated calls with no session id, the
second never having invoked the skill. The gate answered "all gated skills
consumed", because the first call's skill run sat in the shared file. A second
payload shape failed the other way: an agent id with no session made a
per-dispatch file nothing could find again, and blocked for good.

It was latent. No skill was `enforce: block` yet, and the ledger file did not
exist on the measured machine, only the journal (51 records, 50 of them from a
test run against the real state dir). It needs a payload with no session id, a
persona in scope and a blocking skill at once, and agy spells the session field
`conversationId`, which is how the first of those would have arrived.

## Why it happens

The path builder was written to survive hostile ids (`a/b` and `a.b` must not
collide), and its fallback, `if safe == "" { safe = "unknown" }` plus a digest,
made the empty case look handled. It turned "no identity" into "the identity of
nothing", which is valid, stable and shared by everyone who has no identity.

The tempting fix, keying by process instead, fails the other way: a blocked call
can never record the skill that would satisfy it, so a persona could be stuck
for good.

## The rule

1. **The empty key means "cannot store".** `StatePath("")` returns no path,
   writing through it is a no-op, reading through it is empty. Do it in the path
   builder, so every caller is safe by construction and nobody has to remember.
2. **When enforcement needs an identity and there is none, allow and say so.**
   Failing closed on an unidentifiable call makes the gate's normal state red.
   The journal gets its own outcome (`session-unscoped`), so a count of them is a
   count of calls the gate could not police.
3. **Test the consequence against the old code.** Two unrelated calls with no id;
   assert the second is not satisfied by the first and that no state file exists.
   The mutation run showed the old code answering `allow` and writing
   `unknown-e3b0c442.json`.

Relates to lesson 287 (a guard that skips is a guard that passes).
