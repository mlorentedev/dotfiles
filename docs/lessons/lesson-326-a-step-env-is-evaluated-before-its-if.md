---
id: "lesson-326-a-step-env-is-evaluated-before-its-if"
type: lesson
status: active
title: "A step env is evaluated before its if, so parsing a skipped step output throws"
created: "2026-10-01"
owner: manu
tags: [lesson, github-actions, ci, pr-agent]
---

# A step env is evaluated before its if, so parsing a skipped step output throws

## What happened

#1914 added a retry step to PR-Agent's workflow. Its model came from `${{ fromJSON(steps.models.outputs.fallbacks)[0] }}` in the step's `env`, and its `if:` only let it run when the first attempt failed. Every test passed, and the first live run reviewed the PR.

Then "update branch" pushed a merge commit. The push gate does not review merge commits, so the preflight step (`models`) was skipped and its outputs were empty strings. The retry's `if` would have been false. But GitHub evaluates a step's `env` before its `if`, which is why `if: env.X` can read the step's own `env`. So `fromJSON('')` ran and threw "Error reading JToken from JsonReader". The step failed, the publish guard's `env` threw the same way, and the `review` job went red on a push nobody meant to review (run 36812454370).

## Rule

- Never `fromJSON` a step output inside an expression. When the producing step is skipped, the output is `''`, and the `env` that parses it is evaluated anyway. Have the producer emit the scalar the consumer needs (here `retry_model=`, possibly empty), and compare it with `!= ''`.
- A skip path is a path. The tests read the workflow file statically, so they could not see this; only a run where the producer was skipped could. When a step's inputs come from an optional step, ask what they are when that step does not run.

## Guard

`tests/pr-agent-config.bats`: "no expression parses a step output with fromJSON".
