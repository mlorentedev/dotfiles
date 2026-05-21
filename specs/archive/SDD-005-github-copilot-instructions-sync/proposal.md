---
id: "SDD-005-github-copilot-instructions-sync"
type: spec
status: draft
created: "2026-05-19"
tags: [spec, proposal, copilot, docs-drift]
template_version: "1.0"
---

# SDD-005-github-copilot-instructions-sync

## Why

AUDIT-003 detected real drift between `ai/copilot/copilot-instructions.md` (deployed user-global to `~/.copilot/`) and `.github/copilot-instructions.md` (repo-local, read by GitHub Copilot for this repo). PR #60 (AI-019) added a "Model Tier" subsection to the first file but missed the second. As a result, GitHub Copilot operating inside this dotfiles repo currently does NOT see the model-tier policy that every other agent reads — the cross-agent rule is partially broken.

Same drift class as BUG-001 (#40) and BUG-002 (#47): files that must move together silently diverge when an editor touches only one. CI has no enforcement, so the next AI-019-style edit will likely re-introduce the same drift.

## What

After this PR merges:

1. `.github/copilot-instructions.md` contains the same load-bearing content as `ai/copilot/copilot-instructions.md`. Specifically: same H2 section headers (Role & Goal, Execution Preferences, Interaction Style, Quick Reference, Model Tier), same Model Tier wording, same vault paths.
2. **Both files are byte-equivalent in content**, except for the pointer banner (`> First, read AGENTS.md...`). The `.github/` version preserves its markdown link form `[`AGENTS.md`](../AGENTS.md)` because GitHub renders it as a clickable link; the `ai/copilot/` version stays plain because it deploys to `~/.copilot/` where the relative link would not resolve.
3. A new bats parity test in `tests/docs-drift.bats` asserts: stripping the leading `>` blockquote lines from both files yields byte-identical output. CI fails on future drift.

## Out of scope

- Build-time generation of `.github/copilot-instructions.md` from `ai/copilot/...` (a pre-commit hook or `setup-linux.sh` step). Bats parity check is sufficient enforcement; generation adds complexity without proportional benefit.
- Reformatting `ai/copilot/copilot-instructions.md` (the canonical-ish source). The drift fix flows from canonical to drifted, not the other way.
- Auditing other cross-file sync relationships (e.g. setup-linux.sh ↔ setup-windows.ps1 verification strings). Those have their own bats coverage; not in scope here.

## Risks / open questions

- **Risk: future contributors edit only one file again.** Mitigation is the bats parity test — CI fails loud, no merge.
- **Risk: legitimate future divergence** (e.g. a GitHub-repo-specific guideline that does NOT belong in user-global Copilot config). Then the parity test becomes incorrect. **Decision rule**: if divergence is intentional, the spec for that change must update the parity test simultaneously. The test is enforcement, not absolute truth.
- **Open**: which file is "canonical"? **Decision**: `ai/copilot/copilot-instructions.md` is canonical for content; `.github/` is the deploy mirror with one allowed difference (pointer link form). This matches the existing flow (setup-windows + setup-linux deploy `ai/copilot/` → `~/.copilot/`, but `.github/` is repo-local and not deployed by setup).
- **Risk: pointer-banner stripping is fragile.** A future contributor might add a non-blockquote pointer line that the bats `sed '/^> /d'` filter would miss. Mitigation: use `## ` headers as anchors and compare section-by-section instead of stripping. **Final approach**: compare both files with the pointer banner block stripped — simpler and tied to the actual difference.

## Acceptance criteria

- [ ] `.github/copilot-instructions.md` contains a `## Model Tier (per AGENTS.md "Model Selection")` section with the same Top/Mid/Low TBD wording as `ai/copilot/copilot-instructions.md`.
- [ ] `.github/copilot-instructions.md` pointer-banner fallback line matches `ai/copilot/copilot-instructions.md` wording ("If `AGENTS.md` is missing **from the current repo**, default to..."). Only the markdown-link form of the AGENTS.md reference may differ.
- [ ] New file `tests/docs-drift.bats` exists with at least one parity test that fails when content (excluding the pointer banner block) drifts.
- [ ] Bats parity test passes on the synced state.
- [ ] Full existing bats suite remains green (no regression). Target: 645 + new cases.
- [ ] No edits to `ai/copilot/copilot-instructions.md` (it is canonical for this PR — the drift is one-directional).

## References

- Vault: `10_projects/dotfiles/11-tasks.md` SDD-005 entry.
- Vault: [[audit-003-docs-drift]] — the audit that detected this drift.
- Sibling drift class: BUG-001 PR [#40](https://github.com/mlorentedev/dotfiles/pull/40), BUG-002 PR [#47](https://github.com/mlorentedev/dotfiles/pull/47).
- Source of the partial sync: PR [#60](https://github.com/mlorentedev/dotfiles/pull/60) (AI-019 added Model Tier to `ai/copilot/` only).
