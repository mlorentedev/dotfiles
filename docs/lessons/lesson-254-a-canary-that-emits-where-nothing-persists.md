# 254 - A canary that emits where nothing persists accumulates no evidence, and its silence reads as success

**Date:** 2026-09-01
**Area:** harness, orchestrator gate, verification

## What happened

`dotf harness gate` was bound into `~/.claude/settings.json` as a `PreToolUse`
hook at 19:34:05 — the first time the orchestrator gate had ever run on this
machine. The `reviewer` persona carried four skills at `enforce: warn`, the
severity the owner chose to canary all 35 skills at before promoting any to
`block`.

The first probe dispatched a `reviewer` subagent to run one Bash command and
looked for the gate's `[gate] warn: <skill> not consumed` lines. **Nothing
appeared.** The obvious reading was that the payload field `agent_type` is not
the name the gate expects, which would mean the persona resolution shipped in
#1410 rests on a field that never arrives.

That reading was wrong, and the probe could not have produced any other outcome.
A parallel session grepped its own transcript JSONL: Claude Code records hook
execution as entries carrying `hookCount` / `hookErrors` / `hasOutput`, and the
entire session contained **exactly one**, of the Stop family. There is no
`pre_tool_use_hook_summary` for any tool call. **A `PreToolUse` hook's streams on
exit 0 are not persisted at all.** The gate emits `[gate] warn` to stderr and
exits 0, so a warn decision was invisible however correct the field names were.

The gate's other channel is durable but narrower than it looks. It writes a
consumption ledger under `~/.local/state/dotfiles/gate/`, and only on a `Skill`
tool call — an ordinary tool call at `warn` writes nothing. That is why the
directory had been empty since the day the gate was written.

And no persona can make a `Skill` call. `harness/capability-map.json` declares
the neutral vocabulary `[read, search, edit, shell, web]`; **nothing maps to
`Skill`**, so all seven personas deploy with `tools: Read, Glob, Grep, Bash`
(plus `Edit, Write` for some). That map's own audited comment records why this is
decisive:

> claude frontmatter `tools:` — a comma-separated list of native tool names. An
> ALLOW-LIST: a tool not named is unavailable to the agent.

So at `warn` the gate produces no observable signal by any route, and the canary
period would end with zero accumulated evidence. Verified from the other
direction the same session: subagents with the full toolset *can* invoke a skill
and *do* write the ledger, so the gap is precisely and only the seven persona
capability declarations.

## Why this is the lesson and not just a bug

The severity ladder was designed with care — `warn` reports and allows, `block`
refuses, an undeclared skill is neither. What nobody specified with the same rigour
was **where the report goes**, and that turns out to decide whether the ladder
works at all.

`HARNESS-045`'s **AC4** reads *"a skill declared `enforce: warn` **emits** and does
not block, asserted on the same path as AC3 so the two cannot collapse."* The
non-blocking half is verifiable. The emitting half is not, through the channel it
assumes. The AC was written against a mechanism whose observability was never
checked.

**AC3 is worse, because it would pass.** It asks that with a skill at
`enforce: block` unconsumed, a tool call is blocked. Under `block` a persona is
blocked on every call *and cannot reach the one action that would clear it* — the
gate's `isSkillTool` escape ("invoking a skill is never blocked: forbidding it
would deadlock the session") assumes an agent able to emit a call it was never
granted. AC3 would observe a blocked call and record a pass, on a persona that is
now permanently deadlocked. An acceptance criterion satisfied by the broken state
is worse than a missing one.

The session also spent two probes reading a silence as data, in a repository that
has already written that mistake down twice: `dotf pr triage-queue` exiting 0 is
not an empty queue, and a green `review-attestation` check is not a review that
happened. Both were re-derived here at cost. **The tell is identical every time —
a channel is consulted, it says nothing, and nothing is exactly what it says when
it is broken and when it is fine.**

## What this does not license

**The gate's fail-toward-allow design is correct and is not what failed.** A field
name it does not recognise yields an empty `AgentType`, a nil persona, and the
pre-existing allow; a payload it cannot parse takes the same path. That is
deliberate — a wrong guess costs enforcement and can never block a session — and
it is why binding the gate to a live machine was safe before any of this was
measured. The defect is not that failure is silent to the *session*; it is that
success is silent to the *operator*.

**Nor is `--debug` the answer.** It surfaces hook streams and would break this
particular tie once, but it depends on how a human launched the session. A canary
cannot rest on a flag someone has to remember. What the gate needs is a decision
record that survives independently of stderr, of exit code, and of whether the
persona holds the `Skill` tool — covering allow, warn, block, and the
`role did not resolve` path, which today is the loudest thing the gate can say
and is equally invisible.

## Rule

When you design a graduated rollout — warn now, block later — specify the
observation channel with the same rigour as the enforcement, and **prove the
channel persists before the canary period starts counting**. Then check every
acceptance criterion against the failure you are guarding, not only against the
success: an AC that the broken state satisfies is not a check. Silence in a
channel nobody has shown to be durable is not evidence of safety, of correctness,
or of anything at all.
