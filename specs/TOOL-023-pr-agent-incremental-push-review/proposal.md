---
id: "TOOL-023-pr-agent-incremental-push-review"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-09-25"
issue: "mlorentedev/dotfiles#1756"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal, ci, review, pr-agent]
template_version: "1.0"
---

# TOOL-023: a push is reviewed incrementally, and only past three new commits

## Why

pr-agent runs a full `/review` on every push (#1053).

Measured over 2026-09-19 to 2026-09-26:

- 39 reviews across 18 PRs, a mean of 2.2 per PR and a maximum of 7 (#1732: the opening plus six pushes).
- Each review is one request to a NaN pool of 5 concurrent requests, shared with pi, qq, hive and the spec reviews (#1107).
- #1732's seventh review failed. The primary model timed out after 361 s, and the fallback model hit the concurrency limit.
- Every re-review edits the persistent Guide, which reopens `dotf pr triage-queue` even when nothing new was found.

The rest of the field reviews once and again only on request, or past a bound:

- Copilot reviews once unless "Review new pushes" is on.
- Codex reviews on open or on `@codex review`.
- Claude Code Review defaults to "once after PR creation".
- CodeRabbit reviews incrementally and pauses after 5 reviewed commits.

## What

1. **A push runs a gate first,** `scripts/pr-agent-push-gate.sh`. It reads the PR's comments and commits and decides whether PR-Agent runs.
   - It runs once at least **3 non-merge commits** are newer than the previous review.
   - "The previous review" is the comment PR-Agent itself would pick (v0.45.0, `get_previous_review`): the last one carrying `<!-- pr-agent:review:full -->` or `<!-- pr-agent:review:incremental -->` within its first 5 lines, or starting with either Guide heading.
   - A push from a bot is skipped, as PR-Agent skips it.
   - With no previous review, it runs, and PR-Agent falls back to a full review.
   - With input it cannot read, it runs. It never skips a review it cannot justify.
2. **The push command is `/review -i`,** which reviews only the commits since the previous review. PR-Agent's own incremental thresholds stay at their defaults, so the threshold lives in one tested place.
3. **Below the threshold, neither PR-Agent nor the published-review guard runs,** so a skipped push leaves no red check.
   - The guard accepts every review marker the registry declares.
   - The registry declares `## Incremental PR Reviewer Guide` as well, so `dotf pr triage-queue` and `review-attestation` recognise an incremental review.
4. **Opening, reopening and marking ready still run a full review, and `/review` still runs on demand.**
5. **The #1053 rationale in `pr-agent.yml` is replaced,** ADR-040 records the decision, and `pr-stewardship` tells agents to iterate on a draft.

## Out of scope

- **CodeRabbit.** It already reviews incrementally, pauses after 5 reviewed commits, and is rate-limited on this repo anyway.
- **A pause after N reviews per PR.** The commit threshold bounds the same thing without state.
- **Other repositories' pr-agent ports** (kubelab, yt-metrics-cli, svqtriana). They follow when this one has run for a while.

## Risks / open questions

- **`/review -i` is implemented but undocumented upstream.** Its docs have been commented out since v0.24. The pinned v0.45.0 source was read for this design, and the tests pin its headings and identities, so a version bump that changes them fails here.
- **After a rebase, or when no earlier review exists, PR-Agent falls back to a full review.** That is acceptable: a rebase is a new diff.
- **The job gains a checkout, sparse, of the gate script alone.** It already runs the PR-Agent action with the PR's own workflow file, and fork PRs stay excluded, so what a PR can change is unchanged.

## Acceptance criteria

- [ ] AC1: the gate returns `run=false` below 3 new non-merge commits and `run=true` at or past it. It finds the previous review as PR-Agent does, ignores merge commits, skips a bot push, runs when there is no previous review, and runs on unreadable input. Tested offline.
- [ ] AC2: `pr-agent.yml` runs the gate only on `synchronize`. PR-Agent and the guard are skipped when it returns `run=false`, and `push_commands` is exactly `["/review -i"]`.
- [ ] AC3: the registry declares the incremental heading, and the guard accepts any declared marker.
- [ ] AC4: the #1053 comment is replaced, ADR-040 exists, and `pr-stewardship` names the draft practice.

## References

- Bitácora: #1756. Evidence: #1732's pr-agent run on `4b7d126`, and pr-agent runs 2026-09-19 to 09-26.
- PR-Agent v0.45.0 (`f3b385e`): `pr_agent/tools/pr_reviewer.py` (`_can_run_incremental_review`), `pr_agent/git_providers/github_provider.py` (`get_commit_range`, `get_previous_review`), `pr_agent/algo/utils.py` (headers and identities).
- ADR-037 (review runners bounded in time); #1053, #1107, #1618.
