---
id: "lesson-615-git-does-not-check-a-lease-on-a-ref-already-up-to-date"
type: lesson
status: active
title: "git does not check a lease on a ref that is already up to date"
created: "2026-11-20"
---

# git does not check a lease on a ref that is already up to date

## Context

`dotf pr push-lease` (HARNESS-203, #3612) runs `git push --force-with-lease=refs/heads/<b>:<expect>`
and promises that a wrong `--expect` is refused with a non-zero exit.

## Problem

git compares the pushed commit with the remote ref before it looks at the lease. When they are equal
it reports the ref as up to date, prints `Everything up-to-date` and exits 0, whatever `<expect>` says.
Measured 2026-11-20 against a local bare remote: a lease naming a sha the remote never held passed.
A caller reading the exit status would take a stale lease for a confirmed one, and a line built from
`<expect>` would report a move that never happened. `lane.Claimer.Acquire` met the same rule from the
other side: two identical claim commits both "win" because the second push is up to date.

## Solution

Read the remote ref first (`git ls-remote <remote> refs/heads/<b>`) and refuse when it is not the
expected sha; the lease still covers a push that lands between that read and the push. More generally,
a lease is a condition on a change: where no change happens, nothing is checked, so a test of a lease
needs a row where the remote already equals what is pushed.
