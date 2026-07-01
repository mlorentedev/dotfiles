---
tags: [spec, verification, secrets, age, doctor]
created: "2026-07-01"
---

# Verification - CLI-024-secrets-age-discovery

## Evidence

Each acceptance criterion → concrete proof (test name / command / observed behavior).
Commit hash to be stamped on the squash-merge.

- [x] **AC1** (env-contract declares AGE_KEY_PATH + SOPS_AGE_KEY_FILE, no `path_exists`, cross-OS
  defaults; starter template untouched) → `tests/env-contract.bats` (6 new tests, incl.
  "no age key var carries path_exists validation" + "dotf init starter template does not declare
  age key vars") **PASS**; Go `TestRealContractRendersAgeKeyDiscovery` renders both vars from the
  *real* contract into `paths.{sh,ps1}` for linux + windows **PASS**.
- [x] **AC2** (present + valid key → PASS naming the round-trip) →
  `TestSecretsTooling_AllPresent` (now includes `age-keygen`, asserts
  "age root-of-trust verified (round-trip)") **PASS**.
- [x] **AC3** (present but failing round-trip → FAIL with path + cause) →
  `TestSecretsTooling_RoundTripFailIsFail` (1 failure; message contains "round-trip FAILED",
  the cause, "restore a good key") **PASS**.
- [x] **AC4** (absent key → WARN, round-trip skipped) →
  `TestSecretsTooling_KeyMissingSkipsRoundTrip` (seam spy never called, WARN present) +
  `TestSecretsTooling_AgeKeygenMissingWarnsAndSkips` (age present, age-keygen absent → WARN,
  no round-trip, 0 failures) **PASS**.
- [x] **AC5** (verifier wired from secrets seams; no age exec in doctor) →
  `grep 'exec.Command("age'` in `internal/doctor/` returns **nothing**; the round-trip is
  unit-tested via the injected `AgeRoundTrip` fake (no real age/key in CI).
- [x] **AC6** (build/vet/gofmt clean; tests green; production I/O by smoke) →
  `go vet ./...` clean, `go build ./...` clean, all touched `.go` files gofmt-clean in their
  committed LF form; `go test ./...` green except one **pre-existing unrelated** failure (below).

## Test status

- `go test ./...` (from `cli/`): all packages **ok** except `internal/spec`
  `TestEmbeddedTemplatesMatchVault` — **pre-existing, unrelated** to this slice (vendored
  `cli/internal/spec/templates/tasks.md` drifted from vault `spec-tasks.md`; fails identically on
  a clean `main` checkout; flagged in the prior session handoff for its own re-vendor). Not
  touched here (atomic-PR: re-vendoring the spec template is a separate change).
- `bats tests/env-contract.bats`: **19/19 ok** (13 prior + 6 new age-key tests).
- `internal/doctor` secrets tests: **8/8 ok** (`go test ./internal/doctor/ -run SecretsTooling`).
- Windows-worktree note: `gofmt -l .` locally lists CRLF working-tree files (git `autocrlf`);
  the committed form is LF (`git ls-files --eol` → `i/lf`; `.gitattributes` `* text=auto`), so
  CI's gofmt (Linux, LF) is clean. Verified per-file with `tr -d '\r' | gofmt -d` → empty.
- Live smoke (production age I/O — thin, not in CI): **pending on a box with a real key** —
  `dotf doctor` shows "age root-of-trust verified (round-trip)"; temporarily corrupting/renaming
  `~/.config/age/key.txt` flips it to the FAIL. To be captured before archive.

## Decisions made during implementation

- **SOPS retained, not cruft (resolved by the codebase).** SOPS is a deliberately-installed
  general tool (CLI-029, `docs/lessons.md`) consuming the *same* age key; declaring
  `SOPS_AGE_KEY_FILE` alongside `AGE_KEY_PATH` (both → the deployed key) is correct and closes the
  literal #518 bug. No AskUser needed — the code answered it.
- **Scope correction mid-work:** dropped the `dotf init` starter template from AC1 — its
  `env-contract.json` is a generic empty seed for *any* onboarded repo, so an age key path (a
  dotfiles machine fact) must not leak into it. Added a guard test asserting the template stays
  age-var-free.
- **Round-trip proves the operator's key, not a fixture.** Self-encrypt→decrypt a fixed sentinel
  with the deployed key beats #518's literal "known test ciphertext": a committed fixture would
  only test a test key. No committed fixture; sentinel is a non-secret in-code constant.
- **age-keygen gate → WARN, not FAIL.** age present but age-keygen absent can't derive the
  recipient → we can't verify (the key may be fine), so WARN and skip; only a real decrypt
  mismatch/error with both binaries present is a FAIL.
- **Reused the file-oriented `AgeDecrypt` seam** via a temp ciphertext file (0600, removed on all
  paths) rather than adding a symmetric in-memory decrypt — DRY over the shipped seams; the
  encrypt→bytes / decrypt→file asymmetry dedup is noted out-of-scope.

## Promotion candidates

- [ ] Lesson for the repo's `docs/lessons.md`? **maybe** — "a doctor health-check must test
  *function*, not *presence*: a root-of-trust with only an existence check is untested until the
  disaster" (the #518 → round-trip guard pattern). Small, worth capturing on merge.
- [ ] ADR-worthy decision? **no** — implements existing ADR-028 (§4/§5), no new architecture.
- [ ] New cross-project pattern? **no** — repo-specific; the incident→guard rule already lives in
  memory/patterns.

## Deferred / tracked (not folded in — atomic-PR)

- Prior session's deferred lesson (escrow `--help`/`Long` strings are untested literals → smoke
  the real `--help`) — the handoff parked it for "the next slice PR". Kept out of this diff to
  hold one logical change; to be captured in `docs/lessons.md` (flag to user).
- `internal/spec` template re-vendor (pre-existing failure above) — its own change.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-024-secrets-age-discovery/` -> `specs/archive/CLI-024-secrets-age-discovery/`
- [ ] Bitácora board ticket (#518) moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
