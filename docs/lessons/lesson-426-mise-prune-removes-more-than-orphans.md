---
id: "lesson-426-mise-prune-removes-more-than-orphans"
type: lesson
status: active
title: "mise prune removes more than orphans"
created: "2026-10-14"
owner: manu
tags: [mise, converge, tools, idempotence]
---

# mise prune removes more than orphans

**Context:** #2375 asked converge to remove mise installs that no pin asks for. The forcing case was a PowerShell installed by hand on the Mac, whose `pwsh` shim failed with `No version is set for shim`. `mise prune` looked like the whole answer.

**Problem:** On the same machine, `mise prune --dry-run` listed four installs, and only one of them was an orphan:

- `powershell@7.5.11`: "no tracked config or tool stub requires powershell". This is the orphan.
- `jq@1.7.1` and `python@3.12.15`: superseded versions of tools that are pinned at newer ones.
- `agy@1.3.3`: already *scheduled* for removal by `mise upgrade` (`prune_after` in `mise ls --prunable --json`).

A bare `mise prune` in converge would have removed all four. That is more than the owner decided, and it would race mise's own removal schedule.

The other obvious design was a name-diff of `mise ls` against the pins. That fails differently. It would flag `go`, which another project's tracked `mise.toml` requires. It would also flag a tool the sync installs by name and does not pin (for example, `node` for `npm:bats` on Windows). Converge would then prune it and reinstall it on every run, and the second apply would never reach `changed=0`.

**Solution:** Stay downstream of mise's own list, then narrow it. Read `mise ls --prunable --json`, keep only the tools whose *name* is not a pin, and prune exactly those names with `mise prune --yes --tools <names>`, after the install and the upgrade. Each exclusion now holds by construction:

- mise never lists a tool that a tracked config requires;
- a superseded version of a pinned tool is filtered out by name and left to mise's schedule.

**Why:** "Unused" has three meanings: a tool no config wants, a version no config wants, and a version already queued for removal. A destructive step must pick one of them explicitly. Taking the tool's definition wholesale also takes its other two meanings. Measure the dry-run on a real machine before choosing, because only a machine that has lived for a while has all three kinds. A from-zero runner has none of them, so it would pass either design.
