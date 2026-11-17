---
id: "lesson-587-a-running-uvx-holds-the-uv-cache-lock-for-its-whole-life"
type: lesson
status: active
title: "A running uvx holds the uv cache lock for its whole life, so a cache prune never gets it"
created: "2026-11-12"
---

# A running uvx holds the uv cache lock for its whole life, so a cache prune never gets it

## Context

#3367 added the weekly `cache-gc` converge step, which runs each tool's own safe collection. uv's is `uv cache prune`, and the uv cache was the largest prunable cache on the Mac (9.9 GB).

## Problem

The first real run of the step failed after 5 minutes: `Timeout (300s) when waiting for lock on ~/.cache/uv/.lock, is another uv process running?`. `lsof ~/.cache/uv/.lock` listed ten `uv` processes, the oldest 1 day 20 h old. All of them were `uv tool uvx hive-vault` and `uv tool uvx pdf-modifier`, MCP servers that agent sessions start and keep alive. `uvx` holds the cache lock shared for as long as the tool runs, and `uv cache prune` needs it exclusive. On a workstation with any agent session open, the prune never runs. Unit tests with a fake runner could not show this. Only the run on the real machine did.

`uv cache prune --force` ignores the in-use check. It was rejected: it can remove the environment a running MCP server executes from.

## Solution

Before pruning, the step tries the lock itself (`filelock.TryLock` on `<uv cache dir>/.lock`, the same `flock` uv takes on Unix). A held lock skips uv with the reason `cache in use`, and the other pruners still run. The root cause is filed as #3370: launch the MCP servers from installed uv tools, which run outside the cache, instead of through `uvx`.

A maintenance step that wraps another tool's GC inherits that tool's locking. Run it once on a real machine with the usual workload alive before trusting it unattended.
