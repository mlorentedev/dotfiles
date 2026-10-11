---
id: "lesson-372-a-dry-run-proves-the-launch-not-the-request"
type: lesson
status: active
title: "A dry run proves the launch, not the request"
created: "2026-10-09"
---

# A dry run proves the launch, not the request

## Context
#2188 (AI-045 amendment B, ADR-046) added an `anthropic-review` provider to `ai/pi/models.json`.
It lets `dotf spec review --second` and `--fallback-reason` run Claude Haiku 5.5 or Sonnet 5.5
under pi, with `REVIEW_ANTHROPIC_API_KEY`. Before the merge, two checks passed: a
`--fallback-reason --dry-run` that resolved pi, the provider and the model, and a curl probe
that got HTTP 200 from both models with the key.

## The Trap
Each check covered only half of the path. The dry run proved the launch command, and the curl
probe proved the key. Neither one sent the request that pi builds. The first real pi call
after the deploy failed with a 400 on every attempt:

`"thinking.type.enabled" is not supported for this model. Use "thinking.type.adaptive"`.

pi 1.1.0 (`pi-ai/dist/api/anthropic-messages.js`) picks the thinking shape for each model, not
for each API. It sends adaptive thinking only when the model declares
`compat.forceAdaptiveThinking: true`. A model without that flag gets the budget form, which
Claude 5.x rejects. pi's built-in Claude models carry the flag. A custom model in `models.json`
must declare it itself.

## The Solution
Declare `"compat": {"forceAdaptiveThinking": true}` on both `anthropic-review` models, in the
repository source that `dotf deploy` renders. Never patch `~/.pi` by hand.

`tests/pi-config.bats` now requires the flag on every reasoning model that uses the
`anthropic-messages` API. If an older Claude model that needs budget thinking is added, the
test has to make an explicit exception for it.

The fix was verified on the real request, in a throwaway `PI_CODING_AGENT_DIR`, before anything
was deployed:

- With the flag, both models answered.
- Without the flag, the same call reproduced the 400.

## Takeaways
- **A smoke test of a new model path sends the real request through the real client.** A probe
  that takes another route (curl in place of pi) proves the credential and nothing about what
  the client puts on the wire.
- **`PI_CODING_AGENT_DIR` is the test seam for pi configuration.** Point it at a scratch
  directory that holds only the `models.json` under test. Running the fix and the control side
  by side shows the flag is the cause, without touching the deployed agent.
- Running pi without `NAN_API_KEY` prints `No models match nan/...` warnings, because
  `enabledModels` names NaN models that only register when the key is present. A review that
  names its model explicitly is unaffected. The warnings are noise, not this failure.
