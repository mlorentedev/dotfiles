---
id: "lesson-604-a-pin-check-that-reports-the-symptom-hides-the-config-that-outranked-it"
type: lesson
status: active
title: "A pin check that reports the symptom hides the config that outranked it"
created: "2026-11-14"
---

# A pin check that reports the symptom hides the config that outranked it

## Context

On ace2 the converge `tools` step failed with `after mise install, not running at their pin: node,
python` and skipped every later step (#3524). The cause was `~/.config/mise/config.toml`, written by
another repository's role, which outranks dotfiles' `conf.d/dotfiles.toml`. Re-running the sync could
never fix it: mise installs dotfiles' version beside the other one and keeps running the other.

## Problem

The check knew the tool failed, and mise knew why; the message carried only the first fact.
Diagnosing it took `mise config ls` and `mise ls` by hand. A check that compares an outcome with a
pin has to say which input won when the outcome is wrong.

## Solution

Ask mise, not the filesystem: `mise ls --json <tool>` prints an array with one entry per installed
version, and the active one carries `source.path` and `requested_version` (mise 2026.11.6). Query one
tool per call: with several tools the entries come back merged and none names its tool. Ask only
about tools that already fail, and when the answer cannot be read, keep the old message rather than
guess a cause. The converge step and `dotf doctor` share `MiseSync.ShadowNote`, so the two cannot
disagree about who shadows a pin.
