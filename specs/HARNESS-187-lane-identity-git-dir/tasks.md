---
tags: [spec, tasks, templates]
created: "2026-11-11"
id: "HARNESS-187-lane-identity-git-dir"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-187-lane-identity-git-dir

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/3261-lane-identity` (worktree `feat-3261-lane-identity`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (each risk carries its answer)

## Implementation

- [ ] [P] [AC4] Red: `cli/internal/lane/identity_test.go` — `TestIdentity_ReadsTheRecordFromTheWorktreeGitDir`: a linked worktree whose git admin dir holds `dotf-lane` answers that name; a worktree without the file, the main checkout and a plain temp dir answer `("", false, nil)`. Expected: FAIL with `undefined: Identity`.
- [ ] [AC4] Green: `cli/internal/lane/identity.go` — `const LaneIdentityFile = "dotf-lane"`, `Identity(path) (string, bool, error)` reads `git -C path rev-parse --absolute-git-dir` (git >= 2.13) then the file; `writeIdentity(path, name) (bool, error)` writes only when the content differs. Commit.
- [ ] [AC1] Red: `cli/internal/lane/lanes_test.go` — `TestLaneInitRecordsTheIdentityInEachLanesGitDir`: after `InitLanes`, `Identity` answers the lane name for all three lanes; a second `InitLanes` returns `0`. Expected: FAIL, `Identity` answers `""`.
- [ ] [AC1] Green: `provisionLane` writes the record on every exit (create, adopt, and the healthy-lane early return) and counts a rewrite as a change. Commit.
- [ ] [AC1] Red: `TestLaneInitRestoresADeletedIdentityRecord`: init, delete one lane's record, init again returns `1` and the record is back, a third run returns `0`. Expected: FAIL until the healthy-lane exit writes it.
- [ ] [AC3] Red: `TestLaneInitRefusesALaneDirectoryOfAnotherRepository`: two repositories named `dotfiles` under distinct parents, one `wtRoot`; `InitLanes` on the second returns an error containing the lane path and the first repository's git dir; the first repository's `.dotf-worktree.json` files are byte-identical before and after. Expected: FAIL, the second init returns `0, nil`.
- [ ] [AC3] Red: `TestLaneInitRefusesAPlainDirectoryAtALanePath`: a non-git directory at the lane path is refused, not given metadata. Expected: FAIL (metadata written today).
- [ ] [AC3] Green: `provisionLane` requires the existing path to be a worktree toplevel (`rev-parse --show-toplevel` equal to the path, symlinks resolved) whose `--git-common-dir` is the repository's; otherwise an error naming the path and the owner. Commit.
- [ ] [AC2] Red: `cli/internal/hooks/lane_guard_test.go` case 10 rewritten: (a) an unleased worktree named `spike-lane` with no record passes; (b) with `lane.LaneIdentityFile` written into its git admin dir (the Go constant, so sh and Go cannot drift) a real `git commit` with the hook installed as `pre-commit` is refused with `lease metadata` and the lane name; (c) after `git worktree move` to `not-a-lane-name` it is still blocked. Expected: (a) FAIL (blocked today), (c) FAIL (passes today).
- [ ] [AC2] [AC5] Green: `git-hooks/lib/lane-guard.sh` — resolve `git_dir` with `rev-parse --absolute-git-dir`, replace the `canonical_lanes` loop with `[ -f "$git_dir/dotf-lane" ]`, drop the `canonical_lanes` read, update the header comment. `shellcheck`, then `go test ./internal/hooks/ ./internal/lane/`. Commit.
- [ ] [AC5] Docs: runbook `docs/runbooks/guide-persistent-named-lanes.md` (identity section, re-run `lane init` after updating, the guard's rule 5b), ADR `docs/adr/adr-061-lane-identity-in-the-worktree-git-dir.md`, lesson 579 (identity belongs with the owner of the lifecycle; a knob measured with a listener). `dotf lessons fmt`. Commit.
- [ ] File the follow-up ticket: a layout for two lane-managed repositories that share a basename under one `WORKTREE_ROOT`.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] `go build ./... && go vet ./... && GOOS=windows go vet ./...`, `golangci-lint run ./...` (pinned), `shellcheck git-hooks/lib/lane-guard.sh`
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder
