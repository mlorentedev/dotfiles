---
id: "ADR-062-the-lease-lives-in-the-worktree-git-dir"
type: adr
status: proposed
owner: manu
date: "2026-11-12"
issue: "mlorentedev/dotfiles#3307"
extends: [adr-061-lane-identity-in-the-worktree-git-dir]
tags: [architecture, decision, worktrees, lanes, git-hooks, cross-platform]
created: "2026-11-12"
---

# ADR-062: The lease lives in the worktree's git admin dir, next to the identity record

## Status

Proposed 2026-11-12 (direction chosen by the owner on #3307; the spec is
`specs/HARNESS-193-lease-in-git-dir/`). Amends ADR-061's consequence "the lease
stays in the worktree and stays deletable by `git clean -fdx`". The owner
delegated the review of this draft and its open overlap policy to the PR author
on 2026-11-12; the policy below is decided on that delegation and the owner can
overrule it before the mirror ships.

## Context

Every dotf worktree carries a lease, `.dotf-worktree.json` at its root: creator,
issue, creation time, expiry, `reap_ok`, and for a persistent named lane its
name and state (`ACTIVE` or `PARKED`). `dotf worktree sweep` reaps only past the
expiry (CLI-075), `dotf lane status` and the pre-commit guard read the state, and
ADR-061's rule 5b blocks a registered lane that has no lease at all. The file is
kept out of `git status` through `info/exclude`.

