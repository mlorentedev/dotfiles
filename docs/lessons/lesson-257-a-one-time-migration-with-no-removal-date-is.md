# 257 - A one-time migration with no removal date is indistinguishable from live code, and two of eleven turned out not to be finished

**Date:** 2026-09-02
**Area:** setup scripts, technical debt

## What happened

Both setup scripts had accumulated eleven blocks written to converge existing
machines off some retired arrangement. None recorded when it could be removed.
#1333 proposed the obvious cleanup: delete them, they are done.

They were not all done. Classifying each block by **what breaks on a machine that
never runs it** — rather than by what its comment says — split them three ways:

| | Count | |
|---|---|---|
| Dead: skipping costs nothing on any machine | 7 | deleted |
| Correcting, probed converged on the only OS they run on | 2 | deleted |
| Correcting, unverifiable or actively broken | 2 | **kept** |

The two that stayed are the point:

- **HIVE-118** (stale `uvx hive-vault` MCP entry) is correcting and has a Windows
  twin. Skipping it leaves the `hive` MCP pinned to a retired definition that the
  surrounding skip-if-present loop will then never replace. Probed clean on
  Linux; the Windows box cannot be probed from here, so it stays.
- **MEM-002** (claude-mem retirement) **had never worked**. It strips a
  `settings.json` key Claude Code stopped writing; the registry moved to
  `plugins/known_marketplaces.json`, which no code in the repo mentions. The
  `rm -rf` runs, the marketplace re-clones on the next session, forever. Filed as
  #1431.

A twelfth block, the `GEMINI.md` removal, turned out to be deleting a live
harness deploy target — lesson 256.

So of eleven "finished one-time migrations", **one had never functioned, one
could not be shown to have converged, and one was actively destroying a file the
harness deploys.** The ticket's framing — delete ~200 lines of finished work —
would have shipped all three outcomes as a cleanup.

## Why the missing expiry is the root cause

Without a removal date, every reader has to re-derive convergence from scratch,
and the cheapest derivation is reading the comment — which is exactly the step
that fails (lesson 256). The blocks are also *cheap to keep*: each is a guarded
no-op on a converged machine, so nothing ever pressures anyone to check. They
accrete until someone opens a debt ticket, at which point the accumulated
verification cost lands in one PR, on someone with no memory of why any of them
was written.

An expiry converts an unbounded question ("has every machine that will ever run
this already run it?") into a bounded one ("has the date passed?").

## What to do instead

**When you write a one-time migration, write when it dies.** A comment is
enough; the mechanism is the discipline, not the tooling:

```sh
# EXPIRES: 2026-12-01 (OPS-040) - converge machines off X; delete after this
# date once `dotf doctor` reports no machine still needing it.
```

Two things make the date honest rather than decorative:

- **Name what "converged" means, checkable from a machine.** "Delete after
  2026-12-01" is a guess; "delete once `crontab -l` shows no matching line on
  every box" is a probe someone can actually run — and running it is what caught
  MEM-002.
- **Classify by skip-cost when you write it, not when you delete it.** "Skipping
  this leaves harmless cruft" and "skipping this leaves the MCP entry broken" are
  different lifetimes, and only the author knows which one it is.

Deliberately not built here: a `# EXPIRES:` CI check. The convention costs a
comment; enforcing it is construction, and the repo caps meta-work while opened
tickets outnumber closed ones. Proposed to the owner as a decision instead.

## Related

- #1333 — this purge; #1431 — the block that never converged
- Lesson 256 — the block's own description of what it deletes is not evidence
- Lesson 247 — a pointer into a deletable location keeps every check green until
  the deletion, and then the cleanup gets blamed
