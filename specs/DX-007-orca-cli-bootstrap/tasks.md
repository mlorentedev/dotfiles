---
tags: [spec, tasks, templates]
created: "2026-06-20"
---

# Tasks - DX-007-orca-cli-bootstrap

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

> ⚠️ **Fleet-coordination note (2026-06-20):** do NOT branch off the shared `dotfiles/main` working tree while another session has uncommitted `setup-windows.ps1` changes — isolate this work into its own Orca worktree first, or wait until that file is committed by its owner. `setup-windows.ps1` is the contended file.

- [ ] Isolate into a dedicated worktree/branch `feat/DX-007-orca-cli-bootstrap` (NOT in the shared main tree)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions" — ⛔ **R1/R2/R3 still open**

## Implementation

> TDD order. Resolve open questions (Phase 0) before writing install code.

### Phase 0 — resolve the open questions (no production code)

- [ ] **R3** — read `setup-{linux.sh,windows.ps1}` to determine whether the `full`/optional profile mechanism already exists (#143) or DX-007 must define the gating. Decide the gate.
- [ ] **R2** — decide CLI bin-discovery: fixed per-user `Programs\Orca\resources\bin` vs registry uninstall key (Windows); `.deb`/AppImage layout for Linux `orca-ide`.
- [ ] **R1** — verify `orca-windows-setup.exe /S` installs silently per-user on a clean box (smoke).

### Phase 1 — env-contract is the SSOT for PATH

- [ ] Add the Orca CLI bin to `env-contract.json` as the generation source (NOT an ad-hoc PATH append in setup-*).

### Phase 2 — Windows (TDD)

- [ ] Pester: failing test — `orca` resolves on PATH after provisioning; re-run is a no-op.
- [ ] Implement idempotent install (download latest `orca-windows-setup.exe`, silent `/S`, skip if already installed) behind the `full` profile.
- [ ] Implement PATH wiring via env-contract → `paths.ps1` (skip if `orca` resolves), mirroring the Obsidian-CLI pattern (`setup-windows.ps1:~828`).

### Phase 3 — Linux (TDD)

- [ ] bats: failing test — `orca`/`orca-ide` resolves after provisioning; re-run no-op.
- [ ] Implement idempotent install (`.deb`/AppImage; AUR `stably-orca-bin` on Arch) behind the `full` profile.
- [ ] Implement PATH wiring via env-contract → `paths.sh`.

### Phase 4 — doctor + CI

- [ ] Go table-driven test in `cli/internal/doctor`: reports Orca CLI presence/absence.
- [ ] Implement the doctor check.
- [ ] Assert base profile unaffected; `test-windows`/`test-linux` green with the profile off (#350 parity).

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

Minimal `features.json` skeleton (drop into `<repo>/specs/DX-007-orca-cli-bootstrap/features.json`):

```json
[
  {
    "id": "DX-007-orca-cli-bootstrap-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
