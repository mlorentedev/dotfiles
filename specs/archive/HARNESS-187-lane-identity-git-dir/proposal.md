---
id: "HARNESS-187-lane-identity-git-dir"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3261"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-187-lane-identity-git-dir

> **Naming**: file lives at `<repo>/specs/HARNESS-187-lane-identity-git-dir/proposal.md`. `HARNESS-187-lane-identity-git-dir` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3261: lane-guard: lane identity is a leaf name, so a real lane can go unmatched and a same-named stranger is blocked -->

The pre-commit lane guard (`git-hooks/lib/lane-guard.sh`) decides that a directory is a persistent named lane by comparing its last path segment against `canonical_lanes` in `.dotf/lanes.json`, and only when the directory carries no `.dotf-worktree.json`. A name is not an identity: a worktree that merely shares a lane's name is refused commits, and a lane whose leaf changed (`git worktree move`) stops being recognised. Measured before this spec (2026-11-11): `git clean -fdx` inside a lane deletes `.dotf-worktree.json`, which lives only in `info/exclude` (#3307), so the name rule is reached by a routine command and not only by hand; and `dotf lane init` for two repositories sharing a basename under one `WORKTREE_ROOT` silently adopts the first repository's lanes, because `provisionLane` never checks that the directory it finds is a worktree of the repository it is initialising. #3261 blocks the `dotf lane init` rollout to the other lane-managed repositories.

## What

A lane's identity lives where git keeps the worktree's own state: `lane init` writes the lane name to `dotf-lane` inside the worktree's git admin dir (`<repo>/.git/worktrees/<id>/`, the `$git_dir` the guard already resolves). Git owns that directory's lifecycle, measured on a scratch repository: the record survives `git worktree move`, `git gc` and `git worktree repair`; it is gone after `git worktree remove` and after `rm -rf` plus `git worktree prune`; a worktree re-created at the same path starts without it; `git clean -fdx` never touches it. No path is stored anywhere and the committed marker stays exactly as it is.

- `dotf lane init` records the identity for each lane it creates or adopts, idempotently (`changed=0` on re-run), and **refuses** a directory at a lane's path that is a worktree of another repository, naming both.
- The guard's "unleased lane" rule reads `$git_dir/dotf-lane` instead of the leaf name: a registered lane without `.dotf-worktree.json` is blocked whatever its directory is called; an unregistered directory is never blocked by its name. `canonical_lanes` is no longer read by the guard (it stays in the marker as the policy list `lane init` writes and `doctor` will check, #3262).
- `lane.Identity(path)` in Go answers the same question for `doctor` (#3262), `worktree list` and `sweep` (#3293).

## Out of scope

- Treating an app-created worktree under a `third_party_roots` entry as a lane: by decision (owner, 2026-11-11) it is a third-party root and rule 4 keeps blocking it.
- Moving `.dotf-worktree.json` (the lease) into the git admin dir, so that `git clean -fdx` cannot delete it: #3307.
- Locating lanes by enumerating `git worktree list` rather than by the `<WORKTREE_ROOT>/<repo basename>/<lane>` convention, and a distinct layout for two repositories that share a basename: `lane init` refuses that case loudly here; a layout that lets both coexist is #3318, filed from this spec.
- `doctor` checks on the marker or on identity drift: #3262.

## Risks / open questions

- **An existing lane provisioned before this change has no identity record.** `dotf lane init` is idempotent and adopts an existing lane directory (`provisionLane`'s second branch), so re-running it writes the record; until then the guard treats the lane as unregistered, which only means the lease-less case is not blocked for it. The runbook says to re-run `lane init` after updating. No automatic migration: the guard must not write.
- **Guard test case 10 changes meaning.** Today it asserts that an unleased directory merely *named* after a declared lane is blocked. Under this spec that directory is not a lane, so the case is rewritten to assert the opposite for the unregistered directory and the block for the registered one. The ticket's "every existing guard case keeps passing" is amended by this line: one case changes because the contract it pinned is the defect.
- **`git rev-parse --git-dir` may answer a relative path** (`.git` at a main checkout; relative under some `GIT_DIR` exports). The guard already absolutises `common_dir`; `git_dir` gets the same treatment before the record is looked up. A main checkout has no `worktrees/<id>` admin dir and never carries the record, so the main-checkout rules are untouched.
- **Windows is not verified on the authoring box** (macOS). The guard test is POSIX sh and runs on the Linux and macOS legs; Go tests run on all three. The record path is built by git itself, so no separator is composed by hand.

## Acceptance criteria

- [x] AC1: After `dotf lane init`, each canonical lane's git admin dir holds `dotf-lane` with the lane's name; a second run reports `changed=0` and rewrites nothing.
- [x] AC2: With the record present and `.dotf-worktree.json` absent, the guard blocks a commit naming the lane and the fix, whatever the directory's leaf is (verified after `git worktree move`); with no record, a worktree merely named after a canonical lane is not blocked.
- [x] AC3: `dotf lane init` for a second repository whose basename matches one already initialised under the same `WORKTREE_ROOT` fails with an error naming the path and the repository it belongs to, and leaves the first repository's lanes and metadata untouched.
- [x] AC4: `lane.Identity(path)` returns the recorded name for a lane, and `("", false)` for a worktree without a record, a main checkout, and a path outside any repository, with no error for those three.
- [x] AC5: Every other case in `cli/internal/hooks/lane_guard_test.go` and `cli/internal/lane` passes unchanged; the guard no longer reads `canonical_lanes`.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related: #3262 (doctor), #3293 (worktree identity), #3307 (lease deleted by `git clean`), #3318 (same-basename layout), `specs/archive/HARNESS-186-named-lanes-sticky-personas/`
- Runbook: `docs/runbooks/guide-persistent-named-lanes.md`
