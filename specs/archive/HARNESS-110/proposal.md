---
id: "HARNESS-110"
type: spec
status: archived # draft | implementing | verifying | archived
created: "2026-09-02"
issue: "mlorentedev/dotfiles#1436"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-110

> **Naming**: file lives at `<repo>/specs/HARNESS-110/proposal.md`. `HARNESS-110` is `AREA-NNN-slug`.

## Why

<!-- from issue #1436: HARNESS-110: the orchestrator decides and records but nothing triggers it — derive the role from the skills join and bind a suggester -->

The orchestrator can decide (`dotf harness gate`) and, since #1435, record every decision it takes.
Nothing triggers it. `harness/triggers.json` holds 18 rules and `dotf harness suggest` reads them,
but the command is wired to no hook, so it has never fired outside a manual invocation. A session
gets no signal about which persona should be doing the work in front of it — the routing layer
exists end to end except for its first inch.

## What

On every user prompt, a `UserPromptSubmit` hook runs the existing trigger match and prints, as plain
stdout the session can see, which persona (or personas) the matched skills belong to. Advisory only:
it suggests, never dispatches.

The role is **derived, not declared** — `trigger.skills ∩ persona.skills`, joining two SSOTs that
already exist. Re-measured on `745f320` before writing this spec:

| | measured |
|---|---|
| skills named by persona records | 34 |
| skills named by `triggers.json` | 30 |
| intersection | 28 |
| rules resolving to ≥1 persona | **16 / 18** |

The two that resolve to nobody, `shell-standards` and `powershell-ascii-only`, declare no skills at
all. They are pattern-only rules with no owner — a correct answer, not a gap.

A `role:` field in `triggers.json` was rejected: it would be a *second* place stating the role↔skill
relationship the persona records already own, and two places drift silently. Deriving means adding a
skill to a persona updates routing for free — no new file, no new schema, no sync guard.

## Out of scope

- **Autonomous dispatch.** v1 suggests; it does not launch. Launching is gated on #1434 — while a
  named dispatch silently disables its own gate, an auto-launcher multiplies ungated agents instead
  of orchestrating them.
- Phase detection from `Skill` events — a separate spec once this one is live.
- Any change to #1434, or to `triggers.json`'s rule set.
- Repairing the six persona records still on the legacy inline `skills:` form. `parseSkills` reads
  both and `UnmigratedSkills()` already tracks them; migrating them is bookkeeping, not this spec.

## Decisions

**The suggestion shows its derivation.** Owner's call, 2026-09-02, choosing the fullest of three
shapes: role, the pattern and skills that produced it, and the action to consider.
The second field is labelled `pattern:` because that is what `Suggestion` carries — pattern names,
not trigger-rule ids. Labelling it `rule:` would have been a name that does not describe its
contents, the very defect #1448 tracks.

```
[persona] builder  ← pattern: pattern-testing-standards
  skills: test, test-driven-development
  → consider adopting `builder` and invoking test-driven-development

[persona] builder | reviewer  ← pattern: pattern-language-standards
  skills: cyclomatic-complexity
  → each of these declares the matched skill; the work decides which
```

The reasoning: a suggestion that cannot be judged is obeyed on its worst day as readily as on its
best. Showing the matched pattern and skills lets a session dismiss a bad match instead of following
it. The cost is real and accepted — this is charged to every prompt, and it is the shape most likely
to become noise a session learns to skip. If that happens it will show up as suggestions being
ignored, not as an error; revisit on measurement, not on anticipation.

Zero roles prints nothing, which keeps the two pattern-only rules silent rather than announcing that
nobody owns them.

The named entry point is the **composite**, not the alphabetically-first skill. `testing-standards`
matches `[test, test-driven-development]`; naming `test` would point at the prerequisite instead of
the thing worth invoking. `entrySkill` prefers a skill whose `DefaultSkillDependencies` closure
covers another matched skill.

## Risks / open questions

