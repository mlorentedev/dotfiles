---
id: lesson-134-secrets-sync-ci-refreshed-updated-at-on-a-dead-pat
type: lesson
status: active
created: "2026-06-26"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 134: `secrets sync ci` refreshed `updated_at` on a dead PAT — a successful write is not a live credential

**Context**: After rotating-by-redeploy, `dotf secrets sync ci` uploaded `BITACORA_PAT` to the repo's Actions secrets and reported success. The board automation (`add-to-project`/`bitacora-status`) then failed every run with HTTP 401 — the uploaded token was expired at source. `sync` had verified the *write* (`gh secret set` succeeded, `updated_at` refreshed) but never that the *value still authenticates*.

**Problem**: A secret-sync tool's success criterion is "the write landed", not "the payload is live" — those are different claims. Uploading a 401 token looks identical to uploading a good one. Worse, the latent monitor that should have caught it (`pat-expiry.yml`) had been dead-on-arrival: the job has no `actions/checkout`, so its very first `gh` call died with `fatal: not a git repository` before probing anything (fixed by setting `GH_REPO`). So nothing — not the sync, not the monitor — actually checked liveness.

**Solution**: Three tiers. (0) Make `pat-expiry.yml` **fail the job** (red `::error` + `exit 1`) on an invalid/expired token, not just file an ignorable issue; fix the `GH_REPO` checkout bug so it runs at all. (1) Opt-in pre-upload liveness in `sync ci`: a registry entry marked `validate: github-token` is probed with `gh api user` (authenticating *as* the token under test) **before any upload**; a dead token aborts the whole sync. (2) Structural — migrate the board automation to a GitHub App installation token so the long-lived PAT stops existing.

**Rule**: Validating a secret's *liveness* is a separate concern from writing it, and from monitoring its expiry — do all three deliberately. Liveness validation does **not** generalize across providers (each is a bespoke probe, or none), so make it opt-in per credential, scoped to what you can cheaply probe (GitHub tokens via `gh api user`), and fail loud *before* the upload — never push a credential that does not authenticate. And the durable fix for "this PAT keeps expiring" is to delete the PAT (OIDC / GitHub App short-lived tokens), not to monitor it better.
