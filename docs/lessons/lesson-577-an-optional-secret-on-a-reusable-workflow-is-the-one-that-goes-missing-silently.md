---
id: "lesson-577-an-optional-secret-on-a-reusable-workflow-is-the-one-that-goes-missing-silently"
type: lesson
status: active
title: "An optional secret on a reusable workflow is the one that goes missing silently"
created: "2026-11-10"
---

# An optional secret on a reusable workflow is the one that goes missing silently

## Context

`dotf secrets ci-check` (#3290) compares the secrets a repository's workflows read with the ones the registry
declares for it. For a job that calls a reusable workflow with `secrets: inherit`, it reads the callee's
`on.workflow_call.secrets`. The shared pr-agent workflow declares `NAN_API_KEY` as `required: true` and
`PR_AGENT_ANTHROPIC_API_KEY` as `required: false`.

## Problem

The first draft of the spec treated `required: false` the way the keyword reads: an optional secret that is absent
is not missing. That rule made the check's own motivating case unreportable. pr-agent marks the Anthropic key
optional on purpose, so that a review degrades to NaN instead of failing. A repository without the key never
fails a run; it loses reviews under load (#3188, 3 of 8). A required secret that is absent fails loudly at the
first run and gets fixed. An optional one never fails anything, which is why nothing had reported it.

## Solution

An absent `required: false` callee secret counts as missing, marked `(optional in <callee>)`, and fails the check
like any other missing name. `required` describes what the callee does without the secret, not whether the caller
needs it. The fleet run found the case live: `mlorentedev/web` maps the key and the repository does not have it.

## Takeaway

When a check decides what counts as absent, `required: false` (or any "optional" flag) marks the degradations no run
will ever report. Those are exactly what a static check exists to find, so never use the flag to skip a name.
