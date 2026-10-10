---
id: "ADR-047-standing-merge-grant-opt-in-per-repository"
type: adr
status: accepted
owner: manu
date: "2026-10-09"
supersedes: []
extends: [adr-046-anthropic-reviewers-never-sign-alone]
issue: mlorentedev/dotfiles#2178
tags: [architecture, decision, git, merge, review, agents]
created: "2026-10-09"
---

# ADR-047: The standing merge grant is declared per repository, and the owner lands any change that widens it

## Context

On 2026-10-01 the owner granted a standing authority: an agent may merge a non-release PR it opened when CI is
green, the PR has no conflicts and at least one reviewer's output is triaged. The grant was restated on 2026-10-06
and used on 2026-10-07 and 2026-10-08.

The doctrine every agent receives said the opposite. `pattern-git-workflow.md` §9, compiled into
`harness/enforced/no-auto-merge.md` and from there into `AGENTS.md`, allowed an agent to merge "only when the user
has authorized merging that specific PR". The grant lived only in one agent memory file. Other projects carried
conflicting memory rules (`feedback-agent-never-merges`, `manu-always-merges`, and others). Which rule an agent
followed depended on which memory it loaded, not on anything written in the repository.

`dotf pr land` already checked the three conditions on one head commit. It did not check whether the repository
had accepted the grant at all, so the same command would merge in a repository whose owner never agreed to it.

## Decision

1. **The grant is opt-in per repository, declared in `.github/merge-grant.yml` on the default branch.** The file
   names the grant (`prs-the-agent-opened`), its conditions (`ci-green`, `no-conflicts`, `reviewer-triaged`), the
   exclusion (`release-please`), the merge form (`squash --match-head-commit`) and the disclosure the triage comment
   owes (who reviewed, or that no review ran). Without the file, "that specific PR" applies.
2. **`dotf pr land` reads the file from the base branch, never from the PR head,** and refuses without it. A PR
   cannot grant itself the merge by adding the file. A 404 means "not declared"; any other read error stops the
   land rather than guessing.
3. **The declaration is pinned by tests.** `cli/internal/prland/grant_test.go` decodes the file with unknown fields
   rejected and requires exactly the decided content, and checks that `Decide` refuses for each declared condition.
   Widening the file without changing the tests fails CI.
4. **Only dotfiles opts in.** kubelab deliberately does not: its ADR-046 D3 has a human merge anything that reaches
   production, so its `feedback-agent-never-merges` rule stays correct there, not stale.
5. **The owner lands any change that widens the agent's own merge or permission authority,** this ADR's PR
   included. The agent prepares it; the owner reviews, merges, and pushes the matching vault commit. A PR that
   changes the grant is never merged under the grant.
6. **The pattern is the SSOT.** `pattern-git-workflow.md` §1, §4 and §9 and the shipper definition carry the rule;
   `scripts/compile-harness.sh --refresh` regenerates the enforced block and the agent files from them. The memory
   copies point at the pattern instead of restating the rule.

## Consequences

- One written rule replaces several memory rules. An agent in a repository without the file asks for the specific
  PR, whatever its memory says.
- **`dotf pr land` stops merging in every repository without the file,** including the owner's own runs there. In
  those repositories the owner merges with `gh pr merge` as before `pr land` existed.
- Release PRs and PRs the owner said to hold stay the owner's under the grant; `Decide` already refuses the
  release-please prefix, and a hold is honoured by the agent reading the PR, not by the tool.
- Opting another repository in is a one-file diff the owner merges, reviewed like any other authority change.
- The no-auto-merge rule is unchanged: `allow_auto_merge` stays `false` and every merge is a deliberate act on a
  checked head commit. The grant removes the per-PR question, not the checks.
