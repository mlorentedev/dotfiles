---
id: "lesson-357-a-pull-request-closed-after-a-force-push-reopens-only-at-its-old-head"
type: lesson
status: active
title: "A pull request closed after a force-push reopens only at its old head"
created: "2026-10-07"
---

# A pull request closed after a force-push reopens only at its old head

## Context
During a GitHub incident on 2026-10-07 (Git Operations, Pull Requests and Actions), pushes
failed with HTTP 500 for about ten minutes. When they started succeeding, #2108's branch held
the rebased commit `b47513bd`, but the pull request still showed the old head `daa13ec7`, and no
Actions run had started. Closing and reopening a pull request is the usual way to make GitHub
re-read the head and re-run `pull_request` workflows, and it worked for #2070, whose branch had
not been force-pushed.

## The Trap
For #2108 the close succeeded and the reopen did not. `gh pr reopen` answered "Could not open the
pull request", and the REST API said why: "state cannot be changed. The branch was force-pushed or
recreated." GitHub reopens a pull request only when the branch still points at the head it
recorded. A lagging pull request therefore did not go back to syncing after the close; it was
left closed, with its review and triage history attached to something that could not be reopened.

## The Solution
Before closing a pull request to resync it, compare `gh pr view N --json headRefOid` with
`git ls-remote origin <branch>`. If they differ, do not close it. Wait for GitHub to catch up, or
push a new commit, which is a fresh `synchronize` event. If it is already closed, recover it in
three steps: force-push the recorded head back to the branch, reopen the pull request (the API now
accepts the state change), then force-push the real commit with
`--force-with-lease=<branch>:<recorded head>`.
