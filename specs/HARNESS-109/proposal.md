---
id: "HARNESS-109"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-03"
issue: "mlorentedev/dotfiles#1434"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-109

> **Naming**: file lives at `<repo>/specs/HARNESS-109/proposal.md`. `HARNESS-109` is `AREA-NNN-slug`.

## Why

<!-- from issue #1434: HARNESS-109: a named subagent dispatch sends its NAME as agent_type, so the gate resolves no persona and enforcement silently turns off -->

`dotf harness gate` reads `agent_type` off the hook payload and looks up
`harness/agents/<agent_type>/AGENT.md`. When a caller names a dispatch, `agent_type` carries **the
name**, not the persona type, so the lookup misses and the gate fails open. #1434 recorded this from
one deliberate probe. Re-measured on the whole journal it is not an edge case: of the **274**
decision records that carried an `agent_type`, **271 resolved to no persona**, and only **3** of
those 271 were the probe — the other 268 are four ordinary working dispatches
(`kubelab-harness`, `web080-adversarial`, `kubelab-reality`, `kubelab-1e`). The only two records
that ever resolved name `reviewer`, dispatched with no name. Naming a subagent is the normal way
this repository dispatches one, and it turns the gate off every time. Under the current `warn`
canary that costs a warning nobody reads; under `enforce: block` it is a one-word opt-out, which is
why every promotion on the orchestrator's roadmap sits behind this ticket.

**Provenance of that count, stated because the next reader will ask.** The journal holds 4119
records spanning 2026-09-02T04:01Z → 2026-09-04T00:42Z, written by whichever `dotf` was live at the
time — until 2026-09-03 that was a `dev` build off `feat/gate-decision-record`, the branch that
shipped the decision record in #1435. Checked rather than assumed: no record is missing a base
field, so none predates that schema; and the two resolving records (04:02:34Z, 04:04:48Z) are
*interleaved* with the unresolved ones (first at 04:01:37Z), not from a later binary — this is one
population, not a fix landing. The honest limit is the other direction: **every** `agent_type`
record falls in one 3.5-hour window on 2026-09-02, because no subagent has been dispatched on this
machine since. The sample is large in calls and small in sessions.

## What

The gate learns the `name → subagent_type` mapping **at dispatch time**, from a payload it already
receives, and resolves a named child through it. Three observable changes:

1. **A named dispatch is gated exactly like an unnamed one.** `Agent(subagent_type: reviewer, name:
   x)` produces the same `role_resolved: reviewer` and the same warnings as `Agent(subagent_type:
   reviewer)`.
2. **A built-in agent stops being reported as a failure.** `general-purpose`, `Explore` and `Plan`
   have no persona *by design*; today they are recorded `role-unresolved` with
   `ENFORCEMENT IS OFF` on stderr, which is a misclassification. They become `no-role`, the same
   quiet allow as the main thread.
3. **`role-unresolved` becomes rare and therefore meaningful.** After 1 and 2 it names only
   dispatches the gate genuinely never saw, so the journal can be read as a signal instead of noise.

### The measurement that decides the design

Issue #1434 offered two candidate directions and decided neither. The first — *"fall back to a
second payload field if one carries the true type"* — is **refuted**. Read out of the Claude Code
2.1.260 executable, the base hook-payload builder is:

```js
function pa(e,n,r,o){ let d = o?.agentType ?? Sy(); /* … */
  return { session_id:e.id, transcript_path:Hf(e.id), cwd:n, scratchpad_dir:…,
           prompt_id:…, permission_mode:r, agent_id:o?.agentId, agent_type:d, effort:E } }
```

That is the complete base field set (plus the per-event `hook_event_name`, `tool_name`,
`tool_input`). `agent_type` is `toolUseContext.agentType` and is the **only** type-carrying field;
`agent_id` is `a<name>-<hash>` — the name again, hashed. **There is no second field to fall back
to**, on this version or any version with this builder. The direction cannot be implemented and
should not be attempted.

What *is* available is that the gate already sees the **parent's** dispatch. The journal holds 6
records with `tool: "Agent"` in the parent scope, and that call's `tool_input` carries both `name`
and `subagent_type`. PreToolUse is synchronous — the tool does not execute until the hook exits — so
the parent's hook completes **before the child exists**, by construction rather than by luck. (The
journal agrees: dispatch at `04:01:34.087`, first child call at `04:01:37.637`.) One small
session-scoped map closes the gap with no new payload field, no schema change and no doctrine
change.

## Decisions

**The map is written from `tool_input`, and that does not weaken the "no tool input in the record"
property.** The property is *never write tool input into the decision journal*, and it stands: the
map lives in the state dir beside the consumption ledger, not in `*.decisions.jsonl`. The precedent
is already in the code — `skillArg` reads `tool_input.skill` and the gate has recorded it as
`rec.Skill` since #1435. `name` and `subagent_type` are schema-bounded identifiers (the Agent tool
constrains `name` to `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`); both are validated against that pattern
before being written, so no free text can reach disk through this path.

**Keyed by session, not by scope.** A subagent reuses its parent's `session_id` (gate.go:102-104),
which is what makes the lookup work at all — and it is also what makes a *grandchild* resolve, since
every generation shares the one key. Latest-wins on a reused name.

**Resolution order, and why the roster check is half the fix.** `agent_type ∈ roster` → persona;
else `agent_type ∈ dispatch map` → its type → (`∈ roster` ? persona : quiet `no-role`); else
`role-unresolved`, loud. Without the roster check the map would only relabel 271 noisy records as N
differently-noisy ones: `general-purpose` is not a persona and never will be, so a miss on it is a
correct answer, not a fault. `LoadPersonas` (persona.go:117) is the roster enumerator and is reused
verbatim — a second reader of `AGENT.md` is exactly the failure `roles.go` documents at its head.

