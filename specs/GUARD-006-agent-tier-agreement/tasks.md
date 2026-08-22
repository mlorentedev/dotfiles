---
tags: [spec, tasks, templates]
created: "2026-08-22"
---

# Tasks - GUARD-006-agent-tier-agreement

## Setup

- [x] Branch created from main: `fix/doctor-record-tier-agreement`
- [x] `proposal.md` complete, acceptance criteria testable
- [x] No open questions left

## Implementation

- [x] Failing table test: a tier one deploy target cannot answer must FAIL, naming all three facts
- [x] `checkAgentTiersResolve` reading the manifest's `agents.deploy` + `record_dir`
- [x] `readAgentFrontmatter` — minimal, single-line values, first block only
- [x] `recordTargets` with its own table test, because an inverted default turns correct data into noise
- [x] Wired into `checkModelMap` AFTER the map validates, so it never stacks on a load failure
- [x] Silence on inputs it does not own (absent manifest / record dir / no deploy targets)
- [x] Verified against the real tree with a binary built from this branch

## Closing

- [x] Every acceptance criterion covered by at least one test
- [x] `features.json` verifiers propagate the runner's exit status and pin tests by unique name
- [x] `go build` / `go vet` / `GOOS=windows go vet` / `go test ./...` green
- [x] `golangci-lint run` -> 0 issues at the `versions.conf` pin; `gofmt` clean
- [x] `verification.md` filled in with this session's output
- [x] PR opened referencing this spec folder
