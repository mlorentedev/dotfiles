---
tags: [spec, verification, templates]
created: "2026-11-10"
id: "AI-054-review-budget-guard"
type: spec-verification
status: active
owner: manu
---

# Verification - AI-054-review-budget-guard

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof (commit hash, test name, or observed behavior).

- [x] AC1 (token clip) -> `TestCheckReviewBudget_TheCurrentConfigurationFails`,
  `TestCheckReviewBudget_TheTokenClipNeedsRoomForTheAnswer`,
  `TestCheckReviewBudget_TheClipMustLeaveSlackNotJustFit` (160000 fails, 159000
  passes — the boundary is `<`, not `<=`) and
  `TestCheckReviewBudget_AStepThatDeclaresItsOwnClipIsCheckedToo` (the env
  override is read). Observed on the real repo after the fix:
  `[check] OK -> prompt clip in .pr_agent.toml: 150000 + 40000 < 200000`, plus
  one line per Anthropic step that re-declares `CONFIG__MAX_MODEL_TOKENS`.
- [x] AC2 (step ceiling) -> `TestCheckReviewBudget_TheCurrentConfigurationFails`
  (720 s in a 480 s ceiling), `TestCheckReviewBudget_RetryFalseHalvesTheAttempts`
  (the same numbers pass once the retry is declined) and the undeclared
  `ai_timeout` finding. Observed after the fix:
  `720 s of attempts in a 780 s ceiling` on both Anthropic steps and
  `360 s of attempts in a 720 s ceiling` on the NaN step, where the `120` is now
  stated rather than inherited.
- [x] AC3 (job ceiling) -> not in the Go guard by decision, recorded in
  `checkReviewBudget`'s doc comment and pinned by
  `TestCheckReviewBudget_TheGuardLeavesTheJobCeilingToTheBatsTest`; the number
  itself is `tests/pr-agent-config.bats:1189`, red at `False 34 25 5 3` when the
  step ceiling grew and green at `True 37 25 5 3` after the job moved to 37;
  after the audit it counts the two Anthropic probes: `True 38 25 6 3`.
- [x] AC4 (fail closed) -> `TestCheckReviewBudget_AWorkflowWithoutAReviewStepFailsClosed`,
  `TestCheckReviewBudget_HalfAReviewerFailsClosed` (a workflow with the toml
  missing, or the workflow missing, is an `[ERROR]`, not silence) and
  `TestCheckReviewBudget_ACheckoutWithoutAReviewerSaysNothing` (neither file,
  no lines — the behaviour `TestCheckDrift_ACleanCheckoutPasses` already relied
  on).
- [x] AC5 (runbook) -> `docs/runbooks/guide-pr-agent-reviewer.md`: the clip and
  the window are now separate rows (`150000` / `200000` with the room stated),
  `CONFIG__AI_TIMEOUT` is `360` and `120` with the streaming difference named,
  and the step/job rows are `13` and `37` with the sum. No `31-minute`,
  `200000`-as-clip or `8-minute` figure remains in the file.
- [x] AC6 (surfaces agree) -> the bats parity test prints `True True True` for
  cap-for-cap across providers plus `cap + output cap < window`; the routing
  test asserts `13` on both Anthropic steps. Both read the same numbers the Go
  guard reads.

## Test status

- `go test ./internal/harness/` -> `ok github.com/mlorentedev/dotfiles/cli/internal/harness 10.643s`
- `go test ./internal/harness/ -run Review -v` -> 12/12 pass, including
  `TestCheckReviewBudget_TheRepositoryPassesItsOwnGuard`, which runs the guard
  against the real `.pr_agent.toml` and `pr-agent.yml` rather than fixtures.
