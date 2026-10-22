---
id: "lesson-512-a-landing-check-must-read-paths-in-the-frame-the-patch-wrote-them"
type: lesson
status: active
title: "A landing check must read paths in the frame the patch wrote them"
created: "2026-10-21"
---

# A landing check must read paths in the frame the patch wrote them

## Context

`dotf pr reproduce --probe <patch>` applies a patch for one check, then reverts it. It decides whether the patch
landed by hashing the files the patch names before and after `git apply`, because an apply that exits 0 can still
change no bytes. Two sessions were misled by that failure on 2026-08-17.

## Problem

`git apply --numstat` names files from the repository root. The first version hashed those paths relative to the
working directory. Run from `cli/`, as the command's own help example does, it looked for `cli/cli/...`. Both
hashes read "absent", so they matched, and a probe that had landed was reported `did-not-land`. Every test ran
from the repository root, so none of them caught it. A review before the PR found it, not a test.

## Solution

`Run` resolves `git rev-parse --show-toplevel` once. It applies, hashes and reverts the probe there, and runs the
check where the caller asked. A test now applies a probe from a subdirectory. Moving the hashing back to the
working directory turns that test red.

## Rule

A check that compares a tool's output with the file system must read paths in the frame the tool wrote them. Test
it from a frame other than the root. When a guard reports a failure as a safe state ("did not land", "nothing
found"), a bug in its frame looks exactly like a correct answer.
