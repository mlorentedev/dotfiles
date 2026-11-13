---
id: "HARNESS-193-lease-in-git-dir"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-11-12"
issue: "mlorentedev/dotfiles#3307"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
---

# HARNESS-193-lease-in-git-dir

> **Naming**: file lives at `<repo>/specs/HARNESS-193-lease-in-git-dir/proposal.md`. `HARNESS-193-lease-in-git-dir` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

<!-- from issue #3307: lane: git clean -fdx inside a lane deletes .dotf-worktree.json, the lease the guard relies on -->

[AGENT-DRAFT — review before archive]

The lease (`.dotf-worktree.json`) is the liveness and state record of every dotf worktree: `sweep` reaps only past its expiry, `lane status` and the pre-commit guard tell a PARKED lane from an ACTIVE one by it, and rule 5b of the guard blocks a registered lane that has none. It lives at the worktree root and is kept out of `git status` through `info/exclude`, but an excluded file is still an untracked file to `git clean`: measured 2026-11-11 (#3307), `git clean -fdx` inside an ACTIVE lane deletes the lease, and the lane then reads as a registered lane without a lease, blocked by the guard with no diagnostic that names the cause. ADR-061 moved the lane's *identity* into the worktree's git admin dir precisely because `git clean` never reaches it, and left the lease behind as a separate decision; the owner took that decision on 2026-11-12: direction 1, the lease moves next to the identity record.

## What

[AGENT-DRAFT — review before archive]

The lease of a linked worktree lives in the worktree's git admin dir, `<repo>/.git/worktrees/<id>/dotf-worktree.json`, next to `dotf-lane` (ADR-061). This is the seam every dotf worktree goes through (`worktree add`, `adopt`, `detached`, `spec review`, `lane next`, `lane switch`, `lane park`), not only lanes, so the move applies to all of them.

- `worktree.LoadMetadata(path)` reads the admin dir first and the legacy in-tree file only when the admin dir holds no lease; on a path that is not a linked worktree it still reads the in-tree file (the legacy fallback is the same code path) and answers `nil` only when neither exists. `worktree.SaveMetadata(path, meta)` writes the admin dir; what it does with the in-tree copy is the overlap decision in Risks (remove it, or mirror it for one release). A path that is not a linked worktree (a main checkout, a plain directory, a submodule checkout) is refused by `Save` with an error naming the path.
- The admin-dir file gets its own constant (`LeaseFileName = "dotf-worktree.json"`); `MetadataFileName` keeps naming the in-tree file, because `excludeFile`/`ensureExclude` (`lane/lease.go`, `lane/named.go`, `worktree/add.go`, `detached.go`, `adopt.go`), `worktree/clean.go`'s disposable list and `lane/audit.go`'s message all pass that constant and must go on naming the tree file.
- The admin dir is resolved from the worktree's own `.git` pointer file (`gitdir: <path>`, absolute or relative to the worktree), with no subprocess, and only when it points at `…/worktrees/<id>`: the pattern `cli/internal/mem/identity.go` already applies for the memory thread. Git owns that directory, measured for ADR-061 on a scratch repository: the file survives `git worktree move`, `git gc` and `git worktree repair`; it is gone after `git worktree remove` and after `rm -rf` plus `git worktree prune`; a worktree re-created at the same path starts without it; `git clean -fdx` never reaches it.
- The guard (`git-hooks/lib/lane-guard.sh`, rules 5 and 6) reads `$git_dir/dotf-worktree.json` first and `$top/.dotf-worktree.json` only when the first is absent, the same precedence as Go.
- `dotf mem session-end`'s parked-lane check (`isWorktreeParked`) resolves the nearest worktree root and reads the lease where `Load` does; `mem` cannot import `worktree` (the dependency runs the other way), so it reuses its own `readGitdirPointer`.
- `dotf lane init`, the documented re-init step after updating, migrates a lane's in-tree lease into the admin dir and counts it in `changed`; a second run reports `changed=0`.
- `dotf worktree list` names a lease still at the legacy location in its reason text, so a seat that has not re-initialised can see what is left to migrate.

## Out of scope

[AGENT-DRAFT — review before archive]

- Removing the legacy in-tree read path and the `info/exclude` line for `.dotf-worktree.json` (`excludeFile` calls in `lane/lease.go` and `lane/named.go`, `worktree/add.go`): one release after this ships, as its own ticket filed from this spec.
- Re-creating a missing lease on `lane switch` (direction 2 of #3307): the class is removed, not reported.
- A `doctor` entry listing leases left at the legacy location: #3327 (seat-wide sweep entry) is where that belongs.
- Reading the lease from a subdirectory of a worktree in `forge write` and `harness gate` (`LoadMetadata(cwd)` takes the worktree root today and keeps doing so).
- Any change to the lease's fields or to the committed marker `.dotf/lanes.json`.

## Risks / open questions

[AGENT-DRAFT — review before archive]

- **Lesson 272 chose the in-tree location on purpose** ("the liveness signal must live inside the worktree filesystem itself", for containerised agents the host cannot see in `/proc`). The property that lesson protects is that the lease, not a process table, decides liveness, and a reader on the host reads either location. A container that mounts only the worktree and not the repository's `.git` cannot run git at all, so no dotf command renews or reads a lease there today either; the admin dir is reachable wherever git works, which every dotf command already requires. The ADR answers the lesson by name.
- **OPEN DECISION (owner): the overlap policy while old guards exist.** The guard runs from the machine-wide `core.hooksPath`, redeployed by `converge`'s `git-hooks` step, not by the binary; `dotf update` runs `converge`, and on this host that converge failed on 2026-11-12 (#3395), so a seat can run a new binary against an old guard for longer than one command. In that state a lane leased only in the admin dir is blocked by the old rule 5b, and the old FIX (`dotf lane init && dotf lane switch`) no longer writes the in-tree file, so the block does not clear. Two options: **(A) mirror for one release**: `Save` writes the admin dir and the in-tree copy, `Load` prefers the admin dir; an old guard keeps working, `git clean` still deletes the mirror, which with an old guard reproduces today's block and never anything worse, and the mirror goes away with the legacy read under the follow-up ticket. **(B) accept the window**: `Save` removes the in-tree copy; a seat whose converge failed is blocked until `dotf hooks install` runs, and `dotf doctor`'s guard check is the signal. Recommendation: A, one extra write for one release, removed on a ticket that already exists. AC2 is written for A; under B its third clause changes to "removes the in-tree copy".
- **Old binary after a new one** (a `go run` dev build on a seat whose release is newer, or the reverse): the old `ensureLease` finds no in-tree lease (under B) and writes a fresh one with `reap_ok: true` and no lane; the new reader prefers the admin dir; an old `sweep` reading only the stray copy would see a 24-hour lease on a worktree whose branch gates still apply. Under A the old binary finds the mirror and leaves it alone. Not a supported configuration; stated so it can be recognised.
- **Test fixtures write the in-tree file by hand** (29 lines in 16 test files across `worktree`, `lane`, `cmd`, `hooks`). Through the legacy fallback they keep passing unchanged, which means they would keep testing the legacy path. The tasks move the fixtures that run in a real linked worktree to `SaveMetadata`, keep the fake-directory ones on the legacy path as the fallback's own coverage, and add admin-dir cases for `list`, `sweep`, `done`, the guard and `session-end`.
- **Readers outside this repository.** `grep -rl dotf-worktree.json` over the vault and every sibling repository under the projects root (2026-11-12): three vault research notes mention the file, no skill, harness file or sibling repository reads it.
- **`Save` refuses a non-worktree path.** `spec review`, `detached` and `add` call it right after `git worktree add`, so they are linked worktrees; a test that calls `Save` on a plain temp directory fails loudly and is rewritten. Fail closed rather than silently writing in-tree, which is how a second lease location would be born.
- **Windows is not verified on the authoring box** (macOS). The `.git` pointer may carry forward or back slashes; `filepath.Clean` normalises both, and the Go tests run on all three CI legs. The guard is POSIX sh on the Linux and macOS legs.
- **Scratch-repository re-measurement.** This session's lifecycle probe was denied by the sandbox before it ran; the spec relies on ADR-061's measurement of the same directory, and the implementation's Go test runs the `git clean -fdx` case itself.

## Acceptance criteria

[AGENT-DRAFT — review before archive]

- [ ] AC1: `git clean -fdx` inside an ACTIVE lane leaves it ACTIVE for `dotf lane status` and `dotf worktree list`, and a commit through the installed guard is not blocked; a Go test in the lane package runs the clean and asserts all three.
- [ ] AC2: `worktree.LoadMetadata` returns the admin-dir lease when present, the in-tree lease only when the admin dir has none (on a non-worktree path too), and `nil` with neither; `worktree.SaveMetadata` writes the admin dir, writes the in-tree mirror (option A; "removes the in-tree copy" under B), and refuses a main checkout, a plain directory and a submodule checkout with an error naming the path.
- [ ] AC3: On a repository whose lanes carry in-tree leases, `dotf lane init` moves each lease into its admin dir and reports `changed=<n>`; a second run reports `changed=0` and rewrites nothing.
- [ ] AC4: The guard blocks a commit in a PARKED lane and allows one in an ACTIVE lane when the lease is in the admin dir, with the in-tree file absent; rules 5 and 6 fall back to the in-tree file when the admin dir has none; every other case in `cli/internal/hooks/lane_guard_test.go` passes unchanged.
- [ ] AC5: `dotf mem session-end` treats a lane whose admin-dir lease says PARKED as parked, from the lane root and from a subdirectory, and still recognises a legacy in-tree PARKED lease.
- [ ] AC6: `dotf worktree list` reports a lease still at the legacy location in the worktree's reason text, and every existing test in `cli/internal/worktree`, `cli/internal/lane`, `cli/internal/cmd` and `cli/internal/mem` passes.

## References

- Bitácora board: the GitHub issue / Project item tracking this spec (see the `issue:` frontmatter field)
- Related ADR: `docs/adr/adr-062-the-lease-lives-in-the-worktree-git-dir.md` (this spec), amending `docs/adr/adr-061-lane-identity-in-the-worktree-git-dir.md`
- Related: #3261 (identity record), #3327 (seat-wide doctor entry), lesson 272 (in-tree leases), lesson 441 (every constructor owes the lease), `specs/archive/HARNESS-187-lane-identity-git-dir/`, `specs/archive/CLI-075-dotf-worktree-lifecycle/`
- Runbook: `docs/runbooks/guide-persistent-named-lanes.md`
