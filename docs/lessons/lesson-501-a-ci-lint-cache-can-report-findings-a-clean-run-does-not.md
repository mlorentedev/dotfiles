---
id: "lesson-501-a-ci-lint-cache-can-report-findings-a-clean-run-does-not"
type: lesson
status: active
title: "A CI lint cache can report findings a clean run does not"
created: "2026-10-21"
---

# A CI lint cache can report findings a clean run does not

## Context

`cli-lint` runs `golangci/golangci-lint-action`, which by default restores a
golangci-lint cache keyed on `go.sum` and saves one after each run. Main's
`cli-lint` on the pflag bump (#2816) passed with `0 issues` and saved a fresh
cache under the new `go.sum` key.

## Problem

Every pull request touching `cli/` after that restored the cache and failed on
four staticcheck SA5011 findings in `internal/cmd` tests none of them changed:
a dereference after `if x == nil { t.Fatalf(...) }`, which cannot be reached
with a nil. A clean local run with the pinned linter, on darwin and on
`GOOS=linux GOARCH=amd64`, reported `0 issues`, and a rerun of the job failed
the same way. The finding looked like code; it was the cache (#2834).

## Solution

Lint without the cache (`skip-cache: true`). The job takes about 15 s without
it, so the cache bought almost nothing and could turn every PR red at once.

## Rule

A lint result that differs between a clean run and a cached one is a cache
defect, not a finding: reproduce with the cache cleared before touching code.
In CI, a cache that saves seconds on a short job is not worth a failure mode
that reaches every PR; this is lesson 353's shared local cache, in CI.
