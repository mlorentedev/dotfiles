---
id: "lesson-525-an-injection-fix-belongs-to-every-sink-of-the-value-not-the-one-reported"
type: lesson
status: active
title: "An injection fix belongs to every sink of the value, not the one reported"
created: "2026-10-30"
---

# An injection fix belongs to every sink of the value, not the one reported

## Context

MEMORY-008 review round 4 found that `dotf mem resume` put a thread key from `MEMORY.md` raw into the
candidate lines it prints for the operator to copy. #2944 quoted the key there, with a hostile-key test.

## Problem

The same key reached a second copyable command: the one line a resumed or forked session gets at session
start (`sessionResume`). The fix lived in `package cmd` as a private helper, so `package mem` could not call it
even had someone looked. Round 5 failed the spec on that second sink. A fix scoped to the site a reviewer
names leaves every other sink of the same value for the next round to find, one round at a time.

## Solution

The quoting helper moved to `package mem` as `ShellWord`, next to where thread keys are produced, and both
command sinks call it. The pull request records the sweep that proves there is no third:
`grep -rn -- '--thread' cli/` over non-test Go. When fixing an injection, list every place the value is
written into a command, put the helper where all of them can reach it, and keep the sweep as evidence.
