---
id: "lesson-617-a-guard-that-reads-one-repository-must-prove-the-push-writes-only-that-one"
type: lesson
status: active
title: "A guard that reads one repository must prove the push writes only that one"
created: "2026-11-23"
---

# A guard that reads one repository must prove the push writes only that one

## Context

`dotf pr push-lease` reads the remote with `git ls-remote` to refuse a push to the default branch
or under a stale lease, then runs `git push`. Every check is about the repository `ls-remote`
read, so it protects only if `git push` writes that repository and no other (HARNESS-203, #3612).

## Problem

The guard was bypassed three times, each by a different way git picks the push destination, and
each fix closed only the case in front of it:

1. `remote.<r>.pushurl` and `url.<base>.pushInsteadOf` change the push URL while the read uses the
   fetch URL. Fixed by comparing `git remote get-url --all` with `get-url --push --all`.
2. A second `remote.<r>.url` makes both lists equal, so that comparison passed. But `ls-remote`
   reads only the first URL and `git push` pushes to every URL a remote has. A reviewer forced the
   default branch of the second repository.
3. `remote.<r>.receivepack` (or `uploadpack`) names the program run on the remote for one
   direction, so a wrapper sent the push to another repository while every URL agreed.

Fix 1 tested a proxy, "the URL lists are equal", not the invariant, "exactly one push target, and
it is the one that was read".

## Solution

State the invariant first, then list every mechanism git has for choosing a destination, per
direction, and test each one against real bare repositories before calling the guard done. For
git, those mechanisms are:

- the URL lists: count and equality;
- the per-direction programs, `receivepack` and `uploadpack`;
- legacy `.git/remotes/` and `.git/branches/` files;
- the remote resolution order, `pushRemote`, then `pushDefault`, then `branch.<b>.remote`;
- push recursion into submodules.

Code configured for both directions alike, such as `core.sshCommand` or a remote helper, is a
different class. It cannot be told apart from a legitimate transport, and whoever sets it already
runs code at push time. Name it as an accepted risk rather than leaving it unlisted.

A sweep run by someone who did not write the fix found the third case in one pass; the review
rounds had found the first two one at a time.
