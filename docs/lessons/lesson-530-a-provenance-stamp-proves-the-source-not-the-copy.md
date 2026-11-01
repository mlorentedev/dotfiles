---
id: "lesson-530-a-provenance-stamp-proves-the-source-not-the-copy"
type: lesson
status: active
title: "A provenance stamp proves the source, not the copy"
created: "2026-11-01"
---

# A provenance stamp proves the source, not the copy

## Context

Each committed skill and agent record carries `generated_sha`, the hash of the vault source it was rendered
from. `dotf harness check --against-vault` compared that stamp with the vault's current source (#1545).

## Problem

The stamp answers one question: has the source changed since the render? A record edited in the checkout keeps
its stamp, so the source still matches and the check passed. The next `dotf harness refresh` then reverted the
edit without a word (#1545). Enforced records carry no stamp at all, so nothing compared them.

## Solution

With the render in Go, the check renders each source whose stamp matches and compares the result with the
record, file by file and with LF line endings (CLI-026 slice 7b). A difference is `[EDITED]`, and the remedy is
to carry the edit to the vault. Enforced records are compared with the vault section they are cut from.

## Why

A stamp written at render time records the input, not the output. To find a change on the output side, compare
the output itself with a fresh render. A hash of the copy taken at render time would only move the problem: an
edit could update it too.
