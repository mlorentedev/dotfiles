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

That reading was **unsupported** — the probe could not have produced any other
outcome either way. A parallel session grepped its own transcript JSONL: Claude
Code records hook execution as entries carrying `hookCount` / `hookErrors` /
`hasOutput`, and every such record in the session was Stop-family;
`stop_hook_summary` was the only hook-bearing subtype present at all, under any
name. **A `PreToolUse` hook's streams on exit 0 are not persisted to the session
transcript.** The gate emits `[gate] warn` to stderr and exits 0, so a warn
decision was invisible however correct the field names were.

That began as *"we looked and found no record"*, which is the weaker claim. It
became **evidence of absence** an hour later, when a `Skill` call in that same
session provably fired the hook — it wrote a consumption ledger entry — and still
left no transcript record of it. The hook demonstrably ran and the transcript
does not know.

**`agent_type` remains unmeasured, and nothing here changed that.** A later probe
established that `agent_id` arrives on subagent events and is per-invocation, but
it did so through the ledger, and the ledger cannot speak to `agent_type`: the
gate records a skill consumption and `return`s at `harness_gate.go:84-87`,
*before* `loadGatePersona` is reached. Not one ledger entry has ever exercised the
persona-resolution path. The two are sibling fields on one payload struct and are
documented together, so it is strong inference — and inference is what it stays
until the decision record exists.

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

**Nor is `--debug` the answer.** It is *reported* to surface hook streams and
might break this particular tie once — where those streams actually go was never
checked, which is the honest state of it — but either way it depends on how a
human launched the session. A canary cannot rest on a flag someone has to
remember. What the gate needs is a decision
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
