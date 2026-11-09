---
tags: [spec, verification, templates]
created: "2026-11-07"
id: "HARNESS-186-named-lanes-sticky-personas"
type: spec-verification
status: active
owner: manu
---

# Verification - HARNESS-186-named-lanes-sticky-personas

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] Criterion 1 (`AC1` — Persistent Named Lanes & Idempotent Provisioning) -> `TestLaneInitIdempotent`, `TestLaneSwitchRoutingAndWIPCap` (`cli/internal/lane/lanes_test.go`)
- [x] Criterion 2 (`AC2` — Safe In-Place Lane Park & Housekeeping) -> `TestLaneParkInPlaceAndTipVerification` (`cli/internal/worktree/park_test.go`)
- [x] Criterion 3 (`AC3` — Sticky Persona Gate & Read-Only Exemption) -> commit `7027fa36` / `TestDecide_ExemptsReadOnlyDiscoveryTools`, `TestStickyPersona_BindsOncePerBranchAndResetsOnBranchSwitch`, `TestAtomicWriteFile_ConcurrentWritesOnWindowsDoNotFail`, `TestHarnessGateStickyPersonaAndMeeseekResolution`
- [x] Criterion 4 (`AC4` — `meeseek-<role>-<fruit>` Ephemeral Worker Identity & `pi` Pool Fallback) -> commit `7027fa36` / `TestFruitDeck_64SequentialDrawsHaveZeroCollisions`, `TestParseWorkerID_RejectsPluralAndMalformedNames`, `TestDispatch_PopulatesMeeseekWorkerIDAndPropagatesToRequest`, `TestPiBackend_IncludesProviderFlagAndClassifiesUnauthenticatedAsUnavailable`
- [x] Criterion 5 (`AC5` — Scoped `pre-commit` Lane Guard & Rebase Exemption) -> `TestPreCommitLaneGuard` (`cli/internal/hooks/lane_guard_test.go`)
- [x] Criterion 6 (`AC6` — Cross-OS Forge Provenance (`no-attribution` Compliant)) -> `TestForgePRProvenanceNoAttribution` (`cli/internal/forgewrite/provenance_test.go`)

## Test status

- Test suite:
  - `ok github.com/mlorentedev/dotfiles/cli/internal/agent 2.870s`
  - `ok github.com/mlorentedev/dotfiles/cli/internal/harness 2.181s`
  - `ok github.com/mlorentedev/dotfiles/cli/internal/forgewrite 2.181s`
  - `ok github.com/mlorentedev/dotfiles/cli/internal/hooks 48.142s`
  - `ok github.com/mlorentedev/dotfiles/cli/internal/worktree 52.406s`
  - `ok github.com/mlorentedev/dotfiles/cli/internal/lane 182.730s`
  - `ok github.com/mlorentedev/dotfiles/cli/internal/cmd 33.492s`
- Manual smoke test: Provisioned `feat-lane-primary` for `dotfiles`, cleaned 19 safe worktrees on `EGW-LEN029` (`35 -> 16`), verified `dotf agent run --tier mid --role reviewer` dispatches `pi --provider nan` (`meeseek-reviewer-durian` and `meeseek-reviewer-guava`).
- No regressions in existing test suite: yes

## Review-fix evidence (Batch 4 — PR `#3259`)

The guard's opt-in was `.dotf/lane.json`, which already exists on `main` for the unrelated issue-queue
subsystem (`#2532`). Merging as-is would have activated the guard repo-wide with zero lanes initialized.
The named-lane subsystem now owns its own marker.

- [x] `AC5` activation -> marker `dotf lane init` writes, `.dotf/lanes.json`; issue-queue `.dotf/lane.json` must not activate. Proven by scenario 3b of `TestPreCommitLaneGuard`.
- [x] `AC5` POSIX portability -> `sh -n git-hooks/lib/lane-guard.sh` exit 0; `shellcheck -s sh git-hooks/lib/lane-guard.sh` exit 0 (clean). The two bashisms (`${var//\\//}`) that broke the Ubuntu `sh` CI job are gone; path normalization now uses `sed 's|\\|/|g'`.
- [x] `AC5` agnosticism -> scenario 8 declares `third_party_roots: ["acme-sessions"]` and the commit inside it is blocked, proving the script privileges no host, agent or machine name.
- [x] `AC1` marker provisioning -> `TestLaneInitIdempotent` asserts `.dotf/lanes.json` parses, `canonical_lanes` equals `CanonicalLanes`, `third_party_roots` is non-empty, no value contains `/`, `\` or `:` (machine-independence), and the file is byte-identical after a second `dotf lane init`.

Commands run in this worktree:

```
cd cli && go build ./...                                    -> exit 0
cd cli && go vet ./internal/lane/... ./internal/hooks/...    -> exit 0
sh -n git-hooks/lib/lane-guard.sh                            -> exit 0
shellcheck -s sh git-hooks/lib/lane-guard.sh                 -> exit 0
cd cli && go test ./internal/hooks/... -count=1              -> ok  39.297s
cd cli && go test ./internal/lane/...  -count=1              -> ok 182.851s
```

Reference rot: ADR-060's `extends:` and this spec's `## References` cited `adr-032-worktree-first-isolation`,
`adr-049-runtime-skill-enforcement-gate`, `adr-052-deterministic-worktree-sweep` and
`adr-050-self-driving-dev-lanes` — none exist. Repointed to the real ADRs; the phantom ids are documented
and explained in ADR-060's `## References` section.

## Review-fix evidence (Batch 5 — architecture audit of Batch 4)

