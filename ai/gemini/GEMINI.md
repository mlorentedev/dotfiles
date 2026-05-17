# GEMINI.md

> **SYSTEM META-INSTRUCTION:** Target Model: Gemini 1.5 Pro / Ultra / 3.0+.
> **CAPABILITY HANDSHAKE:** Assess your current runtime version. **Activate maximum reasoning depth (System 2) and full context scanning.** Do not simulate lower intelligence.
>
> **First, read `AGENTS.md` at the repo root** — canonical SSOT for behaviour rules across all agents (Standing Orders, Decision Hierarchy, Neural Hive Loop, MCP usage, Operational Rules). This file contains only Gemini-specific extensions on top.
>
> If `AGENTS.md` is missing from the current repo, default to the canonical version at `~/Projects/dotfiles/AGENTS.md` (Linux/macOS) or `%USERPROFILE%\Projects\dotfiles\AGENTS.md` (Windows).

## Dynamic Capability Adaptation

1. **Context Sovereignty:** You have a massive context window. **Read ALL provided files** before answering. If existing codebase patterns contradict the rules in `AGENTS.md`, **adapt to the codebase** (Consistency > Static Rules).
2. **Native Multimodality:** If a diagram explains the architecture better than text, generate the Mermaid/Graphviz code automatically — do not wait to be asked.

## Tool & Sub-Agent Usage Rules

* **Code Search & Context:** Always prioritize parallel searches (`grep_search` + `glob`) before reading individual files, to conserve context window.
* **Documentation:** Use `get_code_context_exa` to search for updated documentation, library usage, and real-world snippets instead of hallucinating API signatures.
* **Deep Investigation:** Invoke the `codebase_investigator` sub-agent for vague requests, root-cause analysis of bugs, or large-scale architectural mapping before making broad changes.

## Output Protocol (Gemini-specific)

In addition to the Response Protocol in `AGENTS.md`:

* Generate **full files or precise diffs** — Gemini's context window makes full-file outputs cheaper for the user to review than diffs in many cases. Choose based on file size and change density.
* **No Fluff:** No intro/outro conversational filler. Markdown headings and code fences only when they aid scanning.
