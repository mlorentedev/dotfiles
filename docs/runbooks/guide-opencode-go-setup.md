---
id: guide-opencode-go-setup
type: runbook
status: active
created: "2026-05-16"
---

> ⚠️ **Stale — describes a retired provider architecture.** The Go subscription
> this runbook sets up (`/connect` → OpenCode Go, the three-layer PAYG guardrail,
> `deepseek-v4-pro` default) is **no longer the deployed configuration**. OpenCode's
> default provider is now `nan` (community provider, `nan/qwen3.6` default,
> per SDD-007-ai-tooling-consolidation) — see `ai/opencode/opencode.jsonc`, the
> live SSOT for provider/model facts. The steps below are dead as instructions,
> not merely stale in the model names; a rewrite for the NaN-era setup flow is
> tracked separately. Kept for historical reference (what the Go-subscription
> onboarding looked like) and because the coexistence-constraint and daily-usage
> sections below still apply regardless of provider.
>
> **Goal (as originally written):** Set up OpenCode with the Go subscription as a secondary AI coding agent (Claude Code stays primary, aider is sunset in PR2). Idempotent installation, three-layer guardrail against accidental PAYG billing.
>
> **Audience:** future-me on a clean Linux box. Cross-OS Windows side ships separately.

## Prerequisites

| Item | Why | Check |
|---|---|---|
| `setup-linux.sh` already ran | Deploys `~/.config/opencode/opencode.jsonc` + binary install | `command -v opencode` resolves |
| `OPENROUTER_API_KEY` exported | Provider `openrouter` autodetects it (frontier on-demand without PAYG to Zen) | `[ -n "$OPENROUTER_API_KEY" ]` |
| Active OpenCode Go subscription | $10/mo (or $5 first month) from opencode.ai/go | API key visible in opencode.ai/zen dashboard |

If `setup-linux.sh` has not run yet, run it first — the binary, PATH, config, and healthcheck section are all wired up there.

## First-time setup (one-time, manual)

The `/connect` flow is interactive and requires pasting an API key, so it is not automated in `setup-linux.sh`. Do this once per machine:

```bash
oc                                  # launches the OpenCode TUI
```

Inside the TUI:

```text
/connect
```

1. The TUI presents a list of providers — select **OpenCode Go**.
2. It opens a browser tab at opencode.ai/auth (or prints the URL if no browser).
3. Authenticate, copy the API key from the dashboard.
4. Paste the key into the TUI prompt and press Enter.

The key is stored at `~/.local/share/opencode/auth.json`. Do **not** commit this file. It is not tracked by dotfiles intentionally — secret automation via age is a follow-up once the format stabilizes.

Verify:

```text
/models
```

Should show the subset of the Go catalog you activated through the UI — pick `deepseek-v4-pro` (default) + `kimi-k2.6` (A/B) at minimum, optionally add others like `glm-5.1` / `qwen3.6-plus` for variety. **If you see Sonnet, GPT-5, Opus, or Gemini Pro under `opencode-go`**, your local activation state got corrupted — clear `~/.local/state/opencode/model.json` and re-pick from the picker. These frontier models should not exist in the Go catalog; their appearance would indicate opencode is misrouting providers (file an upstream bug).

## The three-layer PAYG guardrail (do all three)

Why: a careless `/models` selection of a frontier model under Zen PAYG would trigger auto-recharge ($20 when balance < $5). Defense in depth:

### Layer 1 — config (display metadata only, NOT a whitelist)

> **Empirically validated 2026-05-17:** the `models` block in `opencode.jsonc` enriches display names for the listed models, it does **not** restrict the `/models` picker. The actual whitelist mechanism is **UI-driven model activation** (per-user state in `~/.local/state/opencode/model.json`) — the user picks which subset of the Go catalog to surface. The full Go catalog (12 models: `deepseek-v4-{flash,pro}`, `glm-{5,5.1}`, `kimi-k2.{5,6}`, `mimo-v2.5{,-pro}`, `minimax-m2.{5,7}`, `qwen3.{5,6}-plus`) is inherently bounded to cheap Chinese LLMs covered by the $10/mo Go plan — no PAYG risk inherent to the Go subscription itself.

