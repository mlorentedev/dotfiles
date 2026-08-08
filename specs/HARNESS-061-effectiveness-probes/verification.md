---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - HARNESS-061-effectiveness-probes

## Evidence

- [x] AC1 (effective config, repo named, remedy stated) -> `TestGuardHooks_LocalOverrideDisablesGuard_MustFail`, green direction `TestGuardHooks_NoOverrideGuardFires_MustPass`
- [x] AC2 (gate via dispatcher fallback) -> `TestVaultHooks_GateViaDispatcherFallback_MustPass`; negative `TestVaultHooks_DispatcherWithoutConfig_IsNotAGate`
- [x] AC3 (non-executable resolves absent) -> `TestHookForStage_NonExecutableIsNotAHook`
- [x] AC4 (new stub suite fails) -> `tests/stub-real-pairing.bats` case 1; red-teamed by adding a throwaway stub suite, which the guard named before it was removed
- [x] AC5 (exemptions cannot go stale) -> `tests/stub-real-pairing.bats` case 2, plus case 3 guarding the detector against passing vacuously
- [x] AC6 (DR escrow + drill) -> `TestDR_NoDrillRecorded_Warns`, `TestDR_RecentDrillPasses`, `TestDR_StaleDrillWarns`, `TestDR_AbsentEscrowSkipsNotFails`, `TestDR_StaleEscrowWarns`
- [x] AC7 (red direction everywhere) -> each case above has its inverse; the probe helpers additionally carry `TestEffectiveHooksPath_NilSeamIsUnsetNotPanic`, added after a nil command seam panicked the whole doctor suite on first run

## Test status

- Test suite: `go test ./...` -> 12 packages ok, 0 failures
- Test suite: `bats tests/*.bats` -> 1 failure, `install-dotf.bats` "converges over a running dotf", pre-existing on `main` and tracked as #807 (BUG-054)
- Lint: `shellcheck scripts/*.sh setup-linux.sh` -> clean
- Manual smoke: the two live defects this replaces were both reproduced on this machine before the change and confirmed correctly reported after — the guard's inertness under a repo-local override, and the vault gate running through the dispatcher fallback while the old check called it INACTIVE
- No regressions: yes. Two existing fixtures needed updating rather than the code — `vaultTree` now writes hooks executable (a non-executable hook is one git ignores, so the old fixture modelled a gate that does not fire), and the git fake now reports `config --get` the way git does, effective rather than local-only
