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
   - "The previous review" is the comment PR-Agent itself would pick (v0.46.0, `get_previous_review`): the last one carrying `<!-- pr-agent:review:full -->` or `<!-- pr-agent:review:incremental -->` within its first 5 lines, in either stored form (the HTML comment, or the link reference `[pr-agent:review:full]: https://github.com/The-PR-Agent/pr-agent`), or starting with either Guide heading.
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

- **`/review -i` is implemented but undocumented upstream.** Its docs have been commented out since v0.24. The design was read from the v0.45.0 source and re-checked against the pinned v0.46.0 (#1893).
- **The gate copies PR-Agent's selection rules, and no test reads upstream.** The tests pin the gate's own copies of the headings, identities and date basis, not PR-Agent's. So a version bump that changes upstream's rules does not fail here by itself. What does fail is a mismatch between the version the gate cites and the version the workflow pins (`every PR-Agent version the gate cites is the one the workflow pins`), so a bump forces someone to re-read upstream and update the citation. The earlier claim that the tests pin upstream was false: the v0.45.0 to v0.46.0 bump changed both rules and the suite stayed green (#1893).
- **After a rebase, or when no earlier review exists, PR-Agent falls back to a full review.** That is acceptable: a rebase is a new diff.
- **The job gains a checkout, sparse, of the gate script alone.** It already runs the PR-Agent action with the PR's own workflow file, and fork PRs stay excluded, so what a PR can change is unchanged.

## Acceptance criteria

- [ ] AC1: the gate returns `run=false` below 3 new non-merge commits and `run=true` at or past it. It finds the previous review as PR-Agent does, ignores merge commits, skips a bot push, runs when there is no previous review, and runs on unreadable input. Tested offline.
- [ ] AC2: `pr-agent.yml` runs the gate only on `synchronize`. PR-Agent and the guard are skipped when it returns `run=false`, and `push_commands` follows the gate's `mode` output — `["/review -i"]` when it is `incremental`, `["/review"]` otherwise (amended by the #1757 triage follow-up below).
- [ ] AC3: the registry declares the incremental heading, and the guard accepts any declared marker.
- [ ] AC4: the #1053 comment is replaced, ADR-040 exists, and `pr-stewardship` names the draft practice.

## Follow-up (triage on #1757, comment 5851151835)

Three findings accepted against `scripts/pr-agent-push-gate.sh` after #1757 merged, all fixed in the same follow-up PR (#1756):

1. **CWE-345, insufficient verification of data authenticity.** The baseline was any comment carrying a review marker, regardless of author — on a public repo, anyone can comment the marker text and have it stand in for a review that never happened. Fixed: only a comment authored by `github-actions[bot]` sets the baseline. PR-Agent's own `get_previous_review` (v0.45.0, and still in v0.46.0) has the same gap and is not author-checked either — filed upstream, not fixed here — so if a forged comment is newer than the real baseline, the gate now asks for a FULL review (`mode=full`) instead of incremental, since PR-Agent would pick the forged comment as its own baseline too and `/review` never picks one at all.
2. **Rebase blind spot.** New commits were counted by `.commit.author.date > baseline`; a rebase preserves author dates, and this repo rebases with `--onto` routinely, so a review followed by rebased-but-genuinely-new commits could count as zero new commits. Fixed: when the baseline review's persistent state block (`<!-- pr-agent-review-state:v1 {...} -->`) carries `last_run.head_sha`, commits are counted by POSITION after that sha, immune to date reordering. If the sha is no longer in the PR (rebase or force-push rewrote it), the range cannot be trusted: `mode=full`.
   - **Disclosed limitation.** Only a FULL review's comment carries `last_run.head_sha` — PR-Agent v0.45.0 disables its finding-state machinery entirely for an incremental run (`_review_finding_state_enabled` returns `False` when `self.incremental.is_incremental`), confirmed by reading the pinned source; no real incremental comment exists yet on this repo to double-check against (`in:comments` search, 2026-09-27). So once a PR's most recent bot review is itself incremental, this gate is back on the date-based fallback — and its original rebase blind spot — until the next FULL review. A rebased incremental-on-incremental PR is not fixed by this change.
3. **`push_commands` chooses per push.** The workflow CAN select between `/review` and `/review -i` per run: `github_action_config.push_commands` is a step `env:` value, and GitHub Actions expressions may read an earlier step's output there, the same as `STARTED`/`HEAD_SHA` already do lower in this file. So the gate now emits `mode=full|incremental` and the workflow reads it, rather than only implementing the author filter.

4. **Version drift, found by the archive review (#1893).** The gate mirrored v0.45.0 while the workflow pinned v0.46.0, which changed two rules. `get_commit_range` now orders commits by committer date (`_commit_timeline_date`), and a review identity may be stored as a link reference as well as an HTML comment. Fixed: the date fallback compares the committer date, then the author date, so a rebase (which rewrites committer dates) counts its commits as new and the push is reviewed. The identity predicate accepts both stored forms, so a forged link-reference marker is caught like the HTML one. A test fails when the gate's cited version differs from the pinned tag. The disclosed limitation in item 2 narrows accordingly: after an incremental review, a rebase is no longer counted as zero new commits.

**A fifth, narrower disclosure surfaced by review of this fix itself, not by the original triage:** counting by SHA position (f6) can disagree with PR-Agent's own date-based `get_commit_range`. If every commit after the reviewed sha happens to be authored at or before the review's `created_at` (a cherry-pick with a preserved old author date — not a plain rebase, which rewrites SHAs and lands in `mode=full` already), PR-Agent computes an empty incremental range and publishes nothing. That failure is loud, not silent: the pre-existing "no review published" guard (#1107) catches it and names `/review` as the remedy, which is the safe direction (ADR-037/ADR-040's own standard: "queues or escalates, never degrades silently"). ADR-040 D2 is amended with this caveat rather than left to read as an absolute guarantee it no longer is.

## References

- Bitácora: #1756. Evidence: #1732's pr-agent run on `4b7d126`, and pr-agent runs 2026-09-19 to 09-26.
- PR-Agent v0.46.0 (`1d01f24`, the pinned version): `pr_agent/git_providers/github_provider.py` (`_commit_timeline_date`, `get_commit_range`, `get_previous_review`), `pr_agent/algo/comment_identity.py` (`hidden_marker_forms`, `comment_matches_identity`).
- PR-Agent v0.45.0 (`f3b385e`), read for the original design: `pr_agent/tools/pr_reviewer.py` (`_can_run_incremental_review`, `_review_finding_state_enabled`, `_prepare_review_finding_state`), `pr_agent/git_providers/github_provider.py` (`get_commit_range`, `get_previous_review`), `pr_agent/algo/utils.py` (headers and identities).
- A real state block: `gh api repos/mlorentedev/dotfiles/issues/comments/5851012976 --jq .body`.
- ADR-037 (review runners bounded in time); #1053, #1107, #1618, #1756, #1757.
