---
id: "AI-055-budget-guard-readable-reserve"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-11-11"
issue: "mlorentedev/dotfiles#3301"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
owner: manu
wip_override: "Fixes #3301: the guard declares a reserve it cannot read, so a step with no answer cap reports OK. Lanes are also 11 deep, which is why the limit fires; this is a bounded rescue of the AI-054 follow-up. (10 active, limit 10, 2026-11-11)"
---

# AI-055-budget-guard-readable-reserve

## Why

`dotf harness check` enforces that the prompt clip leaves room for the answer
inside the context window, but it read that answer from the workflow with
`largestOutputCap`, which returns `0` for a step that declares no
`CONFIG__MAX_OUTPUT_TOKENS` or declares one it cannot parse. `0` is then the
amount of room the check insists on, so the comparison `clip + 0 < window`
passes on the widest clip the model will accept — the guard declared a reserve
it could not read and reported the step as healthy (#3301, found in review of
AI-054). The same guard fails closed on an expression `ai_timeout`, so the two
checks disagreed on which direction uncertainty should point.

The same AI-054 review found the arithmetic on the NaN step is a lower bound,
not a ceiling: `ai_timeout = 120` there bounds the gap between streamed chunks,
not the answer, so `120 s × 3 attempts in a 720 s ceiling` reads as coverage
when the step can run past it. That is #3302, and this PR makes the line say so
instead of leaving the reader to know it.

## What

- A `CONFIG__MAX_OUTPUT_TOKENS` the check cannot read as a number is a **GAP**
  naming the step and the raw value, and no clip line prints for that run.
- A workflow where **no** review step declares the cap is a GAP naming the
  variable, instead of a reserve of zero that passes every clip.
- A step whose call streams carries `(streamed: …)` on its OK line, stating that
  the ceiling bounds the gap between chunks and not the answer, so the number is
  a lower bound on that step's wall clock.
- A step that declares no `timeout-minutes` says exactly that, instead of being
  measured against a `0 s` ceiling.
- Unchanged: the reserve is still the **largest** declared cap when steps
  disagree, and every fail-closed path from AI-054 still reports a problem.

## Out of scope

- Measuring the streamed step's real wall clock; #3302 owns that and this PR
  only stops the line from overstating what it covers.
- The job/workflow-level token override skip recorded on #3301 (a step-level
  expression is covered here; the workflow-level case is not).
- Any change to `.pr_agent.toml`, the reviewer pool, the workflow's numbers or
  the bats job-ceiling test.

## Risks / open questions

- **A cap-less step now turns the check red.** Every step in this repository
  declares one, so the repo passes; a future step that relies on the image
  default will be told rather than silently trusted, which is the point.
- **The `timeout-minutes` gap is not in either ticket.** It is part of the same
  defect class (a zero read as a measurement) and rides here so the message is
  correct rather than misleading; it is called out in the PR body.

## Acceptance criteria

- [ ] [AC1] A declared answer cap that is not a number is reported as a GAP naming the step and the value, and no clip line prints.
- [ ] [AC2] A workflow in which no review step declares the answer cap is reported as a GAP naming the variable.
- [ ] [AC3] When steps declare different caps, the reserve is the largest (property preserved from AI-054).
- [ ] [AC4] A step whose call streams has `(streamed: …)` appended to its OK line, and a non-streaming step does not.
- [ ] [AC5] A step with no `timeout-minutes` is reported as declaring none, not measured against `0 s`.
- [ ] [AC6] This repository's committed configuration still passes, and the AI-054 fail-closed paths are unchanged.

## References

- Bitácora: mlorentedev/dotfiles#3301 (this PR), #3302 (streamed ceiling, measurement)
- Supersedes the spec-less commit `5cfc1f54` from the retired `feat/review-budget-guard` branch
- Predecessor: `specs/archive/AI-054-review-budget-guard/` (round-2 PASS-WITH-GAPS)
- Pattern: `00_meta/patterns/pattern-spec-driven-development.md`