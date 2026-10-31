---
id: "lesson-522-a-value-interpolated-into-a-key-value-output-is-an-output-line"
type: lesson
status: active
title: "A value interpolated into a key=value output is an output line"
created: "2026-10-30"
---

# A value interpolated into a key=value output is an output line

## Context

TOOL-031 A4 ports `scripts/pr-agent-push-gate.sh` to Go as `dotf pr gate` (#2893). The script's decision reaches
later steps through `$GITHUB_OUTPUT`: `run=`, `reason=` and `mode=` lines. Its `reason` quotes `head_sha`, a value
read from the state block in a review comment.

## Problem

`head_sha` is JSON, so `"x\nrun=false"` decodes to two lines. The script printed:

```text
run=true
reason=the reviewed commit (x
run=false) is no longer in this PR, most likely a rebase or force-push
mode=full
```

A file command takes the last value of a key, so this line could turn a decision to review into a skip. The fix
that review A3 asked for, checking the inputs for an empty value, would not have found it: the value is present
and well-formed JSON. The gap is in where the value lands.

Three other inputs let a skip rest on no data, and each printed a clean `run=false`: a review with no `created_at`,
commits with no date (`null > "x"` is false in jq, so they counted as old), and a `--threshold` too large for `[`,
whose error fell through to the skip branch.

## Solution

`GateDecision.Output` replaces line breaks in the reason, in one place, whatever the reason quotes. A missing date
is read as unreadable, which means a full review, and an out-of-range threshold exits 2.

The port also detects a forged marker by position, as PR-Agent walks the comments, where the script compared
timestamps with `>` and missed one posted in the review's own second.

## Rule

When a tool writes `key=value` lines for a consumer, single-line the values at the point that prints them, not at
each input. When porting a fail-open tool, list what each comparison does with an absent value: a comparison that
is false on null is a silent decision, and on a fail-open path it is usually the wrong one.
