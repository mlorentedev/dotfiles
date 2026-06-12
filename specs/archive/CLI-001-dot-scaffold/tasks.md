---
tags: [spec, tasks, templates]
created: "2026-05-13"
---

# Tasks - CLI-001-dot-scaffold

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/dot-cli-scaffold` (descriptive name; ticket ID lives in the spec + PR body)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (R1 deferred with an explicit landing point — the bootstrap/PATH PR; R2-R5 resolved by design within this PR)

## Implementation

- [x] Archive `DX-002-dot-umbrella-command` to `specs/archive/_abandoned/` with `status: abandoned` (superseded by ADR-020)
- [x] Write failing table-driven test for the root command: `dot --help` exits 0 + prints usage; `dot version` prints version; unknown subcommand exits non-zero with usage hint (red verified: `undefined: newRootCmd`)
- [x] Scaffold `cli/go.mod` (module `github.com/mlorentedev/dotfiles/cli`, go 1.26.0) + minimal Cobra root in `cli/cmd/dot/` to make the test pass (green verified)
- [x] Lint: golangci-lint defaults, no config file (config-before-need antipattern); local v1.64.8 too old for go 1.26 — `gofmt -l` clean + `go vet` OK locally, full lint delegated to CI
- [x] Add `.github/workflows/cli.yml`: path-filtered on `cli/**`, matrix ubuntu+windows (`go test` + smoke `dot --help`), lint job, goreleaser snapshot job, tag-triggered release job
- [x] Add `cli/.goreleaser.yaml`: plain `v*` release tags (monorepo tag prefix is goreleaser Pro-only — verified empirically), static builds linux/macOS/windows amd64+arm64 — exercised locally with a throwaway tag + by the CI snapshot job
- [x] Write `cli/README.md` (build / test / lint / release one-liners)
- [ ] Manual QA pass: Linux exercised locally (help/version/bogus/released binary); Windows — inspect CI smoke output on the PR; file GitHub issues for findings

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

Minimal `features.json` skeleton (drop into `<repo>/specs/CLI-001-dot-scaffold/features.json`):

```json
[
  {
    "id": "CLI-001-dot-scaffold-f1",
    "behavior": "<one-line copy of an acceptance criterion>",
    "verification": "<single shell command; exit 0 means pass>",
    "state": "pending",
    "evidence": ""
  }
]
```
