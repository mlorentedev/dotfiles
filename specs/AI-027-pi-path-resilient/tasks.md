---
tags: [spec, tasks, templates]
created: "2026-06-18"
---

# Tasks - AI-027-pi-path-resilient

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `fix/pi-launcher-path-resilient`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] Add the doctor guard test (TDD): `TestCheckOpenCode_piPathResilience` — configured-but-not-on-PATH → FAIL, ~/.local launcher off-PATH → FAIL, on-PATH → PASS, truly-absent → SKIP
- [x] Implement the doctor branch in `checks_deploy.go` `checkOpenCode`: `switch` on `sys.has("pi")` / `isExecFile(~/.local/bin/pi)` / `~/.pi/agent/models.json` exists, replacing the bare else-SKIP
- [x] `setup-linux.sh`: install pi with `npm install -g --ignore-scripts --prefix "$HOME/.local"`; guard on `~/.local/bin/pi` (stable location) instead of bare `command -v pi`

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (doctor table test) or a setup-script assertion
- [ ] `features.json` emitted (deferred — repo specs to date do not ship it; not gated by spec-gate)
- [x] Type checks pass (`go build ./...`)
- [x] Lint passes (`shellcheck setup-linux.sh` — no new findings; `go vet` via `go test`)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.

Minimal `features.json` skeleton (drop into `<repo>/specs/AI-027-pi-path-resilient/features.json`):

```json
[
  {
    "id": "AI-027-pi-path-resilient-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
