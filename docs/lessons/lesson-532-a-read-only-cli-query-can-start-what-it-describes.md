---
id: "lesson-532-a-read-only-cli-query-can-start-what-it-describes"
type: lesson
status: active
title: "A read-only CLI query can start what it describes"
created: "2026-11-02"
---

# A read-only CLI query can start what it describes

## Context

The doctor's `[Claude extensions]` check and the `agent-extensions` converge step ask `claude mcp get <name>` for
each server in `mcp-servers.json`. They read only `Scope`, `Type`, `Command`/`Args` or `URL`, to compare the
registration with the declaration (#2393, #2397).

## Problem

`claude mcp get` health-checks the server it describes. That means starting it: `npx -y`, `uvx` or a remote
handshake. The cost hid on a workstation, where the caches are warm and each call took about a second. On a
Windows CI runner with empty caches, the check took ~54 s of `test-windows`' setup step, and the doctor gate paid
again (#2059). The code comment said "about a second per server", which was true only where it was measured. Nobody
read the `Status` line the check paid for.

## Solution

The runner sets `MCP_TIMEOUT=1` on the `mcp get` child, after the parent's environment so it wins. The
registration fields print the same, and a failed connection still exits 0, so "registered" and "not registered" do
not change. Measured locally: ~0.4 s per call instead of 0.6–1.5 s, and `dotf doctor --scope machine` went from 12 s
to 7 s on a warm Mac.

## Why

Before reading a query's output in a loop, check what the query does to answer. A "get" that connects, resolves or
downloads costs what its slowest dependency costs on the coldest machine you run it on. When you read only part of
the answer, find the switch that skips the rest. Measure on the cold machine, not the one you develop on.
