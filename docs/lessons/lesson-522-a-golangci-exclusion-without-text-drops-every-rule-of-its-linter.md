---
id: "lesson-522-a-golangci-exclusion-without-text-drops-every-rule-of-its-linter"
type: lesson
status: active
title: "A golangci exclusion without text drops every rule of its linter"
created: "2026-10-30"
---

# A golangci exclusion without text drops every rule of its linter

## Context

#2812 adds a second forbidigo rule to `cli/.golangci.yml`: `filepath.Glob`, which drops I/O errors (lesson 491).
The first rule, `runtime.GOOS` (#2061), applies to `internal/doctor` only. An exclusion scoped it: forbidigo is
off for every path except `internal/doctor/`.

## Problem

That exclusion names the linter, not the rule. Added as it stood, the Glob rule would have run in
`internal/doctor` and nowhere else, and `golangci-lint run` would have reported `0 issues` with the two blind
Glob sites still in `sshtarget` and `memlink`. A clean lint run cannot tell an empty result from a rule that
never ran.

## Solution

The exclusion now also matches `text: runtime\.GOOS`, so it removes only that rule's findings. The new rule was
proven by mutation: with one `//nolint:forbidigo` removed, the pinned linter reports
`memlink.go:164:17: use of filepath.Glob forbidden` under `internal/memlink`, outside doctor. The full run still
reports `0 issues`, so the GOOS rule stays scoped as before.

## Rule

A linter that holds several rules scopes one of them with `text:` in its exclusion, not with a path alone. When
you add a rule to such a linter, delete a `nolint` (or plant a violation) outside every excluded path and check
that the lint fails before you trust its `0 issues`.
