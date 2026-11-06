---
id: "lesson-550-a-drift-check-that-walks-only-the-checkout-cannot-see-a-new-source"
type: lesson
status: active
title: "A drift check that walks only the checkout cannot see a new source"
created: "2026-11-04"
---

# A drift check that walks only the checkout cannot see a new source

## Context

`dotf harness check --against-vault` compares the committed skill and agent records with the vault sources
they were refreshed from. Each record carries a `generated_from` and `generated_sha` stamp, and the check reads
those stamps.

## Problem

The check iterated over the records that existed, so it saw only what a refresh had already written. A vault
agent added after the last refresh (`hermes-kubelab`, #3098) had no record and therefore no stamp, and the check
had nothing to read. `refresh` wrote the record and `check` said nothing about its absence. With the record
present, the check reported it OK. The two verbs disagreed on whether the record was required, and the green
check meant only "what is committed matches".

## Solution

The check now also walks the source side (`compareCoverage` in `cli/internal/harness/vault.go`). It reports each
vault record that a refresh would write and the checkout lacks as `[MISSING]`. A record that is not shipped is
named in the manifest's `exclude` map with its reason, as an enforced `opt_out` is. `refresh` skips it, and the
check reports a committed copy as stale. The general rule is that a comparison driven from one side detects edits
and deletions on that side, and never detects an addition on the other. Walk both sides, or say which one the
check covers.
