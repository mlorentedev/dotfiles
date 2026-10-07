---
tags: [spec, tasks, templates]
created: "2026-10-06"
---

# Tasks - PLAT-001c-tools-via-mise

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Branch created from main: `feat/plat-001c-installer-goarch` (T1a); each later PR gets its own branch from `main`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [ ] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> One PR per row. Go: `go build ./... && go vet ./... && GOOS=windows go vet ./... && go test ./...` and the pinned `golangci-lint`.

### T1a — installer support (this PR)

- [x] [AC1] Failing test, then `AssetName`: a `goos/goarch` key wins over `goos`; `goos` still resolves alone
- [x] [AC2] Failing test, then `expectedChecksum`: `./name` and `*name` manifest lines verify `name`
- [x] [AC3] Failing test, then `Install`: a `github-release` tool with no asset for this OS/arch is skipped with a message and no error, as `Plan` reports it; `Load` rejects an asset key that names no known platform, so a misspelt key cannot turn into a silent skip everywhere

### T1b — mise in the catalog (after the release carrying T1a is the `DOTF_VERSION` pin)

- [ ] [AC4] `packages.json` entry `mise`, keyed `goos/goarch`, `SHASUMS256.txt`; `MISE_VERSION` in `versions.conf`; installed and probed on the Mac

### T2 — `dotf tools sync` (this PR, stacked on T1a)

- [x] [AC5] Failing tests, then `tools.ParseMiseTools`: pins marked `# mise: cli` on the line before them, name from `NAME_VERSION`; a marker that marks no pin is an error
- [x] [AC5] `tools.MiseSync`: renders `<mise config dir>/conf.d/dotfiles.toml` (owned end to end, written atomically), runs `mise install` through an injected runner (lesson 335), probes every tool through `mise which` and `--version` at or above the pin; a second run writes and installs nothing; the hand-written `config.toml` is never touched
- [x] [AC5] `dotf tools sync [--dry-run] [--versions]`; `env.ResolveVersionsPath`, with the checkout-first rule extracted to one helper (`resolveInCheckout`) shared with the catalog and the secret store
- [x] `versions.conf` marks age, bats, shellcheck, golangci-lint, zoxide and adds pins for jq, direnv, fzf and lazygit; eza stays unmarked (F-056)
- [x] On the Mac: `dotf tools sync` installed age, direnv, fzf, jq, lazygit and zoxide, each runs at its pin, and a second run reported nothing to do

### T2b — committed `mise.lock` and the converge tools step

- [ ] Commit `mise.lock` (verified 2026-10-06: `mise lock -g --platform linux-x64,macos-arm64,windows-x64` writes `~/.config/mise/mise.lock` with per-platform checksums and attestation provenance for a conf.d config); sync deploys it and installs with `--locked`; CI regenerates it
- [ ] [AC6] The `tools` reconciler in `dotf converge`, after the records step (after #2025 merges, which owns `Registry` today); `--plan` lists the tools to install

### W2b — shims on PATH (after #2013 P6/P7)

- [ ] `mise activate` in the rc files, the shims directory for non-interactive callers, and a doctor check

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

Minimal `features.json` skeleton (drop into `<repo>/specs/PLAT-001c-tools-via-mise/features.json`):

```json
[
  {
    "id": "PLAT-001c-tools-via-mise-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
