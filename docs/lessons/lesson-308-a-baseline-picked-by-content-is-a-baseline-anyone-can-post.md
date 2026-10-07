---
id: "lesson-308-a-baseline-picked-by-content-is-a-baseline-anyone-can-post"
type: lesson
status: active
title: "A baseline picked by content is a baseline anyone can post"
created: "2026-09-27"
owner: manu
tags: [lesson, ci, guard, pr-agent, security, cwe-345]
---

# A baseline picked by content is a baseline anyone can post

## What happened

TOOL-023's push gate (#1756) picks "the previous review" the same way PR-Agent itself does: the most recent comment whose body carries a review marker. Mirroring the tool it gates was the right call — it stops the gate from starting a run PR-Agent would then decline.

What it mirrored along with the rule was the rule's own gap: PR-Agent matches that marker by body text, with no check on who posted it. Neither did the gate. On a public repository, `issue_comment` fires for any account, so a comment reading `## PR Reviewer Guide` posted by anyone stood in for a review that never happened — reviewed on #1757's own triage by reproducing it: that exact fixture returned `run=false`.

The same content-shaped hole reopened a second time inside the fix. New commits were counted by comparing each commit's author date to the baseline's timestamp — correct until a rebase, which preserves author dates while giving every commit a new SHA. This repository rebases with `--onto` routinely, so the date comparison was already wrong for its own workflow, not a hypothetical one.

## The rule

A value picked by matching CONTENT (a marker string, a date) rather than IDENTITY (an author, a commit's position in history) is forgeable or reorderable by anything that can produce content shaped the same way. Before mirroring an upstream tool's selection rule, ask what makes that value trustworthy in the upstream's own threat model — here, nothing did, because the upstream never had to defend a baseline against its own repository's public commenters. Anchor a security-relevant decision to something the surface can't cheaply reproduce (an author field checked against a known bot identity; a commit SHA checked for presence, not a date compared for order), and when no such anchor exists for a given case, fail toward the more expensive, more thorough response rather than trust the content.

Refs: TOOL-023 (#1756), #1757 (comment 5851151835), CWE-345.
