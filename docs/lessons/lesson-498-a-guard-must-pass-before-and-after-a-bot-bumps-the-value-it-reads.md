---
id: "lesson-498-a-guard-must-pass-before-and-after-a-bot-bumps-the-value-it-reads"
type: lesson
status: active
title: "A guard must pass before and after a bot bumps the value it reads"
created: "2026-10-20"
---

# A guard must pass before and after a bot bumps the value it reads

## Context

The #2814 guard in `tests/herdr-config.bats` kept the herdr toggle from naming
`dotf herdr explorer` before the pinned release carried it. It had two halves:
the binding present with `DOTF_VERSION` below `floor` failed, and the binding
absent with `DOTF_VERSION` at or past `floor` failed too ("land the binding and
delete this guard").

## Problem

The two halves made the binding and the pin bump one atomic change, but they
have different authors and triggers. release-please raises `DOTF_VERSION`
(`extra-files`) on its own schedule, in a bot-owned branch it force-pushes on
every merge to main. The binding could not land first (the pin was still below
`floor`), and the release PR could not go green without it. `test` is a required
check on main, so release 0.76.0 (#2804) sat red and blocked every PR whose
fixes it carried.

## Solution

Keep only the half that protects a machine: a binding never names a subcommand
the pinned release lacks. The reverse half guarded nothing, since a missing
binding breaks no machine; it only enforced an order. The release then goes
green by itself, and the binding lands in a normal PR once the pin is past
`floor`.

## Rule

A guard that reads a value a bot bumps (`DOTF_VERSION`, any `extra-files` entry)
must pass in the state before the bump and in the state after it, with no other
change. If a check needs a human change to land with the bump, it deadlocks the
release instead of guarding it. Fail on the unsafe state, never on the
"unfinished" one.
