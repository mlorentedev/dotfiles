---
id: dotfiles-troubleshoot-opencode
type: troubleshooting
status: active
tags: [troubleshooting, dotfiles, opencode, tui, performance]
created: "2026-08-12"
---

# Troubleshooting: OpenCode

> ⚠️ **Split out of `guide-opencode-go-setup.md` (D24), which described the
> now-retired OpenCode Go subscription (provider `opencode-go`, PAYG/Zen
> billing). OpenCode's default provider is now `nan` (SDD-007-ai-tooling-consolidation;
> see `ai/opencode/opencode.jsonc`), and `guide-opencode-go-setup.md` has not yet
> been rewritten for that architecture (tracked separately).
>
> **What still applies here:** the diagnostic patterns and log-tailing technique
> below are provider-independent — they hold regardless of which provider is
> configured. **What's era-bound:** any *model name* named in a recovery step
> below (`deepseek-v4-pro`, `kimi-k2.6`, `qwen3.6-plus`, the `opencode-go`
> catalog) is from the Go-subscription era; substitute the current models from
> `ai/opencode/opencode.jsonc` (`nan/qwen3.6` default, `nan/deepseek-v4-flash`
> on-demand, `nan/gemma4` A/B) when applying the same recovery *pattern* today.

## `command -v opencode` fails after `setup-linux.sh`

The install script writes the binary to `$HOME/.opencode/bin/opencode` and `setup-linux.sh` adds that directory to PATH in `.zshrc`/`.bashrc`. Reload the shell or `source ~/.zshrc`. If still missing, re-run `setup-linux.sh` — the `command -v` gate is idempotent and will re-fetch only if the binary is genuinely absent.

## `/models` shows unexpected providers/models

Check the deployed config against the SSOT:

```bash
cat ~/.config/opencode/opencode.jsonc
```

Compare against `ai/opencode/opencode.jsonc` in the repo — `disabled_providers` and each provider's `models` block are what actually restrict the `/models` picker. If the deployed file diverges, re-run `setup-linux.sh` to redeploy the canonical version.

## MCP server registration conflict between `oc` and `claude`

If you see "vault already locked" errors or unexplained merge conflicts in the vault repo after a session, you may have run both agents in parallel against the same vault. Coexistence constraint: don't run `oc` and `claude` in parallel on the same repo until the Hive MCP server adds a lock-file to its auto-commit. Recovery: `cd ~/Projects/knowledge && git status` — resolve conflicts manually, or `git stash && git pull && git stash pop` if remote diverges.

## `opencode --version` reports a different schema than the deployed config expects

OpenCode is pre-1.0 and the config schema can shift. If the TUI logs a schema error on launch, query the latest schema URL and update `ai/opencode/opencode.jsonc` (and re-run setup). The healthcheck section verifies presence of `$schema` but does not enforce a specific version.

## TUI feels noticeably slower than Claude Code, especially under tmux

Empirically confirmed during AI-011-validation (2026-05-17): the opencode TUI does aggressive full-screen refreshes on each streamed token chunk. When stacked under tmux, every refresh has to be parsed and re-emitted through tmux's ANSI layer, adding visible latency. `opencode run "..."` (non-interactive CLI) is **unaffected**. **Recommended workflow:** run opencode TUI sessions in your terminal's native tabs/splits (outside tmux); keep tmux for shell sessions and persistent SSH (`sshmux`). Claude Code TUI is unaffected — its render is more conservative. This is a TUI-rendering behavior, independent of which provider is configured.

## Where to launch opencode (cwd matters a lot)

Surfaced 2026-05-17 during AI-011-validation. opencode performs **per-session work bound to the cwd** that scales with the size and churn rate of that directory:

