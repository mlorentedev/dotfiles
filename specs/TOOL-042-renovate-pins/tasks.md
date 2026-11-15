---
tags: [spec, tasks, templates]
created: "2026-11-15"
id: "TOOL-042-renovate-pins"
type: spec-tasks
status: active
owner: manu
---

# Tasks - TOOL-042-renovate-pins

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.
>
> **Inline markers** (optional, additive — borrowed from `github/spec-kit`, adapt-not-adopt per #141):
> - `[P]` — this task has **no dependency on another unchecked task**, so it is safe to run in parallel (fan out to a `Workflow`, or just batch). TDD chains (test → implement → refactor of the *same* behavior) are sequential and must NOT carry `[P]`; independent behaviors can.
> - `[AC<n>]` — this task helps satisfy **acceptance criterion #`<n>`** from `proposal.md`. Lets `/spec check` map coverage deterministically; omit it and the check falls back to semantic judgment.

## Setup

- [x] Lane branch `feat/3284-renovate-pins`, spec gated on #3284
- [x] Measure the spec-gate on a Renovate-shaped PR (verification.md)
- [x] File the Python ticket: #3541

## Implementation

- [x] [AC1] [AC2] [AC3] [AC6] `cli/internal/tools/renovate_test.go`: six tests reading the committed `versions.conf`, `renovate.json`, `cli/go.mod`, `packages.json` and the spec-gate threshold
- [x] [AC1] [AC6] Annotate all 55 pins in `versions.conf`: 46 with datasource, depName and depType, 9 skipped with a reason; header documents the line
- [x] [AC2] [AC4] [AC6] Root `renovate.json`: three custom managers (versions.conf, cli/go.mod, packages.json), one group per depType, Go patch-only, three-day wait on release binaries
- [x] [AC1] [AC2] [AC3] [AC6] Mutation check: seven deliberate breaks, each failed a test (verification.md)
- [x] [AC4] `renovate-config-validator --strict` and a `--platform=local` extract (verification.md)
- [x] [AC5] `dotf tools sync --dry-run` identical before and after; bats files that read versions.conf pass

## Closing

- [ ] Every acceptance criterion covered by a test or a recorded measurement
- [ ] `go build`, `go vet`, `go test ./...`, golangci-lint (pinned, Linux and Windows)
- [ ] `verification.md` filled in
- [ ] Draft PR referencing this spec; owner flags in the body
- [ ] Independent review before archive
