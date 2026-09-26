---
id: lesson-299
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, review, sdd, agents, prompts, timeout]
---

# 299 — Tell the reviewer its time budget; a deadline alone only kills

## What happened

On pi, the independent review of SKILL-001 had no deadline. Its rounds took 31, 39 and 70 minutes. A hard deadline alone would have killed the slow ones before they wrote a verdict, which loses the whole run. With #1721 the prompt states the stop time and a target at two thirds of it, as absolute clock times the model can check with `date`. The next round was told to aim for 16:30 and to expect a stop at 16:45. It finished at 16:17 with its own mutation runs, so it lost no depth.

## The rule

An agent given a deadline should be told the deadline, and a target before it that leaves room to write the result. Enforce the deadline outside the agent, as lesson 298 describes, and put the target inside the prompt. Absolute times work where durations do not, because the agent does not know when its run began.

Refs: HARNESS-152 (#1710, #1721). The live run is recorded on #1710.
