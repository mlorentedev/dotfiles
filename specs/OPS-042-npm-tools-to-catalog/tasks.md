---
tags: [spec, tasks, templates]
created: "2026-08-29"
---

# Tasks - OPS-042-npm-tools-to-catalog

> TDD order. One task = one focused commit. Tick as you go. Frozen at the start of `implementing`.

## Setup

- [x] Branch created from main: `feat/npm-tools-to-catalog`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] `packages.json`: `obsidian` (npm `obsidian-cli` 0.5.1 — the binary is `obsidian`, which the installer probes) and `yarn` (npm `yarn` 1.22.22). `tests/packages-json.bats` already asserts the shape of every entry.
- [x] [AC2] `setup-linux.sh`: the obsidian-cli block (comment + `if`/`fi`) and the yarn block deleted; `bash -n` clean. `setup-windows.ps1`: sections 2c (Obsidian CLI) and 2d-bis (yarn, with its `versions.conf` parser) deleted; the PATH-state note no longer names them; PSScriptAnalyzer 0 findings. `Test-VersionAtLeast` stays for pi.
- [x] [AC3] `versions.conf`: `OBSIDIAN_VERSION` (never read by anything) and `YARN_VERSION` removed. `dotf doctor`'s yarn row reads `catalogPin(…, "yarn")` through `matchPinFrom`, like copilot and opencode; `TestCheckVersionMatch` feeds a catalog fixture instead of the versions map.
- [x] [AC2] [AC3] bats: `setup-linux.bats` and `setup-windows.bats` assert the catalog entries, the absence of both npm blocks and of the two pins, and that `setup-windows.ps1` still runs `dotf tools install`.
- [x] [AC4] Box: `dotf tools install` with the branch's catalog reports both at pin; `dotf doctor --verbose` shows `yarn version matches packages.json (1.22.22)`.

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (AC4 by the box transcript in `verification.md`)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...`, `GOOS=windows go vet ./...`, `golangci-lint run` (pinned), `bash -n`, PSScriptAnalyzer, bats
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
