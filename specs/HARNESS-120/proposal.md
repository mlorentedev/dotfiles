---
id: "HARNESS-120"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-05"
issue: "mlorentedev/dotfiles#1537"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-120

## Why

The orchestrator has every part it needs and no command that puts them
together. `dotf harness suggest` derives the persona a prompt implies, and
stops; `dotf agent run` dispatches to a pool, but only if a human supplies the
role and the tier by hand. Between those two sits a manual step, and the
measurement says the manual step is not taken: of **11 745** gate decisions
**11 168 (95.1%)** are `no-role`, and **18 of 20** sessions produced no persona
decision at all. The apparatus governs a lane almost nothing travels, because
entering that lane costs a deliberate act nobody performs.

Two fields tell that story precisely, and both were measured for this spec.

**`Persona.Model`** — the tier each record declares for itself — is written at
`cli/internal/harness/persona.go:107` and read by **nothing**. `agent run` asks
its caller for `--tier` instead. A persona states which tier it belongs on, the
dispatcher walks that tier's chain, and the two are never connected.

**`Request.Role` reaches no real backend.** It is threaded from `--role`
through `Options` into `Request` at `dispatch.go:136`, and the only code that
reads it is the dry-run backend's echo string (`dryrun.go:22`). `Subprocess`
sends `--model` and the task (`backends.go:42`); `Hive` sends `--model`,
`--timeout` and `--prompt` (`backends.go:70-78`). Neither sends the persona.

So `dotf agent run --role reviewer` today dispatches a **generic** agent that
happens to be logged as a reviewer. The specialization the six records carry —
mandate, method, boundaries, forced skills — stops at this process boundary.
That is the half of the goal worth more than the routing: a task can already be
sent to another pool; it cannot yet be sent to a *persona*.

## What

A new `dotf agent auto --task "<text>"` that dispatches the persona the task
implies, composing five things that already exist and adding one that does not:

| # | Step | Status |
|---|---|---|
| 1 | `harness.Suggest` + `harness.ResolveRoles` → persona | exists (16/18 rules resolve) |
| 2 | `persona.Model` → tier | **new** — nothing reads that field |
| 3 | the persona's record travels with the task | **new** — nothing sends it |
| 4 | `agent.Dispatch` → walks the tier's chain | exists, 5 pools |
| 5 | one JSON record on stdout | exists (`agent.Record`) |

Observable difference: today, running work as the right persona on the right
tier is `dotf harness suggest`, read the output, decide, then type
`dotf agent run --role X --tier Y --task "…"` — and even then a generic agent
answers. After this it is one command that resolves the persona and its tier
from the task text and the records themselves, sends the persona's own record
to the dispatched process, and refuses rather than guessing when it cannot.

Step 3 is a preamble, not a protocol. Both backends already accept task text
(stdin for `claude`/`pi`, `--prompt` for `hive delegate`), so the record is
prepended to it, delimited, with the task after. No new transport, no
per-harness argv, no change to `harnessFor` — which ADR-035 §4 keeps as a map
rather than a director.

The record on stdout gains the resolution it performed — `role`, the pattern
that matched, and whether each was inferred or dictated — so a caller can tell
a derived route from a supplied one.

## Decisions

**Dispatch is PROMPTED, not automatic.** This answers issue #1537's first
acceptance criterion, which asks for a recorded decision rather than a
particular one.

The `UserPromptSubmit` hook cannot be the dispatcher. Exit 2 on that event
"blocks prompt processing and erases the prompt", so the hook must fail open —
and a dispatcher that cannot refuse is a dispatcher that cannot be trusted with
the decision. It also fires before the session has read the prompt, so it would
be routing on a match rather than on understanding, with the substring matcher
noted below as its only judgement.

So the hook keeps suggesting, and the actor that dispatches is the session
reading that suggestion. What changes here is the cost of acting on it: one
command instead of a manual translation into two flags nobody supplies. The
suggestion names the persona; `dotf agent auto` runs it.

Revisit when the suggestion→dispatch conversion rate is measured. If sessions
still do not dispatch when the act costs one command, the obstacle was never
the friction and automatic dispatch deserves reopening on that evidence.

**Ambiguity refuses.** Carried from HARNESS-110 unchanged, now with a
consequence: an advisory layer may print two personas, a dispatcher may not
pick one.

## Out of scope

- **`--worktree`, and worktree lifecycle generally.** Deliberately deferred to
  a follow-up rather than bundled: it is independent of every criterion here,
  it would push this past the atomic-diff cap, and `agent run --cwd` already
  accepts a working copy the caller made. Nothing in this PR removes, creates
  or leases a worktree.
- **`enforce: block` promotion** — no severity moves here.
- **Multi-step sequencing.** `auto` runs ONE persona and returns. The
  transitions between phases are declared in prose in each record's
  `Boundaries` and remain unexecuted; extracting them is a separate ticket that
  wants this one as its base.
