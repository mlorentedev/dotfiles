---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - HARNESS-067-model-pin-drift

## Evidence

| AC | Proof |
|---|---|
| AC1 registry well-formed, justified, unique ids, locators compile | `go test ./internal/harness/ -run TestModelPinsRegistryIsWellFormed` |
| AC2 every repo routing pin resolves in the map | `go test ./internal/harness/ -run TestEveryRepoRoutingPinResolvesInTheMap` — `resolved 10 routing pins across 6 repo files`. Reads the real repository, not a fixture |
| AC3 the guard fails on a bad pin | `go test ./internal/harness/ -run TestGuardRejectsAnUnresolvablePin` — the two live bad values (`nan/deepseek-v4-flash-0731`, `openrouter/deepseek/deepseek-v4-pro`) are `VerdictUnknown`, and a live id is still `VerdictOK` so it is not simply failing everything |
| AC4 a catalog entry absent from the map is not drift | `TestCatalogEntriesAreNotCheckedAsRouting` (repo half) + `TestModelPinsDoesNotReportAnUnroutedCatalogModel` (deployed half, asserting `gemma4`, `qwen3.8-flash` and `glm5.3-flash` are all silent) |
| AC5 doctor reports a deployed pin that no longer resolves | Live run: `[WARN] $HOME/.pi/agent/settings.json pi-deployed-enabled-models: "nan/deepseek-v4-flash-0731" is a frozen snapshot of "deepseek-v4-flash", and no longer resolves`. Unit: `TestModelPinsReportsAFrozenSnapshot` |
| AC6 a retired provider reads distinctly | Live run: three `[WARN] … names "openrouter", a provider this repository retired`. Unit: `TestModelPinsDistinguishesARetiredProvider`, which also asserts it is **not** misreported as a snapshot |
| AC7 an unreadable registry fails loudly | `TestModelPinsRefusesToReadAsEmpty` (8 broken shapes) + `TestModelPinsFailsLoudlyWithoutARegistry`, which additionally bans the strings `[ OK ]` and `all resolve` from the output |
| AC8 the check performs no writes | `TestModelPinsNeverWrites` — content **and** mtime compared before/after a run that produced findings |

## Test status

```
$ cd cli && go test ./internal/harness/ -run 'ModelPins|EveryRepo|GuardRejects|Catalog|Extract|DeclaredModels'
ok      7 tests, resolved 10 routing pins across 6 repo files

$ go test ./internal/doctor/ -run TestModelPins
ok      7 tests

$ go test ./...
18/18 packages ok

$ go build ./... && go vet ./... && GOOS=windows go vet ./...
OK (linux + windows)

$ golangci-lint run          # pinned 2.12.2, matching versions.conf
0 issues.
```

### The live machine, before any repair

```
$ DOTFILES_DIR=<checkout> dotf doctor
[Model pin drift]
  [WARN] $HOME/.pi/agent/settings.json pi-deployed-enabled-models:
         "nan/deepseek-v4-flash-0731" is a frozen snapshot of "deepseek-v4-flash",
         and no longer resolves
  [WARN] … "openrouter/deepseek/deepseek-v4-pro" names "openrouter", a provider
         this repository retired
  [WARN] … "openrouter/qwen/qwen3-coder-plus"   (same)
  [WARN] … "openrouter/minimax/minimax-m3"      (same)
```

Four findings, all real, none repaired — the check does not write. `nan/gemma4`,
present in the same array, is correctly silent.

## Decisions made during implementation

- **The first version of this check was wrong, and a real run caught it.** It
  reported every catalog id the map does not route, which fired on `nan/gemma4` —
  a live NaN model nobody routes, exactly as `qwen3.8-flash` and `glm5.3-flash`
  are (#1244). That is precisely the failure `harness/model-pins.json`'s own
  `$comment` warns against, written into the check by the same session that wrote
  the warning. The fix is a mechanical distinction rather than a matter of taste:
  a **frozen snapshot** is a declared id plus a date stamp (`deepseek-v4-flash` +
  `-0731`), so it pins a moment of a model that is still alive under its rolling
  name. `gemma4` bears no such relation to anything. Only that shape and a
  retired provider are reported. Regression-guarded by
  `TestModelPinsDoesNotReportAnUnroutedCatalogModel`.
- **Severity follows consequence.** A dead **scalar** routing pin (`defaultModel`)
  decides what a real session runs on, so it FAILs. A dead **catalog** entry costs
  a startup warning and a stale picker row, so it WARNs. Reporting both alike
  would either cry wolf about a picker or under-report a broken default.
- **`DeclaredModels` returns two sets, not one.** The map's `$comment` records
  that tiers are keyed by whatever *consumes* the id — `claude` and `opencode`
  key by harness there, `nan` by pool. Collapsing that into one pool-qualified
  set would invent attributions the map never made and report them as drift. So
  `chains` (unambiguously `pool:id`) populates the qualified set, everything
  contributes to the bare set, and a tier key is qualified only when it is itself
  a declared pool. Asserted by
  `TestDeclaredModelsDoesNotInventPoolsFromHarnessKeys`.
- **A locator that matches nothing is an error, never an empty result.** "Found
  no pins" and "found no drift" are different facts that look identical
  downstream. `Extract` errors on zero matches, `kind: regex` errors when it
  matches more than once rather than silently taking the first, and the doctor
  check FAILs when it read sites but extracted nothing.
- **The PowerShell wrappers are one `regex-all` pin, not two clever ones.** The
  first draft used two multi-line patterns to tell `qq` from `qf`. A pattern
  clever enough to do that across a PowerShell function is a pattern that
  silently matches nothing after an unrelated edit. Both wrappers use
  `opencode run -m <model>`, so every match is checked and zero matches is an
  error.
- **No `--fix`, and the check is not even given the flag.** Repairing means a
  surgical merge into a file pi rewrites at runtime. That is the same disposition
  question the hand-wired extension symlinks raised in #1243, and it gets asked
  rather than defaulted.
- **`expandHome` was reused, not rewritten.** The registry writes `$HOME/...`
  rather than `~/...` so the existing `fs.go` helper resolves it — a second
  path-expansion function in the same package is the drift this spec is about,
  one layer down.

## Promotion candidates

- **Nothing for the vault yet.** The pin sites, the litellm `openai/` prefix and
  the seed-if-missing constraint are all specific to this repository's
  deployment.
- The one genuinely cross-project candidate — *"a guard that detects `not
  declared` fires on every legitimate extension; detect the shape that is
  **wrong**, not the shape that is merely **absent**"* — is a strong pattern and
  should wait for a second instance outside this repo before promotion.
