---
tags: [spec, tasks, secrets, sync, cli, go]
created: "2026-06-26"
---

# Tasks - CLI-024-secrets-sync-verify

> TDD order. One task = one focused commit.

## Setup

- [x] Branch from main: `feat/sync-ci-validate` (worktree)
- [x] `proposal.md` complete; acceptance criteria testable

## Implementation

- [x] Registry schema: optional `validate` field on `Secret` (+ `Validate` on flattened `Entry`).
- [x] `SelectCI` carries the marker onto each Upload entry.
- [x] `GitHubTokenValidator` seam + `GHTokenValidate` (gh api user, authenticates as the
  token under test; strips inherited GH_TOKEN/GITHUB_TOKEN first).
- [x] `sync ci`: pre-upload liveness pass for marked entries, fail loud; `--skip-verify` flag.
- [x] Mark `RELEASE_TOKEN` + `BITACORA_PAT` `validate: github-token` in registry.yaml.

## Tests

- [x] Dead token aborts before any upload (nothing pushed).
- [x] Live token validates then uploads (verified line).
- [x] `--skip-verify` bypasses (validator not called).
- [x] Unmarked entries never call the validator (opt-in only).

## Closing

- [x] Every AC covered by ≥1 test
- [x] `features.json` carries non-vacuous verification commands
- [x] `go test ./...` green; `go vet` + `gofmt` clean; `go build ./...` ok
- [x] `verification.md` filled in
- [ ] PR opened referencing this spec + #635
