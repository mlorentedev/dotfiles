---
id: "lesson-421-a-retry-belongs-to-the-caller-that-knows-its-bound"
type: lesson
status: active
title: "A retry belongs to the caller that knows its bound"
created: "2026-10-14"
---

# A retry belongs to the caller that knows its bound

## Context
`dotf spec audit` asks GitHub about each active spec's issue through `spec.GHIssueStateLookup`. On a Windows runner one TLS handshake timeout made every active spec unanswerable, and the CI leg failed on a network blip (#2450). The same lookup also backs a `dotf doctor` check.

## Problem
The obvious fix is to retry inside the lookup. But the two callers bound the call differently. The CI audit gives each attempt 20 seconds and fails the leg on an unanswerable spec. The doctor kills `gh` after 15 seconds and reports an unanswerable spec as a WARN. A retry in the shared primitive would have cost the doctor nothing on a fast failure and about 48 seconds on a hung call, in an interactive command with more than 180 checks, to protect a WARN.

The retry also cannot tell a failure GitHub reported from one the caller imposed: a timeout the caller enforced is "I stopped waiting", not "GitHub did not answer", and retrying it three times is the retry-storm shape.

## Solution
The lookup asks once by default. `spec.Attempts(n)` opts into a retry, which skips an HTTP 4xx because that is GitHub's answer (401, a 403 rate limit, a 404). `spec.RetryWait` injects the back-off so tests never sleep. The audit passes `Attempts(3)`; the doctor passes nothing.

## Why
Whether to retry depends on what a failure costs the caller and on the deadline the caller has already set, and the primitive knows neither. Put the mechanism in the primitive and the policy at the call site, and test both: the default (one attempt) and the wiring (the audit retries). Otherwise the next caller inherits a policy it never chose.
