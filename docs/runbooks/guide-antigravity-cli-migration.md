---
id: "guide-antigravity-cli-migration"
type: runbook
status: active
tags: [runbook, dotfiles, antigravity, gemini-cli, migration, oauth, ai-plus]
created: "2026-06-03"
owner: manu
---

# Antigravity CLI Migration

Decision record + operating procedure for the `gemini-cli` → Antigravity CLI
(`agy`) migration. Resolves AI-020 (decision) and feeds AI-023 (verification).

## Why this exists

`gemini-cli` upstream-sunsets on **2026-06-18**. Per Google's own announcement
(<https://goo.gle/gemini-cli-migration>), it stops serving requests for **Google
AI Pro and Ultra, plus free** individual tiers on that date — only
Standard/Enterprise licenses, Google Cloud, or paid API keys keep working. So
for any consumer tier (including Google AI Plus), migrating to Antigravity CLI
is **mandatory**, not optional.

Compounding nuance: **Google AI Plus does not support Plan Linking with AI Studio
API keys** (only Pro/Ultra do). An API key issued by an AI Plus account bills
**PAYG**, not subscription quota. This only matters for tools that auth via API
key — `agy` does not.

## Decision (AI-020, 2026-06-03)

**Branch #1 — keep Google AI Plus via `agy` OAuth. No tier upgrade.**

- `agy` authenticates with **browser OAuth** against the Code Assist backend
  `https://cloudcode-pa.googleapis.com` (it issues `loadCodeAssist`) — the same
  subscription-recognizing path `gemini-cli` used. No API key, so the AI Plus
  PAYG trap does not apply.
- **Cost decision (user-owned):** Gemini is low-reliance (3rd agent behind
  Claude Code + OpenCode). If AI Plus quota is ever *not* honored post-cutover,
  the resolution is to fall back to the free Antigravity allotment or **drop
  Gemini** — never a paid tier upgrade.

### Decision matrix (resolved)

| Branch | Condition | Outcome |
|--------|-----------|---------|
| **#1 (chosen)** | `agy` OAuth recognizes AI Plus | Free migration, subscription preserved, no upgrade |
| #2 | `agy` requires an API key | AI Plus → PAYG; **do not upgrade** (low reliance) → drop Gemini |
| #3 | Antigravity uses a different tier system | Re-evaluate; default to free allotment or drop |

## Migration status: already shipped (SDD-007)

The install/config migration is **done** cross-OS — no porting work remains:

- `setup-{linux.sh,windows.ps1}` install `agy` via
  `curl -fsSL https://antigravity.google/cli/install.sh | bash` (idempotent,
  gated on `command -v agy`). They no longer install `gemini-cli`.
- A one-time `gemini-cli → agy` migration removes the legacy `GEMINI.md`.
- Config lives under `~/.gemini/` (agy inherits the path for back-compat):
  `~/.gemini/antigravity-cli/settings.json`, MCP consolidated at
  `~/.gemini/config/mcp_config.json`.
- `ANTIGRAVITY_ENDPOINT=https://cloudcode-pa.googleapis.com` (production Code
  Assist endpoint = the subscription path).
- `healthcheck.sh` section 13 ("Antigravity CLI Health") already guards endpoint,
  app-data path, and MCP config integrity.

## Verify branch #1 (do this once)

```bash
# 1. Trigger / confirm OAuth. If a browser opens, log in with the Google
#    account holding AI Plus. If it drops into a prompt, you are already in.
agy

# 2. Cheap one-shot — confirms requests are served (quota works):
agy -p "reply with exactly: pong"

# 3. Confirm the subscription path (not a PAYG API key):
echo "$ANTIGRAVITY_ENDPOINT"   # expect https://cloudcode-pa.googleapis.com
```

**Interpretation**

- Step 2 returns `pong` and you were never asked for an API key →
  **AI Plus recognized via OAuth = branch #1 verified.**
- Step 2 errors with a quota/billing message → quota not recognized → apply the
  fallback (free allotment or drop Gemini). Do **not** upgrade.

> **Empirical result (2026-06-03): ✅ verified.** `agy -p "reply with exactly: pong"`
> returned `pong` with no API-key prompt → Google AI Plus recognized via OAuth
> (Code Assist backend). Branch #1 confirmed.

## Fallback / recovery

- Auth or update corruption: see [Gemini CLI Recovery](guide-gemini-cli-recovery.md)
  (`~/.gemini/` paths are shared).
- Quota lost on AI Plus: drop Gemini from the daily rotation; Claude Code +
  OpenCode remain primary. Re-evaluate only if Gemini reliance rises.

## References

- Decision spec: `specs/AI-020-gemini-empirical-validation/` (matrix + evidence).
- Follow-up: `specs/AI-023-antigravity-oauth-verification/` (OAuth/quota guard).
- Upstream: <https://goo.gle/gemini-cli-migration>.
- Related: [AI Tools Setup](ai-tools-setup.md), [Gemini CLI Recovery](guide-gemini-cli-recovery.md).
