---
id: audit-003-docs-drift
type: audit
status: active
created: "2026-05-19"
---

# AUDIT-003 — Cross-Agent Docs Drift

> Pointer-style consistency audit across the 6 cross-agent docs + README. Verifies the AI-013 refactor (pointer-style ≤70 lines, no behavioural-rule duplication) is still intact post-AI-019. Generated 2026-05-19.

## TL;DR

**5/6 pointer-style files OK. 1 drift detected: `.github/copilot-instructions.md` is out of sync with `ai/copilot/copilot-instructions.md` after the AI-019 model-tier addition** (the AI-019 PR added a Model Tier section to `ai/copilot/` but not to `.github/copilot/`, plus minor pointer-phrase wording variations). No behavioural rules are duplicated outside `AGENTS.md` — the canonical SSOT is honoured.

**`.claude/CLAUDE.md` (184 lines)** is intentionally project-specific dotfiles content (shell compatibility table, verification commands, file map). NOT subject to the pointer-style budget; correctly placed as the layer-3 project-specific overlay per `ai/claude/CLAUDE.md` "Project Memory Hierarchy".

## File-by-file

| File | LOC | Pointer phrase | Behavioural rules duplicated? | Status |
|---|---:|:---:|:---:|---|
| `AGENTS.md` | 429 | N/A (canonical) | N/A | OK — canonical SSOT |
| `ai/claude/CLAUDE.md` | 78 | ✓ | None | OK (recently bumped 70→80 in AI-019, justified) |
| `ai/gemini/GEMINI.md` | 34 | ✓ | None | OK |
| `ai/copilot/copilot-instructions.md` | 39 | ✓ | None | OK (canonical for Copilot v2 user-global) |
| `.github/copilot-instructions.md` | 33 | ✓ (linked form `[`AGENTS.md`](../AGENTS.md)`) | None | **DRIFT** — out of sync with `ai/copilot/`; AI-019 Model Tier missing |
| `.claude/CLAUDE.md` | 184 | N/A | None | OK — project-specific overlay (not pointer-style by design) |
| `README.md` | 160 | N/A | None | OK — project README (not in cross-agent scope) |

## Drift detail: `.github/copilot-instructions.md` vs `ai/copilot/copilot-instructions.md`

Diff:

```diff
- > **First, read `AGENTS.md` at the repo root** — canonical SSOT…
+ > **First, read [`AGENTS.md`](../AGENTS.md) at the repo root** — canonical SSOT…

- > If `AGENTS.md` is missing from the current repo, default to the canonical version at …
+ > If `AGENTS.md` is missing, default to the canonical version at …

  ## Quick Reference (paths)
  …
- ## Model Tier (per AGENTS.md "Model Selection")
- - **Top / Mid / Low:** TBD — concrete model identifiers pending AI-017 / AI-018 audits …
- When AI-017/AI-018 close, replace this block with the literal model IDs.
+ (MISSING)
```

Three drift types:

1. **Cosmetic pointer-phrase variation** — `.github/` uses markdown-link form; `ai/copilot/` uses plain backticked path. Both convey "First, read AGENTS.md" but a parity grep test only sees one form. Harmless but invisible to automated drift detection without a more careful matcher.
2. **Cosmetic wording variation** — `.github/` says "If AGENTS.md is missing,..."; `ai/copilot/` says "...is missing from the current repo,...". Same meaning, different bytes.
3. **Content drift** — `.github/` is missing the Model Tier subsection added in AI-019. **This is the load-bearing drift**: GitHub Copilot reading the repo will not learn the model-tier rule.

### Why this happened

AI-013 made both files pointer-style with similar content but never established a sync mechanism. AI-019 (model-tier policy) added the Model Tier subsection to `ai/copilot/` but failed to also edit `.github/copilot-instructions.md`. The two files are deployed to different targets but contain ~identical Copilot guidance, so they always need to be kept in sync — yet nothing in CI catches drift.

This is exactly the failure mode AUDIT-003 was designed to catch. BUG-001 + BUG-002 (verify-string drift) were a sibling class — content removed from one file by AI-013 but referenced in setup script grep assertions. Same pattern, different angle.

### Recommended fix (one PR)

**`SDD-005-github-copilot-instructions-sync`** (P1):

- Step 1: re-sync `.github/copilot-instructions.md` content with `ai/copilot/copilot-instructions.md`. Decide single canonical wording for the pointer phrase + missing-AGENTS-fallback line. Add the Model Tier section.
- Step 2: add a **bats parity test** in `tests/agents-md.bats` (or new `tests/docs-drift.bats`): asserts both copilot files have the same section headers + Model Tier mapping. CI fails if a future edit drifts them.
- Step 3 (optional): document in `AGENTS.md` § Spec-Driven Development or in a new pattern that "both copilot files must move together; CI enforces it" — so future contributors know.

**Anti-scope**: do NOT attempt to generate `.github/copilot-instructions.md` from `ai/copilot/copilot-instructions.md` at build time. GitHub Copilot reads the file directly from the checkout; build-time generation adds tooling complexity for marginal benefit over a bats parity assertion.

## Observations (not action items)

- **The pointer-style convention works**, but only when contributors remember to apply it. AI-013 succeeded; AI-019 partially failed. The fix is enforcement (bats parity test), not convention rewrite.
- **`ai/claude/CLAUDE.md` 78 lines after AI-019** is the only ≤80 file. Future per-agent extensions need to add to AGENTS.md (canonical) and to the agent's overlay; the overlay budget is now de facto 80 lines for Claude, 50 for Gemini (test enforces).
- **`.claude/CLAUDE.md` 184 lines is fine.** It serves a different role (project-specific overlay) and contains content that legitimately doesn't fit in `ai/claude/CLAUDE.md` (which is user-global). The Project Memory Hierarchy section in `ai/claude/CLAUDE.md` documents this 4-tier system.
- **README.md 160 lines is fine.** It's the project README, not in cross-agent scope. AUDIT-001 confirmed it as `vault-index` lifecycle.

## Sequenced PR list

| # | PR | Diff | Risk | Notes |
|:---:|---|---:|---|---|
| 1 | `SDD-005-github-copilot-instructions-sync` | ~30 LOC sync + ~20 LOC bats parity test | Low | Specs it; the parity test prevents future drift. |

## Closing

- [ ] SDD-005 opened.
- [ ] Tick AUDIT-003 in the project task backlog with this report's path + finding link.

## References

- [AUDIT-001](audit-001-repo-structure.md), [AUDIT-002](audit-002-cross-os-duplication.md) — sibling audits this cycle.
- [AUDIT-004](dotfiles-architecture-map.md) — locates the 6 docs in the deploy graph.
- AI-013 commit folded into PR [#34](https://github.com/mlorentedev/dotfiles/pull/34) (pointer-style refactor).
- AI-019 PR [#60](https://github.com/mlorentedev/dotfiles/pull/60) (the partial-sync regression source).
- Sibling class: BUG-001 ([#40](https://github.com/mlorentedev/dotfiles/pull/40)) + BUG-002 ([#47](https://github.com/mlorentedev/dotfiles/pull/47)) — verify-string drift after AI-013.
