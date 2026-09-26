---
id: lesson-306
type: lesson
status: active
created: "2026-09-25"
owner: manu
tags: [lesson, sdd, verification, harness, gate]
---

# 306 — "Proven by a dispatch" is a join to the harness's own record, never a grep of a log anyone can write

## What happened

HARNESS-106's AC4 said that a dispatched persona can invoke a skill, "proven by a dispatch … never by a config file containing a key". Its `features.json` verifier grepped the gate's decision ledger for a `skill-consumed` line carrying `agent_type`. The first retroactive review printed one such line into the gate by hand, and the verifier passed. The criterion's own wording forbade that evidence, and its verifier accepted it.

## The rule

When a criterion is about something that happened (a dispatch, a deploy, a run), verify it against two independent records: the one the change writes, and the one the harness writes on its own. For AC4 that is each ledger record joined to Claude Code's subagent transcript of the same session, whose `meta.json` names the same agent type and which invoked the same skill. A hand-written line has no transcript behind it. Then test the verifier both ways: it passes on the real state, and it fails on the forged line.

Refs: HARNESS-106, #1420, #1532.
