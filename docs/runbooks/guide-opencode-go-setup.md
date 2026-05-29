---
id: guide-opencode-go-setup
type: runbook
status: active
created: "2026-05-16"
---

> **Goal:** Set up OpenCode with the Go subscription as a secondary AI coding agent (Claude Code stays primary, aider is sunset in PR2). Idempotent installation, three-layer guardrail against accidental PAYG billing.
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

### `command -v opencode` fails after `setup-linux.sh`

The install script writes the binary to `$HOME/.opencode/bin/opencode` and `setup-linux.sh` adds that directory to PATH in `.zshrc`/`.bashrc`. Reload the shell or `source ~/.zshrc`. If still missing, re-run `setup-linux.sh` — the `command -v` gate is idempotent and will re-fetch only if the binary is genuinely absent.

### `/models` shows Sonnet / GPT-5 / Opus / Gemini Pro under `opencode-go`

The model restriction in `opencode.jsonc` is not in effect. Verify:

```bash
cat ~/.config/opencode/opencode.jsonc
```

The `provider.opencode-go.models` block must list **only** the Go-catalog model IDs. If it lists more, re-run `setup-linux.sh` to redeploy the canonical from `ai/opencode/opencode.jsonc`.

### MCP server registration conflict between `oc` and `claude`

If you see "vault already locked" errors or unexplained merge conflicts in the vault repo after a session, you may have run both agents in parallel. See the coexistence constraint above. Recovery: `cd ~/Projects/knowledge && git status` — resolve conflicts manually, or `git stash && git pull && git stash pop` if remote diverges.

### `opencode --version` reports a different schema than the deployed config expects

OpenCode is pre-1.0 and the config schema can shift. If the TUI logs a schema error on launch, query the latest schema URL and update `ai/opencode/opencode.jsonc` (and re-run setup). The healthcheck section verifies presence of `$schema` but does not enforce a specific version.

### TUI feels noticeably slower than Claude Code, especially under tmux

Empirically confirmed during AI-011-validation (2026-05-17): opencode 1.15.4 TUI does aggressive full-screen refreshes on each streamed token chunk. When stacked under tmux, every refresh has to be parsed and re-emitted through tmux's ANSI layer, adding visible latency. `opencode run "..."` (non-interactive CLI) is **unaffected**. **Recommended workflow:** use Ghostty native splits/tabs (`Ctrl+Shift+T` for new tab, `Ctrl+Shift+E` for split) for opencode TUI sessions; keep tmux for shell sessions and persistent SSH (`sshmux`). Claude Code TUI is unaffected -- its render is more conservative.

### Where to launch opencode (cwd matters a lot)

Surfaced 2026-05-17 during AI-011-validation. opencode performs **per-session work bound to the cwd** that scales with the size and churn rate of that directory:

- `service=snapshot ... tracking` -- a full git snapshot of the cwd is taken at session start (and every time a write tool fires). Stored under `~/.local/share/opencode/snapshot/<projectID>/`. For repos with thousands of files this can add 1-2s per snapshot; for a true monorepo or a heavyweight vault it grows worse.
- `service=file.watcher` (inotify on Linux) -- every file change inside cwd triggers a `bus type=file.watcher.updated` event and may invalidate cached context. If the cwd is being mutated by other processes (Obsidian auto-save, hive MCP auto-commits, vault-maintenance cron, dev server hot reload), opencode is paying constant overhead for nothing relevant to the chat.
- `service=session.prompt` resolveTools -- runs on every prompt; cheap (~20ms) but adds linearly with prompt count.

**Good cwd for opencode:**

- A specific code repo you are actually editing (`~/Projects/dotfiles`, `~/Projects/<some-feature-repo>`)
- Small or medium directory, no constant external mutation
- Git-tracked but with infrequent commits

**Bad cwd for opencode (avoid):**

- `~/Projects/knowledge` (Obsidian vault) -- thousands of `.md` files, constant auto-save churn from Obsidian + hive MCP auto-commits + `dotfiles-vault-maintenance` cron. Snapshot is huge, file watcher fires constantly. Empirically gave 10-20s startup overhead and continuous background noise.
- `$HOME` directly -- everything below it gets watched.
- A multi-gigabyte monorepo with submodules.
- Network-mounted directories (NFS, SMB) -- inotify is unreliable.

**Pattern: query the vault from a code-repo cwd, not the other way around.** If you need vault context while editing code, launch `oc` in the code repo and use the `hive` MCP tools (`vault_query`, `vault_search`) from inside the chat -- those reach the vault without changing the cwd of opencode itself. This matches the architectural intent of having Hive as a per-session-readable knowledge layer (see ADR-009).

### Stream stalls after the first chunk (no `message.part.delta` events following)

Surfaced 2026-05-17 in `~/Projects/resume` cwd. Log pattern:

```
service=llm ... agent=build ... stream         <- request sent
+1327ms bus type=message.part.updated          <- first chunk arrived
+54s    snapshot prune cleanup                 <- next event, unrelated
```

After the first `message.part.updated`, no `message.part.delta` events follow for tens of seconds. This is **upstream provider congestion** or a model that started a long internal reasoning chain without intermediate output -- it is NOT a local opencode bug and NOT a cwd issue (this surfaced with a small clean repo, snapshot finished in 176ms).

**Tactical recovery (in this order):**

