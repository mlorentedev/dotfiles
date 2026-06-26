---
id: "HARNESS-043-curator-dogfood-slice"
type: spec
status: draft # draft | implementing | verifying | archived
created: "2026-06-25"
issue: "mlorentedev/dotfiles#559"   # repo#NNN — GitHub issue / Project item that tracks this spec
tags: [spec, proposal]
template_version: "1.0"
---

# HARNESS-043-curator-dogfood-slice

> **Naming**: file lives at `<repo>/specs/HARNESS-043-curator-dogfood-slice/proposal.md`. `HARNESS-043-curator-dogfood-slice` is `AREA-NNN-slug`.

## Why

<!-- from issue #559: HARNESS-043: Dogfood slice — curator agent end-to-end on claude -->

ADR-027 (the cross-harness agent pipeline) is accepted, but zero invocable agent definitions exist and skill consumption is still probabilistic ("the model remembers sometimes"). The novel, unproven part of the whole epic (#558) is **deterministic consumption** — a forced skill that is present by code, not by memory. Rather than build the full 3-level × 5-harness matrix blind, this slice proves the thinnest end-to-end path: one persona (`curator`), one harness (claude), presence-level determinism only. If this slice lands cleanly, the fan-out (H-044..048) is mechanical; if it fights us, we learned it cheaply.

## What

After this PR:

1. The vault holds the first neutral agent definition — `00_meta/agents/definitions/curator/AGENT.md` (the ADR's named dogfood), `targets: [claude, opencode, pi, copilot]` — and `compile-harness.sh` grows an `agents` target that renders it to Claude's native agent path (`~/.claude/agents/curator.md`) with provenance, exactly as the `skills` target renders skills today.
2. `curator`'s forced skills (`skills:` in its definition) are **deterministically present in every harness you use daily** — claude, opencode, pi, copilot — by **uniform marked-region injection** into each harness's always-loaded instructions file (`~/.claude/CLAUDE.md`, `~/.config/opencode/AGENTS.md`, `~/.pi/agent/AGENTS.md`, `~/.copilot/copilot-instructions.md`). One mechanism, four harnesses; agy excluded (no daily use, presence primitive undocumented).
3. The new engine behavior is covered by `tests/compile-harness.bats`.

## Decision (resolved): presence = uniform marked-region injection

The first build emitted a Claude `SessionStart` hook into `~/.claude/settings.json`. Two problems killed that approach: it is **claude-only** (opencode/pi/copilot have no such hook), and it carries an **OS axis** — the emitted shell command is POSIX, so it never runs on Windows without a separate native-command port. Both are exactly the silent-drift / claude-shortcut failures ADR-027 exists to prevent.

The empirical finding that resolved it: **every harness in daily use already loads an instructions file that the harness pipeline already manages** — `~/.claude/CLAUDE.md`, `~/.config/opencode/AGENTS.md`, `~/.pi/agent/AGENTS.md` (all three carry a `<!-- BEGIN HARNESS GENERATED -->` patterns region today), and copilot's `~/.copilot/copilot-instructions.md` (the skill-catalog target). So **presence ≡ injecting the forced-skills directive into that always-loaded file** — one uniform mechanism, identical across all four harnesses, reusing the engine's marked-region primitive.

Insight that reframes the pattern: a plugin that only *adds text to context* (`chat.system.transform`, `session_start`) is **equivalent** to instructions-injection — both put the directive in context every turn. The plugin primitives earn their place only when they **gate** (Action level) or **activate per-agent** (Invocation/dispatch). So the provider plugins are not the *Presence* level — they are the *Action* level, deferred to **H-045**. Presence is the cheapest, most agnostic primitive: text in an always-loaded file.

**Mechanism.** At `--deploy`, the engine builds a per-harness persona block (each persona that `targets` this harness + its forced skills) and injects it into a **distinct marker namespace** — `<!-- BEGIN HARNESS AGENT-PRESENCE … -->` — that coexists with, and never disturbs, the patterns region. The injection is idempotent (replace-in-place if present, append if absent), and a target file that does not exist is skipped. No shell command, no `settings.json`, no Go, no OS axis: pure text injection, cross-OS by nature.

**Caveat (accepted):** injection is **global per session** — every opencode/pi session sees curator's directive, not only when curator is the active persona (same semantics the old SessionStart hook had). Fine for a single persona; the roster fan-out (H-046) will need per-agent activation, which is precisely where the Action/dispatch plugins come in.

## Out of scope

- **Action-level gate** (`PreToolUse` / `tool.execute.before` / pi `tool_call`, exit 2) and **Invocation dispatch** (slash-router / command) → **H-045**. This slice reads #559's determinism as *Presence* (forced into context by code) — the cheapest level. The plugin primitives that gate or activate per-agent are Action/dispatch.
- **Per-harness agent render beyond claude** — the launchable agent *file* for opencode (`agent-md`), copilot (`catalog`), pi (adapter), plus the agy `agent-yaml` transpose → H-045/H-046/H-047. Presence and render are separable: presence forces the skills into context; render makes the persona launchable by name. This slice does presence on four harnesses, render on claude only.
- **agy** entirely — not in daily use; its presence primitive is undocumented (research) → out.
- Full `capability-map.json` / `model-map.json` → H-044. The render drops `model`/`capabilities` uniformly across harnesses for now (agnostic, just unrefined).
- Roster fan-out (architect/planner/builder/reviewer/shipper) + `hermes-nan` catalog entry → H-046; per-agent presence activation lands there too.
- Library consolidation (patterns→skills) → deferred to a working `curator` (knowledge#125).

## Risks / open questions

- **Global injection** — presence is forced into every session of each harness, not gated on the active persona (accepted; per-agent activation → H-046). Correct for a single persona; watch context bloat as the roster grows.
- **Marker coexistence** — the `AGENT-PRESENCE` region shares a file with the patterns `GENERATED` region. Mitigated by a distinct marker namespace + a bats test asserting the patterns region (and user content) survive injection; verified idempotent across re-deploys.
- **Live-session evidence (Windows)** — acceptance is met by sandbox-`$HOME` deploy proof + the bats suite; touching the live `~/.claude/CLAUDE.md` etc. is deferred (no live mutation while the user works dotfiles in parallel). Injection is text-only and cross-OS, so there is no OS-axis command form to capture.

## Acceptance criteria

- [x] `00_meta/agents/definitions/curator/AGENT.md` exists, neutral, `targets: [claude, opencode, pi, copilot]`, valid against `harness/agent-frontmatter.schema.json`.
- [x] `harness/manifest.json` `agents` block has a `presence[]` list (claude/opencode/pi/copilot → instructions file); `compile-harness.sh --refresh/--deploy/--check` render `curator` to `~/.claude/agents/curator.md` with `generated_*` provenance.
- [x] `curator`'s forced skills are injected as an `AGENT-PRESENCE` marked region into each harness's always-loaded instructions file at `--deploy`, idempotently, without disturbing the patterns region.
- [x] Engine `--check` is clean (record renders, schema validates) — offline, no vault access.
- [x] `tests/compile-harness.bats` covers render + injection across the four harnesses (region present, idempotent, patterns region intact, per-harness `targets[]` respected).

## References

- ADR: `dotfiles/docs/adr/adr-027-cross-harness-agent-pipeline.md` (§1 render target, §3 determinism, Implementation gate steps 1+2+4)
- Pattern: `00_meta/patterns/pattern-cross-agent-agent-pipeline.md`; roster: `00_meta/agents/ROSTER.md` (curator row); doctrine: `00_meta/agents/doctrine.md`
- Engine to clone: `scripts/compile-harness.sh` (`render_skill`, `do_deploy`, `skill_out_path`), `harness/manifest.json` (`skills.deploy[]`), `harness/skill-frontmatter.schema.json`, `tests/compile-harness.bats`
- CLI-025 seam: `cli/internal/cmd/mem.go` (`runClaudeHook`), `cli/internal/mem/session_start_adapter.go` (`ClaudeContext` injectors), `session-start-config.json`
- Epic: #558 · this slice: #559
