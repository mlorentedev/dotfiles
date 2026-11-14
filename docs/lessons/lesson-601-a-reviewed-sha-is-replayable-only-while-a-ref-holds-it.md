---
id: "lesson-601-a-reviewed-sha-is-replayable-only-while-a-ref-holds-it"
type: lesson
status: active
title: "A reviewed SHA is replayable only while a ref holds it"
created: "2026-11-14"
---

# A reviewed SHA is replayable only while a ref holds it

## Context

HARNESS-101 replays archived spec reviews at their own `reviewed_sha` (ADR-063 §2). Every archived
`review*.md` records that SHA in its frontmatter, so the archive looks like a corpus of replayable
trees. Encoding the first six candidate rounds, five SHAs were fetchable through their PR's
`refs/pull/<n>/head`. The sixth, HARNESS-106's `617cbafe` (reviewer `nan/mimo-v2.5`), answers
`422 No commit found for SHA` from GitHub: its branch was rebased after the review, and no ref kept
the commit it reviewed.

## Problem

The SHA in a review's frontmatter proves what was reviewed, not that the tree still exists. A
squash-merged branch keeps its pre-merge commits only through the PR ref, and a branch rebased
after its review drops the reviewed commit from that ref too. A case built on such a SHA validates,
self-grades clean against its source review, and fails only when a run tries to check it out.

## Solution

Resolve every candidate SHA before encoding the case: `git cat-file -e <sha>^{commit}`, else
`gh api repos/<owner>/<repo>/commits/<sha>/pulls` to find the PR and `git fetch origin
refs/pull/<n>/head`. Record that ref as `fetch_ref`. A round whose SHA no ref holds is not a replay
case. Its findings can still seed a planted case.
