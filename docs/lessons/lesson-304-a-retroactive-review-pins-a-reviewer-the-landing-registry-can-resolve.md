---
id: lesson-304
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, review, sdd, secrets, retroactive]
---

# 304 — A retroactive review pins a reviewer whose key the landing commit's registry resolves

## What happened

A retroactive review runs from a worktree at the spec's landing commit, with that commit's own `dotf` first on `PATH`, because `dotf secrets run` reads the registry of the checkout that contains the cwd. The launcher draws a reviewer from the pool at random. On 2026-09-25 it drew `agy/gemini-3.1-pro-high`, whose `GEMINI_API_KEY` the landing commit's registry did not declare. The run died in a second with "the reviewer exited without writing review.md". The next launch, pinned to `nan/mimo-v2.5`, was cut off by the provider's concurrency limit (HTTP 429). Neither was a verdict, and neither counts as a round.

## The rule

For a retroactive review, pin `--reviewer` to a pool member whose key the landing commit's registry resolves. The preflight `dotf secrets run --only NAN_API_KEY -- true` proves that for the NaN members only. When a run ends without `review.md`, read the transcript's last turn before counting it. `stopReason: error` with a 429, or a missing key, is an environment failure to re-run, not a FAIL.

Refs: #1626 (W1.4), SDD-040.
