---
id: lesson-198-a-pr-s-head-sha-not-matching-your-latest-push-can-
type: lesson
status: active
created: "2026-08-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 198: A PR's `head.sha` not matching your latest push can mean the PR is already merged, not that the API is lagging

**Context**: HARNESS-070 (#843/#869/#828, PR #948). After pushing a commit addressing CodeRabbit's review findings, `gh api .../pulls/948 --jq '.head.sha'` kept returning the previous commit even though `git fetch` confirmed the remote branch ref had the new one. The first read was "GitHub API/webhook propagation lag" — plausible after a session that had already hit a real rate-limit earlier — and a background poll was armed to wait for the field to catch up.

**Problem**: it wasn't lag. `gh api .../pulls/948` also carried `"state": "closed"` and `"merged": true`, fields the lag theory never checked. The PR had been squash-merged (by the user, via the GitHub UI) roughly 21 hours *before* the review-fix commit was even authored. A merged PR's `head.sha` is frozen at merge time by definition — pushing more commits to its (now-orphaned) branch updates the branch ref, never the PR record, and triggers no CI, because there is no open PR for a workflow to run against. The poll loop's exit condition (`head.sha == <new commit>`) could never become true; it would have spun until manually killed regardless of how long anyone waited.

**Solution**: checked `state`/`merged`/`merged_at` on the same API object instead of only `head.sha`, which immediately explained the mismatch. Fix-forward: opened a new branch off the now-updated `main`, cherry-picked the orphaned fix commit (applied with zero conflicts, confirming no semantic drift from other PRs merged in between), reran the full verification suite on the cherry-picked result, and opened a follow-up PR referencing (not closing) the already-closed issues.

**Rule**: before diagnosing a mismatched `head.sha` (or any "why hasn't my push shown up" symptom) as replication lag, check the PR's `state`/`merged` fields on the very first query — a closed PR explains a frozen `head.sha` completely and rules out every lag-based theory in one call. Don't build a polling loop around a diff-based condition (`head.sha == X`) without first confirming the object it lives on is still open; a wait condition that assumes "eventually consistent" when the true shape is "permanently fixed" spins forever and burns a task slot for nothing.

**Tags**: `github`, `ci`, `verification`, `debugging`
