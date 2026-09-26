---
id: lesson-303
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, doctor, harness, testing, differential]
---

# 303 — A check that predicts another component uses that component's rule, not a better one

## What happened

GUARD-006's doctor check predicts whether the next deploy can render each agent record. It decided which harnesses a record targets with an exact match on the entries of `targets:`. The render it predicts, `skill_targets_agent` in `compile-harness.sh`, matches the harness name anywhere on the first `targets:` line. The two disagreed on legal records. On `["opencode"]` the check skipped a harness the render deploys to, which hid real drift. On a block-style list the render deploys nowhere, but the check FAILed a record that cannot fail a render. The retroactive review failed the spec on it. Its second round found the same gap on the `model:` key (CRLF, `model : top`, an indented `---`), which is #1740.

## The rule

When a check exists to predict what another component will do, it must apply that component's rule, including the rule's flaws, and ticket the flaws against the component. A more reasonable rule in the check is a check that is wrong. Hold the two together with a differential test that runs the real predicate against the check's on the same inputs (`TestRecordTargetsAgreesWithTheRender`). Then fixing either side turns the test red until the other follows.

Refs: GUARD-006, #1733, #1740.
