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

### T2b — the converge tools step (this PR)

- [x] [AC6] The `tools` reconciler in `dotf converge`, after the records step. Its plan is `MiseSync.Plan`, its apply `MiseSync.Apply`, and its probe re-plans and requires nothing pending. Without mise on PATH it is skipped, naming the remedy, not failed; `Result.Skip` lets any reconciler say so. The mise runners move to `tools.HomeRunners`, shared with `tools sync`
- [x] On the Mac: `dotf converge` -> records-mirror, records-harness, tools all `[ OK ]`, `9 pinned CLI(s) at their pin`

### Wave 3 — committed `mise.lock`

- [ ] Commit `mise.lock` and install with `--locked`. Measured 2026-10-06: mise keeps one lockfile for the whole global config, and `--locked` fails when any configured tool (the Mac's hand-written `config.toml`) is missing from it, so the lock lands when `conf.d/dotfiles.toml` is the only tools config (comment on #2013). The aqua backend already verifies each asset's checksum and attestation

### W2b — shims on PATH (after #2013 P6/P7)

- [x] `mise activate` in `.zshrc`, `.bashrc` and the PowerShell profile, guarded on mise, before direnv and zoxide (#2013 P6, PR #2043)
- [ ] The shims directory for non-interactive callers (setup, cron, an agent's shell), which read no rc file
- [x] The doctor check: T3 below

### T3 — doctor reads a darwin machine as darwin, and checks the mise CLIs (stacked on #2043)

- [x] Failing tests, then one gate for the `~/Applications/<tool>-<version>` toolchain layout: it applies on linux only (`platform.Supports`), and each other OS names how it gets its toolchains instead: winget on windows, mise on darwin (ADR-044 Wave 3)
  - `checkVersionMatch` (the versioned directories) and `checkToolHomeEnvVars` (the `*_HOME` variables must be set) consult it; on the Mac they reported 11 FAILs for a layout darwin does not use
  - `checkVersionedPaths` stays on every OS: a `*_HOME` that is set must point at a real toolchain, which is F-041 (a `JAVA_HOME` pointing at nothing breaks macOS's `/usr/bin/java`)
- [x] tmux is class 3 (ADR-044): the not-installed remedy names the OS's package manager, not `apt` on every POSIX host. `checkTmux` keeps the installed and version half only; `~/.tmux.conf` is the `tmux` deploy entry's, checked by `checkDeployManifest` since #2043
- [x] W2b doctor check: the CLIs marked `# mise: cli` in `versions.conf` run at their pin through `mise which`, the same plan `dotf tools sync --dry-run` prints, on every OS; a changed rendered config is a WARN naming the sync
- [x] On the Mac: the doctor run before and after, the FAIL count and what remains (recorded in `verification.md`)

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
