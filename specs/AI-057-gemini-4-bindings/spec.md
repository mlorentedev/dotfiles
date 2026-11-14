---
id: spec-AI-057-gemini-4-bindings
type: spec
status: implementing
created: "2026-11-14"
owner: manu
issue: "mlorentedev/dotfiles#3471"
---
# AI-057: route the Gemini bindings to Gemini 4 Argon and Hydrogen

## Intent

Every Gemini binding in this repository named `gemini-3.8-flash-high`: both Gemini tiers of `harness/model-map.json`, the agy member of `harness/reviewer-pool.json`, the `agy-model` pin and the agy seed `model`. The recorded reason was that agy served no Pro tier since 2026-10, so top had to name the mid id "until Gemini ships a stronger one" (#2385, #902), and the pool entry called itself PROVISIONAL until that happened. It has: Google ships the Gemini 4 family under element names, `gemini-4-argon` (the Pro slot) and `gemini-4-hydrogen` (the Flash successor), and `agy models` on agy 1.5.8 (2026-11-14) serves `gemini-4-argon-{low,medium,high}` and `gemini-4-hydrogen-{low,medium,high,xhigh}`. Both candidates passed the pool's admission bar, the planted `((count++))` under `set -euo pipefail` with `count=0`: Argon High in 50 s, Hydrogen High in 66 s, each naming the failing input and the fix (recorded on #3471).

## Scope

- `harness/model-map.json`: `tiers.top.gemini` is `gemini-4-argon-high`, `tiers.mid.gemini` is `gemini-4-hydrogen-high`; the two tiers are distinct again and the comments say what was measured and when.
- `harness/reviewer-pool.json` and the byte-identical `cli/internal/prreview/reviewer-pool.json`: the agy member is `agy/gemini-4-argon-high`. One agy member, so the arm's draw weight does not double while #3361 is open; Hydrogen is recorded as the measured replacement; the stall note is marked unmeasured on Argon; `gpt-oss-120b-medium` stays out.
- `ai/agy/settings.json`: the seed `model` is the picker label `Gemini 4 Hydrogen (High)` (mid tier, as before) and `modelConfigs.defaultModel` follows it. `harness/model-pins.json` declares the label-to-id pair.
- `displayNameID` is unchanged; its test table carries the seven Gemini 4 rows `agy models` prints, and `Orchestrator (Argon)` is noted as a label the rule does not cover.
- Out of scope: pi and opencode bind NaN models only; the inline test pools naming `gemini-3.1-pro-high` are fixtures; the ci-workflows copy of the pool is ci-workflows#121; a doctor probe that compares `agy models` with the map is #3475.

## Checklist
- [x] `model-map.json` top and mid Gemini ids and their comments
- [x] `reviewer-pool.json` member swap, `$comment` admission line, embedded copy identical
- [x] `model-pins.json` agy pin `why` and the NOT LISTED note
- [x] `ai/agy/settings.json` `model` and `defaultModel`
- [x] `model_map_test.go` asserts top is Argon and mid differs from top
- [x] `model_pins_test.go` Gemini 4 rows; `model_pins.go` comment
- [x] `dotf spec review` example names the new id
- [x] Lesson 599

## Evidence

```
cd cli && gofmt -l ./internal/harness/ && go build ./... && go vet ./... && GOOS=windows go vet ./...   -> ok
cd cli && go test ./...                                                     -> 59 packages ok, exit 0
golangci-lint cache clean && golangci-lint run ./...                        -> 0 issues
GOOS=windows golangci-lint run ./...                                        -> 0 issues
bats tests/reviewer-pool.bats tests/pr-agent-config.bats tests/check-review-attestation.bats tests/pi-nan-package.bats -> ok
go run ./cmd/dotf harness check                                             -> [check] OK: no harness drift
go run ./cmd/dotf lessons fmt --check                                       -> [OK] lessons are in their canonical form
go run ./cmd/dotf doctor                                                    -> [Model pin drift] 1 checks, all ok
agy models (1.5.8)                                                          -> gemini-4-argon-high  Gemini 4 Argon (High); gemini-4-hydrogen-high  Gemini 4 Hydrogen (High)
cmp harness/reviewer-pool.json cli/internal/prreview/reviewer-pool.json     -> identical
```

## Next

Independent review, then archive. After the PR lands: `dotf deploy agy-settings` on each seat and confirm agy's start log resolves `Gemini 4 Hydrogen (High)`.
