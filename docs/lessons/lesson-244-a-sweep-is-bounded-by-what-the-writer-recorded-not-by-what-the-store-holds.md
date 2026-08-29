# Lesson 244 — A sweep is bounded by what the writer recorded, not by what the store holds; and the store's name rules decide the order

**Date:** 2026-08-29
**Context:** CLI-065 (#1363) — `dotf env persist` (CLI-058) wrote every contract variable into `HKCU\Environment` and never removed one, so a name retired from `env-contract.json` stayed on every box forever, inherited by exactly the profile-less processes the scope exists for. Same class as WIN-013 (#1310): a deployer that adds and never sweeps.
**Category:** deploy, idempotence, windows, registry, ownership

## What happened

The obvious sweep — "delete what is in the store and not in the contract" — is
the one the ticket forbids, and for a good reason: `HKCU\Environment` holds the
user's own variables, third-party installers' variables, and ours, with nothing
in the store telling them apart. A sweep bounded by the store's contents would
delete whatever a past contract *or anyone else* had put there.

The fix records ownership at write time. `dotf env persist` writes a marker
value, `DOTF_MANAGED_ENV`, naming every variable it persisted; the next run
deletes only a name that is **both** in the marker **and** absent from the
contract. A variable dotf never wrote is never touched, whatever its name — and
a box that persisted before the marker existed has no record, so nothing is
deleted there: bounded honesty over guessing, stated in the spec as out of
scope rather than papered over.

Two details were not obvious until the design was written down:

1. **Registry value names are case-insensitive; Go string sets are not.** A
   case-only rename in the contract (`Foo` → `FOO`) is a *rewrite of the same
   value* to the registry. Compared exactly, `Foo` is a leftover and `FOO` is a
   write — and if the writes ran first, the sweep would delete the value the
   same run had just written. Two guards, either sufficient: the comparison is
   `strings.EqualFold`, so a case-only rename is not a leftover at all; and
   every delete precedes every write, pinned by a test that records the
   operation order on the fake store.
2. **`Delete` of an absent name must succeed**, and the fake and the real store
   must agree on it. The sweep is driven by the marker, and a marker can name
   what a hand edit already removed; a store that errored on absence would fail
   every second run. `registry.ErrNotExist` maps to `nil`, the fake mirrors it,
   and a test pins the fake so the two cannot drift.

The three readers — `persist`, `persist --check`, `dotf doctor` — share one
pure `Leftovers(marker, contract)` function, so what the check reports is by
construction what the write deletes.

## The rule

- A sweep needs an **ownership record the writer maintains**, never an
  inference from the shared surface. Where no record exists yet, the first run
  writes one and deletes nothing.
- When the store's identity rules differ from the language's (case-insensitive
  names, normalised paths), the sweep compares under the *store's* rules, and
  removals run before writes so the two cannot cancel.
- One function decides the leftover set; every reporter and the writer call it.

## Refs

- `specs/archive/CLI-065-env-persist-sweep/` (this change); `cli/internal/env/persist.go`
- WIN-013 (#1310) — the same class on the scripts directory, solved with an allow-list because the set was seven known names
- The shared-surface pattern in the project memory: *the writer touches only what it owns*
