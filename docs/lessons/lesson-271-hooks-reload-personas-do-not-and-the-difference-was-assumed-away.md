---
id: 271
title: "Hooks are re-read and personas are not, and one measurement was generalised into both"
date: "2026-09-02"
tags: [lesson, harness, orchestrator, measurement]
---

# 271 — Hooks are re-read, personas are not, and one measurement was generalised into both

## What happened

Measuring the orchestrator gate on 2026-09-01 established, three separate ways, that Claude
Code **re-reads `settings.json` hooks** rather than freezing them at session start. That was a
real result and it retired a standing plan: every "open a fresh session to pick this up" step
was unnecessary.

The next day it got applied one layer over. `harness/capability-map.json` gained a `skill` verb,
`compile-harness.sh --deploy` rewrote `~/.claude/agents/reviewer.md` from
`tools: Read, Glob, Grep, Bash` to `tools: Read, Glob, Grep, Bash, Skill`, and the file on disk
was verified by reading it back. A `reviewer` was then dispatched and asked to invoke a skill.

It answered that it had no Skill tool: `Read, Bash, advisor`. Not the old set, not the new set —
**the roster the dispatching session had loaded at ITS start.**

## The mistake

"Hooks reload" was measured. "The harness reloads its configuration" was inferred from it, and
those are different claims about different files read by different subsystems at different
times. Nothing was measured about persona records, and the earlier result gave no reason to
expect they behave alike — hooks are consulted per tool call, a persona roster is consulted when
a dispatch is constructed.

The tell was available and went unread: the reported tool set matched **neither** the old file
nor the new one. A disagreement with both versions on disk is a signal that the file is not the
source at all, and it was briefly read as a deploy that had half-worked.

## The rule

**A reload measurement covers the file that was measured.** Extending it to a second file is a
new claim needing its own measurement, and the cost of getting it wrong is asymmetric: assuming
a reload that does not happen makes a change look shipped when it is not, and the verification
that "proves" it reads the file rather than the running system.

Concretely, in this repository:

- **Hooks in `~/.claude/settings.json` — re-read.** A `dotf harness bind` takes effect in
  sessions already running.
- **Persona records in `~/.claude/agents/` — frozen at the dispatching session's start.** A
  `compile-harness.sh --deploy` reaches only sessions started afterwards.

## What it cost, and what it did not

One acceptance criterion (`HARNESS-106` AC4 — a dispatched persona invokes a skill and leaves a
consumption record) could not be closed in the session that shipped the mechanism, and stayed
open with a named blocker rather than being ticked on a file's contents. Which is the right
outcome: AC4 was written as *"proven by a dispatch that writes a consumption record — never by a
config file containing a key"*, precisely to refuse the evidence that was available.

Nothing was mis-shipped, because the criterion was worded to reject exactly the shortcut the
wrong inference would have taken. An acceptance criterion that names the evidence it will not
accept is what turned a bad assumption into a deferred item instead of a false claim.

## See also

- `docs/lessons/lesson-254-a-canary-that-emits-where-nothing-persists.md` — the same family: a
  measurement whose instrument could not observe what it claimed.
- `docs/lessons/lesson-255-truncation-not-hostile-input-made-the-digest-load-bearing.md`
- `specs/HARNESS-106-skill-capability/` — AC4's wording, and the decision record that made the
  session's real state readable.
- #1434 — the fail-open the decision record caught on its first day.
