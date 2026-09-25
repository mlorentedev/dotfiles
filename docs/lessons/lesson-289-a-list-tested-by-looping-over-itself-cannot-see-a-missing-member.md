---
id: lesson-289
type: lesson
status: active
created: "2026-09-24"
owner: manu
tags: [lesson, secrets, harness, agnostic, verification, silent-failure]
---

# 289 — A list tested by looping over itself cannot see a missing member

## What happened

`dotf secrets show` refuses to print plaintext in an agent session, and "agent
session" is a list of environment variables. It was wrong twice, both times in
the direction that prints the secret.

- **Round 1 (SEC-001).** The list named `CLAUDE_CODE`. Claude Code exports
  `CLAUDECODE`, so the refusal never fired in the harness it was written for.
- **Round 2.** pi exports `AI_AGENT=pi` and `PI_CODING_AGENT=true`, and the list
  knew neither. The test that should have caught it,
  `TestDetectAgentSession_EachMarkerIsSufficient`, iterates the list and checks
  that each entry is detected. An absent entry is never iterated, so the test
  passed.

Measuring the fix turned up four more ways to get the answer wrong:

- **Static search of a launcher.** `strings` on the `agy` binary found neither
  `ANTIGRAVITY_AGENT` nor `ANTIGRAVITY_CLI`. Live, `ANTIGRAVITY_AGENT` is set:
  the language server that `agy` starts exports it, not the launcher.
- **An inherited environment.** Every harness launched from a Claude session
  inherited `CLAUDECODE` and `AI_AGENT`, so its output alone could not say which
  markers it set.
- **Look-alike variables.** `OPENCODE_HOME`, `COPILOT_HOME`, `GEMINI_HOME` and
  `ANTIGRAVITY_ENDPOINT` look like markers, but the shell profile exports them,
  so a human terminal carries them too.
- **An agent asked to report its own environment.** pi's model declined to run
  `env`, following this repo's own doctrine. It used another method and answered
  that `AI_AGENT` was unset, which contradicts pi's documentation ("child
  processes inherit both markers") and even the variable it had inherited.

## Why it happens

A test built from the thing it checks can only confirm what is already there.
An omission is the one defect it cannot see, and for a denylist-shaped guard the
omission is the dangerous case, because the guard then passes in silence. The
measurements failed for a related reason: each answered from the part it could
see (the launcher, the parent's environment, the model's summary) and returned a
clean answer about it.

## The rule

1. **Test a list against an independent table of facts, and against the
   registry of the things it must cover.** `TestAgentSessionMarkers_CoverEveryHarness`
   holds {harness → marker} separately from `agentSessionMarkers`, and fails for
   any harness in `harness/model-map.json` without a row. A new harness then
   breaks the build instead of shipping a refusal that never fires.
2. **Measure inside the harness, subtract what it inherited, and control with a
   clean login shell.** A marker counts only if the harness adds it and a human
   shell (`env -i HOME=… zsh -lic`) does not carry it. Print variable names,
   never values.
3. **Absence in a binary is not evidence of absence at run time.** A launcher
   may start the process that sets the variable.
4. **An agent's account of its own environment is not a measurement.** When the
   model does not run the exact command, discard the answer and use the
   vendor's code or documentation instead.

Refs: SEC-001, #1646, #1626. Relates to lesson 287 (a guard that skips is a
guard that passes) and lesson 288 (an inventory is a hypothesis until each row
is probed).
