---
id: "lesson-420-a-scratch-home-does-not-isolate-a-tool-that-reads-per-tool-dirs"
type: lesson
status: active
title: "A scratch HOME does not isolate a tool that reads the env contract's per-tool dirs"
created: "2026-10-14"
---

# A scratch HOME does not isolate a tool that reads the env contract's per-tool dirs

## Context
To see what `herdr integration install` writes, it ran with `HOME` and `XDG_CONFIG_HOME` pointed at a scratch directory, once per agent.

## Problem
The installs for pi, opencode and agy landed in the scratch directory. The installs for claude and copilot landed in the real `~/.claude/settings.json` and `~/.copilot/settings.json`. herdr resolves those agents through `CLAUDE_CONFIG_DIR` and `COPILOT_HOME`, and `env-contract.json` exports both on every machine, so they still pointed home. The output said so (`installed claude integration hook to /Users/...`), but only for the two lines nobody expected to differ. Reverting took herdr's own `integration uninstall` plus deleting an empty `hooks` key and directory it left in the copilot config.

## Solution
Isolate by the contract, not by `HOME`: redirect every `*_HOME`, `*_DIR` and `*_PATH` variable `env-contract.json` declares, along with `HOME`. Then check that the tool reports scratch paths, before letting it write. dotf's own tests never execute herdr: `herdr.Apply` takes a `tools.Runner`, and the converge fixture leaves `HerdrRun` unset, which skips that half.

## Why
The env contract exists so every tool finds its config wherever the machine keeps it. That same property defeats a `HOME`-only sandbox. Any tool that honours a per-tool variable writes through the sandbox to the real file.
