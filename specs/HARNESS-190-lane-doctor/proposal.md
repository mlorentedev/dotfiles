---
id: "HARNESS-190-lane-doctor"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3262"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, worktrees, lanes, doctor]
template_version: "1.0"
owner: manu
---

# HARNESS-190-lane-doctor

> **Naming**: file lives at `<repo>/specs/HARNESS-190-lane-doctor/proposal.md`. `HARNESS-190-lane-doctor` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3262: doctor: no check for .dotf/lanes.json validity or for marker-to-lane drift -->

`dotf lane init` writes `.dotf/lanes.json` and the pre-commit lane guard arms itself on that file's
existence, but nothing validates its content and nothing compares it with the lanes that exist. A
hand-edited marker leaves a guard that looks installed and protects nothing; a lane moved by hand, a
lease deleted by `git clean -fdx` (#3307) or a lane provisioned before the identity record (#3261)
is invisible until a commit is refused. After a deploy, no single command answers "is this
repository's lane configuration coherent?".

## What

`dotf lane doctor`, run inside any checkout of a repository, audits that repository's lane
configuration and prints one `ERROR`/`FIX` pair per finding, exiting non-zero when there is one and
printing nothing when there is none. The audit is `lane.AuditLanes(repoRoot, worktreeRoot)` in
`cli/internal/lane`, the one place the marker and the lanes are compared, so a later `dotf doctor`
sweep entry calls the same function. It reports:

- the marker does not parse (the decoder's message), carries an unknown key (named), a `version`
  other than 1, an empty `canonical_lanes` or `third_party_roots`, or an entry that is empty,
  duplicated, carries `/` or `\`, or starts with a drive prefix;
- `canonical_lanes` differs from the lane set this `dotf` provisions (what `lane init` rewrites);
- a declared lane that is absent, is a directory that is not a worktree of this repository, is a
  worktree without the identity record (guard rule 5b does not fire there), is registered by git at a
  path that no longer exists (moved by hand), is at a path other than its canonical one (moved with
  `git worktree move`), is recorded in more than one worktree, or has no lease, a lease naming another
  lane, or a lease in a state the lanes do not use;
- a worktree whose identity record names a lane the marker does not declare.

A repository without the marker is not lane-managed: one line, exit 0. A marker with no lane present
on this machine is "not initialised here", also one line and exit 0: the marker is committed and
travels to every clone, the lanes are per machine.

The drift direction from disk to marker enumerates the worktree admin dirs under the common git dir
(`<repo>/.git/worktrees/*/`), where the identity record lives (ADR-061), not `git worktree list`:
measured on a scratch repository (2026-11-11), a lane moved by `mv` keeps its admin dir, its record
and a `gitdir` file pointing at the old path, `git worktree list` marks it prunable, and
`git worktree add` at the old path refuses ("missing but already registered"). So the FIX for that
case is `git worktree repair <new path>` or `git worktree prune`, never `dotf lane init` alone.

No auto-repair. Every FIX line names the command that repairs the finding.

## Out of scope

- Registering the audit in the `dotf doctor` sweep (`cli/internal/doctor/`): a follow-up ticket;
  this PR ships the function it will call.
- Validating the copy of `.dotf/lanes.json` checked out inside each lane, which the guard reads
  before the main checkout's: a lane on an older branch legitimately carries an older marker. A
  known limit, recorded in the runbook.
- Repairing anything. Repair stays `dotf lane init`, `git worktree repair|move|prune`.

## Risks / open questions

- The audit shells out to git; `dotf doctor` runs on machines with no lane-managed repository.
  Resolved: the function is only reached from `dotf lane doctor` in this PR, and an absent marker is
  the first exit.
- A marker whose `canonical_lanes` is valid but differs from the code's set: a finding, since
  `lane init` always rewrites it (changed=1) and the CLI routes by the code's set, not the marker's.
- A worktree recorded as a lane the marker does not declare has two legitimate repairs (delete the
  record, or declare the lane); the FIX names both and decides neither.

## Acceptance criteria

Observable outcomes. Each must be testable.

- [ ] AC1: a corrupted marker is reported with the offending field named: invalid JSON, an unknown
      key, a wrong `version`, an empty list, an entry with a separator or a drive prefix, a
      duplicate, and a lane set that differs from the one `dotf` provisions.
- [ ] AC2: a lane renamed by `mv` is reported as registered at a path that no longer exists with
      `git worktree repair` as the FIX; a lane moved with `git worktree move` is reported at its new
      path with `git worktree move` back as the FIX; a worktree recorded as an undeclared lane is
      reported.
- [ ] AC3: a lane without its lease, without its identity record, with a lease naming another lane,
      and a lane directory that is not this repository's worktree are each one finding naming the
      lane and the repair.
- [ ] AC4: a coherent lane-managed repository exits 0 with no output from `dotf lane doctor`; a
      repository without the marker and a marker with no lane initialised on this machine each exit
      0 with one line; any finding exits 1 with the ERROR/FIX pairs on stdout and no cobra `Error:`
      line.

## References

- Bitácora board: #3262 (this spec), #3261 (identity record), #3307 (lease and `git clean`), #3318 (same-basename layout), #3320 (`dotf-lane` namespace).
- Related ADR: `docs/adr/adr-061-lane-identity-in-the-worktree-git-dir.md`, `docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md`.
- Runbook: `docs/runbooks/guide-persistent-named-lanes.md`.
- Related patterns: `00_meta/patterns/pattern-agent-oriented-errors.md` (ERROR/WHY/FIX output).
