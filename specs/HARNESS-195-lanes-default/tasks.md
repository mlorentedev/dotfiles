---
tags: [spec, tasks, templates]
created: "2026-11-13"
id: "HARNESS-195-lanes-default"
type: spec-tasks
status: active
owner: manu
---

# Tasks - HARNESS-195-lanes-default

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `feat/lanes-default` (`dotf worktree add feat-lanes-default`, #3432)
- [ ] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> One PR per slice, each at most 300 executable lines. A, D1 and F are independent of each other (`[P]`). B and C depend on A, because both read the activation record A writes. D2 depends on C, because it calls `lane.Where`. E waits on observed launches, and G depends on all of them.

### Slice A: activation through converge (AC1)
- [x] [P] [AC1] Failing test: `lane.Activate` reads `.dotf/lanes.json` from the default branch's committed tree (`git show <remote>/<base>:…`), not from the working tree. A working-tree-only marker yields "not active".
- [x] [AC1] Implement `Activate`: create the missing lanes (reusing `lane init`'s lane creation), and write the activation record under the git common dir. Idempotent.
- [x] [AC1] Failing test, then implementation: the `lanes` converge step (`cli/internal/converge/lanes.go`) reports changed, then ok, on a second run. A test proves the step never activates `DeployDir`, including when `RepoRoot` and `DeployDir` resolve to the same physical path.
- [x] [AC1] Remove the marker write from `dotf lane init`'s default path; `init` never writes the activation record. A test proves that a hand-run `init` leaves no record, both with an uncommitted marker and with a committed one.

> Slice A as built (for the slices that follow):
> - `lane init` keeps `--write-marker`, which writes `.dotf/lanes.json` for the commit that declares a repository. Without the flag, init writes neither the marker nor the record.
> - Plan and apply agree, including for a lane that lacks a lease or its identity record; a lane path another repository occupies is a converge skip carrying the repair, not a failure.

### Slice B: the guard arms from the activation record (AC2, #3437, #3438)
- [ ] [AC2] Failing `TestPreCommitLaneGuard` cases:
  - no record → no block;
  - a record on main in the main checkout → block;
  - a branch worktree → pass;
  - the deploy clone → pass.
- [ ] [AC2] Implement in `git-hooks/lib/lane-guard.sh`: read the record from the git common dir, and drop both marker lookups (`$top/.dotf/lanes.json` and `$main_root/.dotf/lanes.json`).
- [ ] [AC2] An expired lease on a detached HEAD warns on stderr and does not block (#3438). Test first.

### Slice C: tell the session where it stands (AC3)
- [x] [AC3] Failing test: `lane.Where(cwd)` returns in-lane, out-of-lane-in-active-repo, or inactive.
- [x] [AC3] Print the line from `dotf mem resume` and `dotf mem session-start`. Golden tests for all three states.
- [x] [AC3] Failing test: every manifest entry with a session-start presence has a deployed carrier. It fails today for pi, opencode and agy.
  - As built (C1): `TestEveryPresenceTargetHasADeployedCarrier` lists those three in the `carrierGaps` table and asserts it is exact. The slice that deploys a carrier deletes its row.
- [ ] [AC3] pi extension (`session_start`) and opencode plugin (`chat.system.transform`): run `dotf mem session-start` and inject its output into the session context. Keep them small and deploy them through dotf's existing deploy path, not bind (the manifest says `emit: false`).
- [ ] [AC3] agy: bind a `PreInvocation` hook in `bind_named` that prints the line on a session's first invocation only.

### Slice D1: `lane next` without herdr (AC5)
- [x] [P] [AC5] Failing golden test: `manualCommands` in a repository with lanes ends inside the lane before `exec`. Today it prints `lane switch` and then `exec` with no `cd`. Then implement it for the zsh, bash and PowerShell forms. It uses the existing `HasNamedLanes`, so it does not wait for A.

### Slice D2: the `worktree add` hint (AC4)
- [ ] [AC4] Failing test, then implementation: the hint in `cli/internal/worktree/add.go`, through `lane.Where` (Slice C). No hint from `AddDetached`.

### Slice E: launchers (AC6, #3440, #3430)
- [ ] [AC6] Fix #3440: `dotf secrets run` refuses an npm `.cmd` or `.bat` child on Windows, using the same function as `lane exec` (`refuseBatchShim`). Test first.
- [ ] [AC6] Observed launch of copilot (`copilot -i <p>`) and opencode (`opencode --prompt <p>` under `dotf secrets run`). Record `LANE-READY` evidence in `verification.md`, then add the entries.
- [ ] [AC6] codex and gemini: add each when it is installed and observed. Otherwise record "not installed" with the date.
- [ ] [AC6] #3430: check whether a lane-launched pane carries `HERDR_AGENT`. Set it in `Exec`'s environment if it does not.

### Slice F: doctor (AC7, #3327)
- [x] [P] [AC7] Apply the patch preserved on #3327 (detached worktree with no lease) with its tests.
- [x] [AC7] Expired lease and deployed-guard drift: failing tests, then implementation.

### Slice G: doctrine and activation (AC8, AC9)
- [ ] Prerequisite: #3456 (harness re-rendered from the vault) merged, so the doctrine PR renders only its own change.
- [ ] [AC8] Rename the tiers in AGENTS.md and in `cli/internal/initrepo/templates/agents-spec-section.md` (the Lane Decisor rule becomes a tier rule). First check whether AGENTS.md is rendered from a vault source; if it is, edit the source. Update `using-git-worktrees`, `handoff` and `autopilot` in the vault source, then re-render. Add a test that fails if AGENTS.md calls a tier a "lane".
- [ ] [AC8] Runbook `docs/runbooks/guide-persistent-named-lanes.md`: activation through converge, what the guard blocks, and recovery.
- [ ] [AC9] PR adding `.dotf/lanes.json` to dotfiles (the switch). Then `dotf converge` on the Mac and each guest, and sweep the merged worktrees.
- [ ] [AC9] Run a real session of each launchable agent per machine, and record the AC3 line in `verification.md`.

## Closing

- [ ] Every acceptance criterion from `proposal.md` is covered by at least one test
- [ ] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [ ] Type checks pass
- [ ] Lint passes
- [ ] No unrelated changes in the diff (no scope creep)
- [ ] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/HARNESS-195-lanes-default/features.json`):

```json
[
  {
    "id": "HARNESS-195-lanes-default-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
