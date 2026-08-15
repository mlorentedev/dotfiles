---
id: "HARNESS-073-pattern-triggers"
type: spec
status: archived
created: "2026-08-14"
issue: "mlorentedev/dotfiles#980"
tags: [spec, proposal, harness, triggers]
template_version: "1.0"
---

# HARNESS-073: File-Path Pattern Trigger Resolution in dotf

> Implements dynamic pattern trigger resolution for dotf harness per #980.

## Why

Currently, all patterns in `00_meta/patterns/` are loaded statically or by on-demand lookup, which can lead to context bloat or missed pattern application. Resolving relevant patterns deterministically from modified file paths or diffs enables targeted, just-in-time pattern injection.

## What

1. Define default trigger mapping in `harness/triggers.json` (and embedded in `cli/internal/harness/triggers.json`).
2. Implement pattern resolution in Go (`cli/internal/harness/triggers.go`):
   - Glob matching supporting recursive wildcards (`**`, `*`, `?`).
   - Diff parsing to extract changed paths.
   - Deduplicated pattern resolution.
3. Expose `dotf harness triggers [paths...]` CLI command with `--diff` and `--json` flags.
4. Unit test suite covering glob matching, diff extraction, and CLI invocation.

## Out of Scope

- Modifying the vault pattern contents themselves.
- Automatic LLM prompt injection hooks (handled in subsequent workflow steps).
