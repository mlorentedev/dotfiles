---
id: "lesson-378-an-unattended-deploy-must-not-trade-a-secret-for-its-placeholder"
type: lesson
status: active
title: "An unattended deploy must not trade a secret for its placeholder"
created: "2026-10-09"
---

# An unattended deploy must not trade a secret for its placeholder

## Context
`dotf deploy` installs the `ai/deploy.json` configs by hand. To make a template change reach every machine
without that step, `dotf converge` gained a `configs-deploy` reconciler that runs the same loop (#1843 B15).
Converge is meant to run after `dotf update`, when nobody is at the keyboard.

## The Trap
Two behaviours that are harmless by hand turn harmful in a run nobody watches:

- **Render is lenient on purpose.** `secrets.Render` leaves a placeholder when the store cannot answer, so that a
  setup always finishes. When `dotf deploy` runs unattended with a locked store, the staged copy of
  `~/.pi/agent/models.json` still holds `{env:VAR}`. It differs from the installed file, which holds the resolved
  value, so the deploy replaces a working config with a weaker one.
- **A rendered entry stages beside its destination before it compares.** It has to, because an atomic rename needs
  the same filesystem. But a `--plan` that included it created `~/.pi/agent/` on an empty HOME, and a plan must
  write nothing.

## The Solution
- Converge wires a strict renderer. A placeholder the store could not resolve (`Unresolved`, not `Missing`: a
  secret this machine never holds is a permanent state, not a lock) returns `deploy.ErrRenderIncomplete`.
  `deploy.Run` then skips the entry, keeps the installed file, and the report names it as `kept (secrets locked)`.
- `Deploy` stages a plan's copy in a scratch directory. Only a real apply stages beside the destination.

## Takeaways
- A step that is safe interactively has to be re-read as if nobody will see its output. "Leave it and carry on"
  is right with a human watching and wrong without one.
- A plan is only a plan if every path it takes writes nothing, including the scratch work that comes before the
  comparison.
