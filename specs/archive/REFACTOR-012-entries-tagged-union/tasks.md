---
tags: [spec, tasks]
created: "2026-08-15"
---

# Tasks - REFACTOR-012-entries-tagged-union

> TDD order. One task = one focused commit.
> `[P]` — no dependency on another unchecked task, safe to run in parallel.
> `[AC<n>]` — helps satisfy acceptance criterion `<n>` in `proposal.md`.

## Setup

- [x] Branch created from origin/main: `fix/refactor-012-backend-tagged-union` (worktree `dotfiles-wt-r12`)
- [x] `proposal.md` complete, acceptance criteria testable
- [x] Coordination: peer session `dotfiles-wt-h72-dd` holds HARNESS-072. Off-limits here:
      `harness/manifest.json`, `AGENTS.md`, `ai/claude/CLAUDE.md`,
      `pattern-change-lifecycle.md`, `specs/HARNESS-072-*`. No overlap with this scope.

## Implementation

### Backend SSOT (the durable guard)

- [ ] [P] [AC5] Failing test: every backend the registry accepts has a resolver, and `""` maps to age
- [ ] [AC5] Add exported backend constants + one canonical list in `internal/secrets`
- [ ] [AC5] Point `registry.go`'s two validation switches and `resolve.go`'s `resolvers()` at the list
- [ ] [AC5] Replace bare `"bw"` literals at their comparison sites (`registry.go`, `github.go`, `checks_deploy.go`, `checks_bw_reach.go`)

### Entry identity (instance 2)

- [ ] [P] [AC1] Failing test: two distinct bw secrets exposing one var must be rejected by `envSourceMap` (passes wrongly today)
- [ ] [AC1] Add `Entry.SourceID()`, backend-qualified, on the union's own type
- [ ] [AC1] `render.go`'s duplicate-var guard compares `SourceID()`, and the error names both sources legibly for a bw entry

### PAT selection (instance 1a)

- [ ] [P] [AC2] Failing test: a registry fixture with a bw-backed `validate: github-token` secret is returned by `githubPATSecrets` (dropped today)
- [ ] [AC2] `Entries()` carries `Validate` through both `ageEntries` and `bwEntries`
- [ ] [AC2] `githubPATSecrets` selects on `Validate == "github-token"` and dedupes on `SourceID()`
- [ ] [AC2] Add the missing `validate: github-token` to `GITHUB_PERSONAL_ACCESS_TOKEN` in `secrets/registry.yaml`

### PAT resolution (instance 1b — the dead check)

- [ ] [P] [AC3][AC4] Failing table test for the resolution branches: resolved → probe; absent → SKIP; backend unavailable → SKIP; other error → WARN; 401 → FAIL
- [ ] [AC3] Add the `ResolveSecret` seam to `System` (production impl builds a `Loader`; reader selected from the bw-serve probe, never an optimistic shellout)
- [ ] [AC3] `probePATSecret` resolves through the seam; delete `resolvePATToken`'s ambient-env read and the `secrets_refresh` message
- [ ] [AC4] Classification per the taxonomy in AC4, one branch each

### Close

- [ ] Every AC covered by at least one test
- [ ] `features.json` written, every entry with a non-vacuous verification command
- [ ] `go build ./... && go vet ./... && go test ./...` green
- [ ] `golangci-lint run` green **with the pin from `versions.conf`** (BUG-071: a local binary on a different major reports 0 issues on code CI rejects)
- [ ] [AC6] Live: `dotf doctor` PAT section on this machine, and one run with the bw daemon down (timed)
- [ ] `verification.md` filled with the evidence above
- [ ] PR opened referencing this spec folder
- [ ] Propose `/adversarial-review REFACTOR-012-entries-tagged-union` before archive — never self-served (`dotf spec review`)

## Machine-readable features

Emitted as `features.json` beside this file ([[pattern-feature-list-as-primitive]]).
The agent may not write `"state": "passing"` — only the harness may, after running
`verification` and capturing exit 0.
