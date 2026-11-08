---
tags: [spec, verification, templates]
created: "2026-11-07"
id: "BUG-3253-gittest-quiet-appends-inherited-config"
type: spec-verification
status: active
owner: manu
---

# Verification - BUG-3253-gittest-quiet-appends-inherited-config

## Evidence

- [x] Criterion 1 -> `cli/internal/cmd` `-count=1` under `GIT_CONFIG_COUNT=3` with three inherited pairs: the binary starts and exits cleanly, where the pre-fix control aborted at `gittest.go:50` (the A/B below). `go test ./internal/cmd -run '^$'` -> exit 0 in 1.36s. The criterion is **startup only** — `-run '^$'` selects no tests, so it cannot speak for the suite. That tests now run at all is evidenced separately: the adversarial reviewer reproduced two real fixtures to completion under the same environment — `TestDeclaredCapacity` (2.0s) and `TestHarnessRefresh_AFailedRenderIsAnErrorAndWritesNothing` (67.7s) — and CI owns the full run (suite note below).
- [x] Criterion 2 -> `TestQuietKeepsTheCallersConfig`, `TestQuietAppendsAfterAForeignEnvironment`, `TestQuietOutranksAConflictingCallerPair` (the last two in this commit): each inherited pair still reads back through a child `git config`, and `gc.auto=0` / `maintenance.auto=false` apply on top — including where the caller pins the same keys to the opposite value.
- [x] Criterion 3 -> `TestQuietAcceptsItsOwnKeys`, `TestQuietDoesNotIncludeItsOwnFile` (commit `8881f845`): a second `Quiet` leaves `GIT_CONFIG_COUNT` unchanged and does not rewrite the system config.
- [x] Criterion 4 -> `TestQuietRefusesAMalformedCount`, `TestQuietRefusesBeforeItTouchesTheEnvironment` (this commit), `TestQuietRefusesNoSystem` (pre-existing): `abc` and `-1` still panic, as do `GIT_CONFIG_NOSYSTEM`, a count promising a pair the environment does not carry, and a count too large to be real.

## Test status

- `go test ./internal/gittest -run '^TestQuiet' -v -count=1` with `GIT_CONFIG_COUNT=3` in the environment -> 14 top-level tests PASS, `ok ... 8.993s`.
- `go test ./internal/cmd -count=1` with `GIT_CONFIG_COUNT=3` in the environment -> the binary starts and runs its tests instead of aborting in `TestMain`; the package itself cannot finish inside the default timeout on this host (see the suite note below, measured cause).
- Feature predicates: all four `features.json` entries ran under bash with every `GIT_CONFIG_*` cleared -> `PASS` x4, exit 0.
- **A/B control:** with the harness environment intact, `git stash push` of the change makes `go test ./internal/cmd` panic at `gittest.go:50`; `git stash pop` restores exit 0. The panic is the bug, and it is the only difference between the two runs.
- **Precedence is observable, so both precedence tests are non-vacuous:** with `GIT_CONFIG_COUNT=2` and both pairs on `gc.auto`, `git config gc.auto` reads `0` when `1` is at index 0 and `1` when `0` is at index 0 — the last pair wins. A `Quiet` that appended at the wrong index would fail `TestQuietOutranksAConflictingCallerPair`, and one that skipped on "the pair appears anywhere" would fail `TestQuietOutranksACallerThatSetsTheSameKeyTwice`: reverted to that check, the test reports `gc.auto = "1", want "0"` and `GIT_CONFIG_COUNT = "3", want "4"`; restored, it passes.
- No regressions: `go test ./internal/gittest -count=1` green; `go vet ./internal/gittest` clean.

## Review dispositions (round 1, `review.md`, verdict FAIL)

| Finding | Disposition |
|---|---|
| Major — the proposal's duplicate-key precedence is claimed as asserted but untested; AC2's "apply on top" was only proven on non-conflicting keys | **Fixed.** `TestQuietOutranksAConflictingCallerPair` pins `gc.auto=1` and `maintenance.auto=true` as caller pairs and asserts the child reads Quiet's values and `GIT_CONFIG_COUNT == 2+len(quietPairs)`. The `proposal.md` sentence is now true as written, so no contract-set edit was needed. |
| Minor — `quietSystem()` ran before the count was validated, so a refusal was not side-effect-free | **Fixed.** `callerPairs()` runs first; `TestQuietRefusesBeforeItTouchesTheEnvironment` asserts `GIT_CONFIG_SYSTEM` is unchanged after each refusal. |
| Minor — `callerPairs` allocated `n × 32B` from a caller-controlled count (`1e9` -> ~32 GB) | **Fixed.** Pairs are appended, not preallocated, and reading stops at the first pair the environment does not carry — which is also what git refuses. The `huge count` subtest would have OOMed under the old code. |
| Minor — the full `internal/cmd` suite never completes on this host | **Dispositioned, not fixable here.** Documented in the suite note; CI is the authoritative full-suite check. Cause is the global `core.hooksPath`, #3199 territory, pre-existing. |
| Question — `strconv.Atoi` accepts forms git's parser may read differently (`+3`, `03`) | **Declined.** The reviewer classed it speculative and explicitly said not to gate on it. `03` reads as 3 in both (git parses base 0); `0x3` would differ, and no harness emits one. Widening the parser to match `strtol` base 0 is a change with no observed failure behind it. |
| Process — `review-request.json` recorded `base_sha: 8881f845`, the fix commit, not the merge-base | **Not a defect in the resolver.** `ResolveReviewBaseDetail` returns the parent of the commit that *added* the spec folder, which is the documented contract (BUG-108, CLI-132). The folder landed in the second commit because the pre-push gate demanded a spec only after the fix existed. The prompt states the base, and this reviewer widened to `0b1eb901` from it. The fix is procedural: `dotf spec init` before the first code commit. |
| Process — a foreign `stash@{0}` from `feat/mise-windows` is present in the repo | **Left alone deliberately.** Stashes live in `refs/stash`, shared by every worktree, and that entry is someone's unlanded work from 2026-10-12. Dropping it is the owner's call, not a reviewer's or this change's. |

