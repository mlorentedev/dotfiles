---
id: lesson-311
type: lesson
status: active
created: "2026-09-23"
owner: manu
tags: [lesson, cli-ux, error-messages, verification, secrets]
---

# 311 — A printed remedy is run verbatim, so it must reproduce the invocation that failed

## What happened

During Bitwarden dedupe (2026-09-24), the owner ran `dotf secrets backup --out <worktree>/sensitive/dr` against a locked vault to put the DR escrow in the worktree whose PR would carry it.

The error's remedy was a fixed string, `BW_SESSION="$(bw unlock --raw)" dotf secrets backup`, which dropped `--out`. The owner copied it as printed. The escrow was written and verified in the wrong checkout, so it looked successful and the owner reported "backup done". Only checking the file's mtime and the worktree's git status showed that the backup the apply depended on was not where the PR expected it. A test pinned the fixed string, so nothing caught the loss.

## The rule

Build a remedy from the failed invocation (its flags included), or phrase it as a prefix to "the command you ran". Test that a hint built for an invocation with a flag still carries that flag. When an operator reports that a prerequisite step is done, check its artifact, not the report.

Refs: mlorentedev/dotfiles#1647.
