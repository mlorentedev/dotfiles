---
tags: [spec, verification, templates]
created: "2026-08-27"
---

# Verification - HARNESS-045-hook-emission

> **Partial by design.** This records what the first sitting built and proved.
> AC1 (setup wiring), AC2 (presence) and AC7 (the block-style guards) are open,
> and `tasks.md` says which and why. Nothing below is claimed beyond what ran.

## Evidence

| AC | Proof |
|---|---|
| AC1 (declaration half) | `harness/manifest.json` carries `agents.bind`; `bats tests/compile-harness.bats` 72/72 exit 0, so adding the key disturbs no render |
| AC3 (decision half) | `go test ./internal/harness/ -run TestGate` — blocks on an unconsumed `enforce: block`, opens once consumed, never blocks the skill invocation itself |
| AC4 | `TestGateWarnDoesNotBlock` asserts `warn` reports and allows, on the same path as AC3 so the two cannot collapse |
| AC5 | `TestMergeHooksIsIdempotent` — a second identical merge reports `changed=false` and leaves exactly one entry |
| AC5+ | `TestMergeHooksReplacesOurOwnEntryRatherThanAccumulating` — idempotence under **change**, the assertion AC5/AC6 imply but neither states |
| AC6 | `TestMergeHooksPreservesForeignEntries` and `TestMergeAgainstTheRealDeployedSettings` — **12 foreign hook entries in the real `~/.claude/settings.json`, all preserved** |
| AC6+ | `TestMergeHooksSurvivesAForeignGroupAtIndexZero` — the ordering the positional shell path silently overwrites |
| AC7 (loader half) | `TestPersonaRefusesToReadSkillsAsEmpty` — 7 malformed shapes, each a loud error, never an empty list |
| AC8 | Every `Decide` test runs with no harness installed |

## The gate, driven as a binary

```
claude    {"session_id":"c","tool_name":"Bash",...}     EXIT=2
agy       {"session_id":"a","tool_name":"Bash",...}     EXIT=2
pi        {"session":"p","tool":"bash"}                 EXIT=2
opencode  {"session":"o","tool":"bash"}                 EXIT=2

pi  invoke skill adversarial-review                     EXIT=0
pi  retry the blocked call                              EXIT=0

garbage on stdin                                        EXIT=0   ("payload not recognised")
a real call, same run                                   EXIT=2   (the fix did not neuter the guard)
```

## Decisions made during implementation

- **agy is not presence-only.** #561 defers it on the assumption it has no gate.
  `~/.gemini/settings.json` declares `BeforeAgent`, `AfterAgent`, `BeforeTool`,
  `AfterTool` in claude's exact command-hook format. It is the **cheapest**
  harness to add after claude. Its field names remain unverified, so the agy leg
  degrades to allow on a mismatch and its AC must stay structure-level until a
  real payload is captured.
- **opencode's gate is `permission.ask`, not `tool.execute.before`.** The latter
  is typed `(input, output: {args}) => Promise<void>` — it mutates arguments and
  returns nothing. #561 and ADR-027 name an event that cannot deny.
- **Two families, not four cases**, so the wrapper this repo generates for pi and
  opencode emits ONE canonical payload. That deletes two per-harness parsers
  instead of adding them.
- **No default severity.** An undeclared `enforce` resolves to `EnforceUnset`;
  the gate refuses to act and `UnmigratedSkills()` surfaces it. `warn` would make
  every unmigrated persona silently inert while checks reported it wired; `block`
  would turn 35 existing skills into hard gates overnight.
- **Ambiguity resolves to Allow, everywhere.** A gate that blocks on input it
  cannot read blocks on every harness upgrade.

## Reviewer findings on #1272, applied

PR-Agent raised two, both real, both fixed with a regression test each.

1. **A skill invocation could deadlock the session.** On a *well-formed* payload
   naming the skill primitive but carrying no readable argument, the skill name
   came back empty, the call fell through to enforcement, and the gate blocked
   the one action that could satisfy it — permanently, since a blocked call never
   records consumption. The guard keyed on the skill's **name** when it had to
   key on the tool's **identity**. `TestGateNeverBlocksTheSkillInvocationItself`
   passed throughout, because it always supplied a name — the case that was never
   in danger. Pinned by `TestGateNeverBlocksASkillToolWithAnUnreadableName`.
2. **State paths could collide.** Character-mapping alone flattens `a/b` and
   `a.b` to one file, so one session's consumption would open another's gate.
   UUIDs never collide, but a session id is attacker-adjacent input landing in a
   path. A digest is appended and the readable prefix kept. Pinned by
   `TestGateStatePathDoesNotCollideAcrossDistinctSessions`.

A third came from checking whether the pin guard shipped in #1256 had gone red on
main: `tiers.low` acquired a `$comment`, and `DeclaredModels` treated every string
in a tier as a model id, so the whole sentence entered the declared set. It only
ever widened the set — no check could fail wrongly — but a registry that treats
prose as an id is one coincidence away from masking real drift.

**That check worked as designed**, and it is worth recording: `qwen3.8-flash` was
promoted to pi's routed default **and** added to the map in the same change,
which is exactly the coupling the pin registry exists to force.

## Three defects found by running it, not by reading it

1. **A malformed payload blocked every call.** `normaliseToolCall` returned a
   zero `ToolCall` and the doc comment claimed `Decide` would resolve that to
   Allow. `Decide` does not: a valid persona with nothing consumed blocks. The
   nine `Decide` unit tests all passed, because the defect lived in the **seam**
   between normalisation and decision, which none of them crossed. Prose
   asserting behaviour the code did not have — caught only by driving the binary.
2. **Identity by repository evicted a live hook.** After adopting
   `dotf mem session-start` on `SessionStart`, emitting the gate on that event
   matched the memory hook's marker and replaced it. Identity had to become
   per-purpose.
3. **Adoption by exact command was missing**, so `bind` taking over a hook the
   positional path wrote would have appended a duplicate rather than adopting it.

## Promotion candidates

- Nothing yet. The one cross-project candidate — *"a config surface shared with a
  live third-party writer is merged by marker, never by position"* — is strong,
  but it wants a second instance outside this repository first.
