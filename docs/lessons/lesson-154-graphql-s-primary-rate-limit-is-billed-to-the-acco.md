---
id: lesson-154-graphql-s-primary-rate-limit-is-billed-to-the-acco
type: lesson
status: active
created: "2026-08-06"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 154: GraphQL's primary rate limit is billed to the account, not the token

**Context**: `dotfiles#530` already documented an agent-side fallback (runbook §8a) for when the bitácora board's GraphQL pool runs dry: fall back to REST for issue data, degrade to waiting for the reset for board fields. `add-to-project.yml` and `bitacora-status.yml` kept failing anyway whenever an interactive Claude Code session ran a few full `gh project item-list` sweeps — three separate incidents across two days.

**Problem**: The obvious fix — give the interactive session and the CI automation separate tokens, so one exhausting its pool can't touch the other — does not work, and OPS-007 (per-purpose PAT convention) would not have prevented this. `BITACORA_PAT` and a personal OAuth token are different token strings, but both authenticate as the same GitHub account, and GitHub's primary GraphQL limit (5,000 points/hour) is billed to the *account*, not the token: the error is literally `API rate limit exceeded for user ID <id>`, never "for this token". Confirmed live, not just inferred: filing `dotfiles#774` from an interactive session that had just drained the pool, `add-to-project.yml` fired on the `issues: opened` webhook 3 seconds later and failed with the identical error — a different credential, same account, same exhausted bucket.

**Solution**: The one mechanism that would truly isolate the pools — a GitHub App installation token, which gets its own budget — is a dead end here per ADR-031: Apps can't write to a user-owned Projects v2 board, only an org-owned one, and moving the board to an org is out of scope. With isolation closed off, the fix is tolerance instead: soft-fail the board mutations on a rate-limit-specific 403 rather than hard-failing the job, and run the already-idempotent `scripts/bitacora-rollout.sh backfill` on a schedule so a soft-failed event gets reconciled later instead of silently lost. Filed as `dotfiles#774` (OPS-022).

**Rule**: Before reaching for "split the credential" as a fix for a shared rate limit, check who the tokens authenticate *as*, not just what string they are. Two PATs owned by the same account still share that account's primary limit — separating tokens only isolates blast-radius (OPS-007's concern), never quota. When a shared-account collision can't be engineered away (no App-token path available for the resource in question), degrade gracefully — soft-fail plus scheduled reconciliation — rather than treating every transient exhaustion as a hard failure.
