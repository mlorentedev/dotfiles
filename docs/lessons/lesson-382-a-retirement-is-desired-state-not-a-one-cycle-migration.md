---
id: "lesson-382-a-retirement-is-desired-state-not-a-one-cycle-migration"
type: lesson
status: active
title: "A retirement is desired state, not a one-cycle migration"
created: "2026-10-10"
---

# A retirement is desired state, not a one-cycle migration

## Context
MEM-002 retired the claude-mem plugin with a "one-cycle cleanup, prune after rollout" block in each
setup script: uninstall the plugin, delete its directories, strip its marketplace key from
`settings.json`. #2007 then asked to delete both blocks once every machine was confirmed clean.

## The Trap
On msi, nothing was clean. Claude Code had moved the marketplace registry to
`plugins/known_marketplaces.json`, so the `settings.json` strip found nothing to remove. The
registration survived with `autoUpdate: true` and re-cloned the retired repository on every session
for months (#1431). The block reported nothing, because it checked the file it edited, not the
registry Claude reads. That is the proxy mistake of
[[lesson-051-healthcheck-must-validate-end-state-not-proxy-arti]], repeated by the change that
retired the plugin. "Prune after rollout" also assumes someone can tell when rollout finished. A
block that reports nothing never says.

## The Solution
The retirement became data: `retired_marketplaces` in `ai/claude/plugins.json`. The `claude-plugins`
step of `dotf deploy`, which both setups already run, removes each retired marketplace through the
CLI that owns the registry (`claude plugin marketplace remove`). It then lists again, and fails by
name if one survived. Both setup blocks are deleted. One Go path serves every OS, and a re-run
reports nothing.

## Takeaways
- **Declare what must not be there**, next to what must be. A converge step can check an absence on
  every run. A one-cycle block can only hope.
- **Remove through the owner's interface, then ask the owner again.** Editing another tool's state
  file breaks the day that tool moves the state; its own `list` does not.
- **A ticket that says "confirm, then delete" is a measurement first.** Here the measurement
  inverted the ticket.
