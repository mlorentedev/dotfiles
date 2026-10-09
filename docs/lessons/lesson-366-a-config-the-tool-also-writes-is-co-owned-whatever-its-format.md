---
id: "lesson-366-a-config-the-tool-also-writes-is-co-owned-whatever-its-format"
type: lesson
status: active
title: "A config the tool also writes is co-owned, whatever its format"
created: "2026-10-08"
---

# A config the tool also writes is co-owned, whatever its format

## Context
#2013 H4 deploys herdr's `config.toml` through `dotf deploy`. The manifest has two strategies:
`replace`, where the repo file becomes the whole destination, and `merge`, where the repo owns the
keys it names and the tool keeps the rest. `merge` read JSON only, so a TOML config looked like a
`replace` entry by default.

## The Trap
Whether a deploy can own a whole file is a fact about the tool, not about the file's format.
herdr writes `config.toml` itself: finishing onboarding writes `onboarding = false`, and its
Settings screen saves what the user picks (herdr 0.9.3 configuration docs). A `replace` entry over
that file overwrites the user's in-app choices on every setup run, and doctor reports drift from
the moment the user changes a setting until the next deploy. That is the `.gitconfig` case again:
every `git config --global` writes that file, and doctor exempts it from the content check for that
reason (PLAT-001a, measured drifting on a converged box 2026-09-02). The format chose the strategy,
and the format was the wrong question.

## The Solution
Before adding a deploy entry, establish whether the tool writes the destination: an onboarding
flow, a settings screen, a `config set` command or a `reset` command are each evidence that it
does. If it does, the entry is a `merge`. `dotf deploy` now merges TOML too: the format follows the
destination's extension and the merge rules stay the same. Equality is semantic, because a tool
that rewrites the file in its own layout must not read as drift.

The cost is visible and bounded. A merge that changes a value re-encodes the file, so comments in
the destination do not survive it. An in-sync merge writes nothing.
