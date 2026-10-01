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

## Archive pass (2026-09-30)

Implementation merged in #1256 (`d7e5ddcc`). Re-verified on `main` at archive
time: every test `features.json` names exists (one `func` each, checked by name,
since `go test -run` with a name that matches nothing still exits 0), and they
pass:

```
$ cd cli && go test ./internal/harness/ ./internal/doctor/ -run 'ModelPins|EveryRepo|GuardRejects|Catalog|ExtractRefuses'
ok      github.com/mlorentedev/dotfiles/cli/internal/harness
ok      github.com/mlorentedev/dotfiles/cli/internal/doctor
```

The proposal's acceptance boxes were never ticked when #1256 merged; this pass
ticks them against the evidence table above. Generation for the
pipeline-owned surfaces stays open on #902 as phase 2, so this archive does
not close that issue.


## Round-1 review fixes (#902, 2026-10-01)

Round 1 (`nan/deepseek-v4-flash`, sha `f1c2b18`) returned FAIL; its files are
kept byte-identical as `review-round-1.md` and `review-request-round-1.json`.
The owner chose path (a) on #902: make "every routing pin" true. Code, tests and
registry changes landed in `21c3a0f0`; this section and the contract edits are
the commit after it.

### Disposition of the round-1 findings

| # | Finding | Disposition | Proof |
|---|---|---|---|
| 1 | agy, claude and copilot settings carry undeclared pins | Applied: four pins declared (`agy-model`, `claude-model`, `claude-advisor-model`, `copilot-model`). agy's `modelConfigs.defaultModel` is **excluded**, not declared: the agy binary carries no `modelConfigs` string, its log resolves the top-level `model`, and `gemini-3.7-flash` is not an id `agy models` lists | `TestEveryRoutingKeyUnderAIIsDeclared` |
| 2 | Nested pins and `fallback_models` undeclared; a dead id passed | Applied: `opencode-small-model`, `opencode-provider-model`, `opencode-agent-models` (`agent.*.model`), `pr-agent-fallbacks`; extraction reaches dotted paths, wildcards and TOML lists | `TestEveryModelKeyInATOMLSiteIsDeclared`, `TestEveryDeclaredPinRejectsADeadID`, `TestExtractReachesNestedAndArrayPins` |
| 3 | `cli.yml` skips the Go guard on data-only changes | Out of this spec: **#1888** (P1), wider than model pins | — |
| 4 | `Normalize` trims an absent prefix | Applied: a declared prefix or suffix is required (`VerdictMisspelled`), and the repo guard fails on it and on any verdict it does not handle | `TestNormalizeRequiresTheDeclaredPrefix`, `TestDeclaredSuffixIsSpellingNotPartOfTheID`, and the reviewer's own mutation through `TestEveryRepoRoutingPinResolvesInTheMap` (below) |
| 5 | `pi-default-model`'s `why` describes `qwen3.6` | Applied: rewritten from the file's history (`qwen3.6` until #1255, `qwen3.8-flash` until #1471) | data |
| 6 | doctor prints WARN, then "all resolve" | Applied: a rotted deployed locator FAILs and counts as a finding | `TestModelPinsFailsOnARottedLocatorAndClaimsNothingClean` |
| 7 | AC1 says "schema-validated" | Applied: AC1 reworded, no schema file (owner decision) | `proposal.md` |
| 8 | `retiredProvider` hardcodes two providers | Declined as a gate (round 1's own disposition) | — |

### Spelling decided before declaring (the precondition for finding 1)

- **`opus[1m]`**: a declared `suffix: "[1m]"`, required like the prefix. The map
  routes the `opus` alias; `[1m]` is Claude Code's 1M-context spelling of it.
- **`Gemini 3.7 Flash (High)`**: a declared `spelling: display-name`. Measured
  2026-10-01 with `agy models`: every gemini row maps label to id by lowercasing,
  dropping the parentheses and joining with `-` (`Gemini 3.1 Pro (High)` is
  `gemini-3.1-pro-high`). The rule does not hold for every row
  (`Claude Sonnet 4.6 (Thinking)` is `claude-sonnet-4-6`), which is why a pin
  opts in rather than the guard applying it.

### Two routing additions the pins needed

Both values were already in use; the map did not route them.

- `tiers.mid.gemini = gemini-3.7-flash-high`, agy's interactive default.
- `services.advisor = {pool: claude, model: fable}`, Claude Code's
  `advisorModel`. No tier describes a role pinned to one model, and a new tier
  would need a chain; the schema's `services` description now names it.

No chain changed, so `dotf agent run` dispatch is untouched.

### Catalog is declared, not inferred

`fallback_models` is a list and a routing decision, so "a `[]` locator is a
catalog" stopped holding. `pi-deployed-enabled-models` now carries
`catalog: true`, the loader rejects `catalog` on a scalar locator, and doctor
reads the field instead of the locator's suffix.

A side effect, covered by the existing tests: a deployed catalog entry such as
`openrouter/minimax/minimax-m3` is now `VerdictMisspelled` (it lacks `nan/`)
rather than `VerdictUnknown`. Doctor names a retired provider before it reports
spelling, so `TestModelPinsDistinguishesARetiredProvider` still reads "retired".

### Not declared, recorded

The deployed copies of the agy, claude and copilot settings stay out. The live
`~/.gemini/antigravity-cli/settings.json` carries `Gemini 3.8 Flash (High)`,
which agy's picker wrote and the map does not route. Declaring it would fail
doctor on a choice made in the UI, and that is a routing decision for the
owner, not a registry edit.

### Evidence

RED before the fix (the sweeps reproduce findings 1 and 2 by name; finding 6's
contradiction verbatim):

```
--- FAIL: TestEveryRoutingKeyUnderAIIsDeclared
    ai/agy/settings.json: "model" is a routing-shaped key that no pin in harness/model-pins.json locates.
    ai/agy/settings.json: "modelConfigs.defaultModel" is a routing-shaped key ...
    ai/claude/settings.json: "advisorModel" ...      ai/claude/settings.json: "model" ...
    ai/copilot/settings.json: "model" ...            ai/opencode/opencode.jsonc: "agent.plan.model" ...
    ai/opencode/opencode.jsonc: "provider.nan.options.model" ...   ai/opencode/opencode.jsonc: "small_model" ...
--- FAIL: TestEveryModelKeyInATOMLSiteIsDeclared
    .pr_agent.toml: "fallback_models" carries a model id that no pin locates
--- FAIL: TestNormalizeRequiresTheDeclaredPrefix
    "mimo-v2.6-flash" dropped the declared prefix "openai/" and came back 0, want VerdictMisspelled
--- FAIL: TestModelPinsFailsOnARottedLocatorAndClaimsNothingClean
  [WARN] $HOME/.pi/agent/settings.json: pin "pi-deployed-default-model": key "defaultModel" not present — the file changed shape
  [ OK ] 1 deployed routing pins across 1 files all resolve in the map
```

Finding 4 re-run the way the reviewer ran it: `.pr_agent.toml`'s
`model = "openai/mimo-v2.6-flash"` edited to `model = "mimo-v2.6-flash"`, then
restored with `git checkout`. A first cut taught `Check` the new verdict but
left the guard's switch without a case for it, and the mutation still passed:

```
--- PASS: TestEveryRepoRoutingPinResolvesInTheMap
    resolved 18 routing pins across 9 repo files
```

With the case added, and a `default` that fails on any verdict the switch does
not know:

```
--- FAIL: TestEveryRepoRoutingPinResolvesInTheMap
    .pr_agent.toml pin "pr-agent-primary": "mimo-v2.6-flash" is not spelled "openai/"…"", the form this site declares
```

The map additions are load-bearing. With `harness/model-map.json` reverted to
`HEAD~1` and everything else kept:

```
--- FAIL: TestEveryRepoRoutingPinResolvesInTheMap
    ai/claude/settings.json pin "claude-advisor-model": "fable" normalizes to "claude:fable", which harness/model-map.json does not declare
    ai/agy/settings.json pin "agy-model": "Gemini 3.7 Flash (High)" normalizes to "gemini:gemini-3.7-flash-high", which harness/model-map.json does not declare
```

GREEN at `21c3a0f0`:

```
$ cd cli && go test ./internal/harness/ ./internal/doctor/ -run '<the nine tests above>' -v -count=1
--- PASS: TestEveryRoutingKeyUnderAIIsDeclared
--- PASS: TestEveryModelKeyInATOMLSiteIsDeclared
--- PASS: TestEveryDeclaredPinRejectsADeadID        (19 subtests, one per declared pin)
    resolved 18 routing pins across 9 repo files
--- PASS: TestEveryRepoRoutingPinResolvesInTheMap
--- PASS: TestNormalizeRequiresTheDeclaredPrefix
--- PASS: TestDeclaredSuffixIsSpellingNotPartOfTheID
--- PASS: TestDisplayNameSpellingMatchesTheAgyCatalog
--- PASS: TestExtractReachesNestedAndArrayPins
--- PASS: TestModelPinsFailsOnARottedLocatorAndClaimsNothingClean
ok  github.com/mlorentedev/dotfiles/cli/internal/harness
ok  github.com/mlorentedev/dotfiles/cli/internal/doctor

$ go build ./... && go vet ./...            # exit 0
$ go test ./...                             # exit 0, 27 packages ok
$ GOOS=windows go vet ./...                 # exit 0
$ golangci-lint run                         # 2.12.2 (the pin), 0 issues
$ bats tests/model-map.bats tests/triggers-registry.bats \
       tests/compile-harness-real.bats tests/compile-harness.bats   # exit 0, 105 ok
```

Round 1 measured 10 pins across 6 repo files; there are now 18 across 9.

## Round-2 review (2026-10-01)

Verdict PASS-WITH-GAPS (`nan/glm5.3-flash`, reviewed `7a018fca`). This is the last round under the two-round cap. No Blocker, and no REAL Major.

| Finding | Disposition |
|---|---|
| Major, THEORETICAL: `toml-key` takes the first match file-wide | Ticketed: #1924. Today's `.pr_agent.toml` has one `model` key. |
| Minor, THEORETICAL: an unreadable deployed site ends as a Skip | Ticketed: folded into #1924. |
| Minor, SPECULATIVE: `staleSnapshotOf` numeric-tail false positive | Declined. The WARN names the base id it assumed, so a misclassification shows. No such id exists in the map. |
| Question: AC5/AC6 not re-run live | Accepted. The named unit tests pass, and the live findings were recorded before round 1. |
