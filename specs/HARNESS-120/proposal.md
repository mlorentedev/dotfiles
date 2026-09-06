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

The gap is smaller than it looks. `Persona.Model` — the tier each record
declares for itself — is written at `cli/internal/harness/persona.go:107` and
read by **nothing**: `agent run` asks its caller for `--tier` instead. So a
persona already states which tier it belongs on, and the dispatcher already
walks that tier's chain, and the two are never connected.

## What

A new `dotf agent auto --task "<text>"` that dispatches the persona the task
implies, composing five things that already exist and adding one that does not:

| # | Step | Status |
|---|---|---|
| 1 | `harness.Suggest` + `harness.ResolveRoles` → persona | exists (16/18 rules resolve) |
| 2 | `persona.Model` → tier | **new** — the only missing link |
| 3 | `worktree.Add` → isolated working copy (opt-in) | exists, with leases |
| 4 | `agent.Dispatch` → walks the tier's chain | exists, 5 pools |
| 5 | one JSON record on stdout | exists (`agent.Record`) |

Observable difference: today, running work as the right persona on the right
tier is `dotf harness suggest`, read the output, decide, then type
`dotf agent run --role X --tier Y --task "…"`. After this it is one command
that resolves both from the task text and the personas' own records, and
refuses rather than guessing when it cannot.

The record on stdout gains the resolution it performed — `role`, the pattern
that matched, and the worktree it ran in when it made one — so a caller can
tell an inferred route from a dictated one.

## Out of scope

- **The `UserPromptSubmit` hook still only suggests.** Making the hook itself
  dispatch is issue #1537's first open question and is deliberately left for a
  follow-up: this ticket builds the mechanism a dispatcher would call, and the
  decision about who calls it is cheap once the call exists and expensive to
  reverse once wired into a hook that cannot fail closed.
- **`enforce: block` promotion** — no severity moves here.
- **Multi-step sequencing.** `auto` runs ONE persona and returns. The
  transitions between phases are declared in prose in each record's
  `Boundaries` and remain unexecuted; extracting them is a separate ticket that
  wants this one as its base.
- **`dotf worktree done`.** See the risk below: `auto` never removes a
  worktree.
- **agy / the second harness.** #1532's territory.

## Risks / open questions

- **Removing a worktree after a dispatch is a data-loss mode, so `auto` will
  not do it.** The advisory shape for this command was `worktree add → run →
  worktree done`, and the last step is wrong: if the dispatched agent left
  uncommitted work, `done` destroys it, and the caller learns nothing. This
  repository has already ruled on the equivalent case — `UserPromptSubmit`
  exit 2 "blocks prompt processing and erases the prompt", and that data-loss
  mode is why the hook must fail open. So `--worktree` creates and reports;
  teardown stays a separate, human-authorised `dotf worktree done`.
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

- [ ] `dotf agent auto --task "<spec-ish text>" --backend dry-run` resolves a
      single persona, reads that persona's declared tier from the deployed
      record, and emits one JSON object carrying `role` and `tier` consistent
      with it — with no `--role` and no `--tier` supplied.
- [ ] A task resolving to two personas dispatches nothing, exits non-zero, and
      names both personas in its refusal.
- [ ] A task matching no trigger rule dispatches nothing and refuses with a
      reason distinguishable from the ambiguous case.
- [ ] A persona whose record declares no tier, or one outside `top|mid|low`,
      is refused rather than defaulted.
- [ ] `--role` overrides resolution and skips the join entirely; `--tier`
      overrides the persona's declared tier. Both are recorded in the output as
      dictated rather than inferred.
- [ ] `--worktree <slug>` creates an isolated worktree, dispatches with the
      task's working directory inside it, reports its path and branch in the
      JSON, and leaves it on disk whatever the dispatch's outcome.
- [ ] One REAL dispatch (not `dry-run`) is observed answering from a pool
      chosen by the persona's own declared tier, and its record is quoted in
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