`ai/opencode/opencode.jsonc` declares the default model (`opencode-go/deepseek-v4-pro`) and friendly display names for the A/B candidates. **Real PAYG protection lives in Layers 2 and 3 below.**

### Layer 2 — workspace cap at opencode.ai/zen

1. Sign in to <https://opencode.ai/zen>.
2. Open **Workspace → Billing**.
3. If a **PAYG spending cap** option is available for individual accounts: set it to `$0`.
4. If only team-level caps exist: skip this layer; Layer 3 is the safety net.

### Layer 3 — no payment method for PAYG

1. In the Zen dashboard, **Workspace → Billing → Payment methods**.
2. Confirm there is **exactly one** payment method, vinculated to the **Go subscription** (not to PAYG/Zen general billing).
3. If a separate "Zen workspace billing" payment method exists, remove it.
4. With no card on file for PAYG, any inadvertent call to a frontier model **fails** rather than charging. This is the strongest guarantee.

## Daily usage

```bash
oc                                  # launch TUI from any repo
```

Default model: `deepseek-v4-pro` (set in `opencode.jsonc`). To switch:

```text
/models                             # picker, restricted to Go catalog
```

A/B comparison Kimi K2.6 vs DeepSeek V4 Pro is tracked as a follow-up — first month of real use will inform the default.

For frontier (Sonnet, GPT-5, Gemini Pro) on demand: switch provider to `openrouter` via `/models` (uses the existing `OPENROUTER_API_KEY` and the $5 credit balance — once exhausted, calls fail rather than auto-recharge).

### Quick-question wrappers (`qq` / `qf`)

Non-interactive one-shot aliases (shipped 2026-05-18, dotfiles PR #56). Wrap `opencode run -m <model>` so you can ask a question from any shell without opening the TUI. Each call starts a fresh session — for follow-ups use `opencode run -c` or the TUI.

| Alias | Model | When |
|---|---|---|
| `qq` | `opencode-go/qwen3.6-plus` | Default — multilingual (ES-friendly), balanced |
| `qf` | `opencode-go/deepseek-v4-flash` | Faster (~97 tok/s, **never-rate-limited** per opencode-go docs); good for strict technical Qs |

```bash
qq por que tardas tanto?            # no quotes needed in zsh
qf "explain the C10k problem"       # quoted form also works
```

Source files: `.zsh/aliases.zsh`, `.bashrc`, `powershell/profile.ps1`.

**Design notes worth keeping:**

- Name is `qq` (not `??`) because PowerShell 7+ reserves `??` as the null-coalescing operator — `Set-Alias`/`function` cannot use it. `qq` is the cross-platform compromise.
- zsh aliases are wrapped in `noglob` so trailing `?` chars don't trigger zsh's `nomatch` error. Bash and PowerShell defaults are permissive enough; no `noglob` needed there.
- Shared helper `_qq_call <model> <name> "$@"` factors the usage check + invocation. The two aliases differ only by pinned model.

## Coexistence constraint with Claude Code

Until the lock-file follow-up on the Hive MCP server lands, do **not** run `oc` and `claude` in parallel on the same repo. Both agents have the Hive MCP server registered, and both auto-commit vault edits to git — racing writes can corrupt the working tree.

Safe patterns:

- Same repo, sequential: finish one agent's task fully before launching the other.
- Different repos, parallel: fine — Hive serializes within a single vault but agents in different working dirs don't conflict on the vault itself the same way as on a single repo's working tree.

This constraint disappears once the hive flock follow-up is merged.

## Troubleshooting

Moved to [`docs/troubleshooting/opencode.md`](../troubleshooting/opencode.md) — cwd/snapshot
forensics, stream-stall recovery, TUI-under-tmux latency, log tailing, and the rest
of the deep troubleshooting content that used to live in this section (D24 split).

## References

- Spec (archived): `specs/archive/AI-011-opencode-bootstrap/` (initial bootstrap, 2026-05-17 — the era this runbook's setup steps describe)
- ADR: [ADR-009](../adr/adr-009-multi-agent-runtime.md)
- Current provider/model SSOT: `ai/opencode/opencode.jsonc`
- Troubleshooting: [`docs/troubleshooting/opencode.md`](../troubleshooting/opencode.md)
- Upstream docs: <https://opencode.ai/docs/> and <https://opencode.ai/go>
