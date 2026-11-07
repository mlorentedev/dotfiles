---
tags: [spec, tasks, templates]
created: "2026-11-06"
id: "TOOL-035-manage-ts-bridge-lifecycle"
type: spec-tasks
status: active
owner: manu
---

# Tasks - TOOL-035-manage-ts-bridge-lifecycle

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch series created from `origin/main`: state foundation -> process identity -> read-only status -> start/stop
- [x] Work-gate verified: #3244 is open, assigned and In Progress
- [x] ADR-059 and the three-reference audit are complete
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### State and health foundation

- [x] [P] [AC4] Add table tests in `cli/internal/bridge/state_test.go` for safe profile names, atomic descriptor/process state, one-profile defaulting and ambiguous profiles. Expected: FAIL because the package does not exist.
- [x] [AC4] Implement `cli/internal/bridge/state.go` under `env.StateDir()/bridge/<profile>` with strict validation and atomic JSON writes. Expected: focused state tests PASS.
- [x] [P] [AC1] Add shared listener/destination probe tests in `cli/internal/bridge/health_test.go`. Expected: FAIL before the bridge health package exists.
- [x] [AC1] Extract the TOOL-034 listener/destination interpretation into `cli/internal/bridge/health.go`; refactor doctor to consume it without changing its report. Expected: doctor and bridge focused tests PASS.
- [x] Run `go test ./internal/bridge ./internal/doctor -run '^(TestStore.*|TestLaunchDescriptor.*|TestProcessRecord.*|TestProbeHealth.*|TestCheckTsBridge.*)$'` and focused vet; commit the state foundation.

### Exact process identity

- [x] [P] [AC3] Add process-identity tests for exact ownership, stale PID, PID reuse, executable mismatch, absent state and inspection failure on platform seams.
- [x] [AC3] Implement Windows, Linux and macOS process observation using PID, executable and a platform-native process-start identity.
- [x] [AC3] [AC4] Cross-compile the bridge package for Windows, Linux and macOS.
- [ ] [AC3] [AC4] Run `TestCaptureAndInspectCurrentProcess` on Windows, Linux and macOS CI hosts. Windows and Ubuntu pass; a live macOS run is pending.

### Read-only status

- [x] [P] [AC1] Add status tests for stopped, starting, destination-failed, healthy, unowned and unavailable states plus JSON `error`/`why`/`fix`.
- [x] [AC1] Implement the read-only status classifier and service using exact process ownership plus the shared health probe.
- [x] [AC1] Add `dotf bridge status [--profile] [--json]` and command tests; non-healthy states exit non-zero.
- [x] [AC5] Update `docs/runbooks/ts-bridge-client.md` with the status command while preserving stale Copilot App remove/re-add recovery.
- [x] Run focused bridge, doctor and command tests plus vet; commit the read-only status command.

### Owned start/stop lifecycle

- [ ] [P] [AC2] Add fake-daemon tests for `start`: first launch records state/log, waits for READY and destination, ERROR/exit/timeout fail, healthy second start spawns nothing.
- [ ] [AC2] Implement the launch descriptor resolver and detached start manager, reusing the existing Windows/POSIX detach semantics and ts-bridge profile lock.
- [ ] [P] [AC3] Add graceful-stop, hard-stop timeout and identity recheck tests on Windows/POSIX seams.
- [ ] [AC3] Implement owned stop; never kill by process name and never act on an identity mismatch.
- [ ] [AC4] Add bounded log rotation and tests proving credentials/profile bodies never enter state, logs or command output.
- [ ] [AC2] [AC3] Wire `dotf bridge start|stop`, including `ERROR` / `WHY` / `FIX` human and JSON output.
- [ ] [AC2] [AC3] Run process-level helper tests on Windows and CI, then `go test ./internal/bridge ./internal/cmd ./internal/doctor` and focused vet.
- [ ] [AC2] [AC3] Record a live start/status/stop/start smoke against the synthetic profile only; never use live credentials in tests. Commit the lifecycle command.

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

Minimal `features.json` skeleton (drop into `<repo>/specs/TOOL-035-manage-ts-bridge-lifecycle/features.json`):

```json
[
  {
    "id": "TOOL-035-manage-ts-bridge-lifecycle-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
