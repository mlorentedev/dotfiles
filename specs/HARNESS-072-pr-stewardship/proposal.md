---
id: "HARNESS-072-pr-stewardship"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-08-15"
issue: "mlorentedev/dotfiles#963"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, harness, review]
template_version: "1.0"
---

# HARNESS-072-pr-stewardship

## Why

Opening a PR is treated as the end of a change, and it is not: checks report
afterwards, and reviewer bots report after *that*. Nothing currently obliges an
agent to still be there when they do. The worked example is PR #959 — all checks
went green, a session would have been justified in walking away, and CodeRabbit
then posted **10 comments including four Majors**: a `| tee` pipeline discarding
the reviewer's exit status, an `sh -c` fallback that cannot run on the Windows it
was documented for, a pool whose deletion silently disabled a security gate, and
an acceptance criterion ticked without its evidence. Every one of those shipped
into a follow-up PR instead of the one that introduced them.

The procedure to handle this already exists (`pr-review-triage`) and is correct.
What it lacks is a *binding trigger* and a *correct end condition*: it is a
skill, so an agent that does not think to load it simply does not, and its
trigger fires at "checks finished", which is demonstrably too early.

## What

Two changes, one obligation and one procedure.

**1. A new `enforced` harness region.** Sourced from a section in
`pattern-change-lifecycle.md` beside `definition-of-done`, whose Review item this
elaborates, and injected verbatim into every agent surface that region reaches.
It states two rules:

- *A PR you open is watched, not abandoned.* The window closes at the first of:
  an actionable reviewer comment (attend to it), or N minutes after checks
  settle. Pushing a fix reopens it, because the reviewer re-reviews.
- *A PR that touches a `specs/<id>/` folder gets an adversarial review by
  default.* This names an obligation that already binds mechanically — spec-gate
  refuses to merge a PR closing a spec's issue without archiving it,
  `dotf spec archive` refuses without a passing `review.md`, and since #958 the
  pool refuses one signed by the wrong model. Stating it converts three
  mechanical refusals into one intention an agent can act on *before* being
  refused by any of them.

**2. `pr-review-triage` amended** in its vault source: the end condition covers
the reviewer bot rather than only CI, and the wait is expressed as a *contract*
with `gh` commands — agent-neutral — rather than a polling implementation, since
Claude has background tasks and pi and opencode do not.

After this, an agent that opens a PR and walks away at CI-green is violating a
rule injected into its own instructions, rather than merely failing to load an
optional skill.

## Out of scope

- Making the babysit a scheduled or background mechanism. The contract is what
  binds; how a given harness waits is its own business.
- Auto-applying reviewer comments. `pr-review-triage` already forbids acting
  without human confirmation, and that stays.
- Detecting or working around a reviewer's quota exhaustion. The wording below
  must make an agent *report* that state honestly, but building a gate that
  recognises it is #906.

## Risks / open questions

- **The obvious end condition is already known to be wrong, and this is the main
  design risk.** "An actionable comment arrived" is satisfied by a comment that
  is not a review at all. Observed on PR #973 the same day this spec was written:
  every check green, and CodeRabbit posted *"Review limit reached — you've
  reached your PR review limit, so we couldn't start this review."* That is a
  comment, it is not review activity, and a naive babysit rule treats the PR as
  reviewed. The region's wording must make that non-lawyerable: **a reviewer-bot
  notice that no review ran is the PR being unreviewed, and saying so out loud is
  part of closing the window.** This is the same shape as BUG-077 and the
  `pattern-verification-fails-toward-unproven` family — absence of review
  presenting as clean review.
- **A rule injected everywhere is expensive to get wrong.** It lands verbatim in
  every agent's instructions, so it must be terse and free of Claude-specific
  vocabulary. The mitigation is the same one `definition-of-done` uses: bind
  existing obligations, do not restate them.
- **N is a guess.** 10 minutes fits CodeRabbit's observed latency on this repo
  and is short enough not to park an agent. It is a number in prose, not a
  constant, so it costs nothing to revise.
- **A region added to `enforced` but missing from a target's `inject` list
  silently misses that surface.** Exactly the producer-updated /
  consumer-forgotten class that BUG-077 (#969) was. Mitigated by AC5 below:
  `compile-harness.sh --check` is the test, not a hand count.

## Acceptance criteria

- [ ] **AC1** An `enforced` region exists, sourced from a vault pattern section,
      and is listed in `harness/manifest.json` with an `id` and a `source`.
- [ ] **AC2** The region is injected into every surface `definition-of-done`
      reaches — both `targets` entries — verified by `compile-harness.sh --check`
      passing, not by counting files by hand.
- [ ] **AC3** The region's text states that a reviewer-bot notice reporting that
      no review ran (rate limit, quota) leaves the PR **unreviewed**, and that
      the agent must say so rather than close the window on it.
- [ ] **AC4** "Important PR" resolves to a test an agent applies without
      judgement: the PR touches a `specs/<id>/` folder.
- [ ] **AC5** `pr-review-triage`'s trigger and end condition cover the reviewer
      bot, not only CI, and the wait is expressed with `gh` commands runnable by
      an agent with no Claude-specific primitives.
- [ ] **AC6** The vault source and the committed harness records agree —
      `compile-harness.sh --check` reports no drift after `--refresh`.

## References

- Bitácora board: mlorentedev/dotfiles#963
- `harness/manifest.json` — `enforced` + `targets`; `definition-of-done` is the precedent
- `harness/skills/pr-review-triage/SKILL.md` — the procedure this makes binding
- `00_meta/patterns/pattern-change-lifecycle.md` — Definition of Done §4, the parent rule
- `00_meta/patterns/pattern-verification-fails-toward-unproven.md` — why the quota-notice case is a design requirement and not an edge case
- PR #959 (CI green, then 4 Majors), PR #973 (CI green, no review at all), #906 (the quota gap)
