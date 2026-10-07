---
id: "lesson-360-a-pr-inferred-from-the-checkout-is-not-the-pr-being-pushed"
type: lesson
status: active
title: "A PR inferred from the checkout is not the PR being pushed"
created: "2026-10-07"
---

# A PR inferred from the checkout is not the PR being pushed

## Context
The local spec-gate pre-push hook reads the labels and body of "this branch's PR" so that
`skip-sdd` and archive-on-merge apply before CI does. It asked with a bare `gh pr view`. On
2026-10-07 a branch was created from another PR's remote branch to stack on it
(`git worktree add -b fix/x ... origin/test/y`). Git set `origin/test/y` as its upstream. On push,
`gh pr view` resolved the PR through that upstream, read the other PR's labels (no `skip-sdd`)
and refused the push. The `SDD_LABELS` and `SDD_PR_BODY` values set by hand were overridden,
because the live lookup wins over them.

## The Trap
`gh pr view` with no argument does not answer "which PR does this push belong to". It infers a PR
from the checkout: the upstream branch, then the branch name, in any state. That is the same root
cause as #1152, where a branch whose PR had merged kept resolving to the merged PR and every later
push was refused. Both failures make `--no-verify` the visible way out, which disables the gate
for every ref in that push.

## The Solution
Ask for the PR by the ref being pushed, and ask only for an open one:
`gh pr list --head "$branch" --state open --limit 1`. pre-commit exports the pushed ref as
`PRE_COMMIT_REMOTE_BRANCH`. Outside a hook, the current branch stands in, and a detached HEAD asks
nothing. An empty answer is real `gh`'s "no open PR", so it takes the same fall-through as a
failure. To mutation-check the fix, ignore the pushed ref, drop the empty check or widen the state
to `all`; each one turns a test red.

When creating a stacked branch, run `git branch --unset-upstream`. Several tools other than this
hook follow the upstream.