- `go build ./...` clean, `go vet ./internal/harness/` clean, `gofmt -l` clean on
  both new files (the package's pre-existing CRLF noise aside).
- `go run ./cmd/dotf harness check --repo-root ..` -> exit 0, `[check] OK: no
  harness drift`, with the six review-budget lines quoted above.
- **Red before the fix**, observed on the real configuration: six findings —
  the toml clip, the two `CONFIG__MAX_MODEL_TOKENS` step overrides, the two
  Anthropic step ceilings and the undeclared NaN `ai_timeout`. Exit non-zero.
  The shape is pinned by `TestCheckReviewBudget_TheCurrentConfigurationFails`,
  which asserts four findings on a three-step fixture (no step overrides there).
- bats: the file cannot run on this Windows box (`bats.cmd` produces no output
  for minutes and was killed), so every `run python3 -c` block in
  `tests/pr-agent-config.bats` was extracted and executed directly, comparing
  each `[ "${lines[N]}" = ... ]` assertion: **pass=20 fail=0 skipped=4**. The
  four skips read shell locals built earlier in their `@test`
  (`$has_exclusion`, `$GATE`, `$SUITE`) and are unrelated to timeouts or tokens.
  The job, routing, bounded-in/out and step-order tests are among the 20.
- No regressions: the full `internal/harness` package is green, and
  `dotf harness check` reports no drift on the repository.

## Decisions made during implementation

- **The job ceiling is not duplicated in Go.** The first version modelled the
  job as "the two largest step ceilings plus 300 s of setup" and passed at 34 on
  a job that needs 36 — a guard agreeing with the broken number, which is the
  defect class this spec exists to remove. `jobCeiling`, `prWorstPath`,
  `prSetupSeconds` and their test were deleted; the bats test owns the
  arithmetic because it reads the probe timeouts where they are set. A test now
  pins the guard's silence so the weak duplicate cannot return.
- **150000, not the 120000 the proposal first stated.** Raising
  `custom_model_max_tokens` to 260000 is arithmetically fine but amends a
  documented owner decision (200000 is "a deliberate floor well inside each");
  120000 clips 30000 earlier for no benefit, because the answer cap is reserved
  above the clip rather than out of it. Measured prompts here peak near 97000
  tokens (AI-045 AC9), so nothing that runs today gets clipped.
- **The job ceiling moved to 38 in the same change as the step ceilings.** The
  pair needs 25 of the 34 available minutes and the probes, setup and publication
  calls take the other 11; growing a step ceiling spends slack that was already
  gone. The step fix and the job fix are one fix.
- **The bats parity test was rewritten, not adjusted.** It asserted the prompt
  cap equalled the window — the inert state, written down as the invariant — so
  it would have failed the fix. It now asserts parity across providers plus the
  room the answer needs.
- **The NaN step's `ai_timeout` is stated, not changed.** `120` is what it
  already ran; the finding was that no file in this repository says so.

## Promotion candidates

Answer each line `yes: <path>`, naming the file you promoted, or `no: <reason>`. `dotf spec archive` refuses a line left unanswered, a `no` without a reason, and a `yes` whose file does not exist; a `00_meta/` path is looked up in the vault.

- [x] Lesson for the repo's `docs/lessons/`? yes: `docs/lessons/lesson-573-a-budget-that-cannot-fire-is-not-a-budget.md`
- [x] ADR-worthy decision for the repo's `docs/adr/adr-XXX.md`? no: the decision (guards live in the Go CLI, per ADR-020) already exists; this adds an instance of it, and lesson-573 carries the reasoning.
- [x] New pattern candidate for `00_meta/patterns/`? no: the transferable rule — a limit must be compared against a ceiling it can reach, and a guard must not use a bound looser than the real one — is a review reflex, not a workflow; it belongs in the lesson, and the vault would get a one-paragraph pattern out of it.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/AI-054-review-budget-guard/` -> `specs/archive/AI-054-review-budget-guard/`
- [ ] Bitácora board ticket for this spec moved to Done / closed with PR link (ADR-018)
- [ ] Promotions above executed (if any)
