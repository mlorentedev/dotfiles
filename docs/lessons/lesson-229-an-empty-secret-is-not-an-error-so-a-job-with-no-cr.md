---
id: lesson-229-an-empty-secret-is-not-an-error-so-a-job-with-no-cr
type: lesson
status: active
created: "2026-08-24"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 229: An empty secret is not an error, so a job with no credential fails as if the work failed

**Context**: Every Dependabot PR opened since 2026-08-07 carried red checks — `add-to-project`, `review`, and `review-attestation` downstream of them. Measured on #1219 and #1220 (HARNESS-080, #1221). GitHub serves Dependabot-triggered `pull_request` runs from a **separate** secrets store; this repository's was empty (`gh api repos/OWNER/REPO/dependabot/secrets` → `total_count: 0`), while the Actions store held all three secrets.

**Problem**: `secrets.BITACORA_PAT` and `secrets.NAN_API_KEY` do not raise when the store lacks them — they expand to the **empty string**. So each job proceeded to do its work with no credential, and each failed at the point where the credential was used rather than at the point where it was missing. The resulting red says *"could not add this item to the board"* and *"PR-Agent published no review"* — both true, both describing a symptom three layers from the cause, and both indistinguishable from the genuine failures those messages were written for. Neither job was ever going to succeed, so the red was permanent, and a check that is always red is one everybody learns to scroll past. It went unread for seventeen days.

Two adjacent traps sharpened it. `add-to-project.yml` already skipped **fork** PRs for precisely this reason (`head.repo.fork == false`), and that guard could never fire here: a Dependabot PR is a branch in the repo, not a fork. And re-running the job does not help — secrets access follows the actor of the *original* event, so a human pressing "re-run" gets the same empty store (verified: run 32698096060, `actor=dependabot[bot]`, `triggering_actor=mlorentedev`, failed identically).

**Solution**: Skip the jobs that cannot work, keyed on `github.actor != 'dependabot[bot]'` — the **same field GitHub reads to choose the secrets store**, so the skip fires exactly when the credential is absent and never when it is present. A human pushing to a Dependabot branch is a different actor with the secrets available, and gets the full treatment. The guard is a class assertion, not two patches: any workflow that triggers on `pull_request` and reads a secret other than `GITHUB_TOKEN` must carry the exclusion (`tests/dependabot-ci.bats`), so the next workflow to reach for a credential cannot rediscover this the same way.

What was deliberately **not** done: mirroring the key into the Dependabot store, and exempting the PR from `review-attestation`. The first spends scarce inference on a change whose verdict a human makes anyway; the second turns "nobody reviewed this" green, which is the failure GUARD-002 exists to prevent. The attestation stays `pending` until a human reviews — which the classifier already supports and which `dependabot.yml` already declares as policy.

**Rule**: When a job's credential can be absent **by construction** rather than by accident, gate the job on the same signal the platform uses to decide, and let it skip rather than fail. A red that can never go green is not a finding; it is a broken instrument, and it costs the credibility of every check beside it. Corollary, paid for twice here: an existing guard against "no credentials in this context" (the fork test) is evidence the context has *more than one* entrance — check whether it covers all of them before assuming it covers yours.