Auditing Batch 4 found the marker was written but never actually read: `json_list()` is a line-based
`sed`, `dotf lane init` writes indented JSON with one element per line, and the fixture kept each array
on a single line. Against the real file the parser returned nothing and the guard fell back to
hardcoded lane names and one hardcoded host — the duplication the marker exists to delete, with a green
suite and no error anywhere.

- [x] `AC5` reader/writer agreement -> the guard's fixture is generated from `lane.CanonicalLanes` and `lane.ThirdPartyWorktreeRoots` with `json.MarshalIndent` (production bytes), and `json_list()` flattens newlines before matching, so both indented and compact markers parse.
- [x] `AC5` no privileged host -> hardcoded fallbacks removed from the shell. New scenario 9 (marker with no `third_party_roots` leaves `copilot-worktrees/` allowed) pins the absence of a fallback; new scenario 10 (marker declaring `canonical_lanes: ["spike-lane"]` blocks an unleased `spike-lane/`) pins that lane names are data too.
- [x] `AC1` host-owned vs code-owned data -> `writeLanesMarker()` now unions `third_party_roots` with the on-disk file instead of overwriting it, so a hand-added session host survives `dotf lane init`. `canonical_lanes` stays code-owned. Proven by `TestLaneInitPreservesDeclaredThirdPartyRoots`, which also asserts the file stays indented one element per line.
- [x] Code Quality Rules -> `initLanesLocked` was ~75 lines against the repo's `< 40 lines` rule; extracted `provisionLane()` and `saveLaneMetadata()` with behaviour, error strings and the `changed == 3` contract unchanged.

Commands run in this worktree:

```
cd cli && go build ./...                                              -> exit 0
cd cli && go vet ./internal/lane/... ./internal/hooks/...              -> exit 0
sh -n git-hooks/lib/lane-guard.sh                                     -> exit 0
shellcheck -s sh git-hooks/lib/lane-guard.sh                          -> exit 0
cd cli && go test ./internal/hooks/... -run TestPreCommitLaneGuard     -> ok  81.083s  PASS (10 scenarios)
cd cli && go test ./internal/lane/... -run 'TestLaneInitIdempotent|TestLaneInitPreservesDeclaredThirdPartyRoots|TestLaneSwitchRoutingAndWIPCap'
                                                                      -> ok  99.630s  PASS
cd cli && go test ./internal/spec/... -count=1 -timeout 40m            -> ok 1105.827s
```

Knowledge added: `docs/runbooks/guide-persistent-named-lanes.md` (layer model, marker contract, guard
decision flow, lane lifecycle, use cases, activation checklist, agnostic validation runbook) and
`docs/lessons/lesson-569-generate-test-fixtures-from-the-writer-not-by-hand.md`.

## Decisions made during implementation

- Added `.dotf-lane-prompt.md` to `isDisposableExactPath` in `cli/internal/worktree/clean.go` so `Dirt()` never marks a named lane dirty solely due to a handoff prompt file before `Park()` removes it.
- Used `--no-track` when `SwitchLane` checks out a new branch from `origin/main` so `git` does not set `origin/main` as the upstream tracking branch `@{u}`, allowing `branchLanding` to verify squash-merged PR heads accurately.
- Guarded `atomicWriteFile` in `cli/internal/harness/dispatch.go` with `atomicWriteMu sync.Mutex` + 15-attempt `os.Rename` retry to eliminate concurrent `Access is denied` errors on Windows.
- Executed independent code-level adversarial reviews via subagent (`meeseek-reviewer-avocado`) and `dotf secrets run --only NAN_API_KEY -- dotf agent run --tier mid --role reviewer` (`meeseek-reviewer-durian` on `nan:deepseek-v4-flash`) and applied all 8 findings:
  - `M-CODE-1`: Released `refs/dotf/claims/<issue>`, `refs/dotf/seen/<issue>`, and remote `:refs/dotf/claims/<issue>` across `done.go:Park()`, `sweep.go:parkSingleNamedLane()`, and `named.go:ParkLane()`.
  - `M-CODE-2`: Ordered `State="PARKED"` write before `git checkout --detach` / `git branch -D` in `done.go` and `sweep.go`, and checked `branchLanding` tip SHA containment in `sweep.go:parkSingleNamedLane()`.
  - `M-CODE-3`: Wired active worktree branch, `.dotf-worktree.json` `persona`, and skill-consumption persona inference into `cli/internal/cmd/harness_gate.go`.
  - `M-CODE-4`: Wired `maybeAppendLaneProvenance` into `cli/internal/cmd/forge_write.go` (`forge pr create` and `forge pr/issue comment`) and added `machine`, `os`, and `lane` fields to `cli/internal/lane/claim.go`.
  - `m-CODE-1`..`m-CODE-4`: Forwarded `DOTF_WORKER_ID`/`DOTF_ROLE` to `Hive.Dispatch`, emitted `dotf lane switch` in `handoff.go:manualCommands`, ran best-effort `git fetch --quiet origin` before `Park()`, and exempted `REVERT_HEAD` and `sequencer` in `git-hooks/lib/lane-guard.sh`.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: docs/lessons/lesson-567-in-place-worktree-parking-and-windows-atomic-rename-mutex.md docs/lessons/lesson-568-key-a-guards-activation-on-a-signal-its-own-feature-writes.md docs/lessons/lesson-569-generate-test-fixtures-from-the-writer-not-by-hand.md
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? yes: docs/adr/adr-060-persistent-named-lanes-sticky-personas-and-meeseek-workers.md
- [x] New pattern candidate for `00_meta/patterns/`? Only if this recurs in >1 project. no: documented in repo-scoped ADR-060 and lesson-567 first until validated across multiple repositories.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/HARNESS-186-named-lanes-sticky-personas/` -> `specs/archive/HARNESS-186-named-lanes-sticky-personas/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
