---
tags: [spec, tasks, secrets, age, doctor]
created: "2026-07-01"
---

# Tasks - CLI-024-secrets-age-discovery

> TDD order. One task = one focused commit. Tick as you go. Reorder freely while spec is in `draft` state; freeze once you start `implementing`.

## Setup

- [x] Branch created from main: `feat/age-key-discovery` (external worktree `dotfiles-wt-age-discovery`)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions" (SOPS-vs-age resolved by the codebase)

## Implementation

> TDD: red → green → refactor, one commit each.

### A. Env-contract discovery (AC1)

- [x] Extend `tests/env-contract.bats`: assert `AGE_KEY_PATH` + `SOPS_AGE_KEY_FILE` exist, are `required:false`, default to `~/.config/age/key.txt` per OS, and carry **no** `path_exists` validation (red → green).
- [x] Add both entries to the dotfiles `env-contract.json` only — the generic `initrepo/templates/env-contract.json` starter stays empty (guard test added) (green).
- [x] Confirm `dotf env generate` renders them: `TestRealContractRendersAgeKeyDiscovery` (real contract → `paths.{sh,ps1}`, both OSes). Live smoke pending in verification.

### B. Doctor round-trip verifier (AC2–AC5)

- [x] Added `AgeRoundTrip func(keyPath string) error` seam to `System`; `realSystem` wires `ageRoundTrip` (reuses `secrets.AgeRecipient`/`AgeEncrypt`/`AgeDecrypt`); `newSys` default = success. Mirrors the `HTTPGet` seam idiom.
- [x] Doctor tests (table): key+valid → PASS; key+failing seam → FAIL; key absent → WARN + seam-spy never called; age-keygen absent → WARN + skip (red → green).
- [x] Upgraded `checkSecretsTooling`: key present + `age`/`age-keygen` present → invoke seam, map to PASS/FAIL; WARN on absent key; skip (WARN) when age-keygen absent.
- [x] Verdict logic is a single readable flat block (< 40 lines, < 4 nesting — Code Quality Rules).

### C. Guard the guard (AC5)

- [x] `grep 'exec.Command("age'` in `internal/doctor/` returns nothing — all age I/O behind the `secrets` seams (features.json f4).

## Closing

- [x] Every acceptance criterion from `proposal.md` is covered by ≥1 test
- [x] Every acceptance criterion has a matching `features.json` entry with a non-vacuous verification command
- [x] `go vet ./...` clean; `go test ./...` green (except a pre-existing unrelated `internal/spec` failure — see verification.md); `env-contract.bats` green; touched `.go` gofmt-clean in LF form
- [ ] Live smoke: real key present → `dotf doctor` shows the round-trip PASS; temporarily corrupt/rename the key → FAIL (capture both in verification.md)
- [x] No unrelated changes in the diff (no scope creep)
- [x] `verification.md` filled in with evidence
- [ ] PR opened referencing this spec folder and #518

## Machine-readable features

This spec emits a sibling `features.json` following [[pattern-feature-list-as-primitive]]. Each acceptance criterion maps to ≥1 feature with `id`, `behavior`, `verification` (executable command), `state` (lifecycle), and `evidence` (harness-captured output).

**Pass-state gating:** the agent CANNOT write `"state": "passing"` — only the harness, after running `verification` and capturing exit code 0, may set that terminal state.
