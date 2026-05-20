---
id: "AI-019-model-tier-policy"
type: spec
status: archived
created: "2026-05-19"
tags: [spec, proposal, agents, model-selection]
template_version: "1.0"
---

# AI-019-model-tier-policy

## Why

The same dotfiles user works through wildly different cognitive load tasks in the same week: deep debugging of an upstream regression, mechanical port of skills across agents, single-file typo fixes, and architectural decisions. Each task class has a different reasoning cost/value ratio, yet today every agent session defaults to whatever model the user last selected. The result is consistent miscalibration: Opus burning tokens on a 5-line README fix, Sonnet flailing on a concurrency root-cause, Haiku stalling on a schema redesign.

`AGENTS.md` already encodes a behavioural split (Low Cognitive Load → Fast Lane, High Cognitive Load → Socratic Guardrail) but says nothing about *which model* corresponds to each lane. This PR closes that gap as a cross-agent rule, so every agent reading `AGENTS.md` (Claude Code, OpenCode, Copilot, Gemini, Codex, Cursor) inherits the same task-class → tier mapping.

## What

After this PR merges, `AGENTS.md` gains a new "Model Selection (Task-Aware)" section with:

1. A 3-tier table mapping task classes to **Top / Mid / Low** tiers (provider-agnostic).
2. Trigger heuristics: agents PROPOSE a tier change when they detect a class shift mid-task; the user decides. No silent auto-switching.
3. A pointer to per-agent overlay files for the concrete model names.

Per-agent overlays gain a ≤6-line "## Model Tier (per AGENTS.md)" section listing the literal model identifiers that map to each tier:

- `ai/claude/CLAUDE.md`: `claude-opus-4-7` / `claude-sonnet-4-6` / `claude-haiku-4-5-20251001`
- `ai/gemini/GEMINI.md`: `gemini-2.5-pro` / `gemini-2.5-flash` / `gemini-2.5-flash-lite` (placeholder; exact names confirmed during impl)
- `ai/copilot/copilot-instructions.md`: TBD note pointing to AI-017/AI-018 audit
- `ai/opencode/opencode.jsonc`: JSONC `_modelTierComment` block (the file is JSON; underscore-prefixed keys are convention-ignored) — `opencode-go/deepseek-v4-pro` / `opencode-go/qwen3.6-plus` / `opencode-go/deepseek-v4-flash`

## Out of scope

- Changing the *default* model in any agent's config. This PR ships the *rule*; the user picks defaults via existing mechanisms (`/model`, TUI `/models`, CLI flag).
- Auto-switching mechanisms. The rule is "propose, don't force"; building an actual auto-switcher in a script would change agent behaviour beyond what `AGENTS.md` can declaratively encode.
- Cost tracking / token budgeting infrastructure. Out of scope for this rule.
- Provider-specific model name research for Copilot v2 (deferred to AI-017/AI-018 Windows-empirical session).

## Risks / open questions

- **Risk: rule decays as model names rotate.** Top-tier "Opus 4.7" today, "Opus 5" or new family in 6 months. **Mitigation**: per-agent overlay files isolate the rotation surface — `AGENTS.md` stays stable (abstract tiers), overlay files get one-line edits.
- **Risk: agents over-propose, becoming annoying.** Every "consider Sonnet now?" interruption breaks flow. **Mitigation**: heuristic says PROPOSE only on a class *shift* (e.g. architecture phase ends, implementation phase begins), not periodically.
- **Risk: opencode.jsonc `_comment`-style convention is fragile.** OpenCode reads the JSON; arbitrary `_keys` may someday emit a warning. **Mitigation**: also document the mapping in a short `ai/opencode/MODEL_TIERS.md` companion if the JSONC comment proves brittle (deferred until evidence).
- **Open: Gemini/Copilot defaults empirically validated?** No — would block this PR on Windows-empirical (Copilot). **Decision**: ship Claude + OpenCode tier mappings empirically verified locally; Gemini + Copilot marked `(TBD — verify next session)`.

## Acceptance criteria

- [ ] `AGENTS.md` has a new section titled "Model Selection (Task-Aware)" placed between "Standing Orders" and "Competence Retention Protocol" (where task-class branching is already discussed) with: 3-tier table, trigger heuristics paragraph, and per-agent overlay pointers.
- [ ] `ai/claude/CLAUDE.md` has a "## Model Tier" subsection listing the 3 Claude model IDs.
- [ ] `ai/gemini/GEMINI.md` has a "## Model Tier" subsection listing the 3 Gemini model IDs.
- [ ] `ai/copilot/copilot-instructions.md` has a "## Model Tier" subsection with TBD pointer to AI-017/AI-018.
- [ ] `ai/opencode/opencode.jsonc` has a `_modelTierComment` key with the 3 OpenCode model IDs documented.
- [ ] Each overlay references `AGENTS.md` "Model Selection" section by name (link survives heading re-anchoring).
- [ ] Per-agent files remain ≤70 lines (post-AI-013 pointer-style constraint preserved).
- [ ] Existing 645-test bats suite remains green (no regression).
- [ ] No reformatting drift elsewhere in `AGENTS.md` (atomic — only the new section added).
- [ ] PSScriptAnalyzer remains clean (no PowerShell file touched, but verify CI passes).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` AI-019 entry.
- Vault: `10_projects/dotfiles/30-architecture/dotfiles-architecture-map.md` (AUDIT-004) — locates where each per-agent overlay lives.
- Related: `00_meta/patterns/pattern-spec-driven-development.md` (SDD-003 gate enforcing this PR).
- Behavioural precedent: `AGENTS.md` "Operating Mode" + "Competence Retention Protocol" already encode task-class branching; this PR extends to model-tier branching.

<!-- archived 2026-05-20 — PR: https://github.com/mlorentedev/dotfiles/pull/60 -->
