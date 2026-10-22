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

The same harness then failed on the Windows runner in a second frame: bytes on disk against content as git sees
it. Under `core.autocrlf` a file git called clean was LF on disk, and every git write (apply, revert, checkout)
rendered it CRLF. The bytes never came back, although the content never changed. A first fix replaced the reverse
patch with a checkout from HEAD, on the theory that `git apply -R` was at fault. It failed the same way, because
the comparison was what was wrong. The macOS reproduction had also re-checked the file out before the run, which
removed the trigger.

## Solution

`Run` resolves `git rev-parse --show-toplevel` once. It applies, hashes and reverts the probe there, and runs the
check where the caller asked. A test now applies a probe from a subdirectory. Moving the hashing back to the
working directory turns that test red. Each target is compared by its git blob id (`git hash-object`), which
undoes the checkout's eol conversion, and restored from HEAD. A `TestRunWithAProbeThatDoesNotLand` row sets
`core.autocrlf` on a file whose bytes are not git's rendering, so every OS now runs the Windows case.

## Rule

A check that compares a tool's output with the file system must read it in the frame the tool wrote it: paths
relative to the same root, and content as the tool sees it, not as the disk holds it. Test it from a frame other
than the default one. When a fix does not change the failure, the frame of the comparison is the first suspect. When a guard reports a failure as a safe state ("did not land", "nothing
found"), a bug in its frame looks exactly like a correct answer.
