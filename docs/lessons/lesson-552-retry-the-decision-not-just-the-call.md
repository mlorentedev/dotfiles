---
id: "lesson-552-retry-the-decision-not-just-the-call"
type: lesson
status: active
title: "Retry the decision, not just the call"
created: "2026-11-05"
---

# Retry the decision, not just the call

## Context

The fixes for #3160 gave every call in `bitacora-status.yml` its own retry, then rolled the workflow out to
eight repositories. Their reviewers, reading the same file eight times, agreed on defects the dotfiles review
had not found.

## Problem

Each call was retried alone, so a retried write acted on a read made before the wait. A rate-limited write
waits a minute or more, and an issue parked or closed in that minute was still moved. The wait itself mixed two
clocks: `Math.max(retry-after, x-ratelimit-reset, 60)`. A secondary limit arrives while the primary reset is up
to an hour away, so the wait went past the five-minute cap and the run failed instead of waiting the minute
GitHub asked for. And the headers were read from `err.response.headers`, which only REST errors carry:
`@octokit/graphql` keeps the parsed body on `err.response` and the headers on `err.headers`, so every GraphQL
rate limit waited a flat minute. The stub in the tests shaped every error as a REST error, so it agreed with
the code.

## Solution

Retry one unit that reads everything it decides on and then writes, so each attempt re-checks the live state
and the Status. Take `retry-after` when given, the primary reset only when `x-ratelimit-remaining` is `0`, and
a minute otherwise. Read headers from `err.response?.headers ?? err.headers`. Retry only API errors: octokit
reports a network failure as status 500, so an error with no status is the script's own. Shape test errors the
way each client raises them, or the stub tests the stub.
