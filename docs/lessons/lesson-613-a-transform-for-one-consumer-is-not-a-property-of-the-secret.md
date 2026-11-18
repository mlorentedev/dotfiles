---
id: "lesson-613-a-transform-for-one-consumer-is-not-a-property-of-the-secret"
type: lesson
status: active
title: "A transform for one consumer is not a property of the secret"
created: "2026-11-17"
---

# A transform for one consumer is not a property of the secret

## Context

`Loader.EnvFor` strips every CR and LF from a value, because a child-process env token is one line
and `age -d` appends a newline. `dotf secrets sync ci` reused `EnvFor` to resolve what it uploads
with `gh secret set`.

## Problem

The first multi-line CI secret, a GitHub App PEM key, would have been uploaded with its line
breaks removed. The sync reports `set NAME → repo` and exits 0; the failure appears only when the
workflow's `actions/create-github-app-token` cannot parse the key. `migrate` hit the same class
before and moved to `RawResolve` (#612 B6), but the upload path kept the env-injection function.

## Solution

Each consumer resolves the raw value and applies its own transform. `sync ci` now calls
`Loader.UploadFor`, which keeps interior line breaks and trims only a trailing CR/LF, so
single-line tokens upload unchanged (SEC-019, #3581). Before reusing a resolver for a new
consumer, read what it does to the value, and add a test with a value that the transform would
change.
