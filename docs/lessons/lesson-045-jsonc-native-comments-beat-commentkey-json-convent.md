---
id: lesson-045-jsonc-native-comments-beat-commentkey-json-convent
type: lesson
status: active
created: "2026-05-19"
owner: manu
tags: [lesson, dotfiles]
---

# Lesson 045: JSONC native // comments beat _commentKey JSON convention for documentation

**Context:** AI-019 needed to document the model-tier mapping inside ai/opencode/opencode.jsonc. The proposal weighed two conventions: (a) a `_modelTierComment` JSON key with underscore prefix (convention says parsers ignore it); (b) native JSONC `//` line comments. OpenCode reads the file as JSONC (the file already used `//` for the schema URL + 6 other comment blocks).</context>
<problem>Documenting structured config inside JSON has no native syntax — comments aren't part of the JSON spec. Many projects invent the `_commentKey` convention (`"_comment": "..."`, `"_doc": "..."`). It works in practice because consumers ignore unknown keys, but it has two downsides: (1) it pollutes the parsed JSON namespace, so any tooling that enumerates keys sees noise; (2) underscore-key convention is unofficial — a future JSON schema validator might reject it. JSONC (JSON with Comments) is a different file format where `//` and `/* */` are first-class syntax.</problem>
<parameter name="solution">When the file is `.jsonc` (or any consumer accepts JSONC), prefer native `//` comments over `_commentKey` JSON keys. Rationale: (a) comments stay out of the parsed key namespace; (b) diff-friendly (line comments don't bracket-wrap content); (c) consistent with existing convention in that file. Example from ai/opencode/opencode.jsonc head:

```jsonc
// OpenCode configuration — managed by dotfiles.
// SSOT: ai/opencode/opencode.jsonc (this file).
// Schema: https://opencode.ai/config.json
//
// Model Tier (per AGENTS.md "Model Selection"):
//   Top: opencode-go/deepseek-v4-pro
//   Mid: opencode-go/qwen3.6-plus
//   Low: opencode-go/deepseek-v4-flash
{
  "$schema": "https://opencode.ai/config.json",
  ...
}
```

Risk mitigation: if the consumer ever rejects line comments (unlikely for any tool that accepts the .jsonc extension), fall back to a sibling Markdown file (e.g. ai/opencode/MODEL_TIERS.md) — explicit documentation file, no convention games inside the data.

Inverse: if the file is pure .json (no JSONC support), use a top-of-file `_comment: ` key as a least-bad option. The pollution is real but acceptable when the alternative is no documentation at all.</parameter>
<parameter name="tags">["json", "jsonc", "config", "documentation"]</parameter>
</invoke>
**Problem:** 
**Solution:**
