---
id: "lesson-529-a-guard-over-a-manifest-driven-writer-reads-the-manifest"
type: lesson
status: active
title: "A guard over a manifest-driven writer reads the manifest"
created: "2026-11-01"
---

# A guard over a manifest-driven writer reads the manifest

## Context

`dotf harness refresh` stops when the files the refresh writes hold uncommitted work, and afterwards lists what
the refresh changed so it gets committed (#2162, #295). The refresh writes `harness/` and every target in
`harness/manifest.json`. The guard kept its own list: `harness/`, `AGENTS.md` and `ai/claude/CLAUDE.md`.

## Problem

The manifest gained a third target, `ai/orca/ORCA.md`, and the list did not. An uncommitted edit there was
overwritten without a warning, and the refresh's own change to it was left out of the "commit them" report. The
writer and the guard each worked, so nothing failed. They had stopped describing the same files.

## Solution

`harness.RefreshPaths` derives the paths from the manifest the refresh reads, and the command guards and reports
exactly those (CLI-026 slice 7). The test fixture has two targets, and one of them is not named anywhere in the
command. Under the old fixed list that test fails.

## Why

A guard that copies its writer's scope from the code is a second copy of the manifest. It drifts the first time
someone edits the manifest and not the guard. Asking the source the writer uses keeps the two in step. A test
needs a target the guard's author never typed, or it only confirms the copy matches itself.
