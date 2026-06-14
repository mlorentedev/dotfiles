---
tags: [spec, tasks, templates]
created: "2026-06-13"
---

# Tasks - CLI-009-setup-install-dot

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/setup-install-dot`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> TDD order, bats-first. `install_dot` lives in a sourceable `scripts/install-dot.sh` (run-guard), so bats exercises it against a `file://` fixture with no network.

- [x] `versions.conf`: add `DOT_VERSION=0.1.0`
- [x] bats for `_dot_arch` (x86_64->amd64, aarch64/arm64->arm64, i686->error)
- [x] Implement `_dot_arch` + OS detection in `scripts/install-dot.sh`
- [x] bats for `install_dot` happy path (file:// fixture: download + sha256 verify + extract -> dest)
- [x] bats for checksum mismatch + missing entry (must abort, no binary left in dest)
- [x] bats for idempotence (pinned version present -> no re-download)
- [x] Implement `install_dot` (artifact name, curl download, checksums.txt sha256 verify, tar extract, chmod, version skip/converge) + robust `(return)` run-guard
- [x] Wire `setup-linux.sh`: source `install-dot.sh`, call `install_dot` in the developer-tools section (graceful warn on failure)
- [x] `healthcheck.sh`: add `dot` presence + version-match check (section 6)
- [x] `cli.yml`: gate the `release` job on `needs: [test, lint]`
- [x] shellcheck changed scripts; run full bats suite

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-009-setup-install-dot/features.json`):

```json
[
  {
    "id": "CLI-009-setup-install-dot-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
