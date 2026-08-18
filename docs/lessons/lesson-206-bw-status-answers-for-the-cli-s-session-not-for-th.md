---
id: lesson-206-bw-status-answers-for-the-cli-s-session-not-for-th
type: lesson
status: active
created: "2026-08-15"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 206: `bw status` answers for the CLI's session, not for the daemon your code actually uses

**Context**: every `bw`-backed secret was failing at once — `dotf secrets run -- true` died on `dockerhub`, and `dotf secrets verify` showed a wall of FAILED with `bw serve returned no parseable envelope: invalid character 'I'`. Looking for a single cause behind a mass failure, I ran `bw status`, got `{"status":"locked"}`, and reported the blocker as a locked vault needing the user's master password.

**Problem**: the vault was not locked. ADR-028's runtime path does not go through the `bw` CLI's own session at all — `dotf secrets` resolves through the long-lived `bw serve` daemon, which holds a *separate* unlocked session. Both readings were true about different subjects, and the one I measured was not the one the failing code uses. A parallel session refuted it with the measurement I should have run: `dotf secrets run --only NAN_API_KEY -- true` exits 0 while unscoped resolution dies. The real cause was one broken item mapping killing an unscoped batch read (#985/#988), and "ask the user to unlock" would have fixed nothing while looking like progress — a plausible cause that explains the symptom is not the same as the cause.

**Solution**: measure the path the failing code takes. `dotf secrets run --only <ID> -- true` exercises exactly the daemon, the mapping and the item that production uses; `bw status` exercises a CLI session nothing in ADR-028's hot path reads. When a mass failure has an obvious single explanation, prefer the probe that is *specific to one working element* over the one that reports global state — a scoped success falsifies "everything is down" in one command.

**Rule**: before believing a diagnostic, name the subject it reports on and check it is the subject that is failing. Tools that front a daemon, a cache, a proxy or a pool almost always have two states — the client's and the server's — and the human-facing status command usually reports the client's, because that is the one it can see without asking. `git status` vs the remote, `docker ps` vs the daemon, `kubectl config` vs the cluster, `bw status` vs `bw serve`: same shape. The tell is a global explanation ("it's locked", "it's down") arriving before any single-element probe was tried.

**Tags**: `secrets`, `bitwarden`, `diagnosis`, `verification`
