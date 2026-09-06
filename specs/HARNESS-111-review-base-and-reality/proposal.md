---
id: "HARNESS-111-review-base-and-reality"
type: spec
status: implementing # draft | implementing | verifying | archived
issue: "mlorentedev/dotfiles#1533"
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-111-review-base-and-reality

## Why

BUG-093 took **four rounds and four FAILs** against a repository history of 30 PASS,
7 PASS-WITH-GAPS, 1 FAIL, and only 4 specs ever needing more than one round. It is a 4x outlier,
and the cause is two architectural defects in the gate rather than anything about that spec.

### Root 1 — the review has no base, so on `main` it reviews nothing

`harness/skills/adversarial-review/SKILL.md` defines the scope as `git diff <base>...HEAD`.
**The launcher recorded no base.** `review-request.json` carried `reviewed_sha`, `reviewer`,
`requested_at` and `review_digest_before` — no base concept existed anywhere in
`cli/internal/spec/`. The reviewer had to guess, guessed `main`, and a review launched *on* `main`
after the work merged therefore diffed nothing.

Measured, not argued:

| Round | Launched on | Diff | Evidence verbs in its own findings |
|---|---|---|---|
| 3 | branch `dab7b6e` | real | "Mutation: … rc=0, NOT CAUGHT", "Measured through the production path", "Reproduced with `docker run`" |
| 4 | `main`, post-merge | **empty** | "Code read", "Code read" — nothing executed |

Round 4 named `git diff main...HEAD` as its source and that diff did not exist. It returned one
THEORETICAL finding and one that was **factually wrong**: it cited `Inside: false`, a string that
appears nowhere in `cli/`, having read a mutation payload out of the spec folder's committed
harness as if it were the implementation.

**The documented workflow guarantees this.** The doctrine requires the review before archiving;
archiving follows the merge; so the review lands post-merge, where its primary input is gone.

### Root 2 — the skill contradicts itself about the verdict

The verdict rule is stated twice and the two disagree:

- Reality classification: *"weight each finding by `severity × reality`. A **REAL** Blocker/Major
  forces FAIL; a **SPECULATIVE** finding cannot, by itself, move the verdict below PASS."*
- Verdict list: *"**FAIL** — at least one blocker or major OR rubric has at least one D."*

A reviewer following the second returns FAIL on a Major it has itself labelled THEORETICAL. That
is round 4 exactly: a race-window narrowing with no demonstrated exploit, costing a full
merge-and-re-review cycle.

This is not a missing policy. It is a contradiction inside one document, and the fix is to resolve
it in favour of the rule that was already stated more precisely.

## What

1. `dotf spec review` resolves a base and records it as `base_sha` in `review-request.json`, then
   states it in the reviewer's prompt.
2. The launcher **refuses** when no base resolves, or when the base is HEAD.
3. The skill's verdict list is made consistent with its own Reality rule.

### How the base is resolved, and why not from the PR

`ResolveReviewBase` returns the **parent of the commit that first added the spec folder**.

The PR route (`issue:` → PRs closing it → oldest PR's base) needs the network, `gh` auth, and PR
metadata a human can edit, and it answers differently depending on which of a spec's several PRs
you ask. The chosen route is local, offline and deterministic, and it is correct in both
situations that matter:

- **on the work branch** — the adding commit is on the branch, so the parent is where the branch
  left `main`;
- **on `main` after a squash merge** — the adding commit *is* the squash commit, so the parent is
  `main` immediately before the work landed.

The second case is the entire point: it is the one the old code got wrong by having no answer.

## Out of scope

- **Softening Majors in general.** Round 1's Blocker on BUG-093 was a real data-loss bug
  (`filepath.Abs` does not resolve symlinks, so a symlinked worktree failed open with a process
  inside) and round 3's six REAL Majors were the gate earning its keep. Only the reality axis is
  honoured; the severity bar is untouched.
- **Blockers.** A Blocker blocks whatever its reality tag. The cost of being wrong about a Blocker
  is the thing the gate exists to prevent.
- **Delta-since-last-round scope.** The base is fixed and the head moves. A per-round delta would
  stop round N seeing a fix that round 1 forced, so it could not judge whether a late change
  interacts with an early one.
- **Backfilling `base_sha` into archived specs.** They carry the old shape and must keep parsing.

## Risks / open questions

- **The skill is deployed, not read from the repo.** The reviewer reads
  `~/.pi/agent/skills/adversarial-review/SKILL.md`, which `compile-harness.sh` writes. Root 2's fix
  has no effect until a deploy runs. Called out in `verification.md` rather than assumed.
- **The base resolver trusts the spec folder's add-commit.** A spec folder created in one commit
  and then rewritten by an interactive rebase would resolve to the rebased commit's parent. That is
  the correct answer for the history that exists, which is the only history a reviewer can read.
- **Command-level tests drive a seam.** `makeRepo` builds a fake `.git` directory, not a
  repository, so the production resolvers answer "" there. The resolvers are tested against real
  git histories in `internal/spec`; the seam proves the caller honours the answer. Neither alone is
  sufficient — the lesson BUG-093 round 3 taught.

## Acceptance criteria

1. `ResolveReviewBase` returns the parent of the commit that added the spec folder, and `""` when
   no such commit exists.
2. On a repository where `git diff main...HEAD` is **empty**, the resolved base still produces a
   non-empty diff — asserted with the naive-diff emptiness as an anchor, so the test cannot pass on
   a fixture that fails to reproduce the defect.
3. `dotf spec review` refuses when no base resolves, and refuses when base equals HEAD, with a
   message naming which.
4. The reviewer prompt states the base and tells the reviewer not to substitute one.
5. `review-request.json` records `base_sha`, and a sidecar written before the field existed still
   parses.
6. The skill's verdict list qualifies Major by reality; a Blocker still blocks at any reality; a
   guard fails if either half regresses.
7. `go build`, `go test ./...`, `golangci-lint run`, `GOOS=windows go vet ./...` clean; the new
   guard passes and is mutation-proven.
