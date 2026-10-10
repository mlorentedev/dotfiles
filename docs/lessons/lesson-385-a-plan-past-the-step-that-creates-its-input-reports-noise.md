---
id: "lesson-385-a-plan-past-the-step-that-creates-its-input-reports-noise"
type: lesson
status: active
title: "A plan past the step that creates its input reports noise"
created: "2026-10-10"
---

# A plan past the step that creates its input reports noise

## Context
PLAT-001b PR 4a put a `checkout` step first in `dotf converge`: from zero it clones the dotfiles
checkout, otherwise it fast-forwards it. Every later step (records, tools, configs, git config)
reads files from that checkout.

## The Trap
An apply is sequential: the clone runs, and the next step finds its files. A plan writes nothing, so
on a machine from zero the checkout is still absent when the plan reaches the next step. Each later
step then plans against a directory that does not exist yet: whatever it reports (a read error, a
full set of "to install") describes the absent checkout, not what the apply would do. And since
`Run` stops at the first failure, one such error hides the rest of the plan. The same holds for any
step whose output is the next step's input.

## The Solution
`converge.Result` carries a `Gate`. A step that has not converged yet under a plan sets it (the
checkout sets `waits for checkout: not cloned yet`, or names the fast-forward it would apply), and
`Run` reports every later step skipped with that reason instead of reconciling it. A checkout that
is merely behind counts too: the later steps would read the old tree, and the apply reads the new
one. An apply ignores the gate, since the step really ran. A plan
from zero now reads: the clone, then the remaining steps waiting for it.

## Takeaways
- **A dry run is only honest up to the first step whose effect a later step reads.** Past that
  point, say what is waiting and why; do not evaluate it against the state that has not changed.
- **Make the dependency a field of the result, not an ordering assumption.** The gate is declared
  by the step that knows it has not converged, so a new dependent step needs no special case.
