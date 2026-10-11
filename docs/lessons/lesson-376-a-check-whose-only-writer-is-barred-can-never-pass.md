---
id: "lesson-376-a-check-whose-only-writer-is-barred-can-never-pass"
type: lesson
status: active
title: "A check whose only writer is barred on an OS can never pass there"
created: "2026-10-09"
---

# A check whose only writer is barred on an OS can never pass there

## Context
On the Mac, `dotf doctor` kept seven `drift:` FAILs (`.zshrc`, `versions.conf`, four scripts)
after every converge (#2224). `dotf converge --plan` reported `0 to change` at the same time.

## Problem
Doctor compared the checkout with `~/.dotfiles` over a path list of its own
(`isManagedDeployPath`). The only thing that wrote those paths was the copy block in
`setup-linux.sh`, and ADR-045 bars that script on macOS. Converge mirrored a different set
(`harness/` plus the manifest targets). So on darwin the check had no writer at all: no command
could make it pass, and its remedy ("run setup") named the barred script. The stale copies were
not cosmetic: `.zshrc` sources `~/.dotfiles/versions.conf`, so every shell exported old pins.

## Solution
One set owns both sides: `harness.DeployDirFiles` and `harness.DeployDirTrees`. `harness.Mirror`
(run by `dotf converge` and by both setups through `dotf harness mirror`) copies them, and
doctor's drift check compares exactly them. The remedy now names `dotf converge`. A test pins the
set to setup's remaining copy block, and another asserts every entry exists in the checkout,
because the mirror skips an absent path.

## Why
A checker and a writer that keep separate lists agree only by coincidence, and the coincidence
breaks on the first OS that runs one without the other. When a FAIL has no command that clears it
on the machine reporting it, ask who writes the thing being checked on that OS, not what drifted.
