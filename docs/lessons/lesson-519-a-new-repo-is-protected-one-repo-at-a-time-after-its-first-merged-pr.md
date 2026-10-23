---
id: "lesson-519-a-new-repo-is-protected-one-repo-at-a-time-after-its-first-merged-pr"
type: lesson
status: active
title: "A new repository is protected one repository at a time, after its first merged PR"
created: "2026-10-22"
---

# A new repository is protected one repository at a time, after its first merged PR

## Context

GUARD-027 (#2967) declared `mlorentedev/ci-workflows` in `forge/branch-protection.json`, requiring `lint`. The
repository was a day old: its only commit went straight to `main`, because an empty repository cannot receive a
pull request, and `lint` had reported on that push.

## Problem

Two things in the plan "merge the declaration, then `dotf forge protection apply`" were wrong.

`check --unmerged` reported drift on a second repository: kubelab's live protection requires `deploy-shape`,
which the declaration does not (#2969). `apply` writes the complete declared object to every repository that
differs, so a bare run would have removed a required check from kubelab while protecting ci-workflows.

`apply --dry-run --unmerged --repo mlorentedev/ci-workflows` refused: `lint` had not reported on any of the
branch's last 5 merged pull requests. The preflight exists because, with `enforce_admins`, a required context
that never reports locks the branch. A push run does not prove the context reports on pull requests, so a
repository with no merged PR cannot pass it.

## Solution

The declaration landed ahead of its apply, which is the documented order. The apply waits for the repository's
first merged pull request, and runs with `--repo`.

## Rule

Run `check` before `apply`, and pass `--repo` whenever any other repository shows drift: apply converges every
repository it reads, not only the one you edited. A new repository cannot become protected until a pull request
on it has merged with the required context reporting; plan its first real PR before the apply, never a
content-free one to feed the preflight.
