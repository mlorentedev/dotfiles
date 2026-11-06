---
tags: [spec, verification, templates]
created: "2026-11-04"
id: "CLI-129-initrepo-lane-contract"
type: spec-verification
status: active
owner: manu
---

# Verification - CLI-129-initrepo-lane-contract

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

Each AC is covered by a named test. Three of them were also mutation-checked (see Decisions).

- [x] AC1 -> `TestBootstrapLane_ConfigLoads` (loads the scaffolded file with `lane.LoadConfig`)
- [x] AC2 -> `TestBootstrapLane_PromptRenders` (renders it with `lane.RenderPrompt`)
- [x] AC3 -> `TestBootstrapLane_AgentsSection` (create, and append after prior content)
- [x] AC4 -> `TestBootstrapLane_Idempotent` (second run byte-identical, all `unchanged`)
- [x] AC5 -> `TestBootstrapLane_KeepsExisting`
- [x] AC6 -> `TestBootstrapLane_Force` (prompt and section replaced in place, `lane.json` untouched)
- [x] AC7 -> `TestInit_LaneStep`, `TestInitLane_DryRun`, `TestBootstrapLane_DryRunWritesNothing`
- [x] AC8 -> kubelab rollout follow-up filed as #3154

## Test status

- `go test ./internal/initrepo ./internal/cmd -count=1` -> `ok` for both; the lane tests all `--- PASS`
- `go build ./... && go vet ./...` -> clean; `GOOS=windows go vet ./...` -> clean
- `golangci-lint run ./...` (pinned, cache cleaned first) -> `0 issues.`
- Smoke: `TestInitLane_DryRun` runs the real command through cobra, dry-run first (nothing written), then for real (three files written)
- No regressions: the full `internal/cmd` suite passes, including the help-text guard that rejected an `ADR-050` mention in the command's help

## Decisions made during implementation

- Mutations, each failing the named test: an unknown `bootstrap` key in `lane.json` (`ConfigLoads`), `--force` also rewriting `lane.json` (`Force`), a prompt field `PromptData` does not carry (`PromptRenders`).
- `--skip-agents` also skips the lane step, because the `## Lanes` section extends the file `--skip-agents` promises not to write. `--skip-lane` skips it alone.
- `hasSDDSection` and `replaceSDDSection` now delegate to `hasSection` and `replaceSection`, so the lane section reuses the SDD section's surgery instead of copying it.
- The command help does not name ADR-050: `TestHelpTextHasNoInternalReferences` rejects internal references in user-facing help.

## Independent review dispositions

Review `review.md` (nan/deepseek-v4-flash): PASS, four Minor findings.

- `--skip-lane` untested at the CLI boundary -> applied: `TestInit_SkipFlagsReachTheLaneStep` drives `dotf init --skip-lane` / `--skip-agents` through cobra. Mutating `SkipLane: skipLane` to `skipAgents` turns it red.
- `--skip-lane` and the `--skip-agents` coupling absent from the contract -> declined: the contract files are closed under the verdict, and the behaviour is recorded here and in the flag help.
- ADR-022 subcommand table omits `dotf init lane` -> applied: row and `--skip-lane` added to the table.
- `## Lanes` heading quoted inside a fenced block (SPECULATIVE) -> declined: not reproduced, and the SDD section has had the same behaviour since before this change.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? no: nothing surprising; the one trap (an internal reference in help text) is already guarded by a test
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the design is ADR-050, accepted in #3143
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: specific to dotf init

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-129-initrepo-lane-contract/` -> `specs/archive/CLI-129-initrepo-lane-contract/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
