---
tags: [spec, verification, templates]
created: "2026-11-15"
id: "TOOL-041-cache-gc-mise-prune"
type: spec-verification
status: active
owner: manu
---

# Verification - TOOL-041-cache-gc-mise-prune

## Evidence

- [x] AC1 -> commit `aef3f12a` / tests: the order test (`mise ls --installed --json`, then `mise prune --tools --yes go node`) and `TestCacheGC_MisePruneLeavesPythonAlone`. Both failed before the change. A real dry run on the Mac, `mise prune --tools --yes --dry-run <every installed tool but python>`, listed 33 prunable versions and no `python@`.
- [x] AC2 -> commit `aef3f12a` / tests `TestCacheGC_MisePruneRunsNothingWhenOnlyPythonIsInstalled` and `TestCacheGC_MiseListFailureFailsTheStepAndKeepsItDue`.
- [x] AC3 -> commits `4c7688f5`, `aef3f12a`: the runbook row, the ADR-036 pruner list and lesson 605.

## Test status

- Test suite: `cd cli && go test ./internal/converge` -> ok. `go vet` on darwin and with `GOOS=windows` -> ok. `golangci-lint` 2.12.2 with go 1.26.10 (the pins), on darwin and with `GOOS=windows` -> 0 issues.
- Manual smoke test: the real `mise prune --tools --yes --dry-run` above. `--yes` and `--dry-run` combine, and nothing prompts.
- No regressions in existing test suite: yes.

## Decisions made during implementation

- The first commit ran a bare `mise prune --yes`. The venv measurement (hive's uv tool and about 55 kubelab `.venv`s on mise's Python 3.13.16) turned that into a hold. mise has no exclusion flag, so the pruner names the tools to prune instead.
- `cmdsFor` builds the commands at run time; static pruners keep `cmds`. A dry run prints `plan` and does not list the tools, because a plan must not depend on a command that may fail.

- Review (PASS at d282b9df), Minor that the unparseable-listing branch is untested: applied. `TestCacheGC_MiseUnreadableListingFailsTheStepWithoutPruning`.
- Review, Minor that the `plan` literal can drift from `miseHeld` and that the `cmdsFor` arm of `plannedCommands` is untested: applied. The plan is built from `miseHeld` through `heldNames`, and `TestCacheGC_PlanNamesTheHeldMiseToolsAndRunsNothing` covers the arm.
- Review, Minor that the `tools` step's orphan prune has no python hold: recorded, not changed. That prune removes only tools no pin names, and python is a hard dependency that stays pinned (#2062). ADR-036 and the `miseHeld` comment now state the assumption.
- Review, Minor suggesting reuse of `mise ls --prunable --json`: declined. The pruner needs tool names, not versions, and `mise prune` already decides which versions go. Reusing `Orphans` would couple `cache-gc` to the `tools` step's parsing for no change in behaviour.
- Review, question whether python could appear under another key: confirmed bare `python` on the Mac (`core:python`); no change.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-605-a-pin-with-no-way-to-move-it-gets-moved-by-hand.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: ADR-036's `cache-gc` section is amended in place
- [x] New pattern candidate for `00_meta/patterns/`? no: the hold is specific to how this repository's venvs reach mise's Python

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/TOOL-041-cache-gc-mise-prune/` -> `specs/archive/TOOL-041-cache-gc-mise-prune/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
