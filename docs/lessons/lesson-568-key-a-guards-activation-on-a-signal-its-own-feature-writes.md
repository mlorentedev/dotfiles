---
id: "lesson-568-key-a-guards-activation-on-a-signal-its-own-feature-writes"
type: lesson
status: active
title: "Key a guard's activation on a signal its own feature writes"
created: "2026-11-07"
---

# Key a guard's activation on a signal its own feature writes

## Context

The named-lane pre-commit guard (HARNESS-186, #3255) shipped opt-in by the rule
"the repository has `.dotf/lane.json`". That file already existed on `main`: it
was committed earlier by #2532 for a different subsystem — the issue-queue lane,
which has a different schema and no named lanes at all. The named-lane feature
wrote no marker of its own; `dotf lane init` created lane directories and
nothing else.

## Problem

"The file exists" is a boolean any neighbouring feature can flip. The guard was
therefore armed by data it did not own, on a repository where zero named lanes
existed. On merge it would have refused every commit inside a third-party session
root and every commit on `main`, and the only documented remedy (`dotf lane
switch`) was a command from the same unmerged PR — so the repository would have
been uncommit-able with no migration path, no runbook and no doctor check.

Nothing failed at build, test or CI time. The guard's own tests wrote the marker
file themselves, so they proved the guard *worked* and never that it would
*arm*. A test that manufactures its own precondition cannot detect a wrong
precondition. (Its mirror is lesson-564: there, data landed before the capability
that reads it; here, data that belonged to another capability was mistaken for
this one's switch.)

## Solution

An activation marker must be written by the feature that reads it, and the reader
and the writer ship in the same change. `dotf lane init` now writes
`.dotf/lanes.json` and the guard requires exactly that file; a scenario asserts
the borrowed file no longer activates it, so the regression is pinned rather than
remembered.

Keep the marker free of anything machine-specific. It carries no paths and no
host names — only `version`, `canonical_lanes` and `third_party_roots` — so it is
byte-identical on every OS, agent host and seat. Roots are data: a test declares
a root that exists on no machine and asserts it blocks exactly like the real one,
which is what makes the guard agnostic rather than merely configurable.

Two habits follow. First, when a guard's activation is a file, write the test
that proves a repository *without* your marker stays unguarded — the negative is
the whole contract. Second, if activation is silent, say so at the moment it
happens: `dotf lane init` prints the marker path and that it must be committed,
because a guard nobody can tell is armed is a guard nobody trusts.