**`no-role` is reused rather than adding a ninth `Outcome`.** The vocabulary is closed and pinned by
tests on purpose. "A built-in agent was acting and no persona applies" is the same *fact* as "the
main thread was acting and no persona applies", and the raw `agent_type` stays in `role_requested`,
so the journal can still tell the two apart without spending a vocabulary slot.

**Bounding the gate's hook timeout ships here.** `~/.claude/settings.json` runs
`dotf harness gate --harness claude` on PreToolUse with **no `timeout`**, while UserPromptSubmit
carries `timeout: 5` — #1455's review bounded the suggester and left the gate, on the same
interactive path, unbounded. Measured in the same executable: a timed-out hook returns
`blocked: false` unless `timeoutFailsClosed` is set, and that flag is only set for a call served to
a cloud session ("timed out … on the attached machine"). So locally a bound converts a long stall
into a fast fail-open — which is the gate's own documented contract — and cannot cause a block. It
is one field on the manifest's `emit_hooks` entry.

## Out of scope

- **Any change to `enforce` modes.** Declaring `enforce` on the six bare personas and promoting
  `warn → block` is the *next* step and reads the decision journal for evidence; this spec only
  makes the journal worth reading.
- **Blocking on an unresolved role.** Explicitly forbidden by #1434's "Do not close by", and by
  lesson 219: it trades a silent bypass for a session that cannot run.
- **The residual `role-unresolved` population.** Dispatches the gate never observes as an `Agent`
  call still cannot resolve — a `Workflow` `agent()` step, an agent resumed by `SendMessage` from an
  earlier session, an in-process teammate. Naming them is the point; building for them is not.
- **A `dotf doctor` check for the unresolved rate** (#1434's second candidate). It is only worth
  writing once this spec has shrunk the set to that residual; filing it then costs nothing and
  building it now measures noise. The meta-work cap is active.
- **#1421** (the gate hook binds user-level, forking `dotf` in every repo) and **#1467** (registry
  resolved against CWD). Adjacent, separately owned.

## Risks / open questions

- **The map is only as good as the dispatch the gate saw.** If the parent's gate hook times out or
  the state dir is unwritable, no entry is written and the child fails open — the behaviour before
  this change, never worse. This is the reason the timeout bound is in scope rather than deferred.
- **Payload-shape drift.** The design reads `tool_input.name` / `tool_input.subagent_type` from a
  vendor tool schema. A rename degrades to "no entry", hence fail-open, hence today's behaviour; it
  cannot cause a block. Same containment as `skillArg`.
- **A single AC needs a real dispatch.** Everything else is verifiable synthetically now that the
  exact payload field set is known (AC1–AC5 drive the built binary against a scratch `--state-dir`).
  AC6 reproduces #1434's own table on a live named dispatch and requires the operator's assent to
  spawn one; it is recorded as owner-gated rather than quietly skipped.
- **Machine-shared binary.** The live hook runs `~/.local/bin/dotf`, which a parallel session
  rebuilt today to fix two live symptoms. Verification uses a locally built binary and a scratch
  state dir; installing over the shared one requires telling that session first.

## Acceptance criteria

- [ ] **AC1** — A gate call with `tool_name: "Agent"` and `tool_input: {name, subagent_type}` writes
      a `name → subagent_type` entry to the session's dispatch map in the state dir, and writes
      nothing to `*.decisions.jsonl` beyond the existing record.
- [ ] **AC2** — A later call in the same session with `agent_type: <name>` records
      `role_resolved: <subagent_type>` and applies that persona's declared `enforce` modes; a
      control run of the same payload without the preceding dispatch still records
      `role-unresolved`.
- [ ] **AC3** — `agent_type` naming a persona directly (`reviewer`) resolves without consulting the
      map, unchanged from today.
- [ ] **AC4** — `agent_type` naming a built-in agent (`general-purpose`, `Explore`, `Plan`), whether
      reached directly or through the map, records `no-role`, emits no
      `ENFORCEMENT IS OFF` line, and keeps the raw value in `role_requested`.
- [ ] **AC5** — Names are validated: a `name` or `subagent_type` failing
      `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$` writes no map entry, and the call still allows. No other
      `tool_input` key is read or persisted anywhere.
- [ ] **AC6** *(owner-gated)* — On a live named dispatch, the journal shows `role_resolved` where
      #1434's table recorded `role-unresolved`, reproducing that table with the third row fixed.
- [ ] **AC7** — The manifest's `emit_hooks` PreToolUse entry carries a `timeout`, verified by
      consequence on the rendered settings file rather than by string-matching the manifest.

## References

- Bitácora board: `mlorentedev/dotfiles#1434` (see the `issue:` frontmatter field)
- `specs/archive/HARNESS-110/proposal.md` — the suggester and `ResolveRoles`; this spec is its
  enforcement-side counterpart, and reuses `LoadPersonas` for the same reason.
- `docs/adr/adr-027` / `specs/HARNESS-045-hook-emission` — the gate as the agnostic seam.
- `cli/internal/cmd/harness_gate.go:149` (`effectiveRole`), `:245` (`loadGatePersona`),
  `:318` (`commandHookPayload`) — the three sites this changes.
- Lesson 219 (a guard that fails open in silence) and lesson 255 (`agent_id` is caller-influenced).
