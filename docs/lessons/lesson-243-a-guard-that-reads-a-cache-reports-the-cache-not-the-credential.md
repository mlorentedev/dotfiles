# Lesson 243 — A guard that reads a cache reports the cache's age as the credential's health

**Date:** 2026-08-28
**Context:** CLI-056 (#1316) — on the Windows work box `dotf doctor` reported the bitácora PAT as "token invalid or expired (HTTP 401) — rotate it" and `dotf secrets verify` reported the `dockerhub` item as "not found". Both credentials were in daily use by the owner.
**Category:** secrets, bw serve, doctor, diagnosis

## What happened

`dotf secrets unlock` started and unlocked the `bw serve` daemon and never
synced it. The daemon answered every read from the vault cache it booted with,
which on this box was twelve days old (`lastSync 2026-08-15`). A token rotated
after that date resolved to its previous value — GitHub answered 401 — and an
item created after that date did not exist. Doctor's PAT tier then said the
one thing a 401 can mean when the value is assumed current: rotate it.

The owner's objection was the whole diagnosis: "I use both every day." A
`POST /sync` to the daemon advanced `lastSync` to now, `dotf secrets verify`
went from 2 failed to 35 ok, and the PAT tier passed. Nothing about the
credentials had changed.

## The rule

A check that reads through a cache is measuring the cache first and the
thing second. Before a message names the expensive remedy (rotate, re-login,
recreate), the check must either refresh the cache or say that it did not:
"token rejected — if the vault cache is stale run `dotf secrets unlock`;
otherwise rotate it" costs nothing and points at the cheaper cause first.

Two caches, two clocks: `bw status` reports the CLI's cache, the daemon's
`/status` reports its own, and they sync independently. Doctor already aged
the first (BUG-074); the one that decides what a daemon-served read returns
was unmeasured. The fix that closes the class: `unlock` syncs (the daemon
serves current secrets by construction), doctor reports the daemon's cache
age, and the 401 message names the stale-cache possibility before the
rotation.

## How to apply

- When a diagnostic reads through a cache, print the cache's age next to the
  verdict, or refresh it first. A verdict without the age is a guess.
- The remedy in a FAIL message is ordered by cost: refresh before rotate.
- "I use it daily" from the owner is evidence about the credential, not about
  the copy the tool read — reconcile the two before repeating the verdict.
