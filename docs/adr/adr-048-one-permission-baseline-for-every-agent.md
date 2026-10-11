---
id: "ADR-048-one-permission-baseline-for-every-agent"
type: adr
status: accepted
owner: manu
date: "2026-10-09"
supersedes: []
extends: [adr-010-agent-harness-parity]
issue: mlorentedev/dotfiles#1852
tags: [architecture, decision, harness, permissions, security, agents]
created: "2026-10-09"
---

# ADR-048: Every agent shares one trust and deny baseline; allow lists stay per agent, declared in one place

## Context

Every harness gets its permissions from a different file, in a different shape. Nothing compares them. An
inventory of the repo and of the Mac on 2026-10-09 (Copilot CLI 1.0.81) found two opposite policies and four gaps:

| Harness | Allow | Deny | Trust | Approval |
|---|---|---|---|---|
| Claude Code | 35 narrow `Bash(...)`, MCP and web entries | 17 (secrets, force push, `reset --hard`, sudo, `env`, `curl\|sh`, `rm -rf`) | none shipped | prompt (default) |
| agy | `command(*)`, `read_file(*)`, `write_file(*)`, `read_url(*)`, `search_web(*)` (dropped on 2026-10-10, see below) | 2 (`rm -rf /`, `rm -rf ~/*`) | `~/Projects/*`, `~/Projects/Workspace/*` | `always-proceed` |
| Copilot CLI | none set; command allow is per-session `--allow-tool` only, URL allow (`allowedUrls`) persists in `config.json` | none set; command deny is per-session `--deny-tool`, URL deny (`deniedUrls`) persists | `~/Projects`, `~/Projects/*`, `~/Projects/Workspace{,/*}` | manual (default); `defaultPermissionMode` persists but is not set |
| opencode | none at session level | none (only the `plan` agent denies edit and bash) | none | built-in default |
| pi | no permission model: it runs every tool without asking | none | none | none |
| Codex | not managed, not installed (unused since 2026-08-21) | none | none | none |

Two facts constrain any fix:

- **The deploy can only add.** `deepMerge` unions lists (`unionLists`, `cli/internal/deploy/deploy.go`), so a
  machine keeps every entry it already has, and an entry removed from a template stays on every machine that
  received it. That is what protects grants a machine added by hand (the incident behind AI-043). It also
  means a baseline can be widened by a deploy but never narrowed.
- **Not every harness can express the same thing.** Copilot persists URL allow and deny and its approval mode,
  but not command allow or deny, and pi has
  no permission model. A single allow list rendered into every harness would be fiction for two of them.

The owner asked for a homogeneous minimum that lets every agent work. Agy's grants, which an earlier draft of
#2236 removed by mistake, are part of that minimum (owner, 2026-10-09).

## Decision

1. **Common to every harness that can express it:**
   - **Trust roots:** `{HOME}/Projects/*` and `{HOME}/Projects/Workspace/*`. A machine may trust more paths; it
     never trusts fewer.
   - **Deny intents**, named once and rendered into each harness's own syntax: destructive filesystem removal,
     force push and history rewrite, secret exposure (`dotf secrets show`, `env`, `printenv`), pipe-to-shell
     installs, privilege escalation (`sudo`), and the cloud metadata endpoints (`169.254.169.254`,
     `metadata.google.internal`), which stay denied whatever an allow list later grants (#1852).
   - **An approval mode declared for every harness**, even where the value is the harness default, so that a
     mode nobody chose is visible in review.
2. **Allow lists stay per harness,** because their syntax and granularity differ (Claude's `Bash(go test:*)`
   against agy's `command(*)`). They are declared next to the common block in one file, so a change to any
   agent's reach is one diff in one place. This decision widens no harness: Claude keeps its narrow list, and
   agy keeps its baseline.
3. **The single source is `harness/capability-map.json`,** extended with a session-level section beside the
   persona tool grants it already holds. That follows the model-map pattern: a declared map, a renderer per
   leaf, and a guard that fails when a leaf disagrees with the map. The leaves are the existing deploy sources
   (`ai/claude/settings.json`, `ai/agy/settings.json`, `ai/copilot/config.json`, `ai/opencode/opencode.jsonc`).
4. **Gaps are declared, not left silent.** pi has no permission model and Codex is out of scope; the map
   records each as a decision. For Copilot, URL denies (the metadata endpoints) and the approval mode go in
   `config.json`, which persists them; command denies go through the per-session `--deny-tool` flags and a
   wrapper, as #1852 already plans. This ADR widens that ticket to every harness.
5. **The baseline only ratchets up.** Narrowing a grant that machines already hold needs an explicit removal
   list in the map, and the deploy has no such mechanism today. Until one exists, a narrowing is an owner action
   on each machine, recorded in the PR that makes it.
6. **The owner lands every change to a grant.** The agent prepares the diff and the tests; the owner applies
   the settings change and merges, as for any change that widens an agent's authority.

## Consequences

- **agy keeps `permissions.allow` and `trustedWorkspaces`** in `ai/agy/settings.json`. #2236 restores them and
  pins them with `TestAgyTemplateShipsTheBaselineGrants`.
- **Claude gains the two trust roots.** `tests/claude-settings-template.bats` forbids `additionalDirectories`
  today; the renderer PR flips that test on purpose.
- **opencode is the hard case.** The setup scripts replace its file whole, because `{env:VAR}` references are
  rendered by `dotf secrets render`. Moving it to a merge deploy must first prove that the merge and the
  secret render can coexist. Until then a machine-local opencode grant is lost on every setup. That move also
  removes shell from the setup scripts, as the never-grow rule requires.
- **Claude has two merge implementations** (the Go deploy, and `merge_claude_settings` in the setup scripts,
  "Divergence 18"). The renderer targets the Go path; the shell twin is a strangler-fig deletion.
- **agy strips `search_web(*)` when it rewrites its settings,** so doctor reported permanent drift on the Mac.
  Measured on 2026-10-10 on macOS with agy 1.3.3 (#2312): agy ran a web search without asking, and its grant
  list did not change. On that build web search needs no grant, and the kinds agy writes and keeps are
  `command`, `read_file`, `write_file`, `read_url` and `mcp` (it had saved `mcp(hive-vault/vault_list)` itself).
  The template no longer ships `search_web(*)`, and a test fails on any other kind. Linux and Windows were not
  measured; if agy prompts for web search there, the grant it writes names the kind to add.
- ADR-010's parity matrix gains a *Permissions* row.

## Alternatives rejected

- **Every agent at agy's level** (full shell and write, no prompts). That widens Claude's reach sharply, and
  Copilot and pi could still not express it.
- **Every agent at Claude's level** (narrow allow list, prompt by default). The deploy cannot narrow agy on
  machines that already hold `command(*)`, and pi and Copilot cannot express it either.
- **A new SSOT file.** `capability-map.json` already maps grants across harnesses for personas. A second map
  would split one question between two files.
