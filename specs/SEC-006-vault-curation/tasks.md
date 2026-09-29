---
tags: [spec, tasks]
created: "2026-09-27"
---

# Tasks - SEC-006-vault-curation

## Setup

- [x] Branch `feat/secrets-curate` from main, worktree `dotfiles-wt-secrets-curate`
- [x] `proposal.md` complete; the passkey risk is resolved by refusal, not measurement

## Implementation

- [x] [AC1] [AC2] Extend the value-free projection (`ItemSummary`) with id, type, reprompt, field kinds, passkey/attachment counts, TOTP presence and URIs; a projection test plants a TOTP seed, passkey and attachment key material and asserts none survives (`layout_test.go`)
- [x] [AC2] [AC6] `applyMutation`: one typed entry point for every curate edit, refusing a passkey item before any branch (`curate_mutate.go`, `curate_mutate_test.go`)
- [x] [AC1] [AC2] [AC3] Plan parser and planner: per-op converged predicate, blocking rules, gates (`curate.go`, `TestParseCuratePlan`, `TestCurateBlocks`, `TestCurateAppliesEveryOpAndConverges`)
- [x] [AC4] Digest over rows, resolved ids, revision dates and states (`TestCurateDigestTracksRevisions`)
- [x] [AC3] [AC4] [AC5] [AC6] `dotf secrets curate` command: dry run, `--apply --digest`, re-plan to converged, passkey count before/after (`secrets_curate.go`, `secrets_curate_test.go`)
- [x] Mutation check: disabling each of 14 guards (passkey at plan and at mutation, attachments on delete and merge, merge duplicate passkey and TOTP, keeper passkey, drop=uris, registry ownership of a target and of a merge keeper, a delete that keeps itself, revision in digest, digest check, passkey count) fails a test
- [x] [AC2] Review round 1: a registry-declared merge keeper blocks when it would take a URI, and a `delete` cannot keep itself; the four functions over 40 lines are split
- [x] [AC2] `drop=uris`: a merge may declare that the duplicate's URIs are not carried, so a keeper with a passkey stays untouched (`TestCurateDropURIsLetsAPasskeyKeeperStandUntouched`)

## Closing

- [x] Every acceptance criterion is covered by a test and a `features.json` entry
- [x] `go build`, `go vet` (linux and windows), `go test ./...`, golangci-lint (pinned 2.12.2) green
- [x] No unrelated changes in the diff
- [x] `verification.md` filled in
- [x] PR opened (#1790, merged `5fdf472`; the guide in #1794, merged `20aa19e`); the live apply (plan in the knowledge vault) is the owner's ceremony, recorded in #1784
