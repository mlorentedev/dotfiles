---
tags: [spec, verification, decision]
created: "2026-05-13"
decided: "2026-06-03"
---

# Verification - AI-020-gemini-empirical-validation

## Decision (matrix resolved 2026-06-03)

**Chosen branch: #1 — Antigravity (`agy`) OAuth preserves the Google AI Plus
subscription path. No tier upgrade.**

Inputs that resolved the matrix:

- **User tier: Google AI Plus.** AI Plus does not support Plan Linking with AI
  Studio API keys — but `agy` does not use API keys. It authenticates via
  browser OAuth against `cloudcode-pa.googleapis.com` (`loadCodeAssist`), the
  same Code Assist backend that recognized consumer subscriptions under
  gemini-cli. So the subscription path is preserved without an API key.
- **Gemini reliance: nice-to-have, low.** This fixes the user-owned cost
  decision (proposal R2): if AI Plus quota is *not* honored post-cutover, the
  resolution is to fall back to the free Antigravity allotment or drop Gemini
  (Claude Code + OpenCode remain) — **never a paid tier upgrade**.

Two findings that reshaped scope versus the proposal:

1. **The sunset is broader than the proposal assumed.** Google's canonical post
   states gemini-cli stops serving **AI Pro and Ultra, plus free** individual
   tiers on 2026-06-18 — not just free/Google One. Migration is mandatory for
   any consumer tier, which only strengthens branch #1.
2. **The migration infrastructure already shipped (SDD-007).** `setup-{linux,
   windows}` install `agy` (not gemini-cli), run a one-time `gemini-cli → agy`
   migration, and consolidate MCP to `~/.gemini/config/mcp_config.json`;
   healthcheck §13 already guards it. So the downstream follow-up shrinks from
   "port the install" to "verify the OAuth/quota path" → scaffolded as
   **AI-023** (the proposal's "AI-022" id was already taken by #161/#211).

## Evidence

- [x] AC1 Phase-1 research summary (install / config / auth / models / MCP) →
      recorded in this file + PR body; sources: Google Developers Blog
      (goo.gle/gemini-cli-migration), The Register, opencode-antigravity-auth.
- [x] AC2 Decision matrix completed, explicit branch chosen → **branch #1**
      (this file).
- [x] AC3 User tier identified → **Google AI Plus** (low Gemini reliance).
- [x] AC4 Follow-up spec scaffolded → `specs/AI-023-antigravity-oauth-verification/`
      (id corrected from AI-022, which is consumed by #161/#211).
- [ ] AC5 Vault `11-tasks.md` AI-020 entry ticked with decision link → vault
      master commit (this session).
- [ ] AC6 Runbook `40-runbooks/guide-antigravity-cli-migration.md` written →
      vault master commit (this session).

## Test status

- No code change in this ticket (decision + docs + scaffold). No test impact.
- Manual verification (user-run, optional de-risk): `agy -p "reply pong"` under
  AI Plus OAuth, `$ANTIGRAVITY_ENDPOINT == https://cloudcode-pa.googleapis.com`,
  no API-key prompt. Result folded into the runbook when available.

## Decisions made during implementation

- Rejected the proposal's "evaluate AI Pro upgrade" path: low Gemini reliance
  makes any paid upgrade non-justified; drop-or-free-allotment is the fallback.
- Renamed the follow-up AI-022 → AI-023 after verifying AI-022 is already in use
  (verify-before-act).

## Promotion candidates

- [x] Lesson for `90-lessons.md`? yes — "verify a spec's named follow-up id is
      free before scaffolding (AI-022 was consumed); decision specs can resolve
      on documentary evidence when the infra already shipped".
- [ ] ADR-worthy decision? no — operational migration, captured in the runbook.
- [ ] New pattern candidate? no — single-project migration.

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived` (after AC5/AC6 land)
- [ ] Folder moved: `specs/AI-020-gemini-empirical-validation/` → `specs/archive/`
- [ ] Backlog entry in vault `11-tasks.md` ticked with PR link
- [ ] Promotions above executed (lesson)
