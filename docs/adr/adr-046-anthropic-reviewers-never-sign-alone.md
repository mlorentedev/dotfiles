---
id: "ADR-046-anthropic-reviewers-never-sign-alone"
type: adr
status: accepted
owner: manu
date: "2026-10-09"
supersedes: []
extends: [adr-037-review-runners-bounded-and-budgeted, adr-040-automatic-reviews-bounded-per-push]
issue: mlorentedev/dotfiles#1923
tags: [architecture, decision, review, pr-agent, ci, inference, budget]
created: "2026-10-09"
---

# ADR-046: An Anthropic reviewer never signs alone, and every reviewer's share is a declared weight

## Context

`harness/reviewer-pool.json` kept every Anthropic model out of the spec archive gate. Claude implements nearly
every change in this repo, and BUG-074 was reviewed twice by the model family that wrote it before anyone
noticed. The rule bought independence, but it also kept the strongest available reviewer out of the reviews where
quality matters most: high-risk specs and large PRs.

PR-Agent's CI pool drew among its members with equal weight. That made each member's share depend on how many
members there were. A model priced 15 times another would have received the same share, and a cheap model added to
the pool silently diluted every other member's share.

The owner set a monthly budget of $60-65 of Anthropic credit for both flows. Measured volume is about 2,500 PR
reviews a month (10-15 PRs in parallel) and 4-5 adversarial reviews per hour of spec work.

## Decision

1. **An Anthropic model never signs an adversarial review alone.** The first signature is always another vendor's.
   An Anthropic member signs in one of two ways. It can add a second signature beside a first signer of a
   different vendor, with `signs: second` and `dotf spec review --second`. Or it can sign as a fallback when the
   first signers failed for a classified provider reason, with `signs: fallback` and `--fallback-reason`. A FAIL
   verdict is never such a reason.
   - Pool loading refuses an Anthropic member that would sign first.
   - The archive gate checks the roles, the recorded reason and that the two vendors differ.
2. **A spec whose `proposal.md` declares `risk: high` needs a second signature** (`review-second.md`) before it
   archives. Sonnet is the only `signs: second` member.
3. **PR-Agent draws by declared weight.** Each member's `pr_agent` block in the pool gives its model, its weight and
   its reasoning effort. A member that does not answer its probe gives up its weight to the rest for that run.
   Weights at adoption: mimo 27, glm 19, deepseek 19, Haiku 35.
4. **A risky PR reviews first on Sonnet.** A PR is risky when it has at least 1,500 changed lines or carries the
   `deep-review` label. Sonnet is never drawn. When it does not answer, the PR goes to the draw with a note.
5. **The allowlist of Anthropic models CI may spend on lives in `scripts/pr-agent-route.sh`, not in the pool.** A
   pool edit naming another model fails the draw. A test requires a price row and a worst-review ceiling for every
   allowed model.
6. **One key, two names.** The local review reads the same Bitwarden item as PR-Agent, under
   `REVIEW_ANTHROPIC_API_KEY`, so one Console spend limit covers both flows. Splitting them later is a one-line
   change to the registry.

## Consequences

- The quality tier reaches the reviews that justify it: about 3% of PRs and about 8 specs a month. Independence is
  still guaranteed by the first signature. The budget arithmetic (about $50 of $60-65) is in
  `specs/AI-045-nan-catalog-alignment/verification.md`. #2215 and #2216 measure it against real volume.
- **The threshold is the budget lever, not Sonnet's prompt cap.** The first live Sonnet review cost $0.37, not the
  estimated $0.13: a PR on the risk route is large, so its prompt sits near the 200K cap. The threshold was planned
  at 900 lines, which 9.1% of the last 395 merged PRs reach, for about $106 a month. The owner set it to 1,500 lines,
  which 3.0% reach, for about $50. Capping Sonnet's prompt instead would truncate exactly the large diffs the route
  exists for. A smaller PR whose risk is not in its size takes the `deep-review` label.
- **A deliberate single point of failure.** A high-risk spec's second signature depends on the Anthropic key. When
  that key is down, the owner waits, or waives or lowers the risk in `proposal.md` with a stated reason.
- A share in either flow is a number in one file. Changing it is a reviewed diff, and a test checks that every
  point of the total weight draws the member that owns it.
- The weights are a first estimate. They change with data from #2215 and #2216, not by feel, as ADR-037 requires
  for its deadline.
