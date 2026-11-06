---
id: "lesson-553-a-token-loop-is-a-process-loop-on-windows"
type: lesson
status: active
title: "A token loop is a process loop on Windows"
created: "2026-11-05"
---

# A token loop is a process loop on Windows

## Context

The doc-path pre-commit guard scanned 14 instruction files and 971 backticked tokens. Its shell loop launched
several `grep` or `bash` processes for most tokens. On the affected Windows machine, 400 Git Bash process
launches took 36.4 seconds; one 100-token README exceeded a 15-second timeout and the full commit hook could
run for minutes or hours.

## Problem

The algorithm looked linear in shell source, but its real unit of work was process startup. A per-token
pipeline multiplied file count, token count and cross-runtime launch cost. Adding a timeout would have turned
the performance failure into a silent false negative, while optimizing the script would have kept cross-OS
tooling in the layer ADR-020 is retiring.

## Solution

Parse and classify the complete token set in one long-lived process. Keep policy as checkout data, invoke Git
once for discovery, sort findings deterministically, and measure the full real repository on the slowest
supported host. When a shell guard accumulates incident-driven parsing rules, process count is an architectural
metric: batch the work or move the engine to the cross-platform CLI before content growth turns every commit
into an outage.
