---
id: "lesson-496-gitea-names-a-deleted-head-branch-only-in-its-label"
type: lesson
status: active
title: "Gitea names a deleted head branch only in its label"
created: "2026-10-19"
---

# Gitea names a deleted head branch only in its label

## Context

#2759 taught `dotf worktree done` to ask Gitea which merged pull requests had a branch as their head. Gitea 1.25's `GET /repos/{owner}/{repo}/pulls` has no head filter, so the backend reads the closed pull requests and matches them on the client. The first version matched `head.ref == branch`, which the swagger describes as "the git reference of the branch". Its unit tests passed against a fixture built from the swagger.

A read-only probe against `gitea.kubelab.live` returned no heads for two branches that had squash-merged hours earlier (#65 and #68 in imagesensortool), and no error. The raw listing showed why:

```
#65 merged=true head.ref=refs/pull/65/head head.label=docs/decisions-d18
```

When the head branch no longer exists, Gitea sets `head.ref` to the pull ref, and only `head.label` keeps the branch name. A squash merge usually deletes the branch, so the shape the check exists for is the one where `ref` never matches. The result was an empty list, which reads as "never landed": the same answer as before the fix, now with a cause nobody would look for.

## Rule

- On Gitea, match a pull request's head branch on `head.label`, and filter forks with `head.repo_id == base.repo_id`. `head.ref` names the branch only while the branch exists.
- Build a fixture for a forge API from a recorded answer, not from the schema. A schema documents fields, not the states in which they change meaning.
- A read whose empty answer drives a decision gets one live call before it ships. Unit tests prove the code matches the fixture; only the live call proves the fixture matches the forge.

## Check

`TestGiteaLandedHeadsKeepsOnlyMergedPRsOfThisBranchInThisRepo` uses the measured shape (`ref` is `refs/pull/9/head`, `label` is the branch) together with a branch that still exists. If the backend goes back to matching on `ref`, the test fails.
