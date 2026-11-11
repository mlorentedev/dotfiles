---
id: "ADR-061-lane-identity-in-the-worktree-git-dir"
type: adr
status: accepted
owner: manu
date: "2026-11-11"
issue: "mlorentedev/dotfiles#3261"
extends: [adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers]
tags: [architecture, decision, worktrees, lanes, git-hooks, cross-platform]
created: "2026-11-11"
---

# ADR-061: A lane's identity is a record in the worktree's git admin dir

## Status

Accepted 2026-11-11.

## Context

ADR-060 provisions three persistent named lanes per repository at
`<WORKTREE_ROOT>/<repo basename>/<lane>` and arms a POSIX pre-commit guard from
the committed marker `.dotf/lanes.json`. The guard's rule 5b, "a lane without a
lease is blocked", decided what a lane *was* by comparing the directory's last
path segment with the marker's `canonical_lanes`, and only when the lease file
`.dotf-worktree.json` was absent.

A name is not an identity. Three measurements (2026-11-11, #3261) showed the
rule answering wrongly in both directions:

- A worktree that merely shares a lane's name is refused commits; a lane whose
  leaf changed (`git worktree move`) stops being recognised.
- `git clean -fdx` inside a lane deletes the lease, which lives only in
  `info/exclude`, so the name rule is reached by a routine command (#3307).
- `dotf lane init` for two repositories sharing a basename under one
  `WORKTREE_ROOT` silently adopted the first repository's lanes: `provisionLane`
  never checked that the directory it found was a worktree of the repository
  being initialised. The same path convention made `HasNamedLanes` answer true
  for a stranger's lanes, which would route a handoff into them.

The block on the `dotf lane init` rollout to the other lane-managed repositories
was this ambiguity, not a missing feature.

## Options considered

1. **Keep the name rule, add the lease to the common dir.** Fixes `git clean`
   only; a renamed lane and a same-named stranger stay wrong, and two
   repositories with one basename still collide.
2. **Treat an app-created worktree under a `third_party_roots` entry as a
   lane.** Rejected by the owner (2026-11-11): those roots belong to an external
   session host and rule 4 keeps blocking them.
3. **Store the lane's path somewhere.** Every path store is wrong on the next
   `worktree move`, differs per OS and seat, and reintroduces the machine
   knowledge the marker was designed to exclude.
4. **Record the identity where git keeps the worktree's own state.** Chosen.

## Decision

1. **`dotf lane init` writes the lane name to `dotf-lane` inside the worktree's
   git admin dir** (`<repo>/.git/worktrees/<id>/`, the directory
   `git rev-parse --absolute-git-dir` answers from the lane). Git owns that
   directory's lifecycle, measured on a scratch repository: the record survives
   `git worktree move`, `git gc` and `git worktree repair`; it is gone after
   `git worktree remove` and after `rm -rf` plus `git worktree prune`; a
   worktree re-created at the same path starts without it; `git clean -fdx`
   never reaches it. No path is stored anywhere. The committed marker is
   unchanged.
2. **The record is written on every exit of `provisionLane`**: create, adopt and
   the healthy early return, so a record deleted by hand is restored by the next
   init and the command stays idempotent (`changed=0` on re-run; `changed=1`
   when exactly a record was restored).
3. **A directory already at a lane path is adopted only when it is a linked
   worktree of the repository being initialised**, compared by common dir and
   top level with symlinks resolved. A plain directory, a main checkout, a
   directory inside some worktree and a worktree of another repository are
   refused, naming the path and its owner. `HasNamedLanes` applies the same
   membership check before routing a handoff.
4. **The guard's rule 5b reads `$git_dir/dotf-lane` and nothing else.** A
   registered lane without a lease is blocked whatever its directory is called;
   an unregistered directory is never blocked by its name. The guard no longer
   reads `canonical_lanes`; that list stays in the marker as the policy
   `dotf lane init` writes and `doctor` will check (#3262). The guard never
   writes the record: migration is `dotf lane init` run again.
5. **`lane.Identity(path)` is the one Go reader** of the record, answering
   `("", false, nil)` for a worktree without a record, a main checkout and a path
   outside any repository, for `doctor` (#3262), `worktree list` and `sweep`
   (#3293).

## Consequences

- Guard test case 10 changes meaning: it pinned that an unleased directory
  *named* after a declared lane is blocked, which is the defect. It now asserts
  the opposite for the unregistered directory, the block for the registered one
  through a real `git commit` with the guard installed as the pre-commit hook,
  and that the block survives `git worktree move`.
- Lanes provisioned before this record need one `dotf lane init` after updating
  (`changed=3`). Until then rule 5b does not fire for them; nothing else changes.
- Two repositories sharing a basename under one `WORKTREE_ROOT` can no longer
  share lanes by accident; they cannot both have lanes either, until a layout
  that keys the lane directory on more than the basename exists (follow-up
  filed from HARNESS-187).
- The lease (`.dotf-worktree.json`) stays in the worktree and stays deletable by
  `git clean -fdx` (#3307). This record is identity, not state: moving the lease
  is a separate decision.
- The guard absolutises `git_dir` the way it already absolutised `common_dir`,
  since `git rev-parse --git-dir` answers `.git` at a main checkout; a main
  checkout has no worktree admin dir and never carries the record.
- Windows is unverified on the authoring box; the record path is composed by
  git, no separator is built by hand, and the Go tests run on all three CI
  legs.

## References

- [ADR-060](adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md): the lanes, the marker and the guard this record completes.
- [ADR-057](adr-057-project-workspace-boundary-and-worktree-roots.md): the worktree roots a lane may occupy.
- `docs/runbooks/guide-persistent-named-lanes.md`: the operational guide, including the re-init step.
- `specs/archive/HARNESS-187-lane-identity-git-dir/`: measurements and acceptance.
- #3261, #3262, #3293, #3307.
