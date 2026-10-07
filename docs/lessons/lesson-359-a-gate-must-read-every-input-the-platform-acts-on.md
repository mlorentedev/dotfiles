---
id: "lesson-359-a-gate-must-read-every-input-the-platform-acts-on"
type: lesson
status: active
title: "A gate must read every input the platform acts on"
created: "2026-10-07"
---

# A gate must read every input the platform acts on

## Context
The spec-gate's archive-on-merge check refuses a PR that closes an issue while the spec tracking
that issue is still active. It found the closing keywords in the PR body. #1868 wrote `Refs #1241`
in its body, so the gate passed it. One of its branch commits said `Closes #1241`. The repository
squashes with `squash_merge_commit_message=COMMIT_MESSAGES`, so that commit message became part
of the merge commit, and GitHub closed #1241 with its spec still active (#1878).

## The Trap
GitHub closes issues from two texts: the PR body and the merge commit message. Which branch text
reaches the merge commit is a repository setting, not something the PR can see. A gate that reads
one of the two inputs passes exactly the PRs that put the keyword in the other one. Its own advice
("write `Refs #N` instead") fixed only the input it read.

## The Solution
The check now reads the PR body plus `git log --format=%B base..head`. The two-dot range leaves
out commits that came in from the base through an update-branch merge. A keyword in a commit is
enforced the same way as one in the body, including on a local pre-push before the PR exists, and
it earns the same archive credit. The failure message now asks for `Refs #N` in every commit
message as well as in the body.

When a gate's input is "what the platform will act on", list every source the platform reads
before deciding which one to parse. For merge-time behaviour, the repository's merge settings are
part of that list.
