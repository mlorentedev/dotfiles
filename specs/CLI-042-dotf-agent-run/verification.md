---
tags: [spec, verification, templates]
created: "2026-08-23"
---

# Verification - CLI-042-dotf-agent-run

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

**PRs B, C1 and C2.** This spec ships as the sequence `tasks.md` declares (C was split into C1/C2 during
C1, because one PR carrying both AC3 and AC4 does not fit the atomic cap). B covered AC1, AC2 and
AC5; **C1 covers AC3, C2 covers AC4**. With that, **AC1-AC5 are complete**; the rest are named here as open with the PR that owns them, so a
partly-filled section is never read as a partly-met criterion.

- [x] AC1 (JSON contract on stdout, logs on stderr) -> `TestAgentRun_WritesOneJSONObjectToStdout`,
      `TestAgentRun_RefusalsLeaveStdoutEmpty`, `TestAgentRun_ReportsTheRouteTheMapDeclares`,
      `TestStdoutContracts/agent_run`, and `tests/dotf-agent-run.bats` (7 cases) through the compiled
      binary and a real pipe
- [x] AC2 (pool-unavailable advances the chain, task-failed does not) -> `TestDispatch_ChainWalk`
      (6 cases, including the exhausted chain, an unrecognised status and a malformed entry),
      `TestDispatch_RecordsDurationAndAttempts`, `TestExitCode`
- [x] AC3 (timeout abandons the worker and frees the slot before reaping) -> **PR C1.**
      `TestDispatch_TimeoutReleasesSlotBeforeReap` takes the freed slot *while the abandoned worker is
      still running*, which is only possible if the release does not wait on it.
      `TestDispatch_TimeoutDoesNotAdvanceTheChain`, `TestSemaphore_*` (capacity, pool independence,
      idempotent release), `TestDeclaredCapacity` (reserve subtracted; undeclared ≠ zero), and a bats
      case that saturates the pool with real `flock`s and drives the compiled binary.
      **Read the guarantee narrowly**, as the map and ADR-032 §3 both do: this bounds `dotf`-dispatched
      work only. A hand-run `qq`, a pi TUI turn or a hive embedding call takes a slot the semaphore
      never sees. The claim is *`dotf` alone will never be the cause of exhaustion*, not that
      exhaustion cannot happen.
- [x] AC4 (fails closed on an unreadable counter and an unidentifiable machine) -> **the counter half
      in C1** (`TestSemaphore_UnreadableStateIsAnErrorNotZeroInUse` — in this design the semaphore
      state IS the counter); **the machine half and the wording in C2**:
      `TestLoadMachinePolicy_*` (4 tests: unidentified denies everything across four shapes, declared
      identity, identified-with-no-deny-list allows, malformed is an error not an absence),
      `TestMachinePolicy_ValidateDenyNames`, `TestAgentRun_AnUnidentifiedMachineIsRefused` (asserts
      the remedy is IN the message), `TestAgentRun_ADenyTypoIsRefused`,
      `TestAgentRun_EveryPoolDeniedIsItsOwnOutcome`, `TestDispatch_ADenied*` (3), and three bats cases
      through the compiled binary.
      `TestAgentRun_StatesTheNarrowGuaranteeAndNotAWiderOne` asserts the wording in both directions:
      the narrow guarantee verbatim, and four overstatements that must NOT appear.
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

Produced 2026-08-23. PR B on `feat/dotf-agent-run` off `a27dcc8`; PR C1 on
`feat/dotf-agent-brakes` off `7e734c4`. Both figures below are from the C1 run.

- Go: `cd cli && go build ./... && go vet ./... && go test ./...` -> all packages ok, 0 failures
- Windows leg: `GOOS=windows go vet ./...` -> clean (the CI leg that compiles the same tree)
- Format: `gofmt -l .` -> no output
- Lint: `golangci-lint run` (v2.12.2, matching the `versions.conf` pin) -> **0 issues**
- Shell: `~/.local/bin/bats tests/*.bats` -> **1462 tests, exit 0, 0 failures**; `shellcheck` clean on
  the new `tests/dotf-agent-run.bats`
- `features.json` commands executed individually: **16 exit 0, 6 exit non-zero** after C1 (was 10/7
  after B; C1 added five features and turned f8 green). The six that fail are AC4's machine half
  (f10, f11 → C2), AC6 (f14 → D) and AC7–AC9 (f15–f17 → E).

  The B-era figure, for the record: **10 exit 0, 7 exit non-zero**. The ten are f1–f7,
  f9, f12 and f13; note f9 belongs to AC3, which is PR C's — it is the half of that criterion B can
  hold (the deadline reaches the backend), and counting it under B would misreport AC3 as met. The
  seven that fail are the criteria PRs C, D and E own, which is what `pending` has to mean.
  **This was not free** —
  the file's original commands named tests that do not exist, and `go test -run <nonexistent>` exits
  **0** with `[no tests to run]`, so every one of them would have certified an unwritten test. The
  **13 Go commands** were rewritten to capture the status before matching —
  `out=$(go test -run '^X$' -v 2>&1) && printf '%s' "$out" | grep -q '^--- PASS: X'` — which fails
  both when the test is absent and when `go test` exits non-zero after printing the PASS line. The
  pipeline form does neither: `grep` reports the pipeline's status, so a package where one subtest
  passes and another fails still certifies. The remaining four (f2, f15, f16 are bats over a whole
  file; f14 is `dotf … | jq -e`) do not take that form and did not need it — bats fails on a missing
  file, and f14's predicate now compares `.model` against the resolved chain entry rather than
  accepting any non-empty record.

### Mutation checks

A passing test is not evidence that a test *would* fail, so the two rules this PR adds were checked by
breaking them:

- Removing the top-tier no-degrade branch from `Dispatch` -> `TestDispatch_TopTierNeverDegrades` fails
  on all four assertions (status, exit code, entries attempted, and the leaked lower-tier answer).
- Removing the sibling `$ref` from `chains.top` in the schema -> three of the five cases in
  `TestChainsTopIsCappedWithoutLosingTheChainShape` fail: `chains.top: "claude:opus"` (a string) and
  `chains.top: []` are both **accepted**.

C1's three, each dying for a different reason — which is the evidence the assertions are independent
rather than one assertion wearing three hats:

- Waiting for the worker before releasing the slot (`<-done` ahead of `Release`) -> the dispatch never
  returns at all; the test's own 5s guard fires.
- Not releasing on the timeout path -> the slot is still held while the abandoned worker runs, and the
  test names it: *the reserve then under-counts what is free for as long as the worker lives*.
- `IsPoolBusy` returning false -> a full pool stops the walk instead of advancing it; two tests fail.
- And through the binary: making every pool unbounded -> the bats saturation case fails, because the
  dispatch goes to the pool whose slots are all held.

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
