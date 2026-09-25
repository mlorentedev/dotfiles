---
tags: [spec, tasks, templates]
created: "2026-08-29"
---

# Tasks - CLI-055-owner-only-acl

> TDD order. One task = one focused commit. Tick as you go. Frozen at the start of `implementing`.

## Setup

- [x] Branch created from main: `feat/owner-only-acl`
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

- [x] [AC1] [AC2] `fsmode`: `Apply(path, mode)` = `os.Chmod` everywhere; on Windows, for an owner-only mode, a protected DACL of the token user + LocalSystem (`ACLFromEntries`, `SetNamedSecurityInfo` with `PROTECTED_DACL`). `OwnerOnly(mode)` table test; Windows test through `icacls` and `SE_DACL_PROTECTED`; a shared mode leaves the ACL untouched.
- [x] [AC3] The three writers — `deploy.stage`, `deploy.commit`, `secrets.AtomicWriteMode` — call `fsmode.Apply`; no `os.Chmod(` call remains in them.
- [x] [AC3] `fsmode.Needs(path, mode)` — the read-only question — and `Deploy`'s two in-sync returns ask it: content in sync but mode off (an inherited DACL on a 0600 deployed by an older binary) is reported `mode fixed` (`would fix mode` on `--dry-run`) and applied without a content rewrite. Found on the box: the first AC4 run said `in sync` and left the inherited ACL in place.
- [x] [AC4] Box: `dotf deploy pi` with the branch binary → `icacls ~/.pi/agent/models.json` lists the user and SYSTEM only, no `(I)`; the 0644 neighbour keeps its `(I)` entries; a second run is `in sync`.
- [x] [AC5] The Windows-only tests resolve the user from the process token, so they pass under whatever account runs them (an administrator on CI's hosted runner); `GOOS=linux go vet` keeps the POSIX build honest from the Windows box.

## Review round 1 (2026-09-24, retroactive, FAIL)

Reviewed at the landing commit `3f554ad` (#1380) by `nan/deepseek-v4-flash`. Every finding's disposition is in `verification.md`.

- [x] F1: `features.json` f5 names the call the code makes, `OpenProcessToken(CurrentProcess())` + `GetTokenUser()`, and the proposal's AC5 risk note says the same
- [x] F3: the proposal's What names the in-sync mode fix `Deploy` gained
- [x] Round 2 review ran (FAIL, below); it did not pass

## Review round 2 (2026-09-24, FAIL)

`nan/deepseek-v4-flash` at `8a110d7`.

- [x] F1: every verifier that names a Windows-only test runs it on Windows and, elsewhere, requires it to exist and compile, so a missing test fails the verifier
- [x] f4 names its platform, and says why it exits 1 elsewhere
- [x] F7: the account premise matches CI (an administrator on the hosted runner, a domain account on the box)
- [x] Round 3 review passes before archive

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by at least one test (AC4 by the box transcript in `verification.md`)
- [x] Every acceptance criterion has a matching entry in `features.json` with a non-vacuous verification command
- [x] `go build ./... && go vet ./... && go test ./...`, `GOOS=linux go vet ./...`, `golangci-lint run` (pinned)
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
