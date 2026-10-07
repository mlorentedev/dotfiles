---
id: "lesson-335-a-converge-step-is-stubbed-in-the-shared-test-helper"
type: lesson
status: active
title: "A converge step that shells out is stubbed in the shared test helper"
created: "2026-10-04"
---

# A converge step that shells out is stubbed in the shared test helper

## Context
A bare `dotf deploy` runs converge steps after the declared configs: the Orca hook tune
(CLI-093) and, since CLI-063, `claude plugin install` for every plugin the box lacks. The
step gates itself on the tool being on `PATH`, so it does nothing on a CI runner.

## The Trap
The `PATH` gate passes on every developer box, because the developer box is where the tool
lives. The existing bare-deploy tests (`TestDeployCmd_NoArgInstallsEveryDeclaredConfig` and
others) call `runDeploy` with no arguments. Once the plugin step was added, those tests would
have run the real `claude plugin list` and `claude plugin install` against the developer's
own `~/.claude`. CI never sees this, since no runner has `claude`. So the failure shows up
only on the one machine where it does damage, and it looks like a passing test.

## The Solution
Put the seam in the shared helper and default it to "tool not installed":
`runDeploy` calls `runDeployWithClaude(..., nil)`. Only tests of the step itself pass a fake
runner. A test that should drive the step has to opt in, and every other test stays off the
real tool without knowing the step exists. Any new converge step that shells out to a tool
developers have installed gets the same default in the same helper.
