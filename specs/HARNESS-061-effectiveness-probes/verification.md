---
tags: [spec, verification, templates]
created: "2026-08-08"
---

# Verification - HARNESS-061-effectiveness-probes

## Evidence

- [x] AC1 (effective config, repo named, remedy stated) -> `TestGuardHooks_LocalOverrideDisablesGuard_MustFail`, green direction `TestGuardHooks_NoOverrideGuardFires_MustPass`
- [x] AC2 (gate via dispatcher fallback) -> `TestVaultHooks_GateViaDispatcherFallback_MustPass`; negative `TestVaultHooks_DispatcherWithoutConfig_IsNotAGate`
- [x] AC3 (non-executable resolves absent) -> `TestHookForStage_NonExecutableIsNotAHook`
- [x] AC4 (new stub suite fails) -> `tests/stub-real-pairing.bats` case 1; red-teamed by adding a throwaway stub suite, which the guard named before it was removed
- [x] AC5 (exemptions cannot go stale) -> `tests/stub-real-pairing.bats` case 2, plus case 3 guarding the detector against passing vacuously
- [x] AC6 (DR escrow + drill) -> `TestDR_NoDrillRecorded_Warns`, `TestDR_RecentDrillPasses`, `TestDR_StaleDrillWarns`, `TestDR_AbsentEscrowSkipsNotFails`, `TestDR_StaleEscrowWarns`
- [x] AC7 (red direction everywhere) -> each case above has its inverse; the probe helpers additionally carry `TestEffectiveHooksPath_NilSeamIsUnsetNotPanic`, added after a nil command seam panicked the whole doctor suite on first run

## Test status

- Test suite: `go test ./...` -> 12 packages ok, 0 failures
- Test suite: `bats tests/*.bats` -> 1 failure, `install-dotf.bats` "converges over a running dotf", pre-existing on `main` and tracked as #807 (BUG-054)
- Lint: `shellcheck scripts/*.sh setup-linux.sh` -> clean
- Manual smoke: the two live defects this replaces were both reproduced on this machine before the change and confirmed correctly reported after — the guard's inertness under a repo-local override, and the vault gate running through the dispatcher fallback while the old check called it INACTIVE
- No regressions: yes. Two existing fixtures needed updating rather than the code — `vaultTree` now writes hooks executable (a non-executable hook is one git ignores, so the old fixture modelled a gate that does not fire), and the git fake now reports `config --get` the way git does, effective rather than local-only

---

## Re-verification 2026-08-18, by mutation

This spec's thesis is that a passing report is not evidence, so the inherited
guards were re-checked by breaking what each claims to catch, rather than by
re-reading the ticks above.

### The mitigations the proposal asserts

| asserted in "Risks / open questions" | check | result |
|---|---|---|
| dispatcher-as-gate is safe *because* the fallback has a real-dependency test | `bats tests/precommit-fallback-real.bats` | 3/3 pass — the mitigation exists and runs |
| the exemption table cannot go stale silently | mutation, below | holds |

### Mutations

| mutation | expected | observed |
|---|---|---|
| exempt a suite that HAS a real sibling (`precommit-fallback`) | stale-entry test fails | `not ok 2 … no stale entries` |
| exempt a suite that does not exist (`ghost-suite`) | stale-entry test fails | `not ok 2 … no stale entries` |
| add a new stub suite, unpaired and unexempted | pairing test fails, naming it | `not ok 1 … throwaway-canary` |
| drop one row from the documented table | new drift guard fails | `not ok 2 … board-pickup` |
| add to `EXEMPT_SUITES` without documenting it | new drift guard fails | `not ok 2` |

**A false positive worth recording**, because it is the failure this spec names.
The first attempt mutated the *comment table* and nothing failed — which reads as
"AC5 is a claim". It was not: the table is documentation and `EXEMPT_SUITES` is
what the code reads, so the mutation never reached the guard. **An invalid
mutation and an absent guard produce the same green.** Verifying that the
mutation lands is part of the mutation, not a preliminary to it.

### Defect found and fixed: a third copy

Chasing that false positive surfaced a real gap. The file's own header narrates a
drift between two copies of the exemption list and fixes it by single-sourcing
`exempt()` and the stale-entries loop to `EXEMPT_SUITES` — but the
human-readable table above them stayed a **third** copy with nothing comparing
it. Measured in sync (11 entries each) on 2026-08-18.

It does not disable the guard; `EXEMPT_SUITES` remains authoritative. It
misinforms the only reader who consults it — the person deciding whether to add a
row. `pattern-derived-fact-drift`, inside the file that exists to catch this
class.

Closed by a fifth case in `tests/stub-real-pairing.bats`, mutation-tested in both
directions (rows 4 and 5 above).

### Current state

```
$ go build ./... && go vet ./... && go test ./...     # cli/
15 packages ok, 0 failures

$ bats tests/stub-real-pairing.bats
5 tests, 0 failures

$ bats tests/precommit-fallback-real.bats
3 tests, 0 failures

$ dotf doctor | grep -A1 'Disaster recovery'
[Disaster recovery]
  [WARN] no recovery drill recorded — run the chain … then `touch ~/.dotfiles/.dr-drill`
```

AC6 confirmed behaviourally: the check reports, and warns rather than fails, on a
machine where the drill has never run. **That WARN is accurate** — the DR drill
is still unexecuted, which this spec deliberately scopes out (surfacing staleness
is code; performing the drill is a calendar commitment).

## Adversarial review disposition (2026-08-18)

`nan/deepseek-v4-flash`, verdict **PASS**, `reviewed_sha: 4d153a0`. No blockers,
no majors. Three Minor findings, all THEORETICAL/UNTESTED. Each dispositioned:

| # | Finding | Disposition |
|---|---|---|
| 1 | AC1's coverage scope (2 probed repos) is documented in code but not in the spec | **Ticketed, deliberately not applied** — see below |
| 2 | `guardProbeRepos` filters with `isDir(r/".git")` while `checkVaultHooks` uses `isGitCheckout`, so a linked worktree is silently skipped | **Applied** + regression test |
| 3 | `hookForStage`'s green direction has no standalone test | **Applied** |

### Why finding 1 is ticketed rather than applied

It asks for a sentence in `proposal.md`. That file is a **contract file** —
`contractFiles = {proposal.md, tasks.md, features.json}` in `cli/internal/spec/review.go`
— so editing it moves the spec past `reviewed_sha` and the archive gate refuses
the review that demanded the edit. That is #998, and applying the finding here
would reproduce it deliberately.

Findings 2 and 3 touch `checks_guard.go`, `hookprobe_test.go` and
`checks_guard_test.go`; none is a contract file, so the review stands.

### Finding 2 was real, and it is this spec's own thesis

`isGitCheckout` exists specifically to avoid this trap and its docstring names
it: in a linked worktree `<path>/.git` is a `gitdir:` pointer **file**, so
`isDir` answers *"is this a REGULAR checkout"* and the repo is skipped — *"a SKIP
that reads as healthy is worse than a FAIL"*. One call site was still asking the
older question, so two checks in the same package disagreed about the same
worktree.

Mutations, both observed failing before the fixes were trusted:

| mutation | observed |
|---|---|
| restore `isDir(r/".git")` in `guardProbeRepos` | `--- FAIL: TestGuardProbeRepos_LinkedWorktreeIsProbed … got []` |
| make `hookForStage` skip the exec-bit check | `--- FAIL: TestHookForStage_NonExecutableIsNotAHook` |

```
$ go build ./... && go vet ./... && go test ./...
ok  github.com/mlorentedev/dotfiles/cli/internal/doctor  0.479s   (all packages green)
```
