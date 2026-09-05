---
id: "HARNESS-084-doctrine-budget-policy"
type: spec
status: implementing # draft | implementing | verifying | archived
created: "2026-09-05"
issue: "mlorentedev/dotfiles#1241"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-084-doctrine-budget-policy

> **Naming**: file lives at `<repo>/specs/HARNESS-084-doctrine-budget-policy/proposal.md`. `HARNESS-084-doctrine-budget-policy` is `AREA-NNN-slug` (e.g. `TOOL-001-secret-drift`).

## Why

`.gemini/GEMINI.md` is capped at 12000 characters by Antigravity, and AGENTS.md is 27293, so
that surface receives a **compact doctrine payload** instead of the constitution. The payload
has now outgrown the cap twice. #1241 already rejected raising it: 12000 is the vendor's real
limit, so raising the assertion moves truncation from a red CI check to silent loss at the
destination. That left "trim the prose until it fits", which is worse — the compact payload is
what agy and codex get **instead of** the constitution, and shaving words off a prohibition is
how it stops being one.

The immediate trigger: the auto-merge exception paragraphs put the payload 826 characters over.

## What

A doctrine record may mark a region as full-only:

```
<!-- full-only:begin -->  …  <!-- full-only:end -->
```

`render_region_compact` drops it; the full surfaces keep it verbatim. The markers themselves
reach no surface — `render_region` strips them, `render_region_compact` consumes the raw form
because it needs them to find the region.

This follows the compactor's existing rule exactly: **delete lines it can identify exactly,
never paraphrase.** It is a second exactly-identified deletion, not a new kind of behaviour.

**What belongs behind the marker is narrow, and the reason is safety before budget.** A capped
rules file should carry the PROHIBITION. The EXCEPTION to it — its four conditions and the
reasoning that earned them — is a policy decision a human makes on a specific pull request, and
an agent that knows an exception exists may try to qualify for it. The exception stays verbatim
in AGENTS.md, where the human reads it.

## Out of scope

- **A general budget policy for the payload.** #1241 asks for one; this gives the mechanism a
  policy would need, not the policy. Deciding which records may mark regions, and how much may
  leave the capped surface, is a separate judgement.
- **Raising any cap.** Rejected on #1241 and not revisited.
- **Paraphrasing or summarising doctrine.** The compactor has never done it and this does not
  start.

## Risks / open questions

- **A marker that dropped its region from every render would satisfy every cap check while
  silently deleting doctrine.** From the cap's point of view that failure is indistinguishable
  from success. The guard therefore asserts both directions, not just the capped one.
- **A missing end marker would swallow the rest of the record**, and the payload would only get
  smaller — again invisible to a cap assertion. The guard asserts content after the region
  survives.
- **The mechanism makes it easy to move something that should have stayed.** Nothing enforces
  which regions qualify; the boundary is stated here and in the compactor's comment, not
  checked. Recorded as a limitation rather than solved.

## Acceptance criteria

1. A region between `full-only` markers is absent from the capped payload and present, verbatim,
   in the full surface.
2. Content following the closing marker survives the compact render.
3. The markers reach neither surface.
4. `render_region` returns the missing-source error status rather than a pipeline's, so a
   missing record still fails.
5. `.gemini/GEMINI.md` is under its 12000-character cap with the auto-merge exception present in
   AGENTS.md.
6. `shellcheck` reports no new findings and the bats suite is green.
