# pi coding agent (dotfiles overlay)

[pi](https://pi.dev/) (`@earendil-works/pi-coding-agent`) integrated as a managed dotfiles
agent, mirroring `ai/opencode/`. pi reads `AGENTS.md` natively, so it joins the cross-agent
SSOT (see root `AGENTS.md`).

## What dotfiles manages

| Source | Deploy target | Notes |
|--------|---------------|-------|
| `models.json` | `~/.pi/agent/models.json` | The OpenRouter provider. NaN's comes from `pi-nan-provider` (see *NaN provider package*). `apiKey` is `${OPENROUTER_API_KEY}`, which pi resolves from its own environment at request time, so the key is in neither the repo nor the deployed file (mode `0600`). |
| `settings.json` | `~/.pi/agent/settings.json` | UX defaults + curated `enabledModels`. **Seed-if-missing**: pi mutates this file at runtime (`lastChangelogVersion`), so setup deploys it only when absent and never clobbers local edits. |
| `packages.json` | (not deployed — reconciled) | Declared pi packages, each pinned. Setup installs the difference against the live `settings.json` on every run, through `pi install`. See below. |
| `mcp.json` | `~/.pi/agent/mcp.json` | Model Context Protocol servers (`hive`, `context7`, `sequential-thinking`). Deployed via `dotf deploy` (`ai/deploy.json`, `requires: "pi"`). Tools load on-demand via `pi-mcp-client` meta-tool (`mcp_tools`). Note: `context7` routes context over HTTPS to an external API. |
| `nan-provider.json` | `~/.pi/agent/nan-provider.json` | `pi-nan-provider`'s own state file. Only `mediaMcp: false` is managed (`strategy: merge`); `webSearch` stays the box's own `/nan-mcp` toggle. See *NaN provider package* below. |
| (canonical `AGENTS.md`) | `~/.pi/agent/AGENTS.md` | Cross-agent SSOT system prompt, deployed verbatim (same as opencode). |

Not managed: `auth.json` (OAuth/secret state) and `skills/` (runtime symlinks).

`settings.json` is the one deployed config pi itself rewrites — `lastChangelogVersion`,
`theme`, the model picked in the TUI — so setup seeds it and then leaves it alone. Editing
it **here** therefore changes only what a *fresh* machine gets: to adopt a change on a
machine that already has `~/.pi/agent/settings.json`, edit that file too, or delete it and
re-run setup to take the committed defaults wholesale. `tests/pi-config.bats` pins that
contract in both setup scripts — until #754 they compared source against destination and,
because the deployed file always differs, overwrote it on every run.

## Packages

`packages.json` declares the pi packages (extensions, skills, prompts, themes)
this environment wants. `setup-{linux,windows}` reconciles it against the live
`~/.pi/agent/settings.json` on **every** run and installs what is missing, so an
existing machine converges on the next setup and a fresh one on its first
(AI-030, #1224).

It is **not** a deployed file, and the `packages` array deliberately does not
live in `settings.json` above. That file is seed-if-missing, so anything
declared there would reach a fresh machine and never this one. `pi install`
writes the live array itself — and unpacks the package to disk while doing it,
which an array entry written by setup would not.

Every entry is pinned. Upstream is explicit that packages *"run with full system
access — extensions execute arbitrary code and skills can instruct the model to
run executables"*, inside an agent holding `NAN_API_KEY`. `tests/pi-packages.bats`
refuses an unpinned entry. Bumping one is a deliberate edit here, which puts a
diff and a reviewer between upstream's publish and the next setup run.

Adding one by hand (`pi install npm:pkg@1.2.3`) works and writes the live array,
but nothing else will ever know about it — put it in `packages.json` instead.

Docs: <https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md>

## NaN provider package

`npm:@gtrabanco/pi-nan-provider` (`packages.json`, AI-046) registers pi's `nan`
provider from a build-time snapshot merged with NaN's live `/models`. It also
drops cross-model reasoning replays that would overflow `qwen3.6`'s 262K window
on a model switch, and retries truncated streams.

- **The key** reaches it the way it reaches every pi provider: `NAN_API_KEY`
  from the environment, which the `pi` shell wrapper injects through
  `dotf secrets run`. The package's `/login` and key-file paths are not used.
- **The media MCP bridge is off.** It spawns `npx -y nan-mcp-server@<version>`
  per tool call, a second npm package outside this manifest. The package reads
  `NAN_MEDIA_MCP`, then `nan-provider.json`, then defaults to on, so the managed
  file is what keeps it off. `NAN_MEDIA_MCP=1 pi` turns it on for one session.
- **Offline:** with NaN unreachable, pi still starts and lists the snapshot
  models (measured 2026-09-26, `specs/archive/AI-046-pi-nan-provider/verification.md`).
- **Premium models** such as `glm5.3` are filtered by the member's tier, so a
  standard key does not list them.
- **`models.json` defines no NaN model** (AI-046 AC3, `tests/pi-config.bats`). A
  `models` entry there would replace the package's model with the same id, and
  the live catalog would stop applying to it. A per-model change goes in
  `providers.nan.modelOverrides`, which composes above the package.
- **The ids are checked against the package,** not against a copy of it: the
  `pi-nan-package` CI job installs the pinned package and runs
  `tests/pi-nan-package.bats` (`enabledModels`, `defaultModel`, the reviewer
  pool, and opencode's context windows).

## MCP (Model Context Protocol)

Pi integrates MCP servers via `npm:pi-mcp-client@0.8.0` (`packages.json`) and `mcp.json` (`ai/deploy.json`, `requires: "pi"`):
- `hive`: stdio client (`hive client`), connects to the supervised `hive.service` daemon for vault access (`/home/manu/Projects/knowledge`).
- `context7`: remote HTTP endpoint (`https://mcp.context7.com/mcp`).
- `sequential-thinking`: pinned calver package (`@modelcontextprotocol/server-sequential-thinking@2026.8.31`).

`pi-mcp-client` uses deferred tool loading: it injects a single `mcp_tools` meta-tool into the initial prompt (~350 tokens) rather than eagerly loading all schemas. Models discover tools via `mcp_tools({ query: "..." })` and activate them via `mcp_tools({ activate: ["hive.vault_query"] })`. Once activated, tools are callable under their namespaced prefix `mcp__<server>__<tool>`.

## Install

Pinned via `PI_VERSION` in `versions.conf`, installed by `setup-{linux,windows}` (guarded on `npm`):

```sh
npm install -g --ignore-scripts @earendil-works/pi-coding-agent@<PI_VERSION>
```

## Model environment

NaN only — the free tier, with no paid fallback in the picker.

- **NaN** (free, primary): `glm5.3-flash`, `deepseek-v4-flash`, `qwen3.8-flash`, `qwen3.6`, `mimo-v2.6-flash`, `gemma4`

`qwen3.8-flash` and `glm5.3-flash` (added 2026-08-26) are catalog additions only — picker
availability, not a default or routing change. Both are live, reasoning-class, and
independently strong: verified via `scripts/nan-debug.sh` on both a smoke prompt
(surfaced `reasoning_content`) and this repo's own planted `((count++))`/`set -e` bug —
the same defect that admitted `mimo-v2.5` (and on 2026-09-30 its successor `mimo-v2.6-flash`) to `harness/reviewer-pool.json` — which both
identified correctly. Published third-party benchmarks back that up: Qwen3.8-Flash-Next
(Alibaba) and GLM-5.3-Flash (Zhipu) both launched 2026-08-26 as genuine frontier-tier
releases with real 1M context, not marketing inflation — see #1244 for sources. Quality is
therefore *not* the reason they stay out of routing. The reason is that NaN announced them
on a **promotional token allocation that expires end of August 2026**, ahead of a community
vote on whether/how they stay, and neither has been run through `reviewer-pool.json`'s own
admission procedure yet. Neither is wired into `defaultModel`, `harness/model-map.json`,
`.pr_agent.toml` or `harness/reviewer-pool.json` — see #1244.

One capability nuance for `qwen3.8-flash` specifically: Alibaba's own release notes put its
*native* context at 262,144 tokens, extended to 1M via YaRN, and YaRN-extended context can
behave differently at the far end of the window than a natively-1M model
(`deepseek-v4-flash`, `mimo-v2.6-flash`). The package's snapshot therefore carries the **native** 262,144
rather than the served 1M, and opencode matches it: a declared window the model degrades inside is worse than a
smaller honest one, because nothing downstream can tell a degraded answer from a good one.
`glm5.3-flash`'s 1M is native per Zhipu.

The three paid OpenRouter models (`deepseek-v4-pro`, `qwen3-coder-plus`, `minimax-m3`) were
dropped from the picker: a paid model one keystroke away in a model list is a cost you take by
accident, not by decision. Frontier work goes through `ai/opencode/opencode.jsonc`, where reaching
for OpenRouter is an explicit act.

pi's picker also omits the rate-limited `:free` OpenRouter tier that `opencode.jsonc` still lists —
the two sets are curated independently. `tests/pi-config.bats` asserts this list stays equal to
`settings.json`'s `enabledModels`.

Default: `nan/deepseek-v4-flash`, thinking level `high`. Change in `settings.json`. (Per-model
context windows come from `pi-nan-provider`'s snapshot; `tests/pi-nan-package.bats` holds
opencode's to the same numbers.)

## Secret

`NAN_API_KEY` lives only in Bitwarden (item `nan-api-key`, mapped in
`secrets/registry.yaml`, ADR-028), and reaches pi only through the `pi` wrapper's
`dotf secrets run`. The literal never appears in a committed file. Rotate it at the
NaN dashboard if ever exposed.
