---
tags: [spec, verification, templates]
created: "2026-11-04"
id: "CLI-135-init-stack-directories"
type: spec-verification
status: active
owner: manu
---

# Verification - CLI-135-init-stack-directories

## Evidence

- [x] AC1 (each stack's directories are pinned, one row per stack) -> test `TestScaffoldCodeDirectoriesByStack`, with rows for go, python, node, ts, none, "", an unknown stack and workflows
- [x] AC2 (`--stack workflows` writes tests/, specs/, docs/ and lint.yml, and no src/, scripts/ or ci.yml) -> test `TestInitWorkflowsStackScaffoldsNoCodeAndNoCI`
- [x] AC3 (none, the existing stacks and an unknown stack keep src/, tests/ and scripts/) -> the same table test's other rows. `TestScaffoldCreatesStructureAndFiles` and `TestInitOrchestratesFullScaffold` pass unchanged
- [x] AC4 (`StackInit(root, "workflows")` is a no-op with no unknown-stack skip) -> test `TestStackInitWorkflowsIsANoOp`

Mutation check (2026-11-04). Each change was reverted on its own and the three new tests run against it. All three mutations were killed:
- `workflows` given `codeDirs` -> killed
- `workflows` dropped from `StackInit`'s no-op case -> killed
- `Init` passing `"none"` to `ScaffoldOpts` instead of `opts.Stack` -> killed

## Test status

- `cd cli && go test ./...` -> every package ok, no failures
- `go vet ./...` and `GOOS=windows go vet ./...` -> clean; `gofmt -l` -> nothing
- `golangci-lint run` (the versions.conf pin) -> `0 issues.`
- No regressions in the existing suite: yes

## Decisions made during implementation

- The owner's shape is a new `workflows` stack, not "`none` with no code directories". `none` is the default, so changing it would have changed every unflagged `dotf init`.
- `workflows` keeps `tests/`. All three ci-workflows sessions reported that its `tests/` holds the bats shape tests, which the issue's premise had missed.
- An unknown `--stack` value keeps today's behaviour: the full code directories, no ci.yml, and `StackInit`'s "unknown stack" skip. Validating the flag is a separate contract change.
- ADR-022 line 76 lists `--stack {go,python,node,none}`. It is a dated decision record, so it is left as written; the flag help is the live list.
- The PR box in tasks.md stays unticked. The PR is recorded here instead, so the reviewed tasks.md digest does not change (HARNESS-081).

## Review dispositions (round 1: PASS, nan/glm5.3-flash, on ecf307de)

- Minor, the `InitOptions.Stack` comment omits `workflows` (orchestrator.go:18) -> applied
- Minor, the `dotf vault project --stack` help omits `ts` and `workflows` (vault.go:227; pre-existing for `ts`) -> applied
- Minor, ADR-022:76 stack list is stale -> declined: a dated decision record stays as written (see Decisions above); the flag help is the live list

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? no: a missing table entry, no new failure mode
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: it extends ADR-022's stack list without changing its decision
- [x] New pattern candidate for `00_meta/patterns/`? no: single project

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/CLI-135-init-stack-directories/` -> `specs/archive/CLI-135-init-stack-directories/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
