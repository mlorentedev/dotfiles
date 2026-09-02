---
id: "HARNESS-045-hook-emission"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-08-27"
issue: "mlorentedev/dotfiles#561"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-045-hook-emission

> `created:` reads 2026-08-27 for work done the evening of 2026-08-26 — #1225's
> UTC stamp, fixed in #1257 but not yet released. Left as-is; a third instance of
> the evidence.

## Why

Every persona definition states, in its own body:

> *"Your phase's skills are enforced by hook, not left to memory."*

That is false today, on every harness. Measured 2026-08-26:

- **Exactly one file in the CLI reads `harness/agents/`** — `cli/internal/doctor/checks_agent_tiers.go`, a doctor check. `cli/internal/agent/dispatch.go` does not; `Role` is a pass-through string, which is why `--role reviewer` worked before any reviewer existed.
- **`harness/manifest.json` has no hook concept at all.** It has `deploy` and `presence`; the word "hook" does not appear.
- **The repo owns two hook events.** `ai/claude/settings.json` declares `SessionStart` and `SessionEnd`, wired by `merge_claude_settings()` through *hardcoded jq paths* (`(.hooks.SessionStart[0].hooks[0].command) = $cmd`). Adding an event means editing shell.

So a persona is presence: a text block that asks nicely. This spec makes it binding.

## What

A third manifest mode beside `deploy` and `presence` — **`bind`** — that emits each harness's native hook primitive, wrapping one line that calls `dotf`.

**The payload is `dotf`, and that is the agnosticism requirement, not a preference.** The user's standing constraint is that this work across many sessions and different agents. Decision logic lives once, in Go, testable with no harness installed:

```
dotf harness gate --harness <h> --event <e> --role <r>   # exit 0 allow, 2 block
```

Each harness contributes a thin declarative wrapper around that call. A thick per-harness adapter is a second implementation that will drift — the failure this repository has catalogued repeatedly.

### The primitives, verified against installed packages rather than ADR-027

| Harness | Presence | Action (gate) | Evidence |
|---|---|---|---|
| `claude` | `SessionStart` | `PreToolUse` | live in `~/.claude/settings.json` |
| `opencode` | `chat.system.transform` | `tool.execute.before` | `@opencode-ai/plugin/dist/index.d.ts` |
| `pi` | `session_start` | `tool_call` | `pi-coding-agent/docs/extensions.md`, which documents *"`tool_call` errors block the tool (fail-safe)"* |

ADR-027's *render kinds* were found stale in 2 of 5 (#563); its **hook names are accurate**. No cell is lost to a missing primitive.

### Severity is declared per skill

Decided by the user 2026-08-26. `skills:` becomes a list of objects:

```yaml
skills:
  - id: adversarial-review
    enforce: block
  - id: cyclomatic-complexity
    enforce: warn
```

Both failure modes are real and this is the only shape that avoids each: a gate that blocks on everything becomes a gate whose normal state is red — the failure already recorded in `harness/review-attestation.json` about CodeRabbit — and a gate that only warns is presence with extra steps.

## Out of scope

- **Invocation (dispatch).** Listed in #561's Scope, absent from its Acceptance. Deferred to #563: pi's `~/.pi/agent/agents/**/*.md` is its highest-precedence agent layer, so a *rendered* persona is already a dispatchable one — dispatch comes free with that edit rather than needing bespoke routing here.
- **`agy` and `copilot`.** Presence-only per #561, unchanged.
- **Deciding which skills block.** The mechanism is this spec's; the per-persona `enforce` values are a content edit in the vault definitions.

## Risks / open questions

