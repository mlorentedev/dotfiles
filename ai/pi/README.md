# pi coding agent (dotfiles overlay)

[pi](https://pi.dev/) (`@earendil-works/pi-coding-agent`) integrated as a managed dotfiles
agent, mirroring `ai/opencode/`. pi reads `AGENTS.md` natively, so it joins the cross-agent
SSOT (see root `AGENTS.md`).

## What dotfiles manages

| Source | Deploy target | Notes |
|--------|---------------|-------|
| `models.json` | `~/.pi/agent/models.json` | NaN custom provider. `apiKey` is `{env:NAN_API_KEY}` in source; `setup-{linux,windows}` inject the literal at deploy time (SDD-009 pattern) so the deployed config is self-contained cross-OS and the key is never committed. |
| `settings.json` | `~/.pi/agent/settings.json` | UX defaults + curated `enabledModels`. **Seed-if-missing**: pi mutates this file at runtime (`lastChangelogVersion`), so setup deploys it only when absent and never clobbers local edits. |
| (canonical `AGENTS.md`) | `~/.pi/agent/AGENTS.md` | Cross-agent SSOT system prompt, deployed verbatim (same as opencode). |

Not managed: `auth.json` (OAuth/secret state) and `skills/` (runtime symlinks).

Because `settings.json` is seed-if-missing, editing it **here** changes only what a fresh
machine gets. To adopt a change on a machine that already has `~/.pi/agent/settings.json`,
edit that file too (or delete it and re-run setup, losing the runtime keys pi wrote there:
`lastChangelogVersion`, `theme`, and any model picked in the TUI).

## Install

Pinned via `PI_VERSION` in `versions.conf`, installed by `setup-{linux,windows}` (guarded on `npm`):

```sh
npm install -g --ignore-scripts @earendil-works/pi-coding-agent@<PI_VERSION>
```

## Model environment

Curated NaN-first: NaN covers the free tier, with three paid OpenRouter models behind it.

- **NaN** (free, primary): `qwen3.6`, `gemma4`, `deepseek-v4-flash`, `deepseek-v4-flash-0731`, `mimo-v2.5`
- **Paid OpenRouter** (3, none from OpenAI/Google/Anthropic): `deepseek-v4-pro`, `qwen3-coder-plus`, `minimax-m3`

pi's picker omits the rate-limited `:free` OpenRouter tier that `ai/opencode/opencode.jsonc`
still lists — the two sets are curated independently. `tests/pi-config.bats` asserts this list
stays equal to `settings.json`'s `enabledModels`.

Default: `nan/qwen3.6`, thinking level `high`. Change in `settings.json`. (Per-model context
windows live in `models.json` — the one place they cannot drift from.)

## Secret

`NAN_API_KEY` lives only in the age store (`sensitive/nan.api-key.secret.age`, mapped in
`secrets/registry.yaml`). The literal never appears in a committed file. Rotate it at the
NaN dashboard if ever exposed.
