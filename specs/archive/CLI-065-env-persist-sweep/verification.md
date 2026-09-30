---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - CLI-065-env-persist-sweep

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (marker written once, second run inert) -> commit `8de1cd1` / `TestPersist_WritesTheMarkerOnce`, `TestPersist_TouchesOnlyWhatDiffers` (now expects the marker as the third write), `TestMarkerValue`
- [x] AC2 (only the retired name goes, a foreign name never) -> `8de1cd1` / `TestPersist_SweepsOnlyWhatTheMarkerOwns`, `TestPersist_DeletesBeforeItWrites`, `TestLeftovers` (8 cases)
- [x] AC3 (`--check` names `retired: NAME`, non-zero; `persist` prints `removed NAME`) -> `8de1cd1` / `TestEnvPersist_CheckAndSweepOfARetiredName` (cmd), `TestRetired` (env)
- [x] AC4 (doctor WARNs on a leftover with the remedy, PASS once swept) -> `8de1cd1` / `TestCheckPersistedEnv_ByStatus` cases "retired name still persisted → WARN naming it", "marker in sync → PASS"
- [x] AC5 (no marker → nothing deleted, marker written; Delete of an absent name succeeds; off Windows no-op) -> `8de1cd1` / `TestPersist_NoMarkerDeletesNothing`, `TestFakeUserEnv_DeleteAbsentSucceeds`, `TestEnvPersist_UnsupportedScopeIsANoOp`; `registryUserEnv.Delete` maps `registry.ErrNotExist` to nil; `GOOS=windows` and `GOOS=linux` vet clean
- [x] AC6 (box) -> transcript below, Windows work box, 2026-08-29, binary built from `8de1cd1`'s tree

## Test status

- Test suite: `cd cli && go test ./... -count=1` -> every package `ok`, `FAIL_COUNT=0` (run on the box while an adversarial review was also running, ~9 min)
- `go vet ./...` and `GOOS=windows go vet ./...` clean; `golangci-lint run ./...` (pinned 2.12.2, matches `versions.conf`) -> `0 issues`
- Manual smoke test (AC6), binary `dotf-sweep.exe` built from this branch, `DOTFILES_REPO_DIR` switched between the real checkout and a scratch copy of the contract minus `SCRIPTS_DIR`:

  ```text
  --- 1. real contract, first run under the new binary ---
  persisted DOTF_MANAGED_ENV
  user scope: 1 changed, 11 unchanged, 0 removed
  marker: AGE_KEY_PATH;AGY_HOME;CLAUDE_CONFIG_DIR;COPILOT_HOME;DOTFILES_DIR;DOTFILES_REPO_DIR;HIVE_VAULT_PATH;OPENCODE_HOME;SCRIPTS_DIR;SOPS_AGE_KEY_FILE;VAULT_PATH
  --- 2. scratch contract minus SCRIPTS_DIR ---
  check before sweep:
  retired: SCRIPTS_DIR
  Error: 1 retired name(s) still persisted at user scope — run `dotf env persist` to sweep them
  sweep:
  removed SCRIPTS_DIR (retired from the contract)
  persisted DOTF_MANAGED_ENV
  user scope: 1 changed, 10 unchanged, 1 removed
  SCRIPTS_DIR after sweep: <absent>          (Get-ItemProperty HKCU:\Environment)
  --- 3. second run, scratch contract ---
  user scope: 0 changed, 11 unchanged, 0 removed
  --- 4. real contract again (restore) ---
  persisted SCRIPTS_DIR
  persisted DOTF_MANAGED_ENV
  user scope: 2 changed, 10 unchanged, 0 removed
  SCRIPTS_DIR restored: C:\Users\<user>\.dotfiles\scripts
  ```

  The scratch contract is a byte-copy of the real one minus the one entry, so
  nothing but that name differs between runs; the real contract is re-run last
  so the box leaves the test as it entered it.
- No regressions in existing test suite: yes

## Decisions made during implementation

- **Sweep before write, and compare under the registry's rules.** Registry value names are case-insensitive; a case-only rename (`Foo` → `FOO`) is one value to the store. Compared exactly, the old spelling is a leftover and the new one a write, and write-then-delete would delete what the run had just written. Both guards are in: `Leftovers` uses `strings.EqualFold` (so it is not a leftover at all) and every delete precedes every write (pinned by `TestPersist_DeletesBeforeItWrites` on the fake's operation log). Lesson 244.
- **`Delete` of an absent name succeeds**, declared on the interface, mapped from `registry.ErrNotExist` in the store and mirrored by the fake, with a test on the fake — the one behaviour where the fake and the box could disagree and AC5 would pass locally and fail on a real second run.
- **Reader/store split.** `Drift` and `Retired` take `UserEnvReader`; the doctor adapter loses the `Set` no-op it carried only to satisfy an interface wider than its caller.
- **The marker is reported as a result line** (`persisted DOTF_MANAGED_ENV` on the run that writes it, an "unchanged" on the others) rather than hidden: a visible store value should have a visible write.
- **No marker → no sweep**, stated in the proposal as out of scope with the WIN-013 contrast, not silently.