- **The hook surface has a live external writer, and a naive emission would delete it.** The deployed `~/.claude/settings.json` carries 12 events; `SessionStart` runs `dotf mem session-start` (ours) and **`PreToolUse`, `Stop` and nine others run `~/.orca/agent-hooks/claude-hook…` (Orca's)**. `merge_claude_settings` preserves them only because it touches keys declared as ours. The same writer is present in `~/.pi/agent/extensions/` (three `orca-*.ts`, found in #1248). **Resolution**: emission is additive and idempotent — append our entry to an event's array, recognise it by a stable marker on re-run, never duplicate, never drop a foreign entry. A test drives the case where Orca's hook is present.
- **The schema change would silently disable the roster drift guard.** `specs/HARNESS-046/check-roster-consistency.py:62` parses `skills:` with `^skills:\s*\[(.*?)\]` — an inline array — and line 65 reads `... if skills else []`. Under the new block form the regex finds nothing and the guard returns **an empty skill list without complaint**, comparing "no skills" against the roster. The guard HARNESS-046 added to catch drift would stop catching it, silently. **Resolution**: an unparseable-but-present `skills:` key must be a loud error, and AC7 drives it.
- **Open**: whether the gate can observe skill *consumption* on every harness, or only *announcement*. Claude exposes tool names to `PreToolUse`; whether a skill invocation is distinguishable there is unverified, and pi/opencode may differ. If a harness cannot observe consumption, its Action cell degrades to announcement and that must be stated per harness, never averaged.

## Acceptance criteria

- [ ] **AC1** — `harness/manifest.json` declares a `bind` mode, schema-validated, and the emitted hook command is generated from it rather than from hardcoded jq paths.
- [ ] **AC2** — **Presence, per harness**: a dispatch through claude, opencode and pi shows the persona's forced skills in the model's context. Proven by a dispatch, never by a config file containing a key.
- [ ] **AC3** — **Action, per harness**: with a skill declared `enforce: block` unconsumed, a tool call through that harness is **blocked**. Proven per harness; a harness that cannot observe consumption is recorded as such rather than counted as passing.
- [ ] **AC4** — a skill declared `enforce: warn` emits and does **not** block, asserted on the same path as AC3 so the two cannot collapse.
- [ ] **AC5** — emission is idempotent: a second setup run adds no duplicate hook entry.
- [ ] **AC6** — **a foreign hook entry survives.** With Orca's `PreToolUse` present, deployment appends alongside it and both remain after a re-run.
- [ ] **AC7** — a `skills:` key the parser cannot read is a **loud failure**, in both the schema validation and `check-roster-consistency.py`; it never resolves to an empty list.
- [ ] **AC8** — `dotf harness gate` decides with no harness installed, exercised in Go tests.

### AC3 and AC4 are falsified as written — measured 2026-09-01

The gate was bound to a live machine for the first time on 2026-09-01 and
measured. Two of the criteria above cannot be satisfied as stated, and one of
them would **pass on a broken system**. Recorded here rather than quietly
rewritten: the correction is HARNESS-106's subject, and this spec's archive
review must see that these were wrong rather than find them silently adjusted.

**AC4 asserts an emission through a channel that does not persist.** A `warn`
decision is written to stderr with exit 0. A `PreToolUse` hook's streams on exit
0 are not recorded in the session transcript — every hook-summary record in a
measured session was Stop-family, and a `Skill` call that provably fired the hook
(it wrote a consumption ledger entry) left no transcript record of the firing. So
"emits" is not observable on the path AC4 names, and asserting it "on the same
path as AC3" does not rescue it. The gate's only durable channel is the
consumption ledger, which records **only** skill invocations.

**AC3 is satisfied by the failure it exists to detect.** It asks that with a
skill at `enforce: block` unconsumed, a tool call is blocked. No persona holds a
skill-invocation tool — `harness/capability-map.json`'s vocabulary has no entry
that maps to one, and claude's `tools:` is an allow-list. So under `block` a
persona is refused on every call *and cannot reach the one action that clears
it*: `isSkillTool`'s escape ("invoking a skill is never blocked: forbidding it
would deadlock the session") assumes an agent able to emit a call it was never
granted. AC3 would observe a blocked call and record a pass, on a persona that is
permanently deadlocked.

**AC2's discipline is the one that held.** *"Proven by a dispatch, never by a
config file containing a key"* is exactly what caught this: two probes that read
a config file would have reported success.

Both are corrected in `specs/HARNESS-106-skill-capability/` — the capability that
makes consumption possible, and the durable decision record that makes a `warn`
observable. This spec should not archive until AC3 and AC4 are restated against
channels that exist.

## References

- Bitácora: `mlorentedev/dotfiles#561`; scoping comment 2026-08-26 carries the measurements.
- ADR-027 §3 (determinism via hooks), epic #558. Its render kinds are stale (#563); its hook names are not.
- The coexistence constraint's first instance: #1248 (Orca writing into `~/.pi/agent/extensions/`).
- The guard this must not silently disable: `specs/HARNESS-046/check-roster-consistency.py`.
- Design constraints and the agnosticism requirement: the session's memory note `project-orchestrator-agnostic-binding`.