An excluded file is still an untracked file to `git clean`. Measured 2026-11-11
(#3307): `git clean -fdxq` inside an ACTIVE lane deletes the lease. The lane is
then a registered lane without a lease, blocked by the guard, and nothing names
the routine command that caused it. ADR-061 moved the lane's *identity* into the
worktree's git admin dir for exactly this reason and left the lease behind, on
record, as a separate decision.

Lesson 272 put the lease in the tree deliberately: a reaper must decide liveness
from the lease, never from a host process table that cannot see a containerised
agent, and "the liveness signal must live inside the worktree filesystem itself"
so it travels with the mounted volume. What that lesson protects is that a
lease, not a process scan, decides; it is read on the host either way. A
container that mounts the worktree without the repository's `.git` cannot run
git at all, so no dotf command reads or renews a lease there today; the admin
dir is reachable wherever git works, which every dotf command already requires.

## Options considered

1. **Keep the lease in the tree; `lane switch` re-creates a missing one and
   `doctor` reports the gap** (direction 2 of #3307). Cheap; the next
   `git clean -fdx` repeats the class, and the guard keeps blocking until the
   next switch. Rejected by the owner (2026-11-12).
2. **Put the lease in the repository's common dir, keyed by worktree id.** Safe
   from `clean`, but the key is a path or an id this code would have to compute
   and keep in step with `git worktree move`; ADR-061 rejected every path store
   for the identity on the same ground.
3. **Put the lease in the worktree's git admin dir, next to `dotf-lane`.**
   Chosen. Git composes the directory, owns its lifecycle, and already holds the
   identity record there; the lease follows the same measured properties.

## Decision

1. **The lease of a linked worktree is `<repo>/.git/worktrees/<id>/dotf-worktree.json`**,
   the directory `git rev-parse --absolute-git-dir` answers from the worktree
   and the guard already resolves as `$git_dir`. The same file holds the same
   fields; only its home changes. This applies to every dotf worktree, not only
   lanes: `worktree.LoadMetadata` and `worktree.SaveMetadata` are the one seam
   every constructor goes through (lesson 441), and a second location for
   non-lane worktrees would be the next class of this defect.
2. **Read precedence is admin dir, then the legacy in-tree file.** `Load`
   returns the admin-dir lease when present, the in-tree lease only when the
   admin dir holds none (a non-worktree path reaches the same fallback), and
   `nil` with neither. `Save` writes the admin dir and, for one release, the same
   bytes to the in-tree file (the overlap policy below). A lease is *legacy-only*
   when the admin dir holds none and the in-tree file exists. `dotf lane init`,
   the re-init step the runbook already prescribes after updating, copies each
   lane's legacy-only lease into the admin dir and counts that lane in `changed`,
   keyed on the admin dir; a non-lane worktree migrates on its next `Save`. The
   admin-dir file gets its own constant; the in-tree name stays on the exclude,
   clean and audit paths that name the tree file.
3. **`Save` refuses a path that is not a linked worktree** (a main checkout, a
   plain directory, a submodule whose `.git` points into `modules/`), naming the
   path. It never falls back to writing in the tree: a silent fallback is how a
   second lease location would be born.
4. **The admin dir is resolved from the worktree's `.git` pointer file**,
   absolute or relative to the worktree, with no subprocess, and only when it
   points at `…/worktrees/<id>`; the pattern `cli/internal/mem/identity.go`
   applies for the memory thread. `mem` cannot import `worktree` (the dependency
   runs the other way), so its parked-lane check keeps its own pointer reader
   and reads the lease at the same path.
5. **The guard's rule 5 and rule 6's detached-HEAD test read
   `$git_dir/dotf-worktree.json` first and `$top/.dotf-worktree.json` only when
   the first is absent**, the same precedence as Go, so the shell and Go readers
   cannot disagree about which lease is in force.
6. **The mirror, the legacy read path and the `info/exclude` line for the
   in-tree file are kept for one release** and removed under their own ticket,
   filed from the spec, once a release has shipped with the mirror and
   `dotf worktree list` reports no legacy-only lease on the owner's seats.

## Consequences

- `git clean -fdx` inside a lane no longer changes what the lane is; the class
  is removed rather than reported.
- Lifecycle, as measured for ADR-061 on the same directory: the lease survives
  `git worktree move`, `git gc` and `git worktree repair`; it is gone after
  `git worktree remove` and after `rm -rf` plus `git worktree prune`; a
  worktree re-created at the same path starts without a lease, as `sweep` and
  `adopt` already expect of a worktree nobody leased.
- **Mixed versions on one seat, and the overlap policy.** The guard runs from
  the machine-wide `core.hooksPath`, which `converge`'s `git-hooks` step
  redeploys; it does not travel with the binary. `dotf update` runs `converge`,
  and that converge failed on the authoring host on 2026-11-12 (#3395), so a
  new binary against an old guard is a state a seat can stay in; and between
  this merge and the next release the pinned binary on `PATH`, the one the hooks
  call, is the old one. Without a mirror, a lane leased only in the admin dir is
  blocked by the old rule 5b and the old FIX (`lane init && lane switch`) no
  longer writes the tree file, so the block does not clear until
  `dotf hooks install` runs. The alternative, accepting that window with
  `dotf doctor`'s guard check as the signal, was rejected: a mirror can be
  dropped later, a blocked seat cannot be unblocked after the fact. So `Save`
  mirrors for one release and `Load` prefers the admin dir; `git clean` can at
  worst reproduce today's block on an old guard, and the mirror is rewritten by
  the next `Save`.
- **An old binary writing while a new one reads.** The old binary's writes reach
  the mirror alone, so after an old `lane park`, `lane switch`, `sweep` park or
  `lane next` renewal the admin copy is stale and the new reader prefers it.
  Traced through every production writer (spec, Risks): the reachable outcomes
  are a WIP-cap refusal, a guard block with its FIX, or a `sweep` reap verdict
  reached earlier on a worktree that is clean, past grace and merged, the
  verdict today's `sweep` reaches a day later. Named lanes are never deleted.
  The precedence stands; the window closes with the release that puts the new
  binary on `PATH`.
- Test fixtures that wrote the in-tree file by hand keep passing through the
  fallback, which is why the spec moves the ones that run in a real linked
  worktree to the admin dir: a fixture that passes for the wrong reason is
  coverage of the path being retired.
- `dotf worktree list` names a legacy-only lease, so a seat can see which
  worktrees no `Save` has reached since the update; a `doctor` entry for it
  belongs to the seat-wide sweep entry (#3327).
- Windows is unverified on the authoring box; the pointer file is read with
  `filepath.Clean`, which normalises either separator, and the Go tests run on
  all three CI legs.

## References

- [ADR-061](adr-061-lane-identity-in-the-worktree-git-dir.md): the identity record this lease joins, and the measurement of the admin dir's lifecycle.
- [ADR-060](adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md): the lanes, the marker and the guard.
- `docs/lessons/lesson-272-fail-closed-worktree-garbage-collection-and-in-tree-leases.md`: why the lease was in the tree, answered above.
- `docs/lessons/lesson-441-every-path-that-creates-a-worktree-owes-it-the-lease.md`: why the seam is `LoadMetadata`/`SaveMetadata` and nothing else.
- `specs/HARNESS-193-lease-in-git-dir/`: the spec.
- #3307, #3261, #3327.
