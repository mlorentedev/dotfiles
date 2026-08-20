# AGY.md

> **SYSTEM META-INSTRUCTION:** Target tool: Google Antigravity CLI (`agy`). Target model: see `settings.json` (`modelConfigs.defaultModel`, currently `gemini-3.7-flash`).
> **CAPABILITY HANDSHAKE:** Activate maximum reasoning depth and full context scanning. Do not simulate lower intelligence than the active model supports.
>
> **First, read `AGENTS.md` at the repo root** — canonical SSOT for behaviour rules across all agents (Standing Orders, Decision Hierarchy, Neural Hive Loop, MCP usage, Operational Rules). This file contains only agy-specific extensions on top.
>
> If `AGENTS.md` is missing from the current repo, default to the canonical version at `$DOTFILES_REPO_DIR/AGENTS.md` (resolved via `machine.json` per ADR-025; falls back to `~/Projects/Workspace/dotfiles/AGENTS.md`).

## Dynamic Capability Adaptation

1. **Context Sovereignty:** You have a massive context window (`maxContextTokens: 2000000` in `settings.json`). **Read ALL provided files** before answering. If existing codebase patterns contradict the rules in `AGENTS.md`, **adapt to the codebase** (Consistency > Static Rules).
2. **Native Multimodality:** If a diagram explains the architecture better than text, generate the Mermaid/Graphviz code automatically — do not wait to be asked.

## Tool & Sub-Agent Usage Rules

* **Code Search & Context:** Always prioritize parallel searches (`grep_search` + `glob`) before reading individual files, to conserve context window.
* **Documentation:** Use `get_code_context_exa` to search for updated documentation, library usage, and real-world snippets instead of hallucinating API signatures.
* **Deep Investigation:** Invoke the `codebase_investigator` sub-agent for vague requests, root-cause analysis of bugs, or large-scale architectural mapping before making broad changes.

## Output Protocol (agy-specific)

In addition to the Response Protocol in `AGENTS.md`:

* Generate **full files or precise diffs** — agy's context window makes full-file outputs cheaper for the user to review than diffs in many cases. Choose based on file size and change density.
* **No Fluff:** No intro/outro conversational filler. Markdown headings and code fences only when they aid scanning.

## Model Tier (per AGENTS.md "Model Selection")

The active default lives in `ai/agy/settings.json` (`modelConfigs.defaultModel`). Single source of truth for model selection — update there, not here.

- **Default:** `gemini-3.7-flash` — daily driver (see `settings.json`).
- **High-reasoning alias:** `chat-base` profile (`thinkingConfig.thinkingLevel: HIGH`) for hard debug / architecture / root-cause.

Selection: configure via `agy` UI or per-prompt model override. Model IDs reflect the Antigravity CLI runtime; verify availability in `agy` itself if the listed model is rejected.
