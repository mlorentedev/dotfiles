---
id: lesson-084-deploying-workflow-files-to-n-repos-via-the-conten
type: lesson
status: active
created: "2026-06-10"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 084: Deploying workflow files to N repos via the contents API: three gotchas the happy path hides

**Context:** OPS-002's `scripts/bitacora-rollout.sh` converges every non-archived, non-fork repo to the bitácora baseline — it pushes the two canonical workflows (`add-to-project.yml`, `bitacora-status.yml`) into each repo via the GitHub contents API and backfills open issues/PRs onto the board. It worked end-to-end against dotfiles/knowledge, then broke in three distinct ways the moment it met the rest of the fleet.
**Problem:** (1) **Protected branches reject a direct `PUT /contents`** with HTTP 409 — repos with branch protection (kubelab, pollex, pdf-modifier-mcp, yt-metrics-cli) cannot take a straight commit to the default branch. (2) **The deployed `add-to-project.yml` ran on fork PRs**, where `pull_request` from a fork has no access to repo secrets, so the job failed secretless on every fork-originated PR (Codex flagged it as P2 on kubelab#237). (3) **Reusing a fixed branch name (`ci/bitacora-workflows`) across runs** meant that after a run's PR was squash-merged, the next run pushed onto the now-diverged stale branch and opened a **conflicting** PR (Manu caught this on pollex/yt/pdf).
**Solution:** All three fixed in the canonical script + propagated to the deployed copies: (1) detect the 409 and **fall back to branch + PR** (with an auto-merge attempt) instead of a direct push (#317); (2) add a **job-level fork guard** (`if: github.event.pull_request.head.repo.fork == false`) so the secret-dependent job is skipped on fork PRs (#320); (3) **force-reset the working branch to base HEAD** at the start of each run so a reused branch never carries stale diff (#322). The convergence mechanism for any later workflow-template change (or PAT rotation) is simply re-running the script — it is idempotent, second run = `0 changes`.
**Tags:** `#github-actions` `#contents-api` `#protected-branches` `#fork-pr-secrets` `#idempotence` `#multi-repo` `#iac` `#bitacora`
