---
id: "lesson-551-a-deferred-row-outlives-its-blocker"
type: lesson
status: active
title: "A deferred row outlives its blocker"
created: "2026-11-05"
---

# A deferred row outlives its blocker

## Context

PLAT-001b's tasks.md kept an open row: the Windows Claude MCP loop in `setup-windows.ps1` stays, because its
`prerequisite_command` was the only way a Windows box got hive. The row named its blocker, OPS-051 (#1993), and
CLI-096 (#2000) was blocked on the same issue.

## Problem

#1993 closed on 2026-10-22. On that day `mcp-servers.json` dropped `prerequisite_command` and `packages.json`
started installing hive on Windows. The row and the issue still said "the loop stays", so for two weeks the loop
registered the same servers that the bare `dotf deploy` already registered. Every session that read the row trusted
its reason and did not test it again. The plan for #2000 had carried the same assumption ("the Windows MCP loop
stays behind AI-028") until the data was read.

## Solution

Before carrying a deferral forward, read its stated blocker against current data. Check that the issue is still
open, and that the field or entry it names (`prerequisite_command`, the `packages.json` platforms) is still there.
The data answers in one `jq` call. A deferral that names a blocker is a claim with a date on it, and it goes stale
in silence when the blocker closes in another PR.
