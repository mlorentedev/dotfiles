---
id: "lesson-573-a-budget-that-cannot-fire-is-not-a-budget"
type: lesson
status: active
title: "A budget only protects something if its ceiling is reachable, and a guard that agrees with a broken ceiling is worse than none"
created: "2026-11-10"
---

# A budget only protects something if its ceiling is reachable, and a guard that agrees with a broken ceiling is worse than none

## Context

The pr-agent reviewer is a workflow with three review steps and a job that runs
them, and every layer carries a limit: a prompt clip in `.pr_agent.toml`, a
per-call `ai_timeout`, a per-step `timeout-minutes`, and a job
`timeout-minutes`. The same configuration, copied to
`mlorentedev/svqtriana.github.io` without one override, killed run
`42562380120` — the job ran out of budget and published no review at all.

Reading the four layers against each other found four distinct shapes, none of
which any test or reviewer had asked about:

1. **The clip equalled its own ceiling.** `max_model_tokens = 200000` and
   `custom_model_max_tokens = 200000`. The clip exists to cut a prompt before it
   overflows the model window; set equal to the window it has no reachable
   subject. `200000 + 40000 = 240000 > 200000` — a 240000-token request cannot
   exist, so the setting was inert, and every comment around it described
   protection.
2. **The ceiling was shorter than the budget it covered.** Both Anthropic steps
   allowed two attempts of `ai_timeout = 360` inside `timeout-minutes: 8` — 720 s
   of attempts in a 480 s box. The second attempt could never complete.
3. **The value was inherited, not chosen.** The NaN step declared no
   `CONFIG__AI_TIMEOUT`, so it ran the image's 120 s default. The arithmetic
   happened to pass; nothing in the repository said the number, and nothing
   would notice it changing upstream.
4. **The job bound was one minute short of the truth.** Worst pair = 13 + 12 =
   25 min, plus 5 min of preflight probes, 3 of setup and 3 of publication calls
   = 36, against `timeout-minutes: 34`. The ANTHROPIC ceiling growing from 8 to
   13 is what pushed it over — the step fix and the job fix are the same fix.
   An independent audit of the change then found the 5 one probe short: the
   route probes the drawn Anthropic model and, on a risky PR, the risk model
   too (`route.go`), 60 s where the test counted 30, so the sum is 37 and the
   job is 38. The bats test now reads the probed models from the pool.

## Problem

Each shape reads as protection and asserts nothing, and the reason is the same
in all four: **the number was never compared against the thing it bounds.** A
comment saying "the job's 31 minutes cover the worst pair" is a claim about
arithmetic in prose, and prose cannot fail.

The worst part of this episode is the fifth: the guard written to catch the class
had the defect itself. Its first version checked the job as "the two largest step
ceilings plus 300 s of setup" — 26 minutes — and reported **pass** at 34 on a job
that needs 36. It read like coverage, it ran on three CI platforms, and it would
have blessed the broken ceiling forever. A test written by hand at the same time
(bats, from the numbers where they are set) said `False 34 25 5 3` and was the
only thing that disagreed.

Two further traps surfaced while fixing the tests rather than the code:

- **A parity test can codify the bug it looks like it is gating.** The bats test
  asserted the prompt cap equaled the window — the inert state, written down as
  the invariant. It had to be rewritten to assert cap-for-cap across providers
  plus the room the answer needs, or it would have failed the fix.
- **An env override restores what the file fix removed.** The clip is declared
  both in `.pr_agent.toml` and as `CONFIG__MAX_MODEL_TOKENS` in two steps. Lower
  the file alone and the runs still clip at the old number; the arithmetic in
  your head is right and the pipeline ignores it.

## Solution

Forget fixing numbers first — write the comparison, then let it fail.
`cli/internal/harness/prbudget.go` runs from `dotf harness check` on every CI
platform and fails when either invariant stops holding:

- `max_model_tokens + <largest declared output cap> < custom_model_max_tokens`
  (strict `<`), checked on the file **and** on every step that re-declares the
  clip;
- `chain x attempts x ai_timeout <= timeout-minutes x 60` per step, where
  `attempts` is 1 only if the step declares
  `CONFIG__RETRY_SAME_MODEL_ON_TIMEOUT: "false"`, and a step that declares no
  `ai_timeout` is its own finding.

It went red on the real configuration with six findings, the values changed, and
it went green. Then the job: `tests/pr-agent-config.bats` already computed the
job's sum from the probe timeouts themselves, and it went red the moment the step
ceiling grew. The Go job check was **deleted** rather than repaired — reaching
the true pair needs the route's `if:` guards and the probe constants, and the
bats test already has them where they are set. Keeping a weaker duplicate leaves
two answers a minute apart, and the weaker one is the one people read.

## Rule

Before trusting a limit, ask what it is compared against, and whether the other
side of that comparison can be reached:

- A limit equal to its own ceiling is decoration. Give it slack, and assert the
  slack (`<`, not `<=`) so it cannot silently become equal again.
- A ceiling that does not cover the sum of what it contains is a coin flip on
  whether work finishes. Sum the content, don't eyeball it.
- A value the code uses but the repository does not state is a number someone
  else chose for you.
- **Never write a guard whose bound is looser than the real one and call it
  coverage.** The only guard worth having models the same quantity as the thing
  it protects, or it agrees with the bug. And when two surfaces can compute the
  same number, keep the one that reads the inputs where they are set and delete
  the other.

## References

- `cli/internal/harness/prbudget.go`, `cli/internal/harness/prbudget_test.go`
- `.pr_agent.toml`, `.github/workflows/pr-agent.yml`
- `tests/pr-agent-config.bats` (the job arithmetic and the step ceilings)
- `docs/runbooks/guide-pr-agent-reviewer.md`
- `specs/archive/AI-045-nan-catalog-alignment/` (the prompt-size measurement)
- mlorentedev/dotfiles#3281, svqtriana run `42562380120`
- lesson-371 (a paid model is bounded by every layer that supplies a default)
