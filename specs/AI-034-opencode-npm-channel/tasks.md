---
tags: [spec, tasks, templates]
created: "2026-08-27"
---

# Tasks - AI-034-opencode-npm-channel

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/opencode-npm-channel`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [P] [AC5] Write ADR-036 (channel per tool class, `packages.json` pin SSOT, version probe in Go, shadowed copy is a finding, migration by the same mechanism)
- [x] [P] [AC3] Write failing tests `TestProbeVersion_*` (real banners: `OpenCode locked.`, `hive-vault 3.0.0`); implement `tools.ProbeVersion` + `ExecRunner`
- [x] [AC3] Write failing test `TestToolsVersionCmd`; implement `dotf tools version <name>` (exit 1, empty stdout when none)
- [x] [P] [AC4] Write failing tests `TestCatalogPin_*`, `TestCheckShadowedCatalogTools_*`; implement `loadCatalog`, `catalogPin`, `checkShadowedCatalogTools`, `System.dirsProviding`
- [x] [AC4] `checkOpenCode` reads the catalog pin (`matchPinFrom`), names the legacy curl copy, calls the shadowed check
- [x] [P] [AC1] `packages.json` gains opencode (npm, `opencode-ai`); `OPENCODE_VERSION` leaves `versions.conf`
- [x] [P] [AC2] `setup-linux.sh`: delete the curl block + `~/.opencode/bin` variables; post-deploy assertion on PATH only; hive gate via `dotf tools version hive` (by path too)
- [x] [P] [AC2] `setup-windows.ps1`: delete the `$opencodeVersion` block, the winget entry and the loop's version-pin machinery; hive gate via `dotf tools version hive`
- [x] [P] [AC2] `.zshrc` / `.bashrc`: drop the `~/.opencode/bin` PATH export
- [x] [AC1] [AC2] [AC3] Rewrite the bats guards: `opencode.bats`, `setup-windows.bats`, `versions-conf.bats`, `hive-upgrade-timer.bats`
- [x] Refactor for clarity: `matchPin` delegates to `matchPinFrom` (source named in every message)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` (see below) with a non-vacuous verification command
- [x] Type checks pass (`go build ./... && go vet ./...`, Windows and `GOOS=linux`)
- [x] Lint passes (`golangci-lint run ./...` pinned 2.12.2; `shellcheck setup-linux.sh`; PowerShell parser + ASCII-only on the `.ps1` edits)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec folder

## Machine-readable features

This spec emits a sibling `features.json` (alongside this file) following [[pattern-feature-list-as-primitive]]. The JSON is the harness-facing contract: each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state. Reviewers must reject PRs where features.json contains `passing` entries with empty `evidence`.
