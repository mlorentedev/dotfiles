# Lesson 240 — A partial mirror the code believes absent: the check that could have flagged the gap was the one switched off

**Date:** 2026-08-27
**Context:** WIN-007 (#1288) — `dotf doctor` FAILs `harness/model-map.json` and `harness/model-pins.json` after every Windows setup, remedy "re-run setup to mirror it".
**Category:** windows, deploy mirror, doctor, false beliefs

## What happened

`~/.dotfiles` on the Windows box held exactly seven artifacts — `versions.conf`,
`packages.json`, `git-hooks/`, `sensitive/`, `secrets/registry.yaml`,
`env-contract.json`, `paths.ps1` — and no `harness/`. `setup-linux.sh` mirrors
`harness/` with a 60-line bash+jq block; `setup-windows.ps1` had no such block,
so the doctor's remedy could never clear.

The Go doctor made it worse by believing it could not happen:

```go
// Windows has no repo/mirror split — setup-windows.ps1 sets
// `$DotfilesDir = $PSScriptRoot`, i.e. the deploy dir IS the checkout
func checkHarnessMirrorOrphans(...) {
    if sys.GOOS == "windows" { return }
```

The comment was true of a *variable name*: on Windows `$DotfilesDir` is the
checkout — Linux uses `DOTFILES_DIR` for the deploy dir, the two scripts invert
the name. The deploy dir on Windows is `$DotfilesDest = ~\.dotfiles`, the very
path `cfg.DotfilesDir` resolves to. A partial mirror, believed absent, with the
one check that walks the mirror switched off for that OS — and a test locking
the switch-off in ("windows has no separate mirror -> no-op").

The pi-packages count read the same missing mirror and reported "0 packages
declared" as a PASS. Green tag, wrong number.

## The lesson

Never encode a belief about an environment as an early return without a test
that fails when the belief is false. The `GOOS == "windows"` branch had a test —
asserting the belief, not the world.

Two shell implementations of one mirror will diverge, and the divergence will
be discovered by the check that reads the result, printing a remedy the
divergent side cannot execute. One implementation (`dotf harness mirror`,
called by both setups) removes the class; a remedy a check prints must be one
the check can verify clears.

The gap had a cost beyond the two FAILs: with the mirror in place the model-pin
check ran on Windows for the first time and found a real drift in the live pi
settings — a finding the switched-off path had been hiding along with the
false ones.

## Guard

`TestCheckHarnessMirrorOrphans` now asserts Windows reports an orphan like any
OS; `TestPiPackagesManifest_ReadsTheCheckoutBeforeTheMirror`; the mirror's own
tests (idempotent, no prune, missing target named); and the CI doctor gate
(lesson 236) that finally runs the check where it can fail.
