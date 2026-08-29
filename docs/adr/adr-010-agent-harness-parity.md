---
id: adr-010-agent-harness-parity
type: adr
status: active
created: "2026-05-17"
---

# ADR-010: Cross-Agent Harness Parity Strategy

## Status: Proposed

## Date: 2026-05-17

## Context

ADR-009 (2026-05-15) made `AGENTS.md` the canonical system prompt across Claude Code, OpenCode, Gemini, Copilot, Cursor, and Codex. That fixed **prompt drift** but only addresses one of four parity surfaces. Three others remain:

1. **Commands / skills** — Claude Code has `~/.claude/skills/<name>/SKILL.md` auto-loaded via slash commands. OpenCode has `~/.config/opencode/commands/<name>.md` (closed by AI-012, PR #43). Gemini CLI has `~/.gemini/prompts/<name>.md`. Copilot CLI has none directly comparable. **Same conceptual primitive, three different filesystem layouts and discovery mechanisms.**

2. **Memory** — Claude Code has the `claude-mem` MCP for cross-session conversation memory (Anthropic-specific, plugin-managed). OpenCode has nothing equivalent built in; Hive vault MCP fills the role partially (it captures lessons, not raw transcripts). Gemini/Copilot have no memory layer at all. **No portable substrate. Either everyone gets a memory MCP or memory stays Claude-only.**

3. **Sub-agents** — Claude Code supports `subagent_type` in its Task tool for delegated work. OpenCode has the `opencode agent` subcommand and "Plan vs Build" agent modes. Gemini and Copilot have no analog. **Conceptually similar in Claude and OpenCode, completely absent elsewhere.**

4. **Hooks** — Claude Code has `SessionStart` / `PreToolUse` / `PostToolUse` / `Stop` hooks configured via `settings.json`. OpenCode has a `plugin` subcommand for npm-distributed plugins (different mechanism — pre/post-instrumentation via JavaScript, not declarative bash hooks). Gemini and Copilot expose no hook surface at the runtime level. **Different mechanisms, may not be portable in any clean way.**

Empirically these gaps caused friction in the AI-011-validation session (2026-05-17):

- The user could not run their `crystallize` workflow in OpenCode because it depends on `claude-mem` (gap #2 manifested).
- The user could not delegate work to a sub-agent for parallel exploration in OpenCode without re-invoking the TUI manually (gap #3 manifested -- "agent=explore mode=subagent" exists in OpenCode but the trigger semantics differ from Claude's Task tool).
- The user wanted `~/.claude/skills/<X>/SKILL.md` triggers to work in OpenCode -- closed by AI-012 (PR #43, gap #1 resolved for the Claude -> OpenCode direction). Gemini/Copilot directions still open.

This ADR defines the strategy for closing these gaps without forcing brittle adapters, and explicitly marks where parity is not worth pursuing.

## Decision

Adopt a **per-primitive parity matrix** that drives discrete micro-specs, rather than a single mega-spec.

### Parity matrix

For each primitive, the matrix has one of four states per agent: ✅ portable, ⚠ partial / research-needed, ❌ not portable / Claude-only, ➖ not applicable / agent unsupported.

| Primitive | Claude Code | OpenCode | Gemini CLI | Copilot CLI | Cursor | Codex |
|---|---|---|---|---|---|---|
| **System prompt (AGENTS.md)** | ✅ via pointer file (`~/.claude/CLAUDE.md`) | ✅ native | ✅ via pointer (`~/.gemini/GEMINI.md`) | ✅ via pointer (`copilot-instructions.md`) | ✅ native | ✅ native |
| **Commands / skills** | ✅ `~/.claude/skills/*` | ✅ `~/.config/opencode/commands/*` (AI-012) | ⚠ `~/.gemini/prompts/*` — audit needed | ❌ no command system | ⚠ unknown | ⚠ unknown |
| **MCP servers** | ✅ `settings.json` MCP block | ✅ `opencode.jsonc` MCP block (mirrors mcp-servers.json) | ➖ no MCP support in Gemini CLI | ➖ no MCP support in Copilot CLI | ✅ MCP-aware | ⚠ unknown |
| **Sub-agents** | ✅ Task tool with `subagent_type` | ⚠ `opencode agent` + agent=build/explore/plan modes (different trigger semantics) | ❌ not supported | ❌ not supported | ⚠ unknown | ⚠ unknown |
| **Memory layer** | ✅ `claude-mem` MCP (Anthropic-specific) | ⚠ Hive vault MCP partially (lessons, not transcripts) | ❌ no memory | ❌ no memory | ⚠ unknown | ⚠ unknown |
| **Hooks (SessionStart, PreToolUse, etc.)** | ✅ `settings.json` hooks | ⚠ `opencode plugin` (npm-distributed JS, different paradigm) | ✅ `~/.gemini/settings.json` hooks (BeforeAgent/BeforeTool; measured 2026-08-26, `harness/manifest.json` `bind_comment`) | ✅ `~/.copilot/hooks/*.json` (SessionStart/PreToolUse, deny by exit 2 or `permissionDecision`; measured 2026-08-29 on CLI 1.0.81 — ADR-027 amendment) | ⚠ unknown | ⚠ unknown |
| **Status line** | ✅ `/statusline-setup` | ⚠ partial (TUI footer, not user-customisable) | ❌ no TUI | ❌ no TUI | ⚠ unknown | ⚠ unknown |
| **Slash commands UX** | ✅ `/<name>` discovery | ✅ `/<name>` discovery | ⚠ different format | ❌ no slash commands | ⚠ unknown | ⚠ unknown |

### Strategy per gap

**Gap 1 -- Commands / skills (Claude <-> OpenCode <-> Gemini)**

- Claude -> OpenCode: closed by AI-012 (PR #43) with the `scripts/skills-to-opencode.sh` transform (since retired — the deterministic transform now lives in the `harness/skills` `targets[]` render path in setup; ADR-021).
- Claude -> Gemini: open. Gemini prompts already exist at `~/.gemini/prompts/` (audited as 15+ files in healthcheck). A follow-up spec `AI-016-skills-to-gemini-prompts` would write the analogous `scripts/skills-to-gemini.sh` transform. Lower priority than #2 and #3 because Gemini CLI usage is sporadic in the user's workflow.
- Claude / OpenCode -> Copilot: not pursuing. Copilot CLI is meant for shell command suggestions, not multi-step agent work. The pointer file (`copilot-instructions.md`) already covers the system-prompt side; commands have no target.

**Gap 2 -- Memory layer (cross-agent durable memory)**

- Strategy: **Hive vault as the cross-agent memory substrate**, not a port of claude-mem.
- Rationale: claude-mem records raw conversation; Hive captures crystallised lessons. The latter is what survives sessions and what other agents would want to query anyway. claude-mem stays Claude-only by design.
- Action: ensure OpenCode + Claude both have Hive MCP registered (already true via `mcp-servers.json`). Document the `query Hive from any agent` pattern in `AGENTS.md` (already done in AI-013).
- Gap remaining: Gemini and Copilot do not support MCP, so they cannot query Hive at runtime. Their workaround is reading `00-context.md` / `11-tasks.md` / `90-lessons.md` files directly from the filesystem. Adequate but lower fidelity than MCP query.
- No spec needed unless empirically a cross-agent memory need surfaces that current Hive-as-substrate does not cover.

**Gap 3 -- Sub-agents**

- OpenCode's `agent=build/explore/plan` modes are conceptually similar to Claude's `subagent_type` but trigger differently (Claude is tool-invoked, OpenCode is mode-switched).
- Strategy: document the equivalence in `AGENTS.md` and the OpenCode runbook. When a workflow benefits from delegation, both agents have a path. The user picks the appropriate idiom per agent.
- Empirical caveat: during AI-011-validation, OpenCode's `agent=explore mode=subagent` was the stall point during the after-tool-call slowness (runbook entry "Stream stalls after first chunk -- variant after tool call"). Sub-agent reliability depends on provider stream stability, not on the harness.
- No spec needed unless we see a workflow that would clearly benefit from a unified API.

**Gap 4 -- Hooks**

- Strategy: **do not port hooks across agents.** They are too tied to runtime internals to abstract cleanly.
- Each agent gets agent-specific hooks where useful. Claude Code has `SessionStart` registered via `setup-linux.sh` (vault health context injection). OpenCode does not currently need an equivalent because the SessionStart-style context is already delivered via `AGENTS.md` (which OpenCode reads at session start) plus the cwd-scoped `hive` MCP query the user runs explicitly.
- If OpenCode-specific instrumentation becomes useful (auto-log session start to vault, auto-load tasks at session start), write a plugin in OpenCode's npm-distributed plugin format. Out of scope for now.
- Gemini / Copilot: not pursuing.

### Implementation order

Closing the matrix is incremental, not a sprint. PR order:

1. **AI-012 (✅ done, PR #43)** -- Claude skills -> OpenCode commands.
2. **AI-016 (proposed, not yet scaffolded)** -- audit Gemini prompts vs Claude skills, write `scripts/skills-to-gemini.sh` if a divergence is found. Lower priority. **Trigger:** when the user actually starts using Gemini CLI for non-trivial work.
3. **No further specs scheduled.** Memory (gap 2), Sub-agents (gap 3), Hooks (gap 4) are documented decisions to NOT port; revisit only if empirical pain surfaces.

### What we explicitly will NOT do

- Maintain a `harness-parity.sh` mega-script that papers over the gaps. The gaps are real and per-primitive; hiding them is worse than acknowledging them.
- Re-implement claude-mem inside OpenCode. Hive vault is the chosen substrate; claude-mem stays Claude-only and that is fine.
- Force Gemini / Copilot into multi-step agent patterns. They are good at the things they are good at; pushing them beyond their model is a tax with no payoff.

## Consequences

**Positive**

- Each parity gap has a documented status (✅ / ⚠ / ❌ / ➖) and a known follow-up (or explicit non-action). No ambiguity about "is this supposed to work".
- New specs (AI-016, possible TERM-003+) are scoped tight to one primitive each. No mega-PRs.
- The matrix is auditable: any agent the user adopts in the future (Cursor, Codex, future entrants) can be slotted in and its gaps surfaced.
- Hive vault gets reaffirmed as the cross-agent memory substrate without needing a new system.

**Negative**

- Users coming from Claude Code to OpenCode will find some primitives missing (memory transcripts, certain sub-agent patterns). The matrix documents this but does not solve it.
- Maintenance of the matrix itself: every time a runtime adds a primitive (e.g., OpenCode adds a hook system), the matrix needs an update.
- Some gaps (sub-agent semantics) are inherently fuzzy; the equivalence-by-documentation approach in this ADR is best-effort, not formal.

**Neutral**

- This ADR codifies what was already implicit. The user's mental model has been "Claude Code + OpenCode are sibling tools with different strengths" -- the ADR makes that legible.
- AI-015 umbrella in `11-tasks.md` is effectively this ADR's scope; the umbrella entry can now point at the ADR as its design surface.

## References

- ADR-009: Multi-agent runtime + AGENTS.md as SSOT
- AI-011-opencode-bootstrap (archived spec) + runbook [`../runbooks/guide-opencode-go-setup.md`](../runbooks/guide-opencode-go-setup.md)
- AI-012-opencode-commands-port (PR #43 -- gap 1 Claude->OpenCode resolved)
- TERM-001-ghostty-bootstrap (PR #45 -- closes the Ghostty surface for the AI-015 audit)
- agents.md standard: <https://agents.md>
- OpenCode docs: <https://opencode.ai/docs/>
- Claude Code docs: <https://docs.claude.com/en/docs/claude-code/>
