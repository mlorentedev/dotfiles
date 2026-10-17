---
id: "lesson-472-polling-gh-pr-checks-from-parallel-sessions-spends-the-graphql-budget"
type: lesson
status: active
title: "Polling gh pr checks from parallel sessions spends the GraphQL budget"
created: "2026-10-17"
---

# Polling gh pr checks from parallel sessions spends the GraphQL budget

**Context:** During an unattended run on 2026-10-17, several sessions waited on CI at once, each looping `gh pr checks` or `gh pr view --json` every 30 seconds. At about 08:30 `gh api rate_limit` showed `graphql.remaining = 0` with a reset a few minutes out, and `gh pr create` and `gh pr ready` failed with "API rate limit already exceeded".

**Problem:** `gh pr checks`, `gh pr view`, `gh pr create` and `gh pr ready` are GraphQL. The 5000-point budget is per user, shared by every session on the machine, so N sessions polling at 30 seconds burn it N times as fast, and the commands that block progress (opening and readying a PR) fail first.

**Solution:** Poll through REST, which has its own budget: `gh api repos/<o>/<r>/commits/<sha>/check-runs` and `.../commits/<sha>/statuses` (the `review-attestation` verdict is a status, not a check-run), `.../pulls/<n>` for `mergeable_state` (GitHub computes it lazily, so `null` or `unknown` right after a push means retry, not an answer), `.../issues/<n>/comments` for reviewer output. Open a PR with `gh api -X POST repos/<o>/<r>/pulls`. Keep GraphQL for the two things REST cannot do, `gh pr ready` and `dotf pr land`, and spend them once. Check `gh api rate_limit --jq .resources.graphql` before a long wait.

**Why:** A shared quota is a shared resource: a polling loop is cheap for one session and a denial of service for six. Read state with the endpoint that has the budget left, and reserve the scarce one for the actions only it can do.
