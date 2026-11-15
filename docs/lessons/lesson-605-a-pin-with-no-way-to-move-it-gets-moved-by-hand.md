---
id: "lesson-605-a-pin-with-no-way-to-move-it-gets-moved-by-hand"
type: lesson
status: active
title: "A pin with no way to move it gets moved by hand"
created: "2026-11-15"
---

# A pin with no way to move it gets moved by hand

## Context

On the Mac the owner wanted newer tools and ran `mise upgrade --bump` (#3284, comment of
2026-11-15). It rewrote `~/.config/mise/conf.d/dotfiles.toml`, the file `dotf tools sync` renders
from `versions.conf`, and moved 28 pins, majors included (helm 3 to 4, node 24 to 26). It also
bumped the hand-written `~/.config/mise/config.toml`, which outranks `conf.d`.

## Problem

Every pin in `versions.conf` was exact and reviewed, but nothing proposed a newer one: Dependabot
watches Go modules, Actions and Docker only. The one tool that offered newer versions was the one
that bypasses the SSOT, so the SSOT lost. The damage stayed quiet. Doctor reported the `conf.d` drift,
but the self-deploy role would have reverted it at the next full converge without a word. A shell
started before the fix also kept the old install paths on `PATH`, so doctor in that shell still
reported drift after the pins were restored.

## Solution

Restore with `dotf tools sync` and `dotf converge --only tools`, and remove from `config.toml` every
tool `conf.d` already pins. Judge the result in a new shell. The lasting fix is a channel that moves
pins through a reviewed PR (#3284). Without one, an exact pin is a pin people learn to break. The
cleanup half was already automatic (brew in `system-upgrade`, caches in `cache-gc`) except old mise
versions, which #3533 adds to `cache-gc`.
