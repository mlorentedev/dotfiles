---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - POLISH-004-cross-os-polish-bundle

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/polish-004-editorconfig-inputrc`
- [x] `proposal.md` complete; acceptance criteria testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] `.editorconfig` at repo root: `root=true`; `[*]` UTF-8 + LF + trim + final newline + 4-space; markdown no-trim; `*.{ps1,psm1,psd1,bat,cmd}` CRLF; yaml/json 2-space; Makefile/go tab. Anti-scope: existing files not reformatted.
- [x] `.inputrc` at repo root: `$include /etc/inputrc` + completion-ignore-case + show-all-if-ambiguous/unmodified + colored-stats + arrow-key history-search.
- [x] `setup-linux.sh` deploys `.inputrc` → `$HOME/.inputrc` via `deploy_file` (dotfiles deploy block).
- [x] `tests/inputrc.bats` (4 tests): repo files exist + `.inputrc` content + `.editorconfig` root/CRLF + setup deploy wiring.
- [x] README "Features" mentions `.editorconfig` + `.inputrc`.

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

Minimal `features.json` skeleton (drop into `<repo>/specs/POLISH-004-cross-os-polish-bundle/features.json`):

```json
[
  {
    "id": "POLISH-004-cross-os-polish-bundle-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
