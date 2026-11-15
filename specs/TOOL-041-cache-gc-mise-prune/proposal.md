---
id: "TOOL-041-cache-gc-mise-prune"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-11-15"
issue: "mlorentedev/dotfiles#3533"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# TOOL-041-cache-gc-mise-prune

> **Naming**: file lives at `<repo>/specs/TOOL-041-cache-gc-mise-prune/proposal.md`. `TOOL-041-cache-gc-mise-prune` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3533: cache-gc: old mise tool versions are never pruned -->

When a pin moves, the version it replaces stays installed for good. The `tools` step prunes only tools that no pin names, and `cache-gc` runs no mise collection. On the Mac on 2026-11-15, `mise prune --dry-run` listed 33 prunable installs. Keeping a machine current has two halves, updating and then cleaning up, and the cleanup half is missing for mise.

## What

`cache-gc`, the weekly converge step that `dotf update` runs, also prunes old mise tool versions. It lists the installed tools (`mise ls --installed --json`) and runs `mise prune --tools --yes` with every one of them **except python**. mise itself keeps every version a tracked config asks for and every version a running process uses. A plan names the command without running it.

Python is held because a venv records its interpreter by absolute path. On this Mac, hive's uv tool and about 55 kubelab venvs point at mise's Python 3.13.16. Pruning the version a pin moved away from would break all of them.

## Out of scope

- Rebuilding venvs on the new Python after a Python pin bump. Without that, a Python bump breaks them even with the hold, once anything else removes the old version. It belongs with the bump channel (#3284).
- Old versions that `mise upgrade` schedules for deferred removal ("pruned in 24h"). That is mise's own mechanism.
- A per-tool hold list in configuration. One hard-coded entry has one known reason.

## Risks / open questions

- mise has no exclusion flag, and `mise prune` with no tools named prunes every tool. So an empty list must run nothing. That case has a test.
- If the listing fails, the step fails and keeps its stamp unwritten, so the next daily run retries. Nothing is pruned without a listing.

## Acceptance criteria

- [x] AC1: with mise on PATH, an apply runs `mise ls --installed --json` and then `mise prune --tools --yes` naming every installed tool, sorted, except python.
- [x] AC2: when python is the only installed tool, no prune command runs. When the listing fails, nothing is pruned and the step fails.
- [x] AC3: the runbook (`guide-self-deploy-timer.md`) and ADR-036 list the mise pruner and the python hold.

## References

- Issue #3533; companions #3284 (pin bumps by PR) and #3535 (OS updates)
- ADR-036 (`cache-gc` section), ADR-044 (mise pins are exact)
- Lesson 605