- **agy / the second harness.** #1532's territory.
- **Verifying that the dispatched model OBEYED its persona.** This ships the
  record into the process; whether the far-side agent honours it is a property
  of that model, not of this change. The criterion below is that the record was
  sent, provably — not that it worked.

## Risks / open questions

- **Why the deferred worktree step will not be `add → run → done` when it
  comes.** Recorded now because the shape is the obvious one and it is wrong:
  if the dispatched agent left uncommitted work, `done` destroys it and the
  caller learns nothing. It is the same data-loss class that forces the
  `UserPromptSubmit` hook to fail open. Whatever lands later creates and
  reports; teardown stays a separate, human-authorised `dotf worktree done`.
- **The persona record is prepended to the task, so it spends the dispatched
  model's context.** The six records run 60–63 lines. That is affordable per
  dispatch and is the cost of the specialization being real; but a record that
  grows without bound would degrade every dispatch silently. Send the record
  whole in this version — truncating a mandate is worse than paying for it —
  and treat a size ceiling as a follow-up with a measurement behind it, not a
  guess now.
- **The far side may ignore the preamble.** Nothing forces a model to obey a
  role it is handed as text. What this can prove is that the record was sent;
  the criterion is written that way deliberately, and the honest claim in
  `verification.md` must match it.
- **Ambiguity must refuse, not rank.** `code-complexity-and-refactor` resolves
  to `[builder, reviewer]` and `spec-driven-development` to `[planner,
  reviewer]`. HARNESS-110 recorded that ambiguity is a first-class output and
  that determinism comes from sorted output, not from narrowing. A dispatcher
  cannot print two personas and carry on, so it refuses and names both — the
  refusal IS the deterministic behaviour, and `--role` remains the way to
  settle it.
- **A tier the map cannot serve must not be silently downgraded.** `top` does
  not fall back by design (`dispatch.go:88`). `auto` resolving `architect` →
  `top` inherits that, so an unavailable top pool escalates rather than
  answering from a weaker model. This is existing behaviour and must stay
  visible in the record, not smoothed over by the new layer.
- **A persona whose `model:` is absent or not in `top|mid|low` must refuse.**
  Defaulting a tier here would be the same defect the gate's loader avoids by
  applying no default severity: a route nobody chose, silently taken.
- **The keyword matcher is substring-based** (`triggers.go:116`). A task
  mentioning "docker" in passing resolves `shipper`. This is inherited, not
  introduced, and it is a reason `auto` prints what matched — a route that
  cannot be judged is obeyed on its worst day as readily as its best.

## Acceptance criteria

The task text in AC1 is named exactly, not described, so the criterion is
reproducible from this file. It was measured against the shipped rules: it
matches `pattern-bitacora-tracking` alone, whose skills intersect only
`planner`, whose record declares `model: mid`.

- [ ] **AC1** — `dotf agent auto --task "open a ticket for the bitacora"
      --backend dry-run`, with neither `--role` nor `--tier` supplied, emits
      one JSON object reporting `role: planner` and `tier: mid`.
- [ ] **AC2** — a task resolving to two personas dispatches nothing, exits
      non-zero, and names both candidates in its refusal.
- [ ] **AC3** — a task matching no trigger rule dispatches nothing and refuses
      with a reason distinguishable from AC2's. Asserted as *these two
      refusals differ*, never as two fixed strings.
- [ ] **AC4** — a persona whose record declares no tier, or one outside
      `top|mid|low`, is refused rather than defaulted, and the refusal names
      the persona, the offending value, and the three legal tiers.
- [ ] **AC5** — `--role` skips the join entirely and `--tier` overrides the
      persona's declared tier; the output distinguishes each field as inferred
      or dictated.
- [ ] **AC6** — the persona's record reaches the dispatched process. Asserted
      against the backend's received `Request`, not against stdout: the task
      text sent for `--role reviewer` contains reviewer's mandate, and the text
      sent for `--role builder` does not.
- [ ] **AC7** — one REAL dispatch (not `dry-run`) answers from a pool chosen by
      the persona's own declared tier, with its record quoted verbatim in
      `verification.md`.

## References

- Bitácora board: `mlorentedev/dotfiles#1537`
- `docs/adr/adr-032-cross-harness-agent-orchestration.md` — the dispatch
  contract: bounded, denied-by-default, top never degrades
- `docs/adr/adr-035-model-map-routing-registry.md` — `model-map.json`'s two
  cadences (compile-time `tiers`, run-time `chains`)
- `specs/HARNESS-110/proposal.md` — the role is DERIVED, never declared;
  ambiguity is a first-class output
- Related: #1532 (making the gate binding), #1536 (decision-file growth),
  #1538 (`Persona.Model` unread — closed by criterion 1 here)
