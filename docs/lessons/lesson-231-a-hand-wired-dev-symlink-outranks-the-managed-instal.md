# Lesson 231 — a hand-wired dev symlink outranks the managed install, and the host fails closed

**Date:** 2026-08-26
**Context:** AI-030 / #1243 — `pi` would not start on this machine.
**Category:** tooling, extension hosts, declaration-vs-effect

## What happened

`pi -p 'ok'` exited **1**:

```
Error: Failed to load extension ".../npm/node_modules/pi-subagents/index.ts":
       Tool "subagent" conflicts with /home/manu/.pi/agent/extensions/subagent/index.ts
Hint: Start without extensions using "pi -ne".
```

Two providers of one tool name. One came from `npm:pi-subagents@0.56.0`, which
`ai/pi/packages.json` declares and `setup-linux.sh` reconciles on every run. The
other was a symlink placed by hand on 2026-08-09 into pi's *own bundled
examples*, under one specific nvm node version:

```
~/.pi/agent/extensions/subagent/index.ts
  -> ~/.nvm/versions/node/v24.16.0/lib/node_modules/
     @earendil-works/pi-coding-agent/examples/extensions/subagent/index.ts
```

The hand-wired one won. The declared one is the one that never loaded.

## Three things worth keeping

**1. The extension host failed closed, and that was a gift.** pi refused to
start at all rather than picking a winner silently. Had it resolved the conflict
by precedence and carried on, the machine would have run for months on a
year-old example extension while every check reported the packaged one as
installed. A hard failure at startup is the cheapest possible signalling for an
ambiguity the host cannot resolve. Prefer it when building anything similar.

**2. The hand-wired dev symlink is a class, not an incident.** It has three
properties that make it worse than it looks: nothing reproduces it (no fresh
machine gets it, so it is invisible to any container or CI verification), it is
pinned to whatever node version was active the day it was made (so it rots when
that version is uninstalled), and it silently outranks the managed install it
duplicates. `~/.pi/agent/extensions/` has a live external writer too — three
`orca-*.ts` files appeared there during this work — so the guard cannot simply
claim the directory. The rule that works is narrower and mechanical: **an entry
under the extensions dir that resolves through a symlink into a `node_modules`
tree is hand-wired by definition**, because a package manager never links that
way.

**3. Quarantine must leave the tree the host reads.** The obvious repair is a
`.disabled/` directory beside the link. pi's own docs list its discovery
patterns:

| Pattern | Scope |
|---|---|
| `~/.pi/agent/extensions/*.ts` | global |
| `~/.pi/agent/extensions/*/index.ts` | global, subdirectory form |

A `.disabled/index.ts` matches the second one. Whether the glob skips a leading
dot is undocumented, so quarantining there is a bet that the same file under a
new name stops being found — the same conflict, one directory deeper. **A repair
that relocates a defect inside the surface that reads it reports success and
changes nothing.** The quarantine went to `~/.pi/agent/.disabled-extensions/`,
outside the scanned tree, which removes the question instead of answering it.

## Why AI-030's own verification could not see this

AC1–AC10 all held while pi refused to start. They prove the manifest is
well-formed and pinned, that the reconcile is idempotent, that a missing
toolchain warns once — and `verify-reconcile.sh` proves it by driving the real
block extracted from `setup-linux.sh`, which is the right shape. All of it
counts entries in an array. **A package can be installed, declared, counted and
still not load.** The criterion that was missing is AC11: a declared package is
also a loaded one, observed as effect.

This is the same failure this repository has now catalogued repeatedly (see
`lesson-230`, *a config that parses is not a config the consumer reads*). The
variant worth naming here is that **the guard's own verification is not exempt**:
the check that watches for declaration-over-effect was itself nearly written as
"is the package in `settings.json`", which is the identical mistake one layer up.

## The guard

`dotf doctor` → `cli/internal/doctor/checks_pi_extensions.go`, repaired under
`--fix`. It lives in doctor rather than CI deliberately: this is **machine
state**, and CI never sees this machine. A check that cannot run where the defect
lives is not a guard.
