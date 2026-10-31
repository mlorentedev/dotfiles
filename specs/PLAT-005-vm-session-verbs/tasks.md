---
tags: [spec, tasks, templates]
created: "2026-10-22"
---

# Tasks - PLAT-005-vm-session-verbs

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [ ] Branch created from main: `feat/PLAT-005-vm-session-verbs`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Three stages and a PR 1 split into 1a to 1d to stay under the 300-LOC cap (pattern-git-workflow §10); each leaves `main` green on its own. Every refusal test uses `noGuest(t)` (lesson 489).

### PR 1a: the output split and the shared redaction primitive (no new verb)

- [x] [P] [AC5] Failing test: progress and the guest's output are two writers; `==> ` goes to stderr
- [x] [AC5] Split `Options.Out`/`RunOptions.Out`; the tests that read the progress writer follow the stream
- [x] [AC7] Move `redactWriter` out of `cmd/secrets.go` into one package; its tests move with it, not rewritten, and `dotf secrets run` keeps passing
- [x] Own PR, never bundled: the redaction move is a security primitive

### PR 1b: `Session`, `Record` and the Linux verbs

- [ ] [P] [AC9] `noGuest(t)` coverage for every new verb's refusals: failing test per verb, `go -C cli test ./internal/cmd/ -run TestVMVerbRefusals`
- [x] [P] [AC2] Failing test for the record shape and the status, exit and process-code table in `cli/internal/vm/session_test.go`
- [ ] [AC2] `Session` interface and `Record` type in `cli/internal/vm/session.go`; `--json` writes one record, empty lists are `[]`
- [x] [AC1] Linux `up --session` over Lima (default name `<os>-<unix time>`); `status` lists what `limactl list` answers
- [x] [AC1] `up` on a name whose guest is gone: `stale`, exit 2, names `down`
- [ ] [AC2] Linux `exec` over `limactl shell` (argv, no shell) and `--script`
- [ ] [AC3] stdin forwarded on Linux
- [ ] [AC7] `exec --secret <registry-id>`: resolve through the catalog, write on the guest's stdin like `withToken`, key the redaction; a guest that echoes it yields a masked record
- [ ] [AC4] Linux `cp` both ways with `limactl copy`; the isolation tests stay green
- [x] [AC11] Linux `down`: no instance and no state; `no-session` exit 3 for an unknown name; `timeout` exits 1
- [ ] Description names `stale` as an additive status value (no `version` bump, ADR-055 decision 7) and lists the allow rules (#2859)

### PR 1c: the Windows verbs

- [ ] [AC1] One fixed guest: a second `up` is `busy`; `up` and `status` share the `running` check of `vm/run.go`
- [ ] [AC2] Windows `exec` over QGA and the scheduled task
- [ ] [AC3] Windows refuses stdin with `unsupported`, exit 2
- [ ] [AC4] Windows `cp` with `QGA.WriteFile`/`ReadFile` in 48 KiB chunks
- [ ] [AC11] Windows `down`: no overlay and no state, `clean-base` untouched; `stale` cleaned by `down`
- [ ] Verb help states ADR-055's `--elevated` caveat

### PR 1d: `run` recomposed

- [ ] [AC6] `run` composed of the verbs, with `--secret GITHUB_TOKEN`; the assertions of its tests are unchanged
- [ ] [AC5] `run` writes progress where the verbs do; the description names the stream move as the one behaviour change

### PR 2: elevation and the guest prerequisites

- [ ] [AC8] Failing test: `exec --elevated` on Windows creates the task with `/rl highest`; on Linux prefixes `sudo`
- [ ] [AC8] Implement; measure on the real guest that the token is high integrity (`whoami /groups`)
- [ ] [AC10] Decide and implement how the `ssh-target` role and a stand-in mesh address (100.64.0.0/10) reach a session (#2763 obstacle 3)
- [ ] [AC10] Install OpenSSH Server inside a session with `exec --elevated`, never in `clean-base`

### PR 3: the Windows fixture and acceptance

- [ ] [AC10] Measure `sshd -T` 9.5p2 through the verbs; add `sshd-T-windows-9.5p2-{effective,ignored}.txt` and the `platforms` entry
- [ ] [AC10] Acceptance on the guest: converge applies the drop-in, the proof passes, a second converge is `changed` 0; record on #2763 and close it

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

Minimal `features.json` skeleton (drop into `<repo>/specs/PLAT-005-vm-session-verbs/features.json`):

```json
[
  {
    "id": "PLAT-005-vm-session-verbs-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
