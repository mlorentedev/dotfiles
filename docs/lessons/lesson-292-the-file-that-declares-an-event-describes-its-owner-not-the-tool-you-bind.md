---
id: lesson-292
type: lesson
status: active
created: "2026-09-24"
owner: manu
tags: [lesson, harness, agy, hooks, verification]
---

# 292 — The file that declares an event describes its owner, not the tool you are binding

## What happened

The agy gate hook was bound into `~/.gemini/settings.json` as a `BeforeTool` hook,
and `harness/manifest.json` recorded the event names as "measured 2026-08-26".
What was measured was the FILE: it declares `BeforeAgent`, `AfterAgent`,
`BeforeTool` and `AfterTool` in claude's exact hook shape, so agy was filed as
"the cheapest harness to add after claude".

That file is Gemini CLI's. Orca writes it for that product. agy reads its hooks
from `~/.gemini/config/hooks.json`, in another schema (named hooks, `PreToolUse`,
a camelCase payload) and another protocol (a JSON decision on stdout; the exit
code means nothing). For weeks the hook was bound where agy never looked, and had
it run, the gate would not have parsed the payload.

Nothing was hidden. A code comment said "agy's exact field names are unverified,
no agy payload has been captured", while the manifest said "measured". The doubt
lived in one file and the certainty in another, so they never met.

An audit found it with one grep. agy's log prints `loaded N named hooks from M
hooks.json file(s)` at every start, and the binary embeds the hook documentation
it follows (`strings agy | grep hooks.json`).

## Why it happens

A config file is written by someone, for some product. "This file declares X" is
a fact about the writer. The step to "the product I am binding reads it" was
inferred from a family resemblance (same `~/.gemini/` directory, same spelling of
"hooks") and never tested against the consumer.

The instrument that could have caught it existed and was not read. Every gate
call leaves a journal record tagged with its harness. The only agy record on the
machine was hand-made, carrying a `tool_name` field agy does not send. A journal
with one record, and that one synthetic, says the harness was never measured.

## The rule

1. **Verify a hook contract against the consumer, not a file that mentions it.**
   The consumer's log says what it loaded; its binary or documentation says what
   it reads. Ask the product which file, which schema, which protocol.
2. **Name the product in the comment.** "Gemini CLI's settings.json" is a claim a
   reader can check. "The settings file" is not.
3. **A "measured" label names the instrument.** If the thing measured was a file,
   the label says file.
4. **Count the real records before calling wiring done.** A journal whose only
   entry for a harness is synthetic has measured nothing.
5. **Answer the protocol the consumer reads.** agy documents `allow` as approving
   the call without asking anyone, so a gate that answered it whenever it had no
   objection would have approved every call past agy's own prompt. The neutral
   answer is `ask`.
6. **A manifest merged ahead of its binary is read by the OLD binary.** Setup
   mirrors the manifest the moment it merges and the binary is released on its own
   schedule. The first version of the fix put the new `hooks-json` target beside the
   others, and dotf 0.57.0 given that manifest wrote a top-level `hooks` key, with a
   `_managed` sidecar inside a handler, into a copy of the real hooks.json: claude's
   shape, into a file agy decodes as protojson and shares with Orca. Caught by
   running the installed binary against a copy, not by reading the code. A format an
   older binary does not know now lives under `agents.bind_named`, a key it never
   reads, so its effect is nothing. The loader refuses a target in the wrong key,
   and the emitter has no default arm that falls back to claude's shape.

Relates to lesson 288 (an inventory is a hypothesis until each row is probed) and
lesson 287 (a guard that skips is a guard that passes).
