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

`settings.json` is the one deployed config pi itself rewrites — `lastChangelogVersion`,
`theme`, the model picked in the TUI — so setup seeds it and then leaves it alone. Editing
it **here** therefore changes only what a *fresh* machine gets: to adopt a change on a
machine that already has `~/.pi/agent/settings.json`, edit that file too, or delete it and
re-run setup to take the committed defaults wholesale. `tests/pi-config.bats` pins that
contract in both setup scripts — until #754 they compared source against destination and,
because the deployed file always differs, overwrote it on every run.

## Install

Pinned via `PI_VERSION` in `versions.conf`, installed by `setup-{linux,windows}` (guarded on `npm`):

```sh
npm install -g --ignore-scripts @earendil-works/pi-coding-agent@<PI_VERSION>
```

## Model environment

NaN only — the free tier, with no paid fallback in the picker.

- **NaN** (free, primary): `qwen3.6`, `gemma4`, `deepseek-v4-flash`, `deepseek-v4-flash-0731`, `mimo-v2.5`

The three paid OpenRouter models (`deepseek-v4-pro`, `qwen3-coder-plus`, `minimax-m3`) were
dropped from the picker: a paid model one keystroke away in a model list is a cost you take by
accident, not by decision. Frontier work goes through `ai/opencode/opencode.jsonc`, where reaching
for OpenRouter is an explicit act.

pi's picker also omits the rate-limited `:free` OpenRouter tier that `opencode.jsonc` still lists —
the two sets are curated independently. `tests/pi-config.bats` asserts this list stays equal to
`settings.json`'s `enabledModels`.

Default: `nan/qwen3.6`, thinking level `high`. Change in `settings.json`. (Per-model context
windows live in `models.json` — the one place they cannot drift from.)

## Secret

`NAN_API_KEY` lives only in the age store (`sensitive/nan.api-key.secret.age`, mapped in
`secrets/registry.yaml`). The literal never appears in a committed file. Rotate it at the
NaN dashboard if ever exposed.