### Suite note: why the whole package cannot finish here, and why that is not this change

`go test ./internal/cmd` exceeds Go's default 10-minute per-binary timeout on this
machine (observed twice at 601s, `panic: test timed out after 10m0s`). The dump
names the test in flight, and it is not a hang: this host's global
`core.hooksPath` is `.../dotfiles/git-hooks`, so every `git commit` a fixture
makes runs the repo's full hook chain. One test,
`TestHarnessRefresh_AFailedRenderIsAnErrorAndWritesNothing`, takes 46.6s alone
(passes), and ~230 tests at that cost is the whole budget.

Measured, not assumed: the same single test passes in 46.62s with the
`GIT_CONFIG_*` variables cleared, so the cost is the hook chain, and it predates
this change — `Quiet` never touched `core.hooksPath`, and the environment it now
merges is not what makes hooks run. It is #3199's subject (hook process cost on
Windows), not a regression here, and neutralizing `core.hooksPath` inside
fixtures belongs to the fixtures, not to `gittest`.

The acceptance criterion is therefore evidenced by the run reaching the tests at
all (it used to abort in `TestMain`) plus the package-start predicate in
`features.json` f4, not by the package finishing.

## Review dispositions (round 2 — PR review bots on #3254)

Round 2 has not run yet as `dotf spec review`; these are the two bots that
reviewed the PR, triaged and applied before the second adversarial pass so it
sees the corrected contract set.

| Finding | Disposition |
|---|---|
| CodeRabbit **Minor** — `slices.Contains(caller, pair)` skips the append when the pair appears *anywhere*, so a caller pinning `gc.auto=0` at one index and `gc.auto=1` at the next leaves `1` in force and background gc on | **Fixed.** `lastValue()` resolves the caller's last pair for the key, and `Quiet` appends unless that value is already its own — which also preserves the re-execution dedupe, since a re-executed binary's own pairs are last. `TestQuietOutranksACallerThatSetsTheSameKeyTwice` covers it, and the A/B above shows it failing against the old check. |
| CodeRabbit **Major** — f4's `-run '^$'` selects no tests, so it cannot support a claim that the full `internal/cmd` suite runs, which `proposal.md` AC1 made | **Fixed by narrowing the claim, not the predicate.** The bug was `TestMain` aborting the binary, so startup is the regression surface; `proposal.md` AC1, `features.json` f4 and the Criterion 1 evidence now say startup-only and name CI as the owner of the full run. A predicate that took 40 minutes on this host would be a worse artifact than an honest one. |
| pr-agent **REAL** — `quietSystem()` ran before the count was validated, so a refusal was not side-effect-free, and the refusal test passed on *any* panic | **Already fixed in `fc3f37c4`** (the bot reviewed `f87a8731`), and completed here: the refusal test now asserts the panic message names `GIT_CONFIG_COUNT`, so it can no longer pass on an unrelated panic. |
| pr-agent **THEORETICAL** — the `slices.Contains` precedence assumption | **Already fixed in `fc3f37c4`**, and hardened further by the CodeRabbit fix above. |
| pr-agent **THEORETICAL** — `make([][2]string, n)` from a caller-controlled count | **Already fixed in `fc3f37c4`.** |
| Both bots class the duplicate-key trigger as unrealistic ("requires a harness that sets the same key twice") | **Severity accepted, skip rejected.** The guard's contract is "each git this process starts is quiet"; it should hold for the input rather than for today's harnesses, and the fix was one helper plus one test. |

## Review dispositions (round 3 — `review.md`, verdict PASS-WITH-GAPS)

Signed by `nan/deepseek-v4-flash` against `670e6529`. No Blocker, no REAL Major,
no failing rubric item (Correctness B, Verification B, Scope A, Reliability B,
Maintainability A, Handoff-readiness B). It was the third launch: the first
stalled at the default 10m transcript window and the second was stopped at the
45m deadline mid-analysis. Neither wrote a verdict, and neither provisional is
kept here — an unsigned PENDING artifact with no findings is not a review, and
the transcript for both is gitignored. The third finished inside
`--timeout 75m`.

