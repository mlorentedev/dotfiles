---
id: "ADR-064-a-failed-converge-step-holds-back-only-its-dependents"
type: adr
status: accepted
owner: manu
date: "2026-11-16"
issue: "mlorentedev/dotfiles#3554"
extends: [adr-041-update-channels-and-convergence-order, adr-045-one-entrypoint-converges-every-os]
tags: [architecture, decision, converge, self-deploy]
created: "2026-11-16"
---

# ADR-064: a failed converge step holds back only the steps that need it

## Status

Accepted 2026-11-16. The owner chose option 1 on #3554 and delegated the implementation. Decisions 5, 6,
7 and 10 below (the default for an undeclared step, `legacy-setup`, keeping uncertain edges, the
revisit) were settled in the session that wrote this ADR. Each stands until the owner reverses it,
which takes one line on #3554. The spec is `specs/BUG-173-converge-step-dependencies/`.

## Context

`dotf converge` runs an ordered registry of 24 reconcilers. Until now the first failure stopped the
run, and every later step was reported `not reached`. That rule came from ADR-041 decision 4: the
binary never moves unless the records were mirrored in the same run.

The rule was written for three steps and applied to all of them. On 2026-11-16 `system-upgrade`
failed on a Windows workstation because winget could not upgrade three machine-wide packages without
elevation (#3553). It is step 6. The 18 steps after it did not run: git config, git hooks, lanes, the
SSH target, the agent extensions and the setup script. None of them reads anything `system-upgrade`
writes. Every `dotf update` and `dotf converge` run on that machine ended the same way.

#3555 makes that one failure a report. The class stays: any step can fail for a reason local to it
(no network for a plugin, a scheduler that refuses a task), and the order of the registry then
decides how much of the machine is left unconverged.

## Options considered

1. **Steps declare what they need; a failure holds back only its dependents.** Chosen.
2. **Order the leaf steps last.** The order would then carry two meanings, which records precede
   which, and which failures are tolerable. Two leaf steps would still block each other, and
   `agent-extensions` already sits last for this reason while eleven other leaves do not.
3. **Keep fail-fast and fix each failure as it appears.** #3555 is that fix for one step. Every new
   reason a step can fail costs a night of unconverged machines before it is found.

## Decision

1. **The order does not change.** The registry is still one ordered list and steps still run in it.
   ADR-041 decision 4 and ADR-045 decisions 4 and 5 keep their order.
2. **A step declares the earlier steps it needs** (`Needer`, `cli/internal/converge/needs_registry.go`).
   A need is an earlier step whose failure makes running this one unsafe: it reads what that step
   writes, or a decision says it must not run without it.
3. **A failed step holds back the steps that need it, directly or through another.** They are
   reported `[SKIP] not reached: <step> failed`, or `not reached: needs <step>, <step> failed` through
   a chain. Every other step runs.
4. **The run fails if any step failed.** The exit code is non-zero and the error names every failure,
   one per line. A run with a failure is never reported as converged, and the identity epilogue that
   follows a converged run does not run.
5. **A step that declares nothing needs every step before it.** That is the previous behaviour, so an
   undeclared step is safe by default. A test requires every production step to declare.
6. **`legacy-setup` declares nothing, on purpose.** The setup script re-runs native work and ADR-045
   decision 5 runs it after every native step. Any failure still holds it back.
7. **An uncertain edge is kept.** A missing edge runs a step against a prerequisite that failed. An
   extra edge only reports a step as not reached. An edge is removed with the code that shows the
   step does not read what the other writes, recorded in the spec.
8. **The floor from the earlier ADRs is declared, not implied:** every step needs `checkout`;
   `records-harness`, `env-generate` and `tools` need `records-mirror`; `tools` needs
   `records-harness`, so no agent is installed before its instructions exist. A test fixes what each
   of `checkout`, `records-mirror` and `records-harness` holds back.
9. **A need outside the run does not hold anything back:** a step left out by `--only`, one that
   skipped, one this OS does not support. `--only` already ran a step without its predecessors.
10. **A step that failed is not applied again** when a later step changes the machine (the revisit of
    ADR-045 decision 11). A revisit that fails holds back that step's dependents from then on.
11. **A plan's gate is unchanged:** it still skips every step after it.

## Consequences

- A failure in a step nothing native needs costs that step and `legacy-setup`. On the machine in
  #3554 that is 2 steps instead of 19. The setup script is therefore still not run there after such a
  failure: decision 6 is the one to reverse if that is not wanted.
- A failure in `tools` still holds back most of the registry, because most steps use what it
  installs. Narrowing those edges is future work, one edge at a time, each with its evidence.
- More than one step can fail in a run. The report, `last.json` and the failure summary already hold
  several failed entries. `dotf doctor` prints the first line of a recorded error and counts the rest.
- A report can show a `not reached` step between steps that ran. A reader that assumed everything
  after the first `[FAIL]` was skipped is wrong now; none was found outside the tests.
- Adding a step means deciding what it needs. A step added without a declaration fails a test.

## References

- #3554, #3553, #3555
- ADR-041 decision 4; ADR-045 decisions 4, 5 and 11
- `specs/BUG-173-converge-step-dependencies/`
