---
id: "TOOL-023-pr-agent-incremental-push-review"
type: spec
status: implementing # draft | implementing | verifying | archived
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

- [x] AC1: the gate returns `run=false` below 3 new non-merge commits and `run=true` at or past it. It finds the previous review as PR-Agent does, ignores merge commits, skips a bot push, runs when there is no previous review, and runs on unreadable input. Tested offline.
- [x] AC2: `pr-agent.yml` runs the gate only on `synchronize`. PR-Agent and the guard are skipped when it returns `run=false`, and `push_commands` follows the gate's `mode` output — `["/review -i"]` when it is `incremental`, `["/review"]` otherwise (amended by the #1757 triage follow-up below).
- [x] AC3: the registry declares the incremental heading, and the guard accepts any declared marker.
- [x] AC4: the #1053 comment is replaced, ADR-040 exists, and `pr-stewardship` names the draft practice.

## Follow-up (triage on #1757, comment 5851151835)

Three findings accepted against `scripts/pr-agent-push-gate.sh` after #1757 merged, all fixed in the same follow-up PR (#1756):

1. **CWE-345, insufficient verification of data authenticity.** The baseline was any comment carrying a review marker, regardless of author — on a public repo, anyone can comment the marker text and have it stand in for a review that never happened. Fixed: only a comment authored by `github-actions[bot]` sets the baseline. PR-Agent's own `get_previous_review` (v0.45.0) has the same gap and is not author-checked either — filed upstream, not fixed here — so if a forged comment is newer than the real baseline, the gate now asks for a FULL review (`mode=full`) instead of incremental, since PR-Agent would pick the forged comment as its own baseline too and `/review` never picks one at all.
2. **Rebase blind spot.** New commits were counted by `.commit.author.date > baseline`; a rebase preserves author dates, and this repo rebases with `--onto` routinely, so a review followed by rebased-but-genuinely-new commits could count as zero new commits. Fixed: when the baseline review's persistent state block (`<!-- pr-agent-review-state:v1 {...} -->`) carries `last_run.head_sha`, commits are counted by POSITION after that sha, immune to date reordering. If the sha is no longer in the PR (rebase or force-push rewrote it), the range cannot be trusted: `mode=full`.
   - **Disclosed limitation.** Only a FULL review's comment carries `last_run.head_sha` — PR-Agent v0.45.0 disables its finding-state machinery entirely for an incremental run (`_review_finding_state_enabled` returns `False` when `self.incremental.is_incremental`), confirmed by reading the pinned source; no real incremental comment exists yet on this repo to double-check against (`in:comments` search, 2026-09-27). So once a PR's most recent bot review is itself incremental, this gate is back on the date-based fallback — and its original rebase blind spot — until the next FULL review. A rebased incremental-on-incremental PR is not fixed by this change.
3. **`push_commands` chooses per push.** The workflow CAN select between `/review` and `/review -i` per run: `github_action_config.push_commands` is a step `env:` value, and GitHub Actions expressions may read an earlier step's output there, the same as `STARTED`/`HEAD_SHA` already do lower in this file. So the gate now emits `mode=full|incremental` and the workflow reads it, rather than only implementing the author filter.

**A fourth, narrower disclosure surfaced by review of this fix itself, not by the original triage:** counting by SHA position (f6) can disagree with PR-Agent's own date-based `get_commit_range`. If every commit after the reviewed sha happens to be authored at or before the review's `created_at` (a cherry-pick with a preserved old author date — not a plain rebase, which rewrites SHAs and lands in `mode=full` already), PR-Agent computes an empty incremental range and publishes nothing. That failure is loud, not silent: the pre-existing "no review published" guard (#1107) catches it and names `/review` as the remedy, which is the safe direction (ADR-037/ADR-040's own standard: "queues or escalates, never degrades silently"). ADR-040 D2 is amended with this caveat rather than left to read as an absolute guarantee it no longer is.

## References

- Bitácora: #1756. Evidence: #1732's pr-agent run on `4b7d126`, and pr-agent runs 2026-09-19 to 09-26.
- PR-Agent v0.45.0 (`f3b385e`): `pr_agent/tools/pr_reviewer.py` (`_can_run_incremental_review`, `_review_finding_state_enabled`, `_prepare_review_finding_state`), `pr_agent/git_providers/github_provider.py` (`get_commit_range`, `get_previous_review`), `pr_agent/algo/utils.py` (headers and identities).
- A real state block: `gh api repos/mlorentedev/dotfiles/issues/comments/5851012976 --jq .body`.
- ADR-037 (review runners bounded in time); #1053, #1107, #1618, #1756, #1757.
