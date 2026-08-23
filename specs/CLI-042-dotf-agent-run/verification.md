---
tags: [spec, verification, templates]
created: "2026-08-23"
---

# Verification - CLI-042-dotf-agent-run

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

**PR B only.** This spec ships as the five-PR sequence `tasks.md` declares. B covers AC1, AC2 and AC5;
the rest are named here as open with the PR that owns them, so a partly-filled section is never read
as a partly-met criterion.

- [x] AC1 (JSON contract on stdout, logs on stderr) -> `TestAgentRun_WritesOneJSONObjectToStdout`,
      `TestAgentRun_RefusalsLeaveStdoutEmpty`, `TestAgentRun_ReportsTheRouteTheMapDeclares`,
      `TestStdoutContracts/agent_run`, and `tests/dotf-agent-run.bats` (7 cases) through the compiled
      binary and a real pipe
- [x] AC2 (pool-unavailable advances the chain, task-failed does not) -> `TestDispatch_ChainWalk`
      (6 cases, including the exhausted chain, an unrecognised status and a malformed entry),
      `TestDispatch_RecordsDurationAndAttempts`, `TestExitCode`
- [ ] AC3 (timeout kills the worker and frees the slot before reaping) -> **PR C.** The half B can
      hold is proven: `TestDispatch_BackendReceivesADeadline` shows the per-dispatch deadline reaches
      the backend rather than being a parsed-but-inert flag. The kill and the slot release need
      something real to kill.
- [ ] AC4 (fails closed on an unreadable counter and an unidentifiable machine) -> **PR C**
- [x] AC5 (the top tier escalates, never degrades) -> `TestDispatch_TopTierNeverDegrades` (behaviour)
      and `TestChainsTopIsCappedWithoutLosingTheChainShape` (shape, in the schema)
- [ ] AC6 (the hive backend answers and reports pool + model) -> **PR D.** Still unrunnable on this
      machine: no worker endpoint is configured here. `hive delegate` itself is now installed
      (`hive-vault` 4.1.0), so the blocker is this machine's configuration, not the missing verb.
- [ ] AC7 (`dotf secrets run -- hive serve`; no credential in a deployed file) -> **PR E**
- [ ] AC8 (Ollama and OpenRouter removed from what this repo deploys for hive's worker) -> **PR E.**
      Re-measured 2026-08-23 and NOT already satisfied: `HIVE_OLLAMA_ENDPOINT` is still present at
      `ai/agy/mcp_servers.json:10`.
- [ ] AC9 (`dotf doctor` fails a backend that can serve nothing) -> **PR E**

## Test status

Produced 2026-08-23 on branch `feat/dotf-agent-run`, off `origin/main` @ `a27dcc8`.

- Go: `cd cli && go build ./... && go vet ./... && go test ./...` -> all packages ok, 0 failures
- Windows leg: `GOOS=windows go vet ./...` -> clean (the CI leg that compiles the same tree)
- Format: `gofmt -l .` -> no output
- Lint: `golangci-lint run` (v2.12.2, matching the `versions.conf` pin) -> **0 issues**
- Shell: `~/.local/bin/bats tests/*.bats` -> **1462 tests, exit 0, 0 failures**; `shellcheck` clean on
  the new `tests/dotf-agent-run.bats`
- `features.json` commands executed individually: the nine belonging to PR B exit 0; the eight
  belonging to PRs C/D/E exit non-zero, which is what `pending` has to mean. **This was not free** —
  the file's original commands named tests that do not exist, and `go test -run <nonexistent>` exits
  **0** with `[no tests to run]`, so every one of them would have certified an unwritten test. All
  seventeen were rewritten to the form `go test -run '^X$' -v | grep -q '^--- PASS: X'`, which fails
  when the test is absent.

### Mutation checks

A passing test is not evidence that a test *would* fail, so the two rules this PR adds were checked by
breaking them:

- Removing the top-tier no-degrade branch from `Dispatch` -> `TestDispatch_TopTierNeverDegrades` fails
  on all four assertions (status, exit code, entries attempted, and the leaked lower-tier answer).
- Removing the sibling `$ref` from `chains.top` in the schema -> three of the five cases in
  `TestChainsTopIsCappedWithoutLosingTheChainShape` fail: `chains.top: "claude:opus"` (a string) and
  `chains.top: []` are both **accepted**.

## Decisions made during implementation

Brief log of non-obvious trade-offs or course corrections taken during the work. Routine choices belong in commit messages, not here.

- **Exit codes mirror `hive delegate` (0/1/3) rather than being chosen freshly.** One vocabulary spans
  the seam, so a composer that already speaks to hive needs no translation table — and a translation
  table is where the two sides would drift. `escalated` and `chain_exhausted` share exit 3 and stay
  distinct in the record's `status`: the exit code is the coarse class, the record carries the fine one.
- **`main()` had to change.** It could only ever exit 1, which collapsed *no pool could serve this*
  into *the task failed* at the process boundary — the exact collapse AC2 forbids one layer down. A
  tagged error unwrapped in `main`, in preference to the `os.Exit`-inside-`RunE` precedent in
  `secrets.go`, which costs that file a child-process re-exec harness to test at all.
- **A `dry-run` backend ships in B**, so AC1's bats half runs against something rather than being a
  verification command that selects nothing. Rationale and the rejected alternative (exposing the
  scripted test fake as a `--backend` value) are in `proposal.md`'s decision table.
- **The chain arrives at `agent.Dispatch` already resolved.** `harness.ResolveChain` stays the map's
  one reader; a second parse inside the dispatcher would be a second place the routing rules are true.
- **Course correction: the first `chains.top` schema edit was a loosening that read as a tightening.**
  Naming `top` under `properties` exempts it from `additionalProperties`, so adding a bare `maxItems`
  silently dropped the array type, the minimum and the `pool:model` pattern. Caught by writing the
  mutation check before believing the green run.
- **A test that could not be written, and why that is the right answer.** The command-layer case for a
  malformed chain entry is unreachable: the schema rejects such a map before `Dispatch` is called. The
  dispatcher's own fail-closed branch is kept as defence for a chain that did not come from a
  validated map, covered in the `agent` package; the command-layer test was rewritten to assert the
  consequence that IS reachable — no record, empty stdout, exit 1 and never 3.

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault. If all three are "no", archive in repo is the only persistence.

- [ ] Lesson for the repo's `docs/lessons/`? <yes / no - one line of what>
- [ ] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? <yes / no - one line of what>
- [ ] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. <yes / no - one line>

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/CLI-042-dotf-agent-run/` -> `specs/archive/CLI-042-dotf-agent-run/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