- **The prompt-text field name is undocumented.** The published hooks reference names `session_id`,
  `prompt_id`, `transcript_path`, `cwd`, `permission_mode`, `effort` and `hook_event_name` — and not
  the field carrying the prompt. Assuming a spelling is exactly how #1434 happened. Mitigated by AC6:
  accept the plausible spellings and record which one arrived.
- **A malfunctioning hook can destroy user input.** On `UserPromptSubmit`, exit code 2 is documented
  verbatim as *"Blocks prompt processing and erases the prompt"* — a data-loss mode strictly worse
  than the gate's, whose worst case is a refused tool call. Mitigated by AC7 (never exit non-zero).
- **Latency is charged to every prompt**, not to a subset. 18 regexes on each one. Mitigated by AC8.
- **A second parser of `AGENT.md` would reintroduce a solved bug.** `check-roster-consistency.py`
  had to be repaired once for reading `skills:` with a regex that silently returned `[]` on the
  block form. Mitigated by AC1 (consume `LoadPersona`) and AC3 (guard in Go).
- Open: the hook fires on *every* prompt, including ones continuing work already underway, where a
  suggestion is noise. Deliberately not solved in v1 — suppression needs session state this spec
  does not have. Revisit only if the noise is measured, not anticipated.

## Acceptance criteria

- [x] **AC1** `harness.ResolveRoles(suggestion, personas) []string` — a pure, sorted join, no I/O.
      Only `kind: invocable` personas are candidates: `hermes-nan` is `kind: autonomous` and cannot
      be adopted by a session.
      It consumes personas via `harness.LoadPersona`; it never re-parses `AGENT.md`. The correct
      parse exists, is documented at `persona.go:71-80`, and fails loud under C15.
- [x] **AC2** Ambiguity is returned in full, never ranked or narrowed. Both known ambiguous rules are
      fixtures: `code-complexity-and-refactor → [builder, reviewer]` and `spec-driven-development →
      [planner, reviewer]`. A rule with no skills returns empty, and empty is not an error.
- [x] **AC3** A drift guard **on the join itself**, in Go, against `LoadPersona`. Nothing today spans
      `triggers.json` and the persona roster. It asserts the resolving-rule count **as a floor**
      (never an equality — an exact count goes red on coverage *rising*, and its only fix is to bump
      the constant, which is how a guard gets disabled) **and** that every persona contributes ≥1
      skill, so a reader returning an empty set reads red instead of silently shrinking the join.
- [x] **AC4** A `UserPromptSubmit` hook bound from `harness/manifest.json`, so a deploy propagates it.
      Never hand-written into `settings.json`. (Corrected while implementing: `emit_hooks` already
      exists under `agents.bind[]` and carries the gate plus both `mem` hooks. This is a fourth
      entry on an existing mechanism, not a new one — the earlier claim was wrong.)
- [x] **AC5** The hook reads its payload from **stdin**, never from a `--prompt` argument. A
      shell-quoted user prompt on a command line is an injection surface.
- [x] **AC6** The field carrying the prompt text is **measured, not assumed**: accept the plausible
      spellings, and record which one arrived, in the defensive shape of `OutcomePayloadUnrecognised`.
- [x] **AC7** The hook can never exit non-zero, under any input, including a malformed payload and an
      unreadable persona record. Asserted by test, not by inspection.
- [x] **AC8** A stated latency budget (<20 ms) with a test asserting it.

## References

- Bitácora board: `mlorentedev/dotfiles#1436`
- #1434 (HARNESS-109) — hard precondition on any `enforce: block` promotion; not on this spec, which
  is advisory
- #1448 (GUARD-009) — the "claim outlives its referent" detector; this spec's premise was re-probed
  rather than trusted, per that ticket's discipline
- `cli/internal/harness/persona.go:71-80` — why the persona parse uses real YAML
- `cli/internal/harness/decision.go` (#1435) — the record every gate decision writes
