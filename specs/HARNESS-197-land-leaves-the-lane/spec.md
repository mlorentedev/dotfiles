---
id: spec-HARNESS-197-land-leaves-the-lane
type: spec
status: implementing
created: "2026-11-15"
owner: manu
issue: "mlorentedev/dotfiles#3531"
---
# HARNESS-197: landing a PR leaves the lane's worktree alone

## Intent

Twice on 2026-11-14 the persistent lane `feat-lane-secondary` vanished within nine seconds of
`dotf pr land` merging the PR it held (#3526, then #3530). Nothing in `sweep`, `worktree done` or
`lane park` removes a lane that carries its lease: the removal was `gh pr merge --delete-branch`,
which `prland` passes. With it, gh 2.104.0 (the version on the machine) removes the linked worktree that has the
branch checked out ("Removed worktree %s" in the binary and `WorktreeRemove` in `pkg/cmd/pr/merge/merge.go` at v2.104.0, reached only when the flag is set, with "navigate out of the worktree" when
the caller stands inside it). The transcript of the session that landed both PRs shows no other
command anywhere in that window.

After the loss, `lane switch` counted the missing lane as ACTIVE and sent the operator to
`lane park`, which refused with "not a persistent named lane". Only `lane doctor` named the fix.

## Scope

- `prland` merges with `gh pr merge --squash --match-head-commit` and no `--delete-branch`, then
  deletes the head branch over REST (`DELETE repos/<repo>/git/refs/heads/<ref>`, each segment
  escaped). A fork's head is left alone (`isCrossRepository`), a branch already gone is not a
  warning, and any other refusal is a warning on a PR that stays merged. The local branch and its
  worktree are left to `dotf worktree done` and `dotf lane park`, which check cwd, sessions and the
  lease before touching them.
- `lane switch` names each candidate lane with what is wrong with it: ACTIVE (park it) or absent
  or without a lease (`dotf lane init`, which is idempotent). It does not recreate the lane itself:
  `init` also owns the marker and the identity record, and one command that fixes it is simpler
  than two that half do.
- `lane park` on a path that does not exist says it is absent and names `dotf lane init`.

Out of scope: refusing `dotf worktree done` on a lane (it already parks one), and the same
`--delete-branch` in a human's own `gh pr merge`.

## Checklist
- [x] A red test per behaviour, each watched failing first
- [x] `go build`, `go vet`, `go test ./...` on Linux and `GOOS=windows go vet`
- [x] The pinned golangci-lint, Linux and Windows

## Evidence

See the PR's `## Verification`.

## Next

Merge, then re-run `dotf lane init` where a lane was lost.

## Promotion candidates
- [x] Lesson? `yes: docs/lessons/lesson-606-a-landing-tools-flag-can-delete-the-worktree-it-was-run-beside.md`
- [x] ADR? `no: no decision changes; prland still merges the same head under the same gates`
- [x] Pattern? `no: one tool's side effect; the cross-project point is lesson 606`