| Finding | Disposition |
|---|---|
| **Major (THEORETICAL)** — `Quiet` refused two environments git reads without error: `GIT_CONFIG_COUNT=""` (git: zero pairs) and `GIT_CONFIG_COUNT=" 3"` (git skips leading whitespace), so the proposal's "git would read the same number and fail" was false for those two forms. `UNTESTED`. | **Fixed.** git's parser was probed directly rather than assumed — 13 forms, one pair present, `git config --get-all` as the oracle: `""` reads as zero pairs and resolves nothing (exit 1, no output); `"1"`, `" 1"`, `"\t1"`, `"+1"`, `"01"`, `"0001"` all resolve the pair; `"1 "` is `error: bogus count in GIT_CONFIG_COUNT` (exit 128); `"-1"` is `error: too many entries` (exit 128); `"0x1"`, `" "`, `"abc"`, `"1a"` are bogus counts. `parseCount` now reproduces that: empty is zero, one `Atoi` after `TrimLeft` of the C whitespace set, and a refusal on error or a negative. `TestQuietReadsTheCountTheWayGitDoes` is differential — it asks git per form and asserts `Quiet` agrees — so the table cannot drift from git the way a hand-written one would. A/B control: with `parseCount` reverted to a bare `strconv.Atoi(raw)` the test fails on exactly `""`, `" 1"` and `"\t1"`, and the file was restored byte-identical afterwards. |
| **Minor (REAL)** — the missing-pair refusal always named `GIT_CONFIG_KEY_<i>`, so an absent `GIT_CONFIG_VALUE_0` produced "but GIT_CONFIG_KEY_0 is not set". | **Fixed.** The single `!hasKey || !hasValue` branch is split, and each names the variable that is actually absent. `TestQuietNamesTheVariableThatIsMissing` covers both halves and asserts the message contains the expected name, so it fails on the old single-branch form. |
| **Minor (REAL)** — `ResolveReviewBaseDetail` returns the parent of the commit that *added* the spec folder, which lands after the implementation whenever the spec is committed later than the first code commit, hiding the production diff. Both rounds had to widen the base by hand. | **Deferred to its own ticket, not fixed here.** `cli/internal/spec/review_request.go` is a different subsystem with its own tests and its own failure modes; changing it inside a `gittest` bugfix would put an unreviewed behaviour change in a PR whose verdict covers only the `gittest` diff. The reviewer's own note agrees ("not this change"). The procedural workaround — `dotf spec init` before the first code commit — is recorded in the round-1 table above, and the prompt already states the base so a reviewer can widen from it. |
| **Minor (REAL)** — `tasks.md` says the round-1 verdict was recorded in `review.md`, but the harness renamed it to `review-round1.md`. | **Accepted into the archive.** `tasks.md` is a contract file: editing prose in it changes its digest and invalidates the round-3 verdict, forcing a fourth ~50-minute round to fix a stale filename in a historical note. The accurate pointer is this document, which is not in the contract set and is read first at archive time. |
| **Minor (REAL)** — the PR head was `f87a8731`, two commits behind, so the green CI never ran the reviewed code, and the PR body still claimed 9 tests. | **Resolved by the push in this change**, together with a PR-body refresh: 14 `TestQuiet*` tests now, and the reviewed SHA is the branch head. |

## Decisions made during implementation

- **Append, not overwrite and not refuse.** Overwriting would drop keys a harness
  relies on (Copilot CLI pins `safe.bareRepository=explicit` and
  `credential.interactive=never`); refusing aborts the binary, which is the bug.
  Git applies pairs in index order, so appending preserves both the caller's keys
  and `Quiet`'s precedence.
- **Index arithmetic, not a merge of the map.** `Quiet` writes at
  `len(caller)+added` so it never lands on an index the caller uses. A caller
  whose count promises pairs the environment does not carry is **refused**, not
  guessed at: git refuses that environment itself, and inventing the missing
  pair would silently set config the caller never asked for.
- **`alreadyQuiet()` deleted.** The fast path is now "the caller's last pair for the key already carries our value", which is what a re-executed test binary hits; keeping both would have left two notions of "already quiet" to keep in sync.
- **`cleanEnv` clears by prefix.** A test that asserts what `Quiet` wrote needs a
  known starting point, and a harness may have pinned pairs there. Clearing
  `GIT_CONFIG_COUNT` alone would have left orphaned `KEY_i`/`VALUE_i` variables.

## Promotion candidates

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-517-git-background-work-is-stopped-in-the-environment-not-the-fixture-repo.md` (Solution paragraph rewritten: append semantics, cites #3253).
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: a package-internal helper's merge policy is a code decision, recorded here and in the lesson, not an architectural commitment.
- [x] New pattern candidate for `00_meta/patterns/`? no: it recurs inside this package only; the cross-project shape ("a test helper must extend the environment, never own it") is already covered by lesson 517.

## Archive checklist

- [x] `proposal.md` frontmatter set to `status: archived`
- [x] Folder moved: `specs/BUG-3253-gittest-quiet-appends-inherited-config/` -> `specs/archive/BUG-3253-gittest-quiet-appends-inherited-config/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [x] Promotions above executed (if any)
