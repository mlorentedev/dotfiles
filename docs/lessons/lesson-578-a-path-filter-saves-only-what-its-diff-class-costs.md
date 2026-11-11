---
id: "lesson-578-a-path-filter-saves-only-what-its-diff-class-costs"
type: lesson
status: active
title: "A path filter saves only what its diff class costs"
created: "2026-11-11"
---

# A path filter saves only what its diff class costs

## Context

#3149 asked to run the Windows and macOS jobs on pull requests only when they can change, and justified it with
each job's monthly cost: `ci/test-windows` 9.2k minutes, `cli/test (windows)` 8.0k, `ci/test-macos` 5.1k. The
line the repository already draws between code and documentation is ci.yml's `code` filter, so CI-013 used it.

## Problem

The monthly figures are the cost of the jobs over every pull request. A filter on documentation-only diffs saves
only what those jobs cost on documentation-only diffs. Measured before writing the workflows: 15 of the last 150
merged pull requests were documentation only, and none of the 40 most recent green pull-request runs was. On those
diffs `test-windows` already skipped its steps (HARNESS-041), and `test-macos` and `from-zero` were already gated.
What CI-013 could remove was the Windows legs of `cli/test` and `cli-smoke` on a tenth of pull requests. The
headline numbers come from code pull requests, which a docs-only filter never touches.

A second limit came from a guard: `test-windows` is required, and CI-004's checker rejects a job-level `if` on a
required check, because a skipped required check passes without running. The job that cost the most could not be
skipped at all without changing what branch protection requires.

## Solution

The spec states the measured saving and scopes the work to what the filter reaches: a shared docs-only filter
file, dynamic OS lists for the two matrix jobs, and the integration layer cache. The narrower OS-sensitive set
that would reach code pull requests is an owner question in the proposal. Skipping `test-windows` goes through the
`gate` composite (ci-workflows#9): once `gate` is the only required CI check, it can skip at job level.

## Takeaway

Before gating a job on a diff class, count how often that class occurs and what the job costs on it. A job's
monthly cost over all pull requests is the ceiling of the saving, not the saving. Check also whether the job is a
required check: a required check cannot skip at job level, so the filter's saving there is only the steps.