## Review round 1 (FAIL, 2026-09-30) — dispositions

`review.md` round 1 (`nan/deepseek-v4-flash`) failed on one Major. Every finding is dispositioned here:

- **Major, reserved name: applied.** `ValidateNames` (was `checkNames`) now refuses a contract variable spelled like `DOTF_MANAGED_ENV` in any case. Otherwise the marker would overwrite it on every run. Tests: `TestPersist_RefusesTheMarkerNameAsAVariable` (no store op).
- **Minor, `--check` does not validate names: applied.** `checkPersisted` and doctor's `checkPersistedEnv` call `ValidateNames` first. A check can no longer pass on a contract, or name a remedy, that `persist` then refuses. Tests: `TestEnvPersist_CheckRefusesTheMarkerNameAsPersistDoes` (persist and `--check` both refuse; the store is untouched), and doctor case "reserved contract name → WARN naming the refusal".
- **Minor, provenance: applied.** `a1878e0` was never an object in this repository, so it was replaced with `8de1cd1` (#1378).
- **Minor, combined drift + retired doctor case: applied.** Two rows were added to `TestCheckPersistedEnv_ByStatus`, one naming the drifted variable and one naming the retired one.
- **Minor THEORETICAL, `--check` is not a snapshot: declined.** Each box has one writer: `persist` runs from setup, by hand, or from doctor's remedy, never concurrently with itself. `--check` is advisory, and `persist` is idempotent: a check that raced a run is corrected by the next check. A lock around a per-user registry key would cost more than the state it protects.

## Review round 2 (FAIL, 2026-09-30) — dispositions

`review-round-2.md` (`nan/mimo-v2.6-flash`) confirmed every round-1 fix. It then failed on two new Majors from the same class, a contract name the marker cannot record:

- **Major, an empty name: applied.** `Persist` wrote it, but `MarkerValue` skips empty names, so nothing could ever sweep the value. `ValidateNames` now refuses it.
- **Major, a whitespace-padded name: applied.** `MarkerValue` recorded it raw and `ParseMarker` trimmed it. On retirement the sweep looked up the trimmed name, missed the value, and dropped the name from the record, so `--check` reported clean. `ValidateNames` now refuses any name that differs from its trimmed form. Test: `TestPersist_RefusesANameTheMarkerCannotRoundTrip` (`""`, `" "`, `" FOO"`, `"FOO "`, `"\tFOO"`, with no store operation before the refusal).
- **Minor THEORETICAL, validation lives at three call sites rather than inside the readers: declined.** All three production readers call `ValidateNames`, and each call is covered by a test that fails without it. Moving the check into `Drift`, `Retired` and `MarkerStale` would run it three times per check to protect a hypothetical fourth reader.

## Review round 3 (PASS, `nan/mimo-v2.6-flash`, 2026-09-30) — dispositions

| # | Finding | Disposition |
|---|---|---|
| F1 | Minor (theoretical): the dedup key (`strings.ToUpper`) and the equality (`strings.EqualFold`) disagree for a non-ASCII pair such as `I` and `ı` | decline: contract names are environment variable names, ASCII identifiers in every shipped contract, and no plausible typo produces the pair. Aligning the fold is a code change after a passing review, for an unreachable input |
| F2 | Minor: the Setup row in `tasks.md` names `feat/env-persist-sweep` | no action: `tasks.md` is in the contract set, and the reviewer advised against staling the verdict for a cosmetic note. The fixes landed on `fix/env-persist-reserved-marker` (#1862) |
| F3 | Minor (theoretical): the read-path tests feed `ValidateNames` only the reserved name, not a padded or empty one | decline: the guard is one function, covered for every refused shape by `TestPersist_RefusesANameTheMarkerCannotRoundTrip`, and the reviewer's mutations 2 and 3 show each read path's call is pinned |

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-244-a-sweep-is-bounded-by-what-the-writer-recorded-not-by-what-the-store-holds.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: an application of ADR-025's contract, not a new decision
- [x] New pattern candidate for `00_meta/patterns/`? no: third instance of "the writer touches only what it owns" in this repo only; promote when it recurs in another project

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-065-env-persist-sweep/` -> `specs/archive/CLI-065-env-persist-sweep/`
- [x] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018): #1862 carries `Closes #1363`
- [x] Promotions above executed (if any): lesson 244
