---
id: "lesson-329-a-git-fixture-with-identical-content-is-a-rename"
type: lesson
status: active
title: "A git fixture whose new files repeat the deleted ones' content is a rename, and counts zero lines"
created: "2026-10-01"
owner: manu
tags: [lesson, testing, git, spec-gate, bats]
---

# A git fixture whose new files repeat the deleted ones' content is a rename, and counts zero lines

## What happened

#1904 needed a failing test first: delete two 200-line `.patch`/`.diff` files and add two 200-line `.orig`/`.rej` files, then assert that spec-gate passes. The test passed before the fix existed.

Every file was written with the same `printf 'line %d\n' {1..200}`. `git diff --numstat`, which spec-gate reads, detects renames by default, so it reported `stray.diff => leftover.go.orig | 0` and `stray.patch => leftover.go.rej | 0`. The production LOC was 0, below the threshold, so the test passed for a reason that had nothing to do with the exclusion under test. Only the test that deleted a file without adding another was red.

## Rule

A fixture that both deletes and adds files in one commit gives each file distinct content. Otherwise rename detection pairs them, and every line count, the gate's included, reads zero. A test written before its fix must be watched failing. Here, watching it is what showed that the test was wrong.

## Guard

In `tests/check-spec-gate.bats`, the #1904 test writes `orig %d` and `rej %d`, distinct from the deleted `line %d`. It was red before `_excluded` learned the patch-tool extensions and green after.
