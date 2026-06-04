---
id: "AI-024-antigravity-oauth-verification"
status: draft # draft | implementing | verifying | archived
created: "2026-06-04"
tags: [spec, proposal]
template_version: "1.0"
---

# AI-024-antigravity-oauth-verification

> Follow-up to AI-020 (Gemini CLI → Antigravity migration decision). AI-020
> chose **branch #1** (`agy` OAuth via the Code Assist backend preserves the
> Google AI Plus subscription path). This ticket verifies that decision holds
> empirically and guards it.

## Why

AI-020 resolved the migration *decision* on documentary evidence: `agy` is
already installed + configured cross-OS (SDD-007), and it authenticates via
OAuth against `cloudcode-pa.googleapis.com` (`loadCodeAssist`) — the same
subscription-recognizing backend gemini-cli used. But two things are unproven:
(a) that `agy`'s OAuth actually serves requests under a **Google AI Plus** tier
*after* the 2026-06-18 gemini-cli sunset, and (b) that this stays true on
Windows. The migration infrastructure shipped; what remains is a verification +
a regression guard so a silent post-cutover quota break is caught, not
discovered mid-task.

## What

- A documented, reproducible check that `agy -p "..."` returns a served
  response under the user's AI Plus OAuth session (no API key, endpoint is the
  Code Assist production URL).
- A healthcheck assertion (section 13, Antigravity CLI Health) that surfaces
  `agy` auth state / reachability so a deauthenticated or quota-broken `agy` is
  reported, not silent.

## Out of scope

- Re-doing the AI-020 decision matrix (settled: branch #1, no tier upgrade).
- Any paid Google tier upgrade — AI-020 fixed the cost decision: Gemini is
  low-reliance, so a broken quota means fall back to free allotment / drop
  Gemini, never upgrade.
- Replacing the `agy` install/config in setup scripts — already shipped
  (SDD-007). This ticket only verifies + guards it.

## Risks / open questions

- R1: Google may move consumer (AI Plus) tiers to a different post-cutover quota
  model than the Code Assist OAuth path currently honors. The healthcheck probe
  must not assume "auth file present" == "quota served".
- R2: A live quota probe costs a token round-trip; the healthcheck assertion
  must stay offline/cheap (check auth-state presence + endpoint), with the
  paid live probe left to the documented manual step.
- R3: Windows empirical re-check is batched into the dedicated Windows session
  (see [[feedback-batch-windows-work]]); this ticket's Linux part can land first.

## Acceptance criteria

- [ ] Manual verification recorded: `agy -p` serves a response under AI Plus
      OAuth, no API key, endpoint = `https://cloudcode-pa.googleapis.com`.
- [ ] healthcheck.sh section 13 reports `agy` auth-state presence + endpoint
      (offline, cheap) and FAILs loudly if auth state is missing.
- [ ] Cross-OS parity: matching probe in healthcheck.ps1 (Windows-empirical,
      may land in the batched Windows session).

## References

- Predecessor: `specs/archive/AI-020-gemini-empirical-validation/` (decision + matrix).
- Vault: `10_projects/dotfiles/11-tasks.md` → AI-020 / AI-024.
- Runbook: `docs/runbooks/guide-antigravity-cli-migration.md`.
- Upstream: <https://goo.gle/gemini-cli-migration>.
