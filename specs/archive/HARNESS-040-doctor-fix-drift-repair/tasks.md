---
tags: [spec, tasks, templates]
created: "2026-06-25"
---

# Tasks - HARNESS-040-doctor-fix-drift-repair

> TDD order. One task = one focused commit. Tick as you go.

## Setup

- [x] Branch created from main: `feat/doctor-fix-drift-repair` (descriptive, no ticket-ID per AGENTS.md branch rule)
- [x] `proposal.md` is complete and acceptance criteria are testable
- [x] No open questions left in `proposal.md` "Risks / open questions"

## Implementation

### Part 2 — OS-aware env-contract checks (smaller, isolates the contract churn first)

- [ ] Add a `contractOSKey(sys)` helper + table test: `GOOS` "windows"→"windows", ""/"linux"→"linux", "darwin"→"darwin"
- [ ] Write failing test: `checkContractPath` with `GOOS="windows"` checks the `windows` `required_path_entries`
- [ ] Thread the OS key through `checkContractPath`
- [ ] Write failing test: `checkContractEnvVars` with `GOOS="windows"` resolves `windows` defaults + honors `required_on: windows`
- [ ] Thread the OS key through `checkContractEnvVars` (`Default[key]`, `requiredOn` logic)
- [ ] Regression test: `GOOS="linux"` output unchanged (guards the byte-equivalence expectation)

### Part 1 — auto-memory junction check

- [ ] Add non-mutating `memlink.Status(cwd, target, project, vault)` + unit test (linked / real-dir / missing-source / repairable)
- [ ] Write failing test: `checkAutoMemoryLink` → SKIP when no vault source resolves
- [ ] Write failing test: → FAIL when source exists but link missing (no `--fix`)
- [ ] Write failing test: → FIX recreates link under `--fix`; PASS on re-run (idempotent)
- [ ] Implement `checkAutoMemoryLink(sys, rep, fix)` in `package doctor`, share the encode logic with the adapter
- [ ] Wire the check into `doctor.Run` (non-`--quick` sweep, alongside `checkVaultHooks`)

## Closing

- [ ] Every acceptance criterion covered by ≥1 test
- [ ] `features.json` written with executable verification commands
- [ ] `go vet` / `go build` clean; lint passes (heed gopls QF/staticcheck hints — CI golangci-lint v2.x)
- [ ] No unrelated changes in the diff
- [ ] `verification.md` filled in
- [ ] Part 3 (Hive venv) ticketed as a new issue
- [ ] PR opened referencing this spec folder (no auto-merge)
