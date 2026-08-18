---
id: lesson-197-a-health-check-that-reads-local-state-proves-the-l
type: lesson
status: active
created: "2026-08-13"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 197: A health check that reads local state proves the liveness of nothing

**Context**: `dotf doctor` verified the two secret tiers of ADR-028 with opposite rigour. The age floor — the *backup* — was proven by behaviour: derive the recipient, encrypt a sentinel, decrypt it back, compare bytes. Bitwarden — the tier ADR-028 designates the **live SSOT** — was proven by `sys.has("bw")`, a binary on `PATH` (BUG-074, #944).

**Problem**: Bitwarden's refresh token expired server-side and doctor printed a green `bw (Bitwarden CLI — live secrets SSOT) found` for the 45 days that followed. The outage surfaced only when an operator ran `bw unlock` by hand and got `invalid_grant` (HTTP 400). Worse, the obvious "deeper" probes are no better: `bw status` is served from local state and keeps reporting a healthy-looking `locked` indefinitely after the server has revoked the grant, and `bw list` / `bw get` read the local cache, so both pass against a dead token. Every cheap observable was a local one, and every local one was a lie.

**Solution**: three tiers, keyed to what can be established without an operator present. (1) `bw status` catches the definite `unauthenticated` break — and names `bw login`, explicitly ruling out `bw unlock`, whose master-password prompt makes an expired token read as a forgotten password. (2) **Elapsed time since the last successful sync** — the only observable that actually moves while the token rots, and the only one available on a locked vault. Threshold 30d, chosen below the only expiry ever observed (45d) — an educated floor, not a derived one: the incident shows the token dead by 45d, not alive at 30d, and no upstream lifetime is documented. (3) `bw sync` as a real round-trip when a session exists. Severity is keyed to real exposure (count of `backend: bw` registry entries), so an unreachable vault is advisory while everything still resolves through age, and a FAIL from the first migrated secret.

**Rule**: a check on a remote dependency must either exercise the remote path or measure elapsed time since something did. Local status output is a cache of a past success, not evidence of a present one — and a cache that reports "locked" is indistinguishable from one reporting "locked, and also revoked three weeks ago". When the deep probe needs a credential the check cannot assume (an unlocked vault, an API key), the elapsed-time tier is not a consolation prize: it is the only tier that runs in the resting state, so it is the one that catches the silent expiry. Corollary for severity: scale it to what actually breaks today, because a red diagnostic for a harmless condition is how operators are trained to ignore red diagnostics.

**Tags**: `doctor`, `secrets`, `bitwarden`, `health-checks`, `verification`
