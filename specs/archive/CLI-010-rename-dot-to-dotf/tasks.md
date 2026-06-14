---
tags: [spec, tasks, templates]
created: "2026-06-13"
---

# Tasks - CLI-010-rename-dot-to-dotf

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from origin/main: `chore/rename-dot-to-dotf` (worktree ../dotfiles-cli-010)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

> Mostly a mechanical name sweep. Update Go code + its tests first (compiler/`go test` is the guard), then the release pipeline, then bootstrap shell, then docs. One focused commit per logical group.

- [x] Rename binary package dir `cli/cmd/dot/` → `cli/cmd/dotf/`; update Cobra root `Use` string + the version test (`go test ./...` green)
- [x] Update `cli/.goreleaser.yaml` (`project_name`, `builds.id`, `builds.binary`, archive `id`) → `dotf`
- [x] Update `.github/workflows/cli.yml` smoke paths (`./cmd/dotf`)
- [x] Rename `scripts/install-dot.sh`→`install-dotf.sh` — artifact name, extracted binary, installed `~/.local/bin/dotf`, function `install_dotf`, vars `DOTF_*`/`_dotf_*`
- [x] Update `versions.conf` `DOT_VERSION` → `DOTF_VERSION=0.2.0` and `scripts/healthcheck.sh` `dot` check → `dotf` (healthcheck.ps1 has no CLI check — Windows install deferred)
- [x] Update `setup-linux.sh` install wiring → `install-dotf.sh` / `install_dotf` / `dotf`
- [x] Update live docs naming the `dot` binary: `AGENTS.md` (§245/249/250), `README.md:215`, `docs/architecture.md` (table + cli layout tree + rules) — and `tests/architecture-md.bats:31` (`cmd/dotf`) in lockstep
- [x] Amend `docs/adr/adr-020-tooling-cli-go-convergence.md` (banner + Amendment section); body left historical per ADR convention (banner covers inline `dot` examples)
- [x] Rename the matching bats (`tests/install-dot.bats`→`install-dotf.bats`) in lockstep; left `init-spec`/`archive-spec` repoints to CLI-005
- [x] Guard: no live `dot`-binary ref remains outside ADR-020 historical body + provenance/feature-id slugs
- [x] Verify: `go test ./...` green, `go build ./cmd/dotf`, smoke `dotf spec init/archive`, `shellcheck` clean, bats 57/57

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-010-rename-dot-to-dotf/features.json`):

```json
[
  {
    "id": "CLI-010-rename-dot-to-dotf-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