1. `Esc` to cancel the streaming request inside the TUI.
2. `/models` -> switch to a different Go-catalog model (`kimi-k2.6`, `qwen3.6-plus`, `glm-5.1`). If the new model responds fast, the original model/endpoint was the bottleneck (transient -- retry later).
3. If all Go models stall -> the entire `opencode-go` provider is congested. Switch provider to `openrouter` via `/models` (uses the existing `OPENROUTER_API_KEY` and the $5 credit balance for frontier models).
4. If everything stalls -> network or auth issue, not a model issue. Check `tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)"` for ERROR lines and verify `command -v opencode && curl -fsI https://api.opencode.ai/`.

**Probable cause windows for Go-catalog congestion:** Chinese providers (DeepSeek, Kimi, Qwen, GLM, MiMo, MiniMax) follow Asia working hours. UTC 00:00-09:00 is daytime China -- peak load. UTC 16:00-22:00 is night in China -- minimal load. From Europe, slowdowns are most likely around UTC 00:00-04:00 local.

**Empirical fallback data point (2026-05-17, UTC 02:45 = 10:45 China peak):** DeepSeek V4 Pro stream stalled after first chunk; switched to **`qwen3.6-plus`** with same prompt, response was visibly faster and completed normally. Suggests Qwen3.6-plus is a reliable second-line option when DeepSeek is congested. Worth A/B'ing alongside Kimi K2.6 over the first month of use (originally planned default: DeepSeek; A/B candidate: Kimi -- consider adding Qwen as a third).

**Variant of the stall: hang on the second LLM call after a tool result (2026-05-17, UTC 02:48 = 10:48 China peak).** Different signature, same root cause family. Pattern in log:

```
service=llm ... agent=build ... stream        <- first request
... 21s of message.part.delta events          <- first turn streamed fine
tool.registry status=started/completed <X>    <- model issued a tool call, opencode resolved it
service=session.processor ... process         <- preparing second turn with tool result
service=llm ... agent=build ... stream        <- SECOND request sent
[silence for minutes]                         <- second turn never streams
```

The first turn completes normally; opencode runs the tool the model requested (bash/read/grep/etc.) in <200ms; opencode sends a second LLM call containing the tool result; that second call is the one that hangs. So this is **NOT** a "first chunk stall" -- the first turn worked fine. It is congestion on the second roundtrip, often because the tool result inflates the context size and the provider deprioritises the request.

**Tactical recovery same family of moves:** `Esc` to cancel, retry with a prompt that does not need tool calls (`"what is 2+2?"`) to confirm the model itself is reachable, then either switch model with `/models` or switch provider to `openrouter` for that task. If `/models` shows no fast Go model and you cannot wait, OpenRouter (existing `$5` credit, no PAYG auto-recharge thanks to Layer 3) is the safest temporary jump.

### Live log tailing (when "thinking..." takes forever and you want to see what's happening)

The TUI hides backend events. Open a second Ghostty tab/split and tail the most recent log:

```bash
# Quick one-liner
tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)"

# Filtered (skip noisy file-watcher events and per-token deltas)
tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)" \
  | grep --line-buffered -vE "file\.watcher\.updated|bus type=message\.part\.delta"
```

Or persist as alias in `.zshrc` / `.bashrc`:

```bash
alias oclog='tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)" | grep --line-buffered -vE "file\.watcher\.updated|bus type=message\.part\.delta"'
```

Key events to look for: `service=llm ... stream` (request sent), `message.part.updated` (first chunk arrived), `message.part.delta` (token streaming -- if absent for >10s after `stream`, the model is hung). Do NOT use `opencode --print-logs` against the TUI -- it writes to stderr and corrupts the render. Always log via the file path.

### Cost figure in TUI status bar — informational, not actual billing

The number shown at the bottom of the TUI (e.g., `$0.12`) is a **theoretical PAYG equivalent**, calculated as `tokens × per-million-listed-price-of-model`. With the Go subscription, your actual cost is the flat `$10/mo`. The TUI figure is useful only as "what you would pay without the Go plan." For real billing state, always check <https://opencode.ai/zen> → Billing.

### First DeepSeek V4 Pro response time vs Claude Sonnet

DeepSeek V4 Pro on Go infrastructure: ~1–3s first-token latency from Europe, 40–80 tokens/sec sustained. Claude Sonnet 4.6/4.7: ~0.3–0.8s first-token, 80–150 tokens/sec. This is provider geography + inference stack differences, **not a bug**. For latency-sensitive tasks (interactive refactor, line-by-line code review), prefer Claude Code; for cheap bulk iteration (summarisation, scripted edits, mechanical refactors), opencode + DeepSeek wins on $/token.

### Ghostty `theme "<name>" not found` on config validation

Ghostty 1.3.0 theme names use literal capitalization with spaces, not kebab-case. List available: `ghostty +list-themes | grep -i <fragment>`. For Catppuccin family: `Catppuccin Mocha`, `Catppuccin Latte`, `Catppuccin Frappe`, `Catppuccin Macchiato` (each with the literal space). Use `ghostty +validate-config` after editing `~/.config/ghostty/config` to catch typos before reopening the TUI.

## References

- Spec: `~/Projects/dotfiles/specs/AI-011-opencode-bootstrap/`
- ADR: [ADR-009](../adr/adr-009-multi-agent-runtime.md)
- Pattern: `pattern-setup-script-idempotence` (maintainer's cross-project knowledge store)
- Upstream docs: <https://opencode.ai/docs/> and <https://opencode.ai/go>
