---
tags: [spec, verification, templates]
created: "2026-08-29"
---

# Verification - CLI-065-env-persist-sweep

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (marker written once, second run inert) -> commit `a1878e0` / `TestPersist_WritesTheMarkerOnce`, `TestPersist_TouchesOnlyWhatDiffers` (now expects the marker as the third write), `TestMarkerValue`
- [x] AC2 (only the retired name goes, a foreign name never) -> `a1878e0` / `TestPersist_SweepsOnlyWhatTheMarkerOwns`, `TestPersist_DeletesBeforeItWrites`, `TestLeftovers` (8 cases)
- [x] AC3 (`--check` names `retired: NAME`, non-zero; `persist` prints `removed NAME`) -> `a1878e0` / `TestEnvPersist_CheckAndSweepOfARetiredName` (cmd), `TestRetired` (env)
- [x] AC4 (doctor WARNs on a leftover with the remedy, PASS once swept) -> `a1878e0` / `TestCheckPersistedEnv_ByStatus` cases "retired name still persisted → WARN naming it", "marker in sync → PASS"
- [x] AC5 (no marker → nothing deleted, marker written; Delete of an absent name succeeds; off Windows no-op) -> `a1878e0` / `TestPersist_NoMarkerDeletesNothing`, `TestFakeUserEnv_DeleteAbsentSucceeds`, `TestEnvPersist_UnsupportedScopeIsANoOp`; `registryUserEnv.Delete` maps `registry.ErrNotExist` to nil; `GOOS=windows` and `GOOS=linux` vet clean
- [x] AC6 (box) -> transcript below, Windows work box, 2026-08-29, binary built from `a1878e0`'s tree

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

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [x] Lesson for the repo's `docs/lessons/`? yes — lesson 244 (written in this PR): a sweep is bounded by the writer's record, and the store's name rules decide the comparison and the order
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no — an application of ADR-025's contract, not a new decision
- [ ] New pattern candidate for `00_meta/patterns/`? not yet — third instance of "the writer touches only what it owns" in this repo (settings.json hooks, scripts dir, user env); promote when it recurs in another project

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-065-env-persist-sweep/` -> `specs/archive/CLI-065-env-persist-sweep/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
