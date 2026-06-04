---
id: "AI-020-gemini-empirical-validation"
type: spec
status: verifying # draft | implementing | verifying | archived
created: "2026-05-27"
tags: [spec, proposal]
template_version: "1.0"
---

# AI-020-gemini-empirical-validation

> **Naming**: file lives at `<repo>/specs/AI-020-gemini-empirical-validation/proposal.md`. `AI-020-gemini-empirical-validation` is `YYYY-MM-DD-<slug>` or `<TICKET-NN>`.

## Why

<!-- from 11-tasks.md: *(P1 → P0 URGENT 2026-05-21, cross-OS, deadline 2026-06-18)*: ⚠️ **UPSTREAM SUNSET WARNING (surfaced 2026-05-21 in `gemini` TUI on user's Windows):** "We are unifying our tools into a single, multi-agent platform called **Antigravity**, with **Antigravity CLI** now available. **Gemini CLI will stop serving requests for Google One and unpaid tiers starting June 18th.** Please migrate to Antigravity CLI before this date to avoid disruption. Reference: <https://goo.gle/gemini-cli-migration>". ⚠️ **Critical sub-finding 2026-05-21 (web search):** **Google AI Plus tier ($19.99/mo) does NOT support "Plan Linking"** with AI Studio API keys -- only Pro and Ultra do. This means: (a) if user stays on AI Plus and migrates to Antigravity, must verify Antigravity preserves the OAuth flow that gemini-cli uses (which DOES recognize AI Plus); (b) generating an AI Studio API key as AI Plus user gives PAYG billing, NOT subscription quota; (c) one path forward is to upgrade to AI Pro ($19.99/mo same price + Plan Linking) which would unlock API-key-vs-subscription parity AND likely cleaner Antigravity integration. **Decision matrix for the migration:** check (1) does Antigravity CLI support AI Plus OAuth → if yes, free migration; (2) does Antigravity require API key → AI Plus loses subscription billing, evaluate Pro upgrade; (3) does Antigravity have a different tier system entirely → fresh decision. This changes the AI-020 scope dramatically: **validating gemini-cli on a free/Google One tier is wasted effort if it sunsets in <4 weeks**. **Revised scope:** (a) determine user's current tier (paid Google AI Plus vs Google One vs free/no-tier) -- if free/Google One, gemini-cli stops working June 18; (b) if paid Google AI Plus, verify gemini-cli continues working post-deadline OR migrate to Antigravity CLI anyway for the unified platform benefits; (c) read Antigravity CLI docs at goo.gle/gemini-cli-migration to understand: install method (npm? curl? winget?), config schema, OAuth/API key delta vs gemini-cli, model lineup, MCP support; (d) replace `setup-{linux.sh,windows.ps1}` Gemini install + ai/gemini/ deploy with Antigravity-equivalents (OPEN AI-022 atomic PR after research). **Phase A original goal (Gemini CLI 0.42 validation) MOVED TO ANTI-SCOPE** -- not worth validating a sunset tool. **Phase B (Linux mirror)** also blocked on the migration decision. **Done when:** decision made on Antigravity adoption + atomic AI-022 PR opens with the migration cabled in setup scripts.: Empirical shake-down of Gemini CLI + Gemini config + skill-derived prompts deploy on Windows AND Linux. **Why:** Claude Code already validated (primary daily-driver); OpenCode Windows queued as AI-014b. Gemini is the third agent in the multi-agent runtime (ADR-009 / AI-019) and the only one without an explicit empirical-validation ticket. The user wants confidence to swap between Claude Code / Gemini / OpenCode without friction. **Phase A Windows (this session, hands-on with user agent):** (1) `gemini --version` reports version; (2) `~/.gemini/GEMINI.md` exists with `First, read AGENTS.md` pointer (verifies SDD-005-class drift hasn't happened); (3) `~/.gemini/prompts/*.md` has ≥15 skill-derived prompts (matches `ai/skills/*/SKILL.md` count); (4) `g` PowerShell alias resolves to `gemini`; (5) verify auth state at `~/.gemini/.gemini_auth.json` or similar; (6) test invocation `g "ping"` returns a response (one-shot, low-cost). **Phase B Linux (deferred, user hands-on when on Linux machine):** same 6 checks via bash; verify the same skill-prompts file set matches (cross-OS parity); compare default model + response style. **Phase C cross-agent integration (optional, post-validation):** test that the same task gets reasonable answers from Claude Code, Gemini, OpenCode — e.g., "describe this repo's testing strategy in 3 bullets". **Done when:** both Phase A + Phase B documented; any deltas folded into a Windows section of `40-runbooks/guide-gemini-cli-setup.md` (NEW vault runbook if it doesn't exist) or as `## Windows-specific` deltas. **Anti-scope:** do NOT modify setup-windows.ps1 / setup-linux.sh / ai/gemini/ in this ticket — it is empirical only. Any code fix discovered opens an atomic PR. **Companion:** AI-014b (OpenCode validation) runs in parallel scope; both feed into the multi-agent confidence story. -->

**P0 URGENT — hard deadline 2026-06-18.** `gemini-cli` upstream-sunsets for free + Google One tiers on that date in favor of **Antigravity CLI**. Compounding: Google AI Plus tier ($19.99/mo) does NOT support Plan Linking with AI Studio API keys (per official moderator confirmation 2026-05-21) — only Pro/Ultra do. Without a decision before the deadline, the user's Gemini path either breaks (free/Google One) or silently switches to PAYG billing (AI Plus + API key) instead of subscription quota. The original AI-020 scope (validate gemini-cli 0.42) is now wasted effort on a sunset tool.

## What

This ticket is a **decision matrix exercise**, not an empirical-validation pass. Output is a documented decision recorded in PR body + vault, plus a follow-up atomic PR (`AI-022-antigravity-migration`) that cables the decision into `setup-{linux.sh,windows.ps1}`.

Three decision branches:

1. **Antigravity CLI supports AI Plus OAuth** → migrate setup scripts from `gemini-cli` install to Antigravity install; AI Plus continues as subscription billing; no tier upgrade required.
2. **Antigravity CLI requires API key (not OAuth)** → AI Plus loses subscription billing; evaluate upgrading to AI Pro ($19.99/mo same price + Plan Linking unlock).
3. **Antigravity has a different tier system entirely** → fresh empirical pass + tier choice from scratch.

## Out of scope

- **Empirical validation of gemini-cli 0.42 features** — explicitly moved to anti-scope per the vault note. Validating a sunset tool is waste.
- **Modifying `setup-{linux.sh,windows.ps1}`** — those changes ship in the downstream AI-022 PR. AI-020's deliverable is the decision + research doc, not code.
- **Actually upgrading the user's Google subscription** — out of scope for an engineering ticket. Surface the cost tradeoff in the decision doc; user decides.
- **Cross-agent integration test (Phase C of the original scope)** — re-prioritise once Antigravity replaces gemini-cli.

## Risks / open questions

- **R1 (BLOCKING)**: Antigravity's auth model is undocumented in this issue. Phase 1 of the work is reading `goo.gle/gemini-cli-migration` end-to-end and writing up: install path (npm/curl/winget), config schema, OAuth-vs-API-key, model lineup, MCP support.
- **R2**: If AI Plus migration loses subscription billing, the financial-engineering decision (stay on Plus + pay PAYG / upgrade to Pro / drop Google entirely for Claude+OpenCode) is user-owned. Spec records the cost surface, not the answer.
- **R3**: Timeline risk. June 18 deadline ≤ 4 weeks from the spec date (2026-05-27). If R1 research surfaces unexpected blockers (e.g., Antigravity Linux package not yet stable), the user may need to accept a regression window.
- **R4**: `ai/gemini/` config tree may need to be re-laid for Antigravity (different config file paths, prompt-file conventions). Estimate after R1.

## Acceptance criteria

- [ ] Phase 1 research summary written: install method, config schema, auth model (OAuth/API-key), model lineup, MCP support.
- [ ] Decision matrix completed with explicit branch chosen (#1, #2, or #3).
- [ ] User's current tier identified (AI Plus / Pro / Ultra / Google One / free).
- [ ] AI-022 follow-up spec scaffolded with the implementation plan.
- [ ] Vault `11-tasks.md` AI-020 entry ticked with the decision link.
- [ ] Decision documented in `40-runbooks/guide-antigravity-cli-migration.md` (NEW vault runbook).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` → AI-020 (full sunset notice + decision matrix).
- Upstream: <https://goo.gle/gemini-cli-migration>.
- Companion: AI-021 (OpenCode Windows validation, includes a sub-question on Gemini-via-OpenCode that overlaps with this decision).
- Sister sunset issue: `agy` (Antigravity) was auto-installed via PR #121 last session — install path already proven on Linux.
