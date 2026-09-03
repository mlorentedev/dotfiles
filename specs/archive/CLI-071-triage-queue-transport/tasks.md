---
tags: [spec, tasks, templates]
created: "2026-09-02"
---

# Tasks - CLI-071-triage-queue-transport

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `fix/cli-071-triage-queue-transport` (off `origin/main` @ `45e2c61`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" — R1 is resolved by AC5 rather than left open

## Implementation

> TDD order. The seam comes first because nothing else in this spec is testable
> without it — every later task is written against a fake, never the network.

- [x] [AC1] Failing test: `fetchWith` drives the whole path against an injected
      fake runner, no `gh` on PATH, and returns the expected queue
- [x] [AC1] Introduce the `ghRunner` seam and route `FetchWithRegistry` through
      it, keeping the exported signature unchanged
- [x] [AC2][AC4] Failing test: the fake records the requests it receives; assert
      REST paths only, and assert a full page of pull requests still refuses
- [x] [AC2] Replace the `gh pr list` call with `gh api repos/{owner}/{repo}/pulls`
      and map the REST list shape (`html_url`, not `url`)
- [x] [AC3][AC4] Failing test: per-PR comments, with a `[bot]`-suffixed author
      resolving against the bare registry login, and a full comment page refusing
- [x] [AC3] Fetch comments per pull request via
      `gh api repos/{owner}/{repo}/issues/{n}/comments` and map `user.login` /
      `created_at`
- [x] [AC5] Failing test: a fake with a per-call delay, ten pull requests, must
      complete well inside the 5s budget — fails if the calls are serial
- [x] [AC5] Bound the per-PR fan-out with a semaphore-limited worker pool (no new
      dependency; `x/sync` is not in `go.mod` and adding one is its own gate)
- [x] [AC6] Pin every field the renderers consume, since both are pure functions
      of `[]Status` and that type is untouched — plus a production smoke test
      against the live repository, which turned out to be the stronger evidence
- [x] Refactor: delete `ghPR`, fold the wire types into the REST shapes, and keep
      every doc comment honest about what the code now does (lesson 259 —
      grep the prose, not just the callers)

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test
- [x] Every acceptance criterion has a matching entry in `features.json` with a
      non-vacuous verification command — **anchor every `-run` pattern**, because
      `go test -run <no-match>` exits 0 and would make the check unfailable
- [x] Mutation-test each new guard: remove the assertion it defends and watch the
      test fail
- [x] `go build ./... && go vet ./... && go test ./...` green
- [x] `GOOS=windows go vet ./...` green (the Windows leg compiles the same tree)
- [x] `golangci-lint run` with the pinned version from `versions.conf`
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in
- [x] PR opened referencing this spec folder with `Refs #1454` (not `Closes`:
      the spec-gate refuses a closing keyword on a PR that does not archive)

## Machine-readable features

See the sibling `features.json`. Each acceptance criterion maps to ≥1 feature with
an executable `verification` command. The agent may not write `"state": "passing"`
— only the harness may, after capturing exit code 0.