- `service=snapshot ... tracking` — a full git snapshot of the cwd is taken at session start (and every time a write tool fires). Stored under `~/.local/share/opencode/snapshot/<projectID>/`. For repos with thousands of files this can add 1-2s per snapshot; for a true monorepo or a heavyweight vault it grows worse. (Note: `ai/opencode/opencode.jsonc` now sets `"snapshot": false` repo-wide — this specific overhead should no longer apply on a machine running the current deployed config; kept here in case it's re-enabled.)
- `service=file.watcher` (inotify on Linux) — every file change inside cwd triggers a `bus type=file.watcher.updated` event and may invalidate cached context. If the cwd is being mutated by other processes (Obsidian auto-save, hive MCP auto-commits, vault-maintenance cron, dev server hot reload), opencode is paying constant overhead for nothing relevant to the chat.
- `service=session.prompt` resolveTools — runs on every prompt; cheap (~20ms) but adds linearly with prompt count.

**Good cwd for opencode:**

- A specific code repo you are actually editing (`~/Projects/dotfiles`, `~/Projects/<some-feature-repo>`)
- Small or medium directory, no constant external mutation
- Git-tracked but with infrequent commits

**Bad cwd for opencode (avoid):**

- `~/Projects/knowledge` (Obsidian vault) — thousands of `.md` files, constant auto-save churn from Obsidian + hive MCP auto-commits + `dotfiles-vault-maintenance` cron. Snapshot is huge, file watcher fires constantly. Empirically gave 10-20s startup overhead and continuous background noise (pre-`"snapshot": false`).
- `$HOME` directly — everything below it gets watched.
- A multi-gigabyte monorepo with submodules.
- Network-mounted directories (NFS, SMB) — inotify is unreliable.

**Pattern: query the vault from a code-repo cwd, not the other way around.** If you need vault context while editing code, launch `oc` in the code repo and use the `hive` MCP tools (`vault_query`, `vault_search`) from inside the chat — those reach the vault without changing the cwd of opencode itself.

## Stream stalls after the first chunk (no `message.part.delta` events following)

Surfaced 2026-05-17 in `~/Projects/resume` cwd (opencode-go era). Log pattern:

```
service=llm ... agent=build ... stream         <- request sent
+1327ms bus type=message.part.updated          <- first chunk arrived
+54s    snapshot prune cleanup                 <- next event, unrelated
```

After the first `message.part.updated`, no `message.part.delta` events follow for tens of seconds. This is **upstream provider congestion** or a model that started a long internal reasoning chain without intermediate output — it is NOT a local opencode bug and NOT a cwd issue (this surfaced with a small clean repo, snapshot finished in 176ms).

**Tactical recovery (in this order) — the pattern, not the specific era-bound model names:**

1. `Esc` to cancel the streaming request inside the TUI.
2. `/models` → switch to a different model from the current catalog (`ai/opencode/opencode.jsonc`; today that's `nan/deepseek-v4-flash` or `nan/gemma4` as alternatives to the `nan/qwen3.6` default). If the new model responds fast, the original model/endpoint was the bottleneck (transient — retry later).
3. If every `nan` model stalls → the entire `nan` provider is congested. Switch provider to `openrouter` via `/models` (uses the existing `OPENROUTER_API_KEY`).
4. If everything stalls → network or auth issue, not a model issue. Check `tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)"` for ERROR lines and verify `command -v opencode && curl -fsI https://api.opencode.ai/`.

**Historical data point (2026-05-17, opencode-go era, UTC 02:45 = 10:45 China peak):** DeepSeek V4 Pro stream stalled after first chunk; switching to Qwen3.6-plus (then a Go-catalog model) was visibly faster. The general lesson — Chinese-provider load correlates with China working hours (UTC 00:00-09:00 peak, UTC 16:00-22:00 minimal, so from Europe UTC 00:00-04:00 is the likeliest slowdown window) — plausibly still applies to `nan`'s catalog, but has not been re-verified against it.

### Variant: hang on the second LLM call after a tool result

Surfaced 2026-05-17, UTC 02:48 = 10:48 China peak (opencode-go era). Different signature, same root-cause family. Pattern in log:

```
service=llm ... agent=build ... stream        <- first request
... 21s of message.part.delta events          <- first turn streamed fine
tool.registry status=started/completed <X>    <- model issued a tool call, opencode resolved it
service=session.processor ... process         <- preparing second turn with tool result
service=llm ... agent=build ... stream        <- SECOND request sent
[silence for minutes]                         <- second turn never streams
```

The first turn completes normally; opencode runs the tool the model requested (bash/read/grep/etc.) in <200ms; opencode sends a second LLM call containing the tool result; that second call is the one that hangs. So this is **NOT** a "first chunk stall" — the first turn worked fine. It is congestion on the second roundtrip, often because the tool result inflates the context size and the provider deprioritizes the request.

**Tactical recovery, same family of moves:** `Esc` to cancel, retry with a prompt that does not need tool calls (`"what is 2+2?"`) to confirm the model itself is reachable, then either switch model with `/models` or switch provider to `openrouter` for that task.

## Live log tailing (when "thinking..." takes forever and you want to see what's happening)

The TUI hides backend events. Open a second terminal tab/split and tail the most recent log:

```bash
# Quick one-liner
tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)"

# Filtered (skip noisy file-watcher events and per-token deltas)
tail -F "$(ls -t ~/.local/share/opencode/log/*.log | head -1)" \
  | grep --line-buffered -vE "file\.watcher\.updated|bus type=message\.part\.delta"
```

Persisted as the `oclog` alias in `.zsh/aliases.zsh` / `.bashrc`.

Key events to look for: `service=llm ... stream` (request sent), `message.part.updated` (first chunk arrived), `message.part.delta` (token streaming — if absent for >10s after `stream`, the model is hung). Do NOT use `opencode --print-logs` against the TUI — it writes to stderr and corrupts the render. Always log via the file path.

## Cost figure in TUI status bar — informational, not actual billing

The number shown at the bottom of the TUI (e.g., `$0.12`) is a **theoretical PAYG-equivalent**, calculated as `tokens × per-million-listed-price-of-model`. This figure predates the NaN provider switch; whether/how it applies to NaN's pricing model has not been re-verified. For real billing state, check the provider's own dashboard.

## Comparative latency: DeepSeek V4 Pro vs Claude Sonnet (historical, opencode-go era)

DeepSeek V4 Pro on Go infrastructure: ~1–3s first-token latency from Europe, 40–80 tokens/sec sustained. Claude Sonnet 4.6/4.7: ~0.3–0.8s first-token, 80–150 tokens/sec. This was provider geography + inference stack differences, not a bug — kept as a reference data point; not re-measured against the current `nan` default.

## References

- [`guide-opencode-go-setup.md`](../runbooks/guide-opencode-go-setup.md) — one-time setup + guardrail (predates the NaN provider switch; see its own banner)
- SSOT for current provider/model facts: `ai/opencode/opencode.jsonc`
- Upstream docs: <https://opencode.ai/docs/>
