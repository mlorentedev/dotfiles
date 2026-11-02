---
tags: [spec, tasks, templates]
created: "2026-11-01"
---

# Tasks - BUG-110-spec-home-repo-resolver

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/spec-home-repo-resolver`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC4] Failing table test for `spechome.Resolve`: precedence flag > file > env > origin, plus the archive home taken from a GitHub origin only
- [x] [AC3] Failing tests for `specs/.bitacora-repo`: valid, empty, two values, not `owner/name`
- [x] [AC2] Failing tests for the fail-closed error: Gitea HTTPS, Gitea SSH, absent origin; the error carries forge class and host, never the URL
- [x] [AC2] [AC3] [AC4] Implement `spechome.Resolve` in leaf package `spechome` (export `initrepo.OriginURL`, `OriginHost`, `ClassifyURL`)
- [x] [AC1] [AC2] [AC4] Wire `spec init`'s work-gate through the resolver (replaces `workGateRepo`)
- [x] [AC1] [AC2] [AC5] [AC6] Wire `spec audit` through the resolver; archive lookup uses the archive home and reports "could not be asked" when it is empty
- [x] [AC1] [AC2] Wire the doctor `spec-issue-state` check through the resolver
- [x] [AC4] Test that `spec init` and `spec audit` resolve the same issue home for one repo and environment
- [x] [AC5] Test with a recording runner: Gitea origin + override, no call names the Gitea slug
- [x] Document `specs/.bitacora-repo` in `spec audit --help` and the `--repo` flag

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`
- [x] `golangci-lint run` at the pinned version
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder
- [x] Independent review
